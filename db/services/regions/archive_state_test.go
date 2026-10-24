package regions

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/pocketbase/core"
)

func newRegionsTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })

	migrations := slices.Clone(core.AppMigrations.Items())
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

func newArchiveRecord(t *testing.T, app core.App, path string, data map[string]any) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Load(map[string]any{
		"path": path, "name": path, "status": "ready", "dem_status": "ready",
		"min_lon": 7.4, "min_lat": 43.7, "max_lon": 7.5, "max_lat": 43.8,
		"vector_built_date": "20261001",
	})
	record.Load(data)
	if err := app.Save(record); err != nil {
		t.Fatalf("save region_archives %s: %v", path, err)
	}
	return record
}

func writeArchiveFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("pmtiles"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func reload(t *testing.T, app core.App, record *core.Record) *core.Record {
	t.Helper()
	fresh, err := app.FindRecordById("region_archives", record.Id)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func TestFailedRebuildKeepsServingTheExistingArchive(t *testing.T) {
	app := newRegionsTestApp(t)
	// No pmtiles on PATH, so every extract fails.
	t.Setenv("PATH", t.TempDir())

	built := newArchiveRecord(t, app, "monaco", nil)
	writeArchiveFile(t, RegionArchivePath(app, "monaco"))
	writeArchiveFile(t, RegionDemPath(app, "monaco"))
	buildVector(app, built, "monaco", "unused.geojson", "https://build.protomaps.com/20261018.pmtiles", "20261018")
	buildDem(app, built, "monaco", "unused.geojson")

	got := reload(t, app, built)
	if got.GetString("status") != "ready" || got.GetString("dem_status") != "ready" {
		t.Errorf("status = %q, dem_status = %q after failed rebuilds; want both ready", got.GetString("status"), got.GetString("dem_status"))
	}
	if got.GetString("vector_built_date") != "20261001" {
		t.Errorf("vector_built_date = %q, want the previous build's date", got.GetString("vector_built_date"))
	}
	if !strings.Contains(got.GetString("error_message"), "pmtiles") || !strings.Contains(got.GetString("dem_error_message"), "pmtiles") {
		t.Errorf("error messages = %q / %q, want both to name the failed extract", got.GetString("error_message"), got.GetString("dem_error_message"))
	}

	neverBuilt := newArchiveRecord(t, app, "san_marino", map[string]any{"status": "building", "dem_status": "building"})
	buildVector(app, neverBuilt, "san_marino", "unused.geojson", "https://build.protomaps.com/20261018.pmtiles", "20261018")
	buildDem(app, neverBuilt, "san_marino", "unused.geojson")

	got = reload(t, app, neverBuilt)
	if got.GetString("status") != "error" || got.GetString("dem_status") != "error" {
		t.Errorf("status = %q, dem_status = %q after failed first builds; want both error", got.GetString("status"), got.GetString("dem_status"))
	}
	if _, err := os.Stat(RegionArchivePath(app, "san_marino") + ".building"); !os.IsNotExist(err) {
		t.Errorf("failed build left its .building file behind: %v", err)
	}
}

func TestReconcileArchives(t *testing.T) {
	app := newRegionsTestApp(t)

	withFiles := newArchiveRecord(t, app, "monaco", nil)
	writeArchiveFile(t, RegionArchivePath(app, "monaco"))
	orphan := newArchiveRecord(t, app, "andorra", nil)
	failed := newArchiveRecord(t, app, "san_marino", map[string]any{"status": "error", "error_message": "extract failed"})
	leftover := RegionDemPath(app, "monaco") + ".building"
	writeArchiveFile(t, leftover)

	if err := ReconcileArchives(app); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		record *core.Record
		want   bool
	}{{withFiles, true}, {orphan, false}, {failed, true}} {
		_, err := app.FindRecordById("region_archives", test.record.Id)
		if exists := err == nil; exists != test.want {
			t.Errorf("row %s exists = %v after reconcile, want %v", test.record.GetString("path"), exists, test.want)
		}
	}
	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Errorf("interrupted build %s still exists: %v", leftover, err)
	}
	if !archiveFileExists(RegionArchivePath(app, "monaco")) {
		t.Error("reconcile removed a finished archive")
	}
}
