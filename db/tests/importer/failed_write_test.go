package importer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
)

func failedWriteItem() pluginsystem.TrailImport {
	return pluginsystem.TrailImport{
		Name:      "Failing",
		Source:    pluginsystem.TrailImportSource{Provider: "test-provider", ExternalID: "failing"},
		Track:     providerDifficultyGPX(),
		Waypoints: []pluginsystem.Waypoint{{Name: "View", Lat: 46.0005, Lon: 8.0005}},
	}
}

func TestFailingWaypointSaveImportsTheTrailOnce(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	app.OnRecordCreate("waypoints").BindFunc(func(*core.RecordEvent) error {
		return errors.New("waypoint save failed")
	})
	deletes := 0
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		deletes++
		return e.Next()
	})

	for range 3 {
		_, _ = importer.ImportTrail(context.Background(), app, failedWriteItem(), opts)
	}
	trails := providerDifficultyRecordCount(t, app, "trails")
	refs := providerDifficultyRecordCount(t, app, "trail_external_reference")
	if trails != 1 || refs != 1 || deletes != 0 {
		t.Fatalf("three imports: %d trails, %d external references, %d deletes; want 1, 1, 0", trails, refs, deletes)
	}
}

func TestFailingTrailCreateHookKeepsTheExternalReference(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	app.OnRecordAfterCreateSuccess("trails").BindFunc(func(*core.RecordEvent) error {
		return errors.New("search index unavailable")
	})

	for range 3 {
		_, _ = importer.ImportTrail(context.Background(), app, failedWriteItem(), opts)
	}
	trails := providerDifficultyRecordCount(t, app, "trails")
	refs := providerDifficultyRecordCount(t, app, "trail_external_reference")
	if trails != 1 || refs != 1 {
		t.Fatalf("three imports: %d trails, %d external references; want 1, 1", trails, refs)
	}
}
