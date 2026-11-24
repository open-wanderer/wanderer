package util

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func emptyGPXTrail(t *testing.T) (core.App, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := core.NewBaseCollection("geometry_trails")
	collection.Fields.Add(&core.TextField{Name: "gpx"})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(collection)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	trail.Set("gpx", "empty.gpx")
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	if err := fsys.Upload([]byte{}, trail.BaseFilesPath()+"/empty.gpx"); err != nil {
		t.Fatal(err)
	}
	content, err := ReadTrailGPX(app, trail)
	if err != nil || content == nil || len(content) != 0 {
		t.Fatalf("empty file content = %#v, error = %v", content, err)
	}
	return app, trail
}

func TestTrailCoordinatesRejectsEmptyGPX(t *testing.T) {
	app, trail := emptyGPXTrail(t)
	if _, err := TrailCoordinates(app, trail); !errors.Is(err, io.EOF) {
		t.Fatalf("empty GPX error = %v, want EOF", err)
	}
}

func TestComputeTrailGeometryRejectsEmptyGPX(t *testing.T) {
	app, trail := emptyGPXTrail(t)
	_, err := ComputeTrailGeometry(app, trail)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("empty GPX error = %v, want EOF", err)
	}
	if !strings.Contains(err.Error(), `parse gpx file "`+trail.BaseFilesPath()+`/empty.gpx"`) {
		t.Fatalf("parse error has no file path: %v", err)
	}
}
