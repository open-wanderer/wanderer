package importer

import (
	"context"
	"slices"
	"testing"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// Followers learn of a public trail from the activity of the save that makes
// it public, which is built from the photos linked at that moment. An import
// therefore publishes the trail only once its photos and waypoints are stored.
func TestImportTrailPublishesAfterPhotosAndWaypoints(t *testing.T) {
	originalFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = originalFetch })
	fetchPublicPluginMedia = func(_ context.Context, source string, _ int64) (*util.SafeFetchResult, error) {
		return &util.SafeFetchResult{Body: []byte("photo bytes"), ContentType: "image/jpeg", FinalURL: source}, nil
	}

	app, err := pbtests.NewTestApp(t.TempDir())
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
		&core.TextField{Name: "name"}, &core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20},
	)
	waypoints := collection("waypoints",
		&core.TextField{Name: "name"},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	collection("trail_external_reference",
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.TextField{Name: "user"}, &core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	collection("assets",
		&core.TextField{Name: "type"}, &core.TextField{Name: "storage_mode"},
		&core.TextField{Name: "remote_status"}, &core.TextField{Name: "author"},
		&core.TextField{Name: "external_provider"}, &core.TextField{Name: "external_id"},
		&core.JSONField{Name: "metadata"},
		&core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1024},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	collection("trail_assets",
		&core.TextField{Name: "asset"},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
	)
	collection("waypoint_assets",
		&core.TextField{Name: "asset"},
		&core.RelationField{Name: "waypoint", CollectionId: waypoints.Id, MaxSelect: 1},
	)

	// What followers would be told about by each save of the trail.
	type announced struct {
		public                                 bool
		trailPhotos, waypoints, waypointPhotos int
	}
	var saves []announced
	observe := func(e *core.RecordEvent) error {
		state := announced{public: e.Record.GetBool("public")}
		trailPhotos, err := e.App.FindAllRecords("trail_assets", dbx.HashExp{"trail": e.Record.Id})
		if err != nil {
			return err
		}
		state.trailPhotos = len(trailPhotos)
		trailWaypoints, err := e.App.FindAllRecords("waypoints", dbx.HashExp{"trail": e.Record.Id})
		if err != nil {
			return err
		}
		state.waypoints = len(trailWaypoints)
		for _, waypoint := range trailWaypoints {
			photos, err := e.App.FindAllRecords("waypoint_assets", dbx.HashExp{"waypoint": waypoint.Id})
			if err != nil {
				return err
			}
			state.waypointPhotos += len(photos)
		}
		saves = append(saves, state)
		return e.Next()
	}
	app.OnRecordAfterCreateSuccess("trails").BindFunc(observe)
	app.OnRecordAfterUpdateSuccess("trails").BindFunc(observe)

	photo := func(id string) pluginsystem.Photo {
		return pluginsystem.Photo{ExternalID: id, Filename: id + ".jpg", ContentType: "image/jpeg",
			Source: pluginsystem.MediaSource{Type: "url", URL: "https://b.example/" + id + ".jpg"}}
	}
	privacy := "public"
	item := pluginsystem.TrailImport{
		Source: pluginsystem.TrailImportSource{Provider: "test", ExternalID: "public-activity"},
		Name:   "Public import", Track: gpxTrack(), Privacy: &privacy,
		Photos:    []pluginsystem.Photo{photo("first"), photo("second")},
		Waypoints: []pluginsystem.Waypoint{{Name: "Hut", Lat: 46, Lon: 8, Photos: []pluginsystem.Photo{photo("hut")}}},
	}
	// A public import stores each photo before its publication save.
	opts := Options{UserID: "import-test-user", ActorID: actor.Id,
		Manifest: pluginsystem.Manifest{ID: "test"}}
	result, err := ImportTrail(context.Background(), app, item, opts)
	if err != nil {
		t.Fatal(err)
	}

	trail, err := app.FindRecordById(trails, result.TrailID)
	if err != nil {
		t.Fatal(err)
	}
	if !trail.GetBool("public") {
		t.Fatal("public import ended up private")
	}
	published := slices.IndexFunc(saves, func(state announced) bool { return state.public })
	if published < 0 {
		t.Fatalf("no save made the trail public: %+v", saves)
	}
	if want := (announced{public: true, trailPhotos: 2, waypoints: 1, waypointPhotos: 1}); saves[published] != want {
		t.Fatalf("trail became public with %+v, want %+v", saves[published], want)
	}
	assets, err := app.FindAllRecords("assets")
	if err != nil || len(assets) != 3 {
		t.Fatalf("expected the 3 photos to be imported, got %d: %v", len(assets), err)
	}
	for _, asset := range assets {
		if asset.GetString("storage_mode") != "copy" {
			t.Fatalf("photo %s of a public import is stored as %q", asset.GetString("external_id"), asset.GetString("storage_mode"))
		}
	}
}
