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

// Migrations an instance on dev has not applied yet. Some carry timestamps
// older than migrations already on dev, so on upgrade they run after every
// other migration rather than in timestamp order.
var upgradeMigrations = []string{
	userIndexMigration,
	"1781890072_updated_settings.go",
	"1785180139_updated_trails_filter.go",
	"1785200000_updated_subcategories.go",
	"1785300000_add_moving_duration_to_trails.go",
	"1790300000_repair_activitypub_actors_indexes.go",
	"1791200000_updated_settings_create_rule.go",
	"1791200001_updated_profile_feed.go",
	"1791200002_restore_actor_handle_index.go",
}

// The delete-recipient queries join activitypub_actors on inbox; without that
// index SQLite scans every actor for every activity. Earlier versions of
// 1776767721 overwrote the whole index list, so the outcome depended on
// the order an instance happened to apply them in.
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
		"upgraded from dev": func(migrations []*core.Migration) []*core.Migration {
			before, after := splitUpgradeMigrations(migrations)
			return append(before, after...)
		},
	}

	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			app := newApp(t)
			applyMigrations(t, app, order(sortedMigrations()))

			inbox := indexSQL(t, app, "idx_x8xyfe8q8y")
			if !strings.Contains(inbox, "`inbox`") {
				t.Errorf("inbox index = %q, want an index on inbox", inbox)
			}

			user := indexSQL(t, app, "idx_activitypub_actors_user")
			if !strings.HasPrefix(user, "CREATE UNIQUE INDEX") {
				t.Errorf("user index = %q, want it unique", user)
			}

			handle := indexSQL(t, app, "idx_actors_username_domain")
			if !strings.HasPrefix(handle, "CREATE UNIQUE INDEX") {
				t.Errorf("username/domain index = %q, want it unique", handle)
			}
		})
	}
}

func TestUpgradeKeepsTheOldestSettingsRecordPerUser(t *testing.T) {
	app := newApp(t)
	before, after := splitUpgradeMigrations(sortedMigrations())
	applyMigrations(t, app, before)

	user := saveRecord(t, app, "users", map[string]any{
		"username": "owner", "password": "test-password", "email": "owner@example.com",
	})
	other := saveRecord(t, app, "users", map[string]any{
		"username": "other", "password": "test-password", "email": "other@example.com",
	})
	// The user create hook is not registered here, so these are the only rows.
	original := saveRecord(t, app, "settings", map[string]any{"user": user.Id, "language": "de", "unit": "metric", "mapFocus": "trails"})
	if _, err := app.DB().NewQuery("UPDATE settings SET created = '2020-01-01 00:00:00.000Z' WHERE id = {:id}").
		Bind(map[string]any{"id": original.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	saveRecord(t, app, "settings", map[string]any{"user": user.Id, "language": "en", "unit": "metric", "mapFocus": "trails", "bio": "planted"})
	otherSettings := saveRecord(t, app, "settings", map[string]any{"user": other.Id, "language": "en", "unit": "metric", "mapFocus": "trails"})

	applyMigrations(t, app, after)

	rows, err := app.FindAllRecords("settings")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	sort.Strings(ids)
	want := []string{original.Id, otherSettings.Id}
	sort.Strings(want)
	if !slices.Equal(ids, want) {
		t.Errorf("settings after upgrade = %v, want %v", ids, want)
	}

	index := indexSQLOn(t, app, "settings", "idx_rhdd21t4cx")
	if !strings.HasPrefix(index, "CREATE UNIQUE INDEX") {
		t.Errorf("settings user index = %q, want it unique", index)
	}
}

func TestUpgradeNamesUsersWithDuplicateActors(t *testing.T) {
	app := newApp(t)
	before, after := splitUpgradeMigrations(sortedMigrations())
	applyMigrations(t, app, before)

	user := saveRecord(t, app, "users", map[string]any{
		"username": "owner", "password": "test-password", "email": "owner@example.com",
	})
	for _, name := range []string{"owner", "owner2"} {
		saveRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com", "user": user.Id,
			"is_local": true, "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
	}

	migration := after[0]
	if migration.File != userIndexMigration {
		t.Fatalf("first upgrade migration = %s, want %s", migration.File, userIndexMigration)
	}
	err := app.RunInTransaction(migration.Up)
	if err == nil || !strings.Contains(err.Error(), "user "+user.Id) {
		t.Fatalf("apply %s: err = %v, want it to name user %s", migration.File, err, user.Id)
	}
}

func indexSQL(t *testing.T, app core.App, name string) string {
	t.Helper()
	return indexSQLOn(t, app, "activitypub_actors", name)
}

func indexSQLOn(t *testing.T, app core.App, table, name string) string {
	t.Helper()
	var sql string
	err := app.DB().
		NewQuery("SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = {:table} AND name = {:name}").
		Bind(map[string]any{"table": table, "name": name}).
		Row(&sql)
	if err != nil {
		t.Fatalf("index %s not found: %v", name, err)
	}
	return sql
}

func newApp(t *testing.T) *core.BaseApp {
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
	return app
}

func sortedMigrations() []*core.Migration {
	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].File < migrations[j].File
	})
	return migrations
}

// splitUpgradeMigrations returns the migrations an instance on dev had already
// applied, and the ones it applies on upgrade, in the order it applies them.
func splitUpgradeMigrations(migrations []*core.Migration) (before, after []*core.Migration) {
	for _, migration := range migrations {
		if slices.Contains(upgradeMigrations, migration.File) {
			after = append(after, migration)
		} else {
			before = append(before, migration)
		}
	}
	return before, after
}

// applyMigrations skips only the migrations that configure the external
// Meilisearch indexes.
func applyMigrations(t *testing.T, app core.App, migrations []*core.Migration) {
	t.Helper()
	for _, migration := range migrations {
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}
}

func saveRecord(t *testing.T, app core.App, collection string, data map[string]any) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Load(data)
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return record
}
