package importer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestImportTrailKeepsIndependentSuccessfulPhotoLimits(t *testing.T) {
	originalFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = originalFetch })
	for _, test := range []struct {
		name      string
		count     int
		failFirst bool
		want      int
	}{
		{"eight waypoint photos", 8, false, 8},
		{"twenty per target", 25, false, 20},
		{"failed photos allow replacements", 21, true, 20},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, actor := newTrailPhotoLimitTestApp(t)
			fetches := 0
			fetchPublicPluginMedia = func(_ context.Context, source string, _ int64) (*util.SafeFetchResult, error) {
				fetches++
				if test.failFirst && strings.HasSuffix(source, "-00.jpg") {
					return nil, errors.New("photo is unavailable")
				}
				return &util.SafeFetchResult{Body: []byte("photo bytes"), ContentType: "image/jpeg", FinalURL: source}, nil
			}
			photos := func(prefix string) []pluginsystem.Photo {
				result := make([]pluginsystem.Photo, 0, test.count)
				for i := range test.count {
					id := fmt.Sprintf("%s-%02d", prefix, i)
					result = append(result, pluginsystem.Photo{ExternalID: id, Filename: id + ".jpg",
						Source: pluginsystem.MediaSource{Type: "url", URL: "https://b.example/" + id + ".jpg"}})
				}
				return result
			}
			item := pluginsystem.TrailImport{
				Source: pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "many-photos"},
				Name:   "Photo trail", Track: gpxTrack(), Photos: photos("trail"),
				Waypoints: []pluginsystem.Waypoint{
					{Name: "First", Lat: 46, Lon: 8, Photos: photos("first")},
					{Name: "Second", Lat: 46.001, Lon: 8.001, Photos: photos("second")},
				},
			}
			result, err := ImportTrail(context.Background(), app, item, Options{
				UserID: "owner", ActorID: actor.Id,
				Manifest: pluginsystem.Manifest{ID: "komoot", Type: pluginsystem.PluginTypeTrails},
			})
			if err != nil {
				t.Fatal(err)
			}
			links, err := app.FindAllRecords("trail_assets", dbx.HashExp{"trail": result.TrailID})
			if err != nil || len(links) != test.want {
				t.Fatalf("trail photos = %d, %v; want %d", len(links), err, test.want)
			}
			waypoints, err := app.FindAllRecords("waypoints", dbx.HashExp{"trail": result.TrailID})
			if err != nil || len(waypoints) != 2 {
				t.Fatalf("waypoints = %d, %v; want 2", len(waypoints), err)
			}
			for _, waypoint := range waypoints {
				links, err := app.FindAllRecords("waypoint_assets", dbx.HashExp{"waypoint": waypoint.Id})
				if err != nil || len(links) != test.want {
					t.Errorf("%s photos = %d, %v; want %d", waypoint.GetString("name"), len(links), err, test.want)
				}
			}
			wantFetches := test.want * 3
			if test.failFirst {
				wantFetches += 3
			}
			if fetches != wantFetches {
				t.Errorf("downloads = %d, want %d", fetches, wantFetches)
			}
		})
	}
}

func newTrailPhotoLimitTestApp(t *testing.T) (core.App, *core.Record) {
	t.Helper()
	app := newPhotoAssetImportTestApp(t)
	collections := map[string][]core.Field{
		"activitypub_actors": {&core.TextField{Name: "user"}},
		"trails":             {&core.TextField{Name: "name"}, &core.TextField{Name: "author"}, &core.BoolField{Name: "public"}, &core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20}},
		"waypoints":          {&core.TextField{Name: "name"}, &core.TextField{Name: "trail"}, &core.TextField{Name: "author"}},
		"trail_assets":       {&core.TextField{Name: "trail"}, &core.TextField{Name: "asset"}},
		"waypoint_assets":    {&core.TextField{Name: "waypoint"}, &core.TextField{Name: "asset"}},
		"trail_external_reference": {
			&core.TextField{Name: "trail"}, &core.TextField{Name: "user"}, &core.TextField{Name: "provider"}, &core.TextField{Name: "external_id"},
			&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
		},
	}
	for name, fields := range collections {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(fields...)
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(actors)
	actor.Set("user", "owner")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	return app, actor
}
