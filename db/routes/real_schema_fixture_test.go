package routes

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"sync"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/pocketbase/core"
)

var remoteSchemaMigrations []*core.Migration

var (
	realSchemaTemplateOnce sync.Once
	realSchemaTemplateDir  string
	realSchemaTemplateErr  error
)

func TestMain(m *testing.M) {
	// Existing route fixtures create their own collections through TestApp.
	// Keep its automatic migrations separate from the real-schema fixture.
	remoteSchemaMigrations = slices.Clone(core.AppMigrations.Items())
	core.AppMigrations = core.MigrationsList{}
	code := m.Run()
	if realSchemaTemplateDir != "" {
		os.RemoveAll(realSchemaTemplateDir)
	}
	os.Exit(code)
}

// Migrating the real schema takes seconds, so it runs once per test binary
// into a template data directory that every fixture copies.
func buildRealSchemaTemplate() (string, error) {
	dir, err := os.MkdirTemp("", "wanderer-real-schema-")
	if err != nil {
		return "", err
	}
	previousOrigin, hadOrigin := os.LookupEnv("ORIGIN")
	os.Setenv("ORIGIN", "https://local.example")
	defer func() {
		if hadOrigin {
			os.Setenv("ORIGIN", previousOrigin)
		} else {
			os.Unsetenv("ORIGIN")
		}
	}()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: dir})
	app.Settings().Logs.MaxDays = 0
	if err := app.Bootstrap(); err != nil {
		return dir, err
	}
	defer app.ResetBootstrapState()
	migrations := slices.Clone(remoteSchemaMigrations)
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].File < migrations[j].File })
	for _, migration := range migrations {
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			return dir, fmt.Errorf("apply %s: %w", migration.File, err)
		}
	}
	return dir, nil
}

// Build the real schema while leaving external search configuration out.
func newRemoteRealSchemaApp(t *testing.T) *core.BaseApp {
	t.Helper()
	realSchemaTemplateOnce.Do(func() {
		realSchemaTemplateDir, realSchemaTemplateErr = buildRealSchemaTemplate()
	})
	if realSchemaTemplateErr != nil {
		t.Fatal(realSchemaTemplateErr)
	}
	t.Setenv("ORIGIN", "https://local.example")
	dataDir := t.TempDir()
	if err := os.CopyFS(dataDir, os.DirFS(realSchemaTemplateDir)); err != nil {
		t.Fatal(err)
	}
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: dataDir})
	app.Settings().Logs.MaxDays = 0
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
