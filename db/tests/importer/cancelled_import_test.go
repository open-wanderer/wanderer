package importer_test

import (
	"context"
	"testing"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
)

// cancelAfterChecks is a context that reports itself cancelled once Err has
// been called a given number of times, so a test can cancel an import at each
// point where the importer checks its context.
type cancelAfterChecks struct {
	context.Context
	remaining int
}

func (c *cancelAfterChecks) Err() error {
	if c.remaining <= 0 {
		return context.Canceled
	}
	c.remaining--
	return nil
}

func TestCancelledImportKeepsNoTrailWithoutItsReference(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	cancelledBeforeSave := false
	for checks := 0; checks < 10; checks++ {
		item := pluginsystem.TrailImport{
			Name:      "Cancelled",
			Source:    pluginsystem.TrailImportSource{Provider: "test-provider", ExternalID: "cancel-" + string(rune('a'+checks))},
			Track:     providerDifficultyGPX(),
			Waypoints: []pluginsystem.Waypoint{{Name: "View", Lat: 46.0005, Lon: 8.0005}},
		}
		before := providerDifficultyRecordCount(t, app, "trails")
		result, err := importer.ImportTrail(&cancelAfterChecks{Context: context.Background(), remaining: checks}, app, item, opts)
		trails := providerDifficultyRecordCount(t, app, "trails") - before
		if err != nil {
			// Once the trail is saved, a cancelled import keeps it, like any other
			// failure in its waypoints or summit log.
			cancelledBeforeSave = cancelledBeforeSave || trails == 0
			continue
		}
		if !result.Created || trails != 1 {
			t.Fatalf("uncancelled import after %d checks: %#v, %d trails", checks, result, trails)
		}
	}
	if !cancelledBeforeSave {
		t.Fatal("no run was cancelled before the trail was saved")
	}
	if refs := providerDifficultyRecordCount(t, app, "trail_external_reference"); refs != providerDifficultyRecordCount(t, app, "trails") {
		t.Fatalf("%d external references for %d trails", refs, providerDifficultyRecordCount(t, app, "trails"))
	}
}
