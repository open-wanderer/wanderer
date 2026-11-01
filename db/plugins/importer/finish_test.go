package importer

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"pocketbase/pluginsystem"
)

func TestSavedTrailKeepsWaypointsWhenPhotoTimeRunsOut(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	actors := collection("activitypub_actors", &core.TextField{Name: "user"})
	actor := core.NewRecord(actors)
	actor.Set("user", "import-test-user")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	trails := collection("trails",
		&core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20},
	)
	collection("waypoints",
		&core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.NumberField{Name: "lat"}, &core.NumberField{Name: "lon"}, &core.TextField{Name: "icon"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.NumberField{Name: "distance_from_start"},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.FileField{Name: "photos", MaxSelect: 20, MaxSize: 1 << 20},
	)
	collection("trail_external_reference",
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.TextField{Name: "user"}, &core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	original := importFinishTimeout
	importFinishTimeout = 0
	t.Cleanup(func() { importFinishTimeout = original })

	item := pluginsystem.TrailImport{
		Source: pluginsystem.TrailImportSource{Provider: "test", ExternalID: "waypoint-photos"},
		Name:   "Imported route", Track: gpxTrack(),
		Waypoints: []pluginsystem.Waypoint{{
			Name: "View", Lat: 46.0005, Lon: 8.0005,
			Photos: []pluginsystem.Photo{{Source: pluginsystem.MediaSource{Type: "url", URL: "https://photos.example.test/view.jpg"}}},
		}},
	}
	opts := Options{UserID: "import-test-user", ActorID: actor.Id, Manifest: pluginsystem.Manifest{ID: "test"}}
	if _, err := ImportTrail(context.Background(), app, item, opts); err != nil {
		t.Fatal(err)
	}
	waypoints, err := app.FindRecordsByFilter("waypoints", "", "", 0, 0)
	if err != nil || len(waypoints) != 1 {
		t.Fatalf("saved trail has %d waypoints, want 1: %v", len(waypoints), err)
	}
	if photos := waypoints[0].GetStringSlice("photos"); len(photos) != 0 {
		t.Fatalf("photos downloaded after their time ran out: %v", photos)
	}
}
