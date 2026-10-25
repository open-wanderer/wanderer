package routes

import (
	"os"
	"slices"
	"sort"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/pocketbase/core"
)

var remoteSchemaMigrations []*core.Migration

func TestMain(m *testing.M) {
	// Existing route fixtures create their own collections through TestApp.
	// Keep its automatic migrations separate from the real-schema fixture.
	remoteSchemaMigrations = slices.Clone(core.AppMigrations.Items())
	core.AppMigrations = core.MigrationsList{}
	os.Exit(m.Run())
}

// Build the real schema while leaving external search configuration out.
func newRemoteRealSchemaApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	app.Settings().Logs.MaxDays = 0
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	migrations := slices.Clone(remoteSchemaMigrations)
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].File < migrations[j].File })
	for _, migration := range migrations {
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
