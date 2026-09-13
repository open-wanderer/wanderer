package permissions_test

import (
	"slices"
	"sort"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/pocketbase/core"
)

// newRulesTestApp builds the current schema in a temporary database.
// Only migrations that exclusively configure external Meilisearch indexes are
// excluded; all schema and data migrations are applied.
func newRulesTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
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
