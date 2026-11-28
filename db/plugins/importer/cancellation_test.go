package importer

import (
	"context"
	"errors"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/pluginsystem"
	"pocketbase/util"
	"strings"
	"testing"
)

func TestCancelledPhotoImportCanRetryWithoutPartialRecords(t *testing.T) {
	for _, phase := range []string{"trail transfer", "waypoint transfer", "persistence"} {
		t.Run(phase, func(t *testing.T) {
			app, actor := newTrailPhotoLimitTestApp(t)
			original := fetchPublicPluginMedia
			t.Cleanup(func() { fetchPublicPluginMedia = original })
			photo := func(id string) pluginsystem.Photo {
				return pluginsystem.Photo{ExternalID: id, Filename: id + ".jpg", Source: pluginsystem.MediaSource{Type: "url", URL: "https://b.example/" + id + ".jpg"}}
			}
			item := pluginsystem.TrailImport{Source: pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "cancelled-import"}, Name: "Photo route", Track: gpxTrack(), Photos: []pluginsystem.Photo{photo("first"), photo("second")}, Waypoints: []pluginsystem.Waypoint{{Name: "Point", Lat: 46, Lon: 8, Photos: []pluginsystem.Photo{photo("point-first"), photo("point-second")}}}}
			public := "public"
			item.Privacy = &public
			opts := Options{UserID: "owner", ActorID: actor.Id, Manifest: pluginsystem.Manifest{ID: "komoot"}}
			assets, err := app.FindCollectionByNameOrId("assets")
			if err != nil {
				t.Fatal(err)
			}
			existing := core.NewRecord(assets)
			file, err := filesystem.NewFileFromBytes([]byte("existing photo bytes"), "existing.jpg")
			if err != nil {
				t.Fatal(err)
			}
			existing.Load(map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id, "external_provider": "komoot", "external_id": "first", "file": file, "metadata": map[string]any{"caption": "Keep this photo"}})
			if err := app.Save(existing); err != nil {
				t.Fatal(err)
			}
			existingFile, existingMetadata := existing.GetString("file"), existing.GetString("metadata")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			created, published := 0, 0
			app.OnRecordAfterCreateSuccess("trails").BindFunc(func(e *core.RecordEvent) error { created++; return e.Next() })
			app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
				if e.Record.GetBool("public") {
					published++
				}
				return e.Next()
			})
			interrupted := false
			cancelOn := "/second.jpg"
			if phase == "waypoint transfer" {
				cancelOn = "/point-second.jpg"
			}
			fetchPublicPluginMedia = func(_ context.Context, url string, _ int64) (*util.SafeFetchResult, error) {
				if phase != "persistence" && !interrupted && strings.HasSuffix(url, cancelOn) {
					interrupted = true
					cancel()
					return nil, ctx.Err()
				}
				return &util.SafeFetchResult{Body: []byte("downloaded photo bytes"), ContentType: "image/jpeg", FinalURL: url}, nil
			}
			if phase == "persistence" {
				app.OnRecordCreate("waypoint_assets").BindFunc(func(e *core.RecordEvent) error {
					if err := e.Next(); err != nil {
						return err
					}
					if !interrupted {
						interrupted = true
						cancel()
					}
					return nil
				})
			}
			result, err := ImportTrail(ctx, app, item, opts)
			if !errors.Is(err, context.Canceled) || result != nil || !interrupted {
				t.Fatalf("cancelled import result=%+v error=%v interrupted=%t", result, err, interrupted)
			}
			for _, collection := range []string{"trails", "trail_external_reference", "trail_assets", "waypoints", "waypoint_assets"} {
				records, err := app.FindAllRecords(collection)
				if err != nil || len(records) != 0 {
					t.Fatalf("cancelled import left %s records=%d error=%v", collection, len(records), err)
				}
			}
			records, err := app.FindAllRecords("assets")
			if err != nil || len(records) != 1 || records[0].Id != existing.Id || records[0].GetString("file") != existingFile || records[0].GetString("metadata") != existingMetadata {
				t.Fatalf("preexisting asset changed: records=%v error=%v", records, err)
			}
			if created != 0 || published != 0 {
				t.Fatalf("cancelled import reached committed hooks: created=%d published=%d", created, published)
			}
			result, err = ImportTrail(t.Context(), app, item, opts)
			if err != nil || result == nil || !result.Created || result.Skipped {
				t.Fatalf("retry result=%+v error=%v", result, err)
			}
			for collection, want := range map[string]int{"trails": 1, "trail_external_reference": 1, "trail_assets": 2, "waypoints": 1, "waypoint_assets": 2, "assets": 4} {
				records, err := app.FindAllRecords(collection)
				if err != nil || len(records) != want {
					t.Fatalf("retry %s records=%d want=%d error=%v", collection, len(records), want, err)
				}
			}
			trail, err := app.FindRecordById("trails", result.TrailID)
			if err != nil || !trail.GetBool("public") || created != 1 || published != 1 {
				t.Fatalf("retry visibility/hooks trail=%v error=%v created=%d published=%d", trail, err, created, published)
			}
			again, err := ImportTrail(t.Context(), app, item, opts)
			if err != nil || again == nil || !again.Skipped || again.TrailID != result.TrailID {
				t.Fatalf("repeat import result=%+v error=%v", again, err)
			}
		})
	}
}
