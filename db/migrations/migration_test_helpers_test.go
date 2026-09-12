package migrations

import (
	"slices"
	"sort"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// newRulesTestApp applies every registered migration before beforeMigration.
// An empty boundary applies all migrations, so current-schema tests also cover
// later changes. Only migrations that exclusively configure Meilisearch are
// excluded; schema and data migration errors always fail the test.
func newRulesTestApp(t *testing.T, beforeMigration string) *core.BaseApp {
	t.Helper()
	if beforeMigration != "" {
		ruleMigration(t, beforeMigration)
	}
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })

	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].File < migrations[j].File
	})
	for _, migration := range migrations {
		if beforeMigration != "" && migration.File >= beforeMigration {
			break
		}
		switch migration.File {
		case "1742167033_init_meilisearch.go",
			"1744651602_add_polyline.go",
			"1749831369_update_sortable_attributes.go":
			// These migrations change only the external search indexes.
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}
	return app
}

func ruleMigration(t *testing.T, file string) *core.Migration {
	t.Helper()
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == file {
			return migration
		}
	}
	t.Fatalf("migration %s not registered", file)
	return nil
}

func saveRulesTestRecord(t *testing.T, app core.App, collection string, data map[string]any) *core.Record {
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
