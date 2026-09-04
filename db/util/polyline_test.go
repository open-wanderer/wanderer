package util

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const geometryTestGPX = `<?xml version="1.0"?><gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="46" lon="8"></trkpt><trkpt lat="46.001" lon="8.001"></trkpt></trkseg></trk></gpx>`

func TestSavePolylineGuardsGPX(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "name"},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 5 << 20},
		&core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"},
		&core.TextField{Name: "polyline", Max: PolylineMaxLength},
		&core.NumberField{Name: "min_lat"},
		&core.NumberField{Name: "max_lat"},
		&core.NumberField{Name: "min_lon"},
		&core.NumberField{Name: "max_lon"},
		&core.NumberField{Name: "bounding_box_diagonal"},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	gpxFile, err := filesystem.NewFileFromBytes([]byte(geometryTestGPX), "track.gpx")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("name", "before")
	trail.Set("gpx", gpxFile)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}

	stale, err := app.FindRecordById(trails.Id, trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := app.FindRecordById(trails.Id, trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Set("name", "edited meanwhile")
	if err := app.Save(fresh); err != nil {
		t.Fatal(err)
	}

	if err := SavePolyline(app, stale); err != nil {
		t.Fatalf("save geometry from fresh trail: %v", err)
	}
	if stale.GetString("polyline") == "" || stale.GetFloat("bounding_box_diagonal") <= 0 {
		t.Fatal("the caller record must receive the stored geometry for later hooks and saves")
	}
	stored, err := app.FindRecordById(trails.Id, trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("name") != "edited meanwhile" {
		t.Fatalf("geometry save overwrote a concurrent edit: %q", stored.GetString("name"))
	}
	if stored.GetString("polyline") == "" {
		t.Fatal("expected computed geometry")
	}

	oldGPXEvent := stored.Clone()
	replacement, err := filesystem.NewFileFromBytes([]byte(geometryTestGPX), "replacement.gpx")
	if err != nil {
		t.Fatal(err)
	}
	stored.Set("gpx", replacement)
	stored.Set("polyline", "newer-geometry")
	if err := app.Save(stored); err != nil {
		t.Fatal(err)
	}
	if err := SavePolyline(app, oldGPXEvent); !errors.Is(err, ErrTrailGPXChanged) {
		t.Fatalf("expected stale GPX guard, got %v", err)
	}
	latest, err := app.FindRecordById(trails.Id, trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if latest.GetString("polyline") != "newer-geometry" {
		t.Fatalf("stale geometry overwrote the newer track: %q", latest.GetString("polyline"))
	}
}
