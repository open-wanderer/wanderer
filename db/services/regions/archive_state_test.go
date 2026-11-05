package regions

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/dbx"
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

// fakeExtractor puts a pmtiles stand-in first on PATH. It writes its output
// file, or fails while the returned path exists, and calls() returns the source
// URLs it was run against.
func fakeExtractor(t *testing.T) (calls func() []string, failFlag string) {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	failFlag = filepath.Join(dir, "fail")
	script := "#!/bin/sh\necho \"$2\" >> '" + log + "'\n[ -e '" + failFlag + "' ] && exit 1\nprintf pmtiles > \"$3\"\n"
	if err := os.WriteFile(filepath.Join(dir, "pmtiles"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() []string {
		data, err := os.ReadFile(log)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Fields(string(data))
	}, failFlag
}

// enabledRegion enables a seeded leaf and caches a geometry for it whose bbox
// matches newArchiveRecord's.
func enabledRegion(t *testing.T, app core.App, path string) *core.Record {
	t.Helper()
	region, err := app.FindFirstRecordByFilter("regions", "path = {:path}", dbx.Params{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	region.Set("enabled", true)
	if err := app.Save(region); err != nil {
		t.Fatal(err)
	}
	collection, err := app.FindCollectionByNameOrId("region_geometry")
	if err != nil {
		t.Fatal(err)
	}
	geometry := core.NewRecord(collection)
	geometry.Set("path", path)
	geometry.Set("polygon", map[string]any{"type": "Polygon", "coordinates": []any{}})
	geometry.Set("bbox", []float64{7.4, 43.7, 7.5, 43.8})
	if err := app.Save(geometry); err != nil {
		t.Fatal(err)
	}
	return region
}

const (
	testBuildDate = "20261003" // two days after newArchiveRecord's build date
	testBuildURL  = "https://build.protomaps.com/20261003.pmtiles"
)

func TestBuildRegionRestoresMissingFiles(t *testing.T) {
	for _, test := range []struct {
		name         string
		deleteVector bool
		deleteDem    bool
		wantCalls    []string
	}{
		{"nothing missing, nothing built", false, false, nil},
		{"missing vector", true, false, []string{testBuildURL}},
		{"missing DEM", false, true, []string{mapterhornSource}},
		{"both missing", true, true, []string{testBuildURL, mapterhornSource}},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newRegionsTestApp(t)
			calls, _ := fakeExtractor(t)
			region := enabledRegion(t, app, "monaco")
			archive := newArchiveRecord(t, app, "monaco", nil)
			if !test.deleteVector {
				writeArchiveFile(t, RegionArchivePath(app, "monaco"))
			}
			if !test.deleteDem {
				writeArchiveFile(t, RegionDemPath(app, "monaco"))
			}

			buildRegion(app, region, testBuildDate, testBuildURL)

			if got := calls(); !slices.Equal(got, test.wantCalls) {
				t.Fatalf("extracted from %v, want %v", got, test.wantCalls)
			}
			if !archiveFileExists(RegionArchivePath(app, "monaco")) || !archiveFileExists(RegionDemPath(app, "monaco")) {
				t.Error("an archive is still missing after the build")
			}
			got := reload(t, app, archive)
			if got.GetString("status") != "ready" || got.GetString("dem_status") != "ready" {
				t.Errorf("status = %q, dem_status = %q, want both ready", got.GetString("status"), got.GetString("dem_status"))
			}
		})
	}
}

// A file that cannot be restored costs one extract per pass, and the region
// says so instead of staying "ready" without a download.
func TestBuildRegionReportsAMissingFileItCannotRestore(t *testing.T) {
	app := newRegionsTestApp(t)
	calls, failFlag := fakeExtractor(t)
	writeArchiveFile(t, failFlag)
	region := enabledRegion(t, app, "monaco")
	archive := newArchiveRecord(t, app, "monaco", nil)
	writeArchiveFile(t, RegionDemPath(app, "monaco"))

	buildRegion(app, region, testBuildDate, testBuildURL)
	if got := calls(); !slices.Equal(got, []string{testBuildURL}) {
		t.Fatalf("extracted from %v in one pass, want one vector extract", got)
	}
	got := reload(t, app, archive)
	if got.GetString("status") != "error" || got.GetString("dem_status") != "ready" {
		t.Errorf("status = %q, dem_status = %q, want error and ready", got.GetString("status"), got.GetString("dem_status"))
	}

	if err := os.Remove(failFlag); err != nil {
		t.Fatal(err)
	}
	buildRegion(app, region, testBuildDate, testBuildURL)
	got = reload(t, app, archive)
	if got.GetString("status") != "ready" || got.GetString("error_message") != "" {
		t.Errorf("status = %q, error_message = %q after the upstream recovered", got.GetString("status"), got.GetString("error_message"))
	}
}

// Dashboard backups leave the archives out, so a restore brings back rows
// whose files are gone.
func TestRestoredRowsWithoutFilesAreRebuilt(t *testing.T) {
	app := newRegionsTestApp(t)
	calls, _ := fakeExtractor(t)
	region := enabledRegion(t, app, "monaco")
	newArchiveRecord(t, app, "monaco", nil)

	if err := ReconcileArchives(app); err != nil {
		t.Fatal(err)
	}
	buildRegion(app, region, testBuildDate, testBuildURL)

	if got := calls(); !slices.Equal(got, []string{testBuildURL, mapterhornSource}) {
		t.Fatalf("extracted from %v, want both archives", got)
	}
	archive, err := findRegionArchiveRecord(app, "monaco")
	if err != nil || archive == nil {
		t.Fatalf("no archive row after the rebuild: %v", err)
	}
	if archive.GetString("status") != "ready" || archive.GetString("dem_status") != "ready" {
		t.Errorf("status = %q, dem_status = %q, want both ready", archive.GetString("status"), archive.GetString("dem_status"))
	}
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

// v0.21.0-app left status "error" behind on a failed rebuild, with the previous
// archive still on disk.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFirstBuildGeometryFailureIsReportedAndRecoverable(t *testing.T) {
	app := newRegionsTestApp(t)
	calls, _ := fakeExtractor(t)
	region, err := app.FindFirstRecordByFilter("regions", "path = 'monaco'")
	if err != nil {
		t.Fatal(err)
	}
	region.Set("enabled", true)
	if err := app.Save(region); err != nil {
		t.Fatal(err)
	}

	upstreamUp := false
	previous := polyHTTPClient
	polyHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !upstreamUp {
			return nil, errors.New("upstream down")
		}
		poly := "monaco\n1\n 7.4 43.7\n 7.5 43.7\n 7.5 43.8\n 7.4 43.8\n 7.4 43.7\nEND\nEND\n"
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(poly)), Request: r}, nil
	})}
	t.Cleanup(func() { polyHTTPClient = previous })

	for pass := 1; pass <= 2; pass++ {
		buildRegion(app, region, testBuildDate, testBuildURL)

		rows, err := app.FindAllRecords("region_archives", dbx.HashExp{"path": "monaco"})
		if err != nil || len(rows) != 1 {
			t.Fatalf("pass %d: %d archive rows, err %v; want exactly one", pass, len(rows), err)
		}
		row := rows[0]
		if row.GetString("status") != "error" || row.GetString("dem_status") != "error" {
			t.Errorf("pass %d: status = %q, dem_status = %q, want both error", pass, row.GetString("status"), row.GetString("dem_status"))
		}
		if !strings.Contains(row.GetString("error_message"), "raw.githubusercontent.com") {
			t.Errorf("pass %d: error_message = %q, want the upstreams named", pass, row.GetString("error_message"))
		}
		if row.GetFloat("min_lon") != 0 || row.GetFloat("max_lat") != 0 {
			t.Errorf("pass %d: bounds set without a resolved geometry", pass)
		}
	}
	if extracts := calls(); len(extracts) != 0 {
		t.Fatalf("extracted without geometry: %v", extracts)
	}

	if err := ReconcileArchives(app); err != nil {
		t.Fatal(err)
	}
	row, err := findRegionArchiveRecord(app, "monaco")
	if err != nil || row == nil {
		t.Fatalf("reconcile dropped the error row: %v", err)
	}

	upstreamUp = true
	buildRegion(app, region, testBuildDate, testBuildURL)

	rows, err := app.FindAllRecords("region_archives", dbx.HashExp{"path": "monaco"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("%d archive rows after recovery, err %v; want exactly one", len(rows), err)
	}
	row = rows[0]
	if row.GetString("status") != "ready" || row.GetString("dem_status") != "ready" {
		t.Errorf("status = %q, dem_status = %q after recovery, want both ready", row.GetString("status"), row.GetString("dem_status"))
	}
	if row.GetString("error_message") != "" || row.GetString("dem_error_message") != "" {
		t.Errorf("error messages = %q / %q after recovery, want both cleared", row.GetString("error_message"), row.GetString("dem_error_message"))
	}
	if row.GetFloat("min_lon") != 7.4 || row.GetFloat("max_lat") != 43.8 {
		t.Errorf("bounds = %v..%v after recovery", row.GetFloat("min_lon"), row.GetFloat("max_lat"))
	}
}

// A geometry failure on a built region must not touch what it already serves.
func TestGeometryFailureKeepsABuiltRegionReady(t *testing.T) {
	app := newRegionsTestApp(t)
	region, err := app.FindFirstRecordByFilter("regions", "path = 'monaco'")
	if err != nil {
		t.Fatal(err)
	}
	region.Set("enabled", true)
	region.Set("comaps_id", "")
	archive := newArchiveRecord(t, app, "monaco", nil)
	writeArchiveFile(t, RegionArchivePath(app, "monaco"))
	writeArchiveFile(t, RegionDemPath(app, "monaco"))

	buildRegion(app, region, testBuildDate, testBuildURL)

	got := reload(t, app, archive)
	if got.GetString("status") != "ready" || got.GetString("dem_status") != "ready" {
		t.Errorf("status = %q, dem_status = %q, want both ready", got.GetString("status"), got.GetString("dem_status"))
	}
	if got.GetString("error_message") == "" {
		t.Error("the failure was not recorded")
	}
	if got.GetString("dem_error_message") != "" {
		t.Errorf("dem_error_message = %q, a DEM that is never rebuilt would keep it forever", got.GetString("dem_error_message"))
	}
}

func TestOptionalBoundsMigration(t *testing.T) {
	app := newRegionsTestApp(t)
	migration := findMigration(t, "1791200101_optional_region_archive_bounds.go")
	existing := newArchiveRecord(t, app, "monaco", nil)

	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		t.Fatal(err)
	}
	withoutBounds := func() error {
		record := core.NewRecord(collection)
		record.Load(map[string]any{"path": "andorra", "status": "error", "dem_status": "error"})
		return app.Save(record)
	}

	// The schema of an instance that has not run the migration yet.
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	if collection, err = app.FindCollectionByNameOrId("region_archives"); err != nil {
		t.Fatal(err)
	}
	if err := withoutBounds(); err == nil {
		t.Fatal("a row without bounds was accepted before the migration")
	}

	if err := app.RunInTransaction(migration.Up); err != nil {
		t.Fatal(err)
	}
	if collection, err = app.FindCollectionByNameOrId("region_archives"); err != nil {
		t.Fatal(err)
	}
	if err := withoutBounds(); err != nil {
		t.Fatalf("a row without bounds was rejected after the migration: %v", err)
	}
	if got := reload(t, app, existing); got.GetFloat("min_lon") != 7.4 {
		t.Errorf("existing row bounds = %v, want them kept", got.GetFloat("min_lon"))
	}
}

func TestReconcileRecoversErrorRowsWithAnArchive(t *testing.T) {
	app := newRegionsTestApp(t)
	calls, failFlag := fakeExtractor(t)
	writeArchiveFile(t, failFlag)

	region := enabledRegion(t, app, "monaco")
	built := newArchiveRecord(t, app, "monaco", map[string]any{
		"status": "error", "error_message": "old failure",
		"dem_status": "error", "dem_error_message": "old DEM failure",
	})
	writeArchiveFile(t, RegionArchivePath(app, "monaco"))
	writeArchiveFile(t, RegionDemPath(app, "monaco"))
	vectorOnly := newArchiveRecord(t, app, "san_marino", map[string]any{"status": "error", "dem_status": "error"})
	writeArchiveFile(t, RegionArchivePath(app, "san_marino"))
	noFiles := newArchiveRecord(t, app, "andorra", map[string]any{"status": "error", "dem_status": "error"})

	if err := ReconcileArchives(app); err != nil {
		t.Fatal(err)
	}

	got := reload(t, app, built)
	if got.GetString("status") != "ready" || got.GetString("dem_status") != "ready" {
		t.Errorf("status = %q, dem_status = %q, want both ready", got.GetString("status"), got.GetString("dem_status"))
	}
	if got.GetString("error_message") != "old failure" || got.GetString("dem_error_message") != "old DEM failure" {
		t.Errorf("error messages = %q / %q, want them kept", got.GetString("error_message"), got.GetString("dem_error_message"))
	}
	got = reload(t, app, vectorOnly)
	if got.GetString("status") != "ready" || got.GetString("dem_status") != "error" {
		t.Errorf("vector only: status = %q, dem_status = %q, want ready and error", got.GetString("status"), got.GetString("dem_status"))
	}
	got = reload(t, app, noFiles)
	if got.GetString("status") != "error" || got.GetString("dem_status") != "error" {
		t.Errorf("no files: status = %q, dem_status = %q, want both error", got.GetString("status"), got.GetString("dem_status"))
	}
	if extracts := calls(); len(extracts) != 0 {
		t.Errorf("reconcile ran extracts: %v", extracts)
	}

	// The upstream is still down. A recent archive is left alone, and an aged
	// one fails once and stays ready.
	buildRegion(app, region, testBuildDate, testBuildURL)
	if extracts := calls(); len(extracts) != 0 {
		t.Errorf("recovered region with a recent archive was rebuilt: %v", extracts)
	}
	buildRegion(app, region, "20261020", "https://build.protomaps.com/20261020.pmtiles")
	if extracts := calls(); len(extracts) != 1 {
		t.Errorf("extracts for an aged archive = %v, want one", extracts)
	}
	got = reload(t, app, built)
	if got.GetString("status") != "ready" {
		t.Errorf("status = %q after a failed rebuild, want ready", got.GetString("status"))
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
