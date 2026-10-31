package feed_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	_ "pocketbase/migrations"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/dbutils"
)

const (
	feedIndexMigration = "1790260000_add_feed_actor_item_unique_index.go"
	feedIndexName      = "idx_feed_actor_item"
)

// newFeedMigrationTestApp builds the schema in a temporary database by applying
// every migration that sorts before stopBefore, skipping only the migrations
// that configure the external Meilisearch indexes.
func newFeedMigrationTestApp(t *testing.T, stopBefore string) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].File < migrations[j].File
	})
	for _, migration := range migrations {
		if migration.File >= stopBefore {
			break
		}
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}

	return app
}

func insertRawFeedRow(t *testing.T, app core.App, id, actor, item, created string) {
	t.Helper()
	_, err := app.DB().Insert("feed", dbx.Params{
		"id":      id,
		"actor":   actor,
		"item":    item,
		"type":    "trail",
		"created": created,
		"updated": created,
	}).Execute()
	if err != nil {
		t.Fatalf("insert feed row %s: %v", id, err)
	}
}

func feedRowIds(t *testing.T, app core.App, actor, item string) []string {
	t.Helper()
	records, err := app.FindAllRecords("feed", dbx.HashExp{"actor": actor, "item": item})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.Id)
	}
	return ids
}

func TestFeedUniqueIndexMigrationDedupsExistingRows(t *testing.T) {
	app := newFeedMigrationTestApp(t, feedIndexMigration)

	insertRawFeedRow(t, app, "feedrowa0000003", "actorA", "itemA", "2026-01-03 00:00:00.000Z")
	insertRawFeedRow(t, app, "feedrowa0000001", "actorA", "itemA", "2026-01-01 00:00:00.000Z")
	insertRawFeedRow(t, app, "feedrowa0000002", "actorA", "itemA", "2026-01-02 00:00:00.000Z")
	insertRawFeedRow(t, app, "feedrowb0000001", "actorB", "itemB", "2026-01-01 00:00:00.000Z")

	var migration *core.Migration
	for _, candidate := range core.AppMigrations.Items() {
		if candidate.File == feedIndexMigration {
			migration = candidate
		}
	}
	if migration == nil {
		t.Fatalf("migration %s is not registered", feedIndexMigration)
	}

	if err := app.RunInTransaction(migration.Up); err != nil {
		t.Fatalf("run migration: %v", err)
	}

	if ids := feedRowIds(t, app, "actorA", "itemA"); len(ids) != 1 || ids[0] != "feedrowa0000001" {
		t.Fatalf("pair A rows = %v, want only the earliest row feedrowa0000001", ids)
	}
	if ids := feedRowIds(t, app, "actorB", "itemB"); len(ids) != 1 || ids[0] != "feedrowb0000001" {
		t.Fatalf("pair B rows = %v, want feedrowb0000001", ids)
	}

	collection, err := app.FindCollectionByNameOrId("pbc_72164123")
	if err != nil {
		t.Fatal(err)
	}
	parsed := dbutils.ParseIndex(collection.GetIndex(feedIndexName))
	if !parsed.Unique {
		t.Fatalf("index %s missing or not unique: %q", feedIndexName, collection.GetIndex(feedIndexName))
	}
	columns := make([]string, 0, len(parsed.Columns))
	for _, c := range parsed.Columns {
		columns = append(columns, strings.ToLower(c.Name))
	}
	if !slices.Equal(columns, []string{"actor", "item"}) {
		t.Fatalf("index columns = %v, want [actor item]", columns)
	}

	duplicate := core.NewRecord(collection)
	duplicate.Set("actor", "actorB")
	duplicate.Set("item", "itemB")
	duplicate.Set("type", "trail")
	if err := app.Save(duplicate); err == nil {
		t.Fatal("saving a duplicate (actor, item) row succeeded, want a unique violation")
	}

	if err := app.RunInTransaction(migration.Up); err != nil {
		t.Fatalf("second run of the migration: %v", err)
	}

	existing, err := util.InsertIntoFeed(app, "actorB", "", "itemB", util.TrailFeed)
	if err != nil {
		t.Fatalf("InsertIntoFeed on the migrated schema: %v", err)
	}
	if existing.Id != "feedrowb0000001" {
		t.Fatalf("InsertIntoFeed returned %q, want the existing row feedrowb0000001", existing.Id)
	}
}
