package feed_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	_ "pocketbase/migrations"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// correlatedFeedDedupSQL is a per-row correlated dedup statement, the
// reference for which rows survive: of each
// (actor, item) pair the row with the smallest created, then the smallest id.
const correlatedFeedDedupSQL = `
			DELETE FROM feed
			WHERE EXISTS (
				SELECT 1 FROM feed AS other
				WHERE other.actor = feed.actor
				  AND other.item = feed.item
				  AND (
					other.created < feed.created
					OR (other.created = feed.created AND other.id < feed.id)
				  )
			)
		`

// errRollback ends a RunInTransaction callback without committing.
var errRollback = errors.New("rollback on purpose")

func findFeedIndexMigration(t *testing.T) *core.Migration {
	t.Helper()
	for _, candidate := range core.AppMigrations.Items() {
		if candidate.File == feedIndexMigration {
			return candidate
		}
	}
	t.Fatalf("migration %s is not registered", feedIndexMigration)
	return nil
}

type feedSeedStats struct {
	rows      int
	pairs     int
	tieGroups int
	// keep holds the ids of the earliest row of every pair (smallest created,
	// then smallest id), sorted, computed from the generated data alone.
	keep []string
}

// seedFeedRows inserts rows raw feed rows from a deterministic generator:
// every (actor, item) pair gets a duplicate count from a skewed mix, created
// is a random millisecond of 2025, and one in ten duplicates reuses the
// created value of an earlier row of its pair so that same-millisecond ties
// exercise the id tie break.
func seedFeedRows(t *testing.T, app core.App, rows int, seed uint64) feedSeedStats {
	t.Helper()

	type seedRow struct{ id, actor, item, created string }

	const idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	const yearMillis = 365 * 24 * 60 * 60 * 1000
	rng := rand.New(rand.NewPCG(seed, 0))
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	usedIDs := make(map[string]struct{}, rows)
	newID := func() string {
		for {
			b := make([]byte, 15)
			for i := range b {
				b[i] = idAlphabet[rng.IntN(len(idAlphabet))]
			}
			id := string(b)
			if _, taken := usedIDs[id]; !taken {
				usedIDs[id] = struct{}{}
				return id
			}
		}
	}
	randomCreated := func() string {
		at := start.Add(time.Duration(rng.Int64N(yearMillis)) * time.Millisecond)
		return at.Format("2006-01-02 15:04:05.000Z")
	}

	stats := feedSeedStats{rows: rows}
	all := make([]seedRow, 0, rows)
	for pair := 0; len(all) < rows; pair++ {
		actor := fmt.Sprintf("actor%05d", pair%3000)
		item := fmt.Sprintf("item%07d", pair/3000)

		var count int
		switch draw := rng.IntN(100); {
		case draw < 70:
			count = 1
		case draw < 90:
			count = 2 + rng.IntN(4)
		case draw < 99:
			count = 6 + rng.IntN(25)
		default:
			count = 100 + rng.IntN(301)
		}
		count = min(count, rows-len(all))

		group := make([]seedRow, 0, count)
		for i := 0; i < count; i++ {
			created := randomCreated()
			if i > 0 && rng.IntN(10) == 0 {
				created = group[rng.IntN(len(group))].created
			}
			group = append(group, seedRow{id: newID(), actor: actor, item: item, created: created})
		}

		earliest := group[0]
		for _, r := range group[1:] {
			if r.created < earliest.created || (r.created == earliest.created && r.id < earliest.id) {
				earliest = r
			}
		}
		atEarliest := 0
		for _, r := range group {
			if r.created == earliest.created {
				atEarliest++
			}
		}
		if atEarliest >= 2 {
			stats.tieGroups++
		}
		stats.pairs++
		stats.keep = append(stats.keep, earliest.id)
		all = append(all, group...)
	}
	sort.Strings(stats.keep)

	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	const batchSize = 500
	err := app.RunInTransaction(func(tx core.App) error {
		for from := 0; from < len(all); from += batchSize {
			batch := all[from:min(from+batchSize, len(all))]
			params := dbx.Params{}
			values := make([]string, 0, len(batch))
			for i, r := range batch {
				params[fmt.Sprintf("id%d", i)] = r.id
				params[fmt.Sprintf("actor%d", i)] = r.actor
				params[fmt.Sprintf("item%d", i)] = r.item
				params[fmt.Sprintf("created%d", i)] = r.created
				params[fmt.Sprintf("updated%d", i)] = r.created
				values = append(values, fmt.Sprintf("({:id%d}, {:actor%d}, {:item%d}, 'trail', {:created%d}, {:updated%d})", i, i, i, i, i))
			}
			query := "INSERT INTO feed (id, actor, item, type, created, updated) VALUES " + strings.Join(values, ", ")
			if _, err := tx.DB().NewQuery(query).Bind(params).Execute(); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed %d feed rows: %v", rows, err)
	}

	return stats
}

func feedRowCount(t *testing.T, app core.App) int {
	t.Helper()
	var n int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM feed").Row(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func feedSurvivorIDs(t *testing.T, app core.App) []string {
	t.Helper()
	var ids []string
	if err := app.DB().NewQuery("SELECT id FROM feed ORDER BY id").Column(&ids); err != nil {
		t.Fatal(err)
	}
	return ids
}

// oracleSurvivorIDs runs the correlated reference statement in a transaction
// that is rolled back, and returns the ids it leaves behind.
func oracleSurvivorIDs(t *testing.T, app core.App) []string {
	t.Helper()
	before := feedRowCount(t, app)

	var ids []string
	err := app.RunInTransaction(func(tx core.App) error {
		if _, err := tx.DB().NewQuery(correlatedFeedDedupSQL).Execute(); err != nil {
			return err
		}
		ids = feedSurvivorIDs(t, tx)
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("oracle run: %v", err)
	}
	if after := feedRowCount(t, app); after != before {
		t.Fatalf("feed has %d rows after the rolled-back oracle run, want %d", after, before)
	}
	return ids
}

// captureFeedDedupSQL runs the migration's Up in a rolled-back transaction and
// returns the one DELETE FROM feed statement it executed. The statement is
// recorded through the dbx ExecLogFunc hook, so the plan under test is the SQL
// the migration really sends, not a copy of it.
func captureFeedDedupSQL(t *testing.T, app *core.BaseApp) string {
	t.Helper()

	var mu sync.Mutex
	var statements []string
	record := func(_ context.Context, _ time.Duration, query string, _ sql.Result, _ error) {
		mu.Lock()
		defer mu.Unlock()
		statements = append(statements, query)
	}

	dbs := []*dbx.DB{app.NonconcurrentDB().(*dbx.DB)}
	if concurrent := app.ConcurrentDB().(*dbx.DB); concurrent != dbs[0] {
		dbs = append(dbs, concurrent)
	}
	previous := make([]dbx.ExecLogFunc, len(dbs))
	for i, db := range dbs {
		previous[i] = db.ExecLogFunc
		db.ExecLogFunc = record
	}
	restore := func() {
		for i, db := range dbs {
			db.ExecLogFunc = previous[i]
		}
	}
	t.Cleanup(restore)

	migration := findFeedIndexMigration(t)
	err := app.RunInTransaction(func(tx core.App) error {
		if err := migration.Up(tx); err != nil {
			return err
		}
		return errRollback
	})
	restore()
	if !errors.Is(err, errRollback) {
		t.Fatalf("capture run of %s: %v", feedIndexMigration, err)
	}

	mu.Lock()
	defer mu.Unlock()
	var deletes []string
	for _, statement := range statements {
		if strings.Contains(strings.ToUpper(strings.Join(strings.Fields(statement), " ")), "DELETE FROM FEED") {
			deletes = append(deletes, statement)
		}
	}
	if len(deletes) != 1 {
		t.Fatalf("captured %d DELETE FROM feed statements, want exactly 1: %q", len(deletes), deletes)
	}

	collection, err := app.FindCollectionByNameOrId("pbc_72164123")
	if err != nil {
		t.Fatal(err)
	}
	if collection.GetIndex(feedIndexName) != "" {
		t.Fatalf("index %s is still on the feed collection after the rolled-back migration run", feedIndexName)
	}

	return deletes[0]
}

type queryPlanRow struct {
	ID      int    `db:"id"`
	Parent  int    `db:"parent"`
	NotUsed int    `db:"notused"`
	Detail  string `db:"detail"`
}

func TestFeedDedupMigrationKeepsTheSameRowsAsTheCorrelatedDelete(t *testing.T) {
	assertIndexPresent := func(t *testing.T, app core.App) {
		t.Helper()
		collection, err := app.FindCollectionByNameOrId("pbc_72164123")
		if err != nil {
			t.Fatal(err)
		}
		if collection.GetIndex(feedIndexName) == "" {
			t.Fatalf("index %s was not added", feedIndexName)
		}
	}

	t.Run("empty", func(t *testing.T) {
		app := newFeedMigrationTestApp(t, feedIndexMigration)

		if oracle := oracleSurvivorIDs(t, app); len(oracle) != 0 {
			t.Fatalf("oracle kept %d rows of an empty table", len(oracle))
		}
		if err := app.RunInTransaction(findFeedIndexMigration(t).Up); err != nil {
			t.Fatalf("run migration on an empty feed: %v", err)
		}
		if got := feedSurvivorIDs(t, app); len(got) != 0 {
			t.Fatalf("migration left %d rows in an empty table", len(got))
		}
		assertIndexPresent(t, app)
	})

	t.Run("seeded-3000", func(t *testing.T) {
		app := newFeedMigrationTestApp(t, feedIndexMigration)
		stats := seedFeedRows(t, app, 3000, 42)
		t.Logf("seeded %d rows, %d pairs, %d tie groups", stats.rows, stats.pairs, stats.tieGroups)
		if stats.tieGroups == 0 {
			t.Fatal("seed produced no same-millisecond tie group at the earliest created value")
		}
		if got := feedRowCount(t, app); got != stats.rows {
			t.Fatalf("feed has %d rows, want %d", got, stats.rows)
		}

		oracle := oracleSurvivorIDs(t, app)
		if len(oracle) != stats.pairs {
			t.Fatalf("oracle kept %d rows, want one per pair (%d)", len(oracle), stats.pairs)
		}

		if err := app.RunInTransaction(findFeedIndexMigration(t).Up); err != nil {
			t.Fatalf("run migration: %v", err)
		}
		got := feedSurvivorIDs(t, app)
		if len(got) != stats.pairs {
			t.Fatalf("migration kept %d rows, want one per pair (%d)", len(got), stats.pairs)
		}
		if !slices.Equal(got, oracle) {
			t.Fatal("migration kept different ids than the correlated statement")
		}
		if !slices.Equal(got, stats.keep) {
			t.Fatal("migration kept different ids than the earliest row of each pair computed from the seed")
		}
		assertIndexPresent(t, app)
	})
}

func TestFeedDedupMigrationPlanHasNoCorrelatedScan(t *testing.T) {
	app := newFeedMigrationTestApp(t, feedIndexMigration)
	statement := captureFeedDedupSQL(t, app)

	var plan []queryPlanRow
	if err := app.DB().NewQuery("EXPLAIN QUERY PLAN " + statement).All(&plan); err != nil {
		t.Fatalf("explain: %v", err)
	}
	t.Logf("captured statement: %s", strings.Join(strings.Fields(statement), " "))

	var correlated, listSubquery bool
	for _, row := range plan {
		t.Logf("plan: id=%d parent=%d %s", row.ID, row.Parent, row.Detail)
		if strings.Contains(row.Detail, "CORRELATED") {
			correlated = true
		}
		if strings.Contains(row.Detail, "LIST SUBQUERY") {
			listSubquery = true
		}
	}
	if len(plan) == 0 {
		t.Fatal("EXPLAIN QUERY PLAN returned no rows")
	}
	if correlated {
		t.Error("query plan contains a CORRELATED subquery step")
	}
	// The plan text is not a stable SQLite interface and may be reworded when
	// PocketBase bumps its SQLite. Only the absence of a correlated step fails
	// the test; the scaling tests and the row-equivalence test are the gates
	// for the behaviour itself.
	if !listSubquery {
		t.Log("query plan has no LIST SUBQUERY step; check whether SQLite reworded its plan text")
	}
}
