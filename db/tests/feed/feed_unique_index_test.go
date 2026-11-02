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
	feedIndexMigration = "1791713196_add_feed_actor_type_item_unique_index.go"
	feedIndexName      = "idx_feed_actor_type_item"
)

// newFeedMigrationTestApp builds the schema in a temporary database by applying
// every migration that sorts before stopBefore, or every migration when
// stopBefore is empty, skipping only the migrations that configure the
// external Meilisearch indexes.
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
		if stopBefore != "" && migration.File >= stopBefore {
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

func feedIndexMigrationUp(t *testing.T) func(core.App) error {
	t.Helper()
	for _, candidate := range core.AppMigrations.Items() {
		if candidate.File == feedIndexMigration {
			return candidate.Up
		}
	}
	t.Fatalf("migration %s is not registered", feedIndexMigration)
	return nil
}

// newFeedActor stores a remote person actor to own feed entries.
func newFeedActor(t *testing.T, app core.App) string {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(collection)
	actor.Set("iri", "https://remote.example/api/v1/activitypub/user/feedowner")
	actor.Set("username", "feedowner")
	actor.Set("preferred_username", "feedowner")
	actor.Set("domain", "remote.example")
	actor.Set("inbox", "https://remote.example/api/v1/activitypub/user/feedowner/inbox")
	actor.Set("public_key", "-----BEGIN PUBLIC KEY-----\nfeed test\n-----END PUBLIC KEY-----\n")
	if err := app.Save(actor); err != nil {
		t.Fatalf("save actor: %v", err)
	}
	return actor.Id
}

func feedRowCount(t *testing.T, app core.App, actor string, feedType util.FeedType, item string) int64 {
	t.Helper()
	n, err := app.CountRecords("feed", dbx.HashExp{"actor": actor, "type": string(feedType), "item": item})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Record ids are unique only within a collection, so a feed entry is keyed by
// actor, type and item: a trail and a list with the same id are both listed.
func TestFeedIndexKeysOnActorTypeAndItem(t *testing.T) {
	app := newFeedMigrationTestApp(t, "")

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
	if !slices.Equal(columns, []string{"actor", "type", "item"}) {
		t.Fatalf("index columns = %v, want [actor type item]", columns)
	}

	actor, item := newFeedActor(t, app), "sharedid0000001"
	trailRow, err := util.InsertIntoFeed(app, actor, "", item, util.TrailFeed)
	if err != nil {
		t.Fatalf("insert trail entry: %v", err)
	}
	if _, err := util.InsertIntoFeed(app, actor, "", item, util.ListFeed); err != nil {
		t.Fatalf("insert list entry with the same id: %v", err)
	}
	if feedRowCount(t, app, actor, util.TrailFeed, item) != 1 || feedRowCount(t, app, actor, util.ListFeed, item) != 1 {
		t.Fatal("want one trail and one list entry for the same id")
	}

	again, err := util.InsertIntoFeed(app, actor, "", item, util.TrailFeed)
	if err != nil || again.Id != trailRow.Id {
		t.Fatalf("repeated insert returned %v, %v; want the existing trail entry %s", again, err, trailRow.Id)
	}
	duplicate := core.NewRecord(collection)
	duplicate.Set("actor", actor)
	duplicate.Set("type", string(util.TrailFeed))
	duplicate.Set("item", item)
	if err := app.Save(duplicate); err == nil {
		t.Fatal("saving a duplicate (actor, type, item) row succeeded, want a unique violation")
	}
}

// A database that ran a development build of instance federation has an
// (actor, item) index; the migration replaces it.
func TestFeedIndexReplacesActorItemIndex(t *testing.T) {
	app := newFeedMigrationTestApp(t, feedIndexMigration)

	collection, err := app.FindCollectionByNameOrId("pbc_72164123")
	if err != nil {
		t.Fatal(err)
	}
	collection.AddIndex("idx_feed_actor_item", true, "`actor`, `item`", "")
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}

	if err := app.RunInTransaction(feedIndexMigrationUp(t)); err != nil {
		t.Fatalf("run migration: %v", err)
	}

	collection, err = app.FindCollectionByNameOrId("pbc_72164123")
	if err != nil {
		t.Fatal(err)
	}
	if collection.GetIndex("idx_feed_actor_item") != "" || collection.GetIndex(feedIndexName) == "" {
		t.Fatalf("indexes = %v, want only %s", collection.Indexes, feedIndexName)
	}
	actor := newFeedActor(t, app)
	if _, err := util.InsertIntoFeed(app, actor, "", "sharedid0000001", util.TrailFeed); err != nil {
		t.Fatal(err)
	}
	if _, err := util.InsertIntoFeed(app, actor, "", "sharedid0000001", util.ListFeed); err != nil {
		t.Fatalf("a list with a trail's id is refused after the migration: %v", err)
	}

	if err := app.RunInTransaction(feedIndexMigrationUp(t)); err != nil {
		t.Fatalf("second run of the migration: %v", err)
	}
}
