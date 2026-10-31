package feed_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// timeFeedDedup runs the dedup statement three times, each in a transaction
// that is rolled back, and returns the fastest run. The minimum of three damps
// scheduler noise on shared CI runners.
func timeFeedDedup(t *testing.T, app core.App, statement string, rows int) time.Duration {
	t.Helper()

	var fastest time.Duration
	for run := 0; run < 3; run++ {
		var elapsed time.Duration
		err := app.RunInTransaction(func(tx core.App) error {
			start := time.Now()
			result, err := tx.DB().NewQuery(statement).Execute()
			elapsed = time.Since(start)
			if err != nil {
				return err
			}
			deleted, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if deleted <= 0 {
				return fmt.Errorf("deleted %d rows, want at least one", deleted)
			}
			return errRollback
		})
		if !errors.Is(err, errRollback) {
			t.Fatalf("dedup run %d: %v", run, err)
		}
		if got := feedRowCount(t, app); got != rows {
			t.Fatalf("feed has %d rows after rolled-back run %d, want %d", got, run, rows)
		}
		if run == 0 || elapsed < fastest {
			fastest = elapsed
		}
	}
	return fastest
}

func TestFeedDedupMigrationScalesNearLinearly(t *testing.T) {
	const smallRows, largeRows = 20_000, 80_000
	// The 20k baseline is only about 140 ms, so one scheduler stall can push a
	// single measurement past the bound. A quadratic plan exceeds it on every
	// attempt; noise does not.
	const attempts = 3

	type seeded struct {
		app       core.App
		statement string
		rows      int
	}
	seed := func(rows int) seeded {
		app := newFeedMigrationTestApp(t, feedIndexMigration)
		statement := captureFeedDedupSQL(t, app)
		stats := seedFeedRows(t, app, rows, 42)
		if got := feedRowCount(t, app); got != rows {
			t.Fatalf("seeded %d rows, want %d", got, rows)
		}
		t.Logf("seeded %d rows (%d pairs)", rows, stats.pairs)
		return seeded{app, statement, rows}
	}

	small, large := seed(smallRows), seed(largeRows)

	var ratio float64
	for attempt := 1; attempt <= attempts; attempt++ {
		smallTime := timeFeedDedup(t, small.app, small.statement, small.rows)
		largeTime := timeFeedDedup(t, large.app, large.statement, large.rows)
		ratio = float64(largeTime) / float64(smallTime)
		t.Logf("attempt %d: %d rows %v, %d rows %v, ratio %.2f", attempt, smallRows, smallTime, largeRows, largeTime, ratio)
		// A quadratic plan gives about 16 for four times the rows, a near-linear
		// one about 4 to 5.
		if ratio < 8 {
			return
		}
	}
	t.Fatalf("80k/20k runtime ratio is %.2f on each of %d attempts, want below 8", ratio, attempts)
}

// The correlated reference statement is quadratic: it
// needs about 14 s for 20,000 rows where the window statement needs 140 ms.
// Timing both on the same data in the same process cancels the speed of the
// machine, so this holds on a slow runner too.
func TestFeedDedupMigrationBeatsTheCorrelatedStatement(t *testing.T) {
	const rows = 5_000
	// Measured at about 27 on 5,000 rows.
	const minimumFactor = 5

	app := newFeedMigrationTestApp(t, feedIndexMigration)
	statement := captureFeedDedupSQL(t, app)
	seedFeedRows(t, app, rows, 42)

	window := timeFeedDedup(t, app, statement, rows)
	correlated := timeFeedDedup(t, app, correlatedFeedDedupSQL, rows)
	factor := float64(correlated) / float64(window)
	t.Logf("%d rows: window statement %v, correlated statement %v, factor %.1f", rows, window, correlated, factor)
	if factor < minimumFactor {
		t.Fatalf("window statement is only %.1f times faster than the correlated one, want at least %d", factor, minimumFactor)
	}
}

func TestFeedDedupMigration200kRowsWithinCeiling(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds 200,000 rows; skipped under -short")
	}

	const rows = 200_000
	// About 2.6 to 4 s on a development machine. Extrapolated from its
	// quadratic growth, the correlated statement would need over 20 minutes, so
	// the ceiling leaves room for a slow runner and still catches any quadratic
	// plan.
	const ceiling = 30 * time.Second

	app := newFeedMigrationTestApp(t, feedIndexMigration)
	stats := seedFeedRows(t, app, rows, 42)
	if got := feedRowCount(t, app); got != rows {
		t.Fatalf("seeded %d rows, want %d", got, rows)
	}

	migration := findFeedIndexMigration(t)
	start := time.Now()
	err := app.RunInTransaction(migration.Up)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("run migration: %v", err)
	}

	survivors := feedRowCount(t, app)
	t.Logf("migration Up on %d rows took %v (ceiling %v), %d survivors, %d pairs", rows, elapsed, ceiling, survivors, stats.pairs)
	if survivors != stats.pairs {
		t.Fatalf("migration kept %d rows, want one per pair (%d)", survivors, stats.pairs)
	}
	if elapsed >= ceiling {
		t.Fatalf("migration took %v on %d rows, want under %v", elapsed, rows, ceiling)
	}

	collection, err := app.FindCollectionByNameOrId("pbc_72164123")
	if err != nil {
		t.Fatal(err)
	}
	if collection.GetIndex(feedIndexName) == "" {
		t.Fatalf("index %s was not added", feedIndexName)
	}
}
