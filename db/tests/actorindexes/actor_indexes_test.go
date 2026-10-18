package actorindexes_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/pocketbase/core"
)

const (
	userIndexMigration  = "1776767721_updated_activitypub_actors.go"
	inboxIndexMigration = "1778583029_updated_activitypub_actors.go"
)

// The delete-recipient queries join activitypub_actors on inbox; without that
// index SQLite scans every actor for every activity. Both migrations above
// overwrite the whole index list, so the outcome depended on the order an
// instance happened to apply them in.
func TestActorIndexesSurviveEitherMigrationOrder(t *testing.T) {
	orders := map[string]func([]*core.Migration) []*core.Migration{
		"fresh install": func(migrations []*core.Migration) []*core.Migration {
			return migrations
		},
		"upgraded before the user index landed": func(migrations []*core.Migration) []*core.Migration {
			userIndex := slices.IndexFunc(migrations, func(m *core.Migration) bool { return m.File == userIndexMigration })
			moved := migrations[userIndex]
			migrations = slices.Delete(migrations, userIndex, userIndex+1)
			inboxIndex := slices.IndexFunc(migrations, func(m *core.Migration) bool { return m.File == inboxIndexMigration })
			return slices.Insert(migrations, inboxIndex+1, moved)
		},
	}

	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			app := newMigratedApp(t, order)

			inbox := indexSQL(t, app, "idx_x8xyfe8q8y")
			if !strings.Contains(inbox, "`inbox`") {
				t.Errorf("inbox index = %q, want an index on inbox", inbox)
			}

			user := indexSQL(t, app, "idx_activitypub_actors_user")
			if !strings.HasPrefix(user, "CREATE UNIQUE INDEX") {
				t.Errorf("user index = %q, want it unique", user)
			}
		})
	}
}

func indexSQL(t *testing.T, app core.App, name string) string {
	t.Helper()
	var sql string
	err := app.DB().
		NewQuery("SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = 'activitypub_actors' AND name = {:name}").
		Bind(map[string]any{"name": name}).
		Row(&sql)
	if err != nil {
		t.Fatalf("index %s not found: %v", name, err)
	}
	return sql
}

// newMigratedApp builds the schema in a temporary database, applying the
// migrations in the order the reorder func returns and skipping only the
// migrations that configure the external Meilisearch indexes.
func newMigratedApp(t *testing.T, reorder func([]*core.Migration) []*core.Migration) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	// The regions migration reads its seed relative to the db directory.
	t.Chdir("../..")
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
	for _, migration := range reorder(migrations) {
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
