package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type assetImportLoggingApp struct {
	core.App
	logger *slog.Logger
}

func (app assetImportLoggingApp) Logger() *slog.Logger {
	return app.logger
}

func TestPartialAssetImportLogsOriginalStorageFailure(t *testing.T) {
	for _, targetImport := range []bool{false, true} {
		t.Run(map[bool]string{false: "waypoint import", true: "target import"}[targetImport], func(t *testing.T) {
			app, auth := newAssetImportRouteTestApp(t)
			storageErr := errors.New("database write failed: storage diagnostic")
			app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
				if e.Record.GetString("external_id") == "failed" {
					return storageErr
				}
				return e.Next()
			})
			collection, err := app.FindCollectionByNameOrId("waypoints")
			if err != nil {
				t.Fatal(err)
			}
			waypoint := core.NewRecord(collection)
			if err := app.Save(waypoint); err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			e := &core.RequestEvent{App: assetImportLoggingApp{App: app, logger: slog.New(slog.NewJSONHandler(&logs, nil))}, Auth: auth}
			e.Request = httptest.NewRequest(http.MethodPost, "/asset-import", nil)
			recorder := httptest.NewRecorder()
			e.Response = recorder
			plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "immich"}}
			config := map[string]any{"host": map[string]any{"photoMode": "link_private"}}
			output := pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{
				{ExternalID: "good", Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/good.jpg"}},
				{ExternalID: "failed", Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/failed.jpg"}},
			}}
			data := pluginAssetLibraryRequest{WaypointID: waypoint.Id}
			if targetImport {
				err = importAssetPluginPhotosToTarget(e, plugin, nil, config, output, data)
			} else {
				err = importAssetPluginPhotos(e, plugin, nil, nil, config, output, data, waypoint.Id)
			}
			if err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || !strings.Contains(logs.String(), storageErr.Error()) || !strings.Contains(logs.String(), `"level":"WARN"`) {
				t.Fatalf("partial import lost diagnostic: HTTP %d, logs %s", recorder.Code, logs.String())
			}
			if strings.Contains(recorder.Body.String(), storageErr.Error()) {
				t.Fatalf("storage diagnostic leaked into response: %s", recorder.Body.String())
			}
		})
	}
}

func TestAssetImportToTargetReportsEveryFailedPhoto(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "all failed", true: "partial success"}[partial], func(t *testing.T) {
			app, auth := newAssetImportRouteTestApp(t)
			photos := []pluginsystem.Photo{{ExternalID: "failed", Source: pluginsystem.MediaSource{Type: "unsupported"}}}
			if partial {
				photos = append(photos, pluginsystem.Photo{ExternalID: "good", Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/good.jpg"}})
			}
			e := &core.RequestEvent{App: app, Auth: auth}
			e.Request = httptest.NewRequest(http.MethodPost, "/plugins/assets/immich/import-to-target", nil)
			recorder := httptest.NewRecorder()
			e.Response = recorder
			err := importAssetPluginPhotosToTarget(e, pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "immich"}}, nil,
				map[string]any{"host": map[string]any{"photoMode": "link_private"}}, pluginAssetLibraryOutput{Photos: photos}, pluginAssetLibraryRequest{})
			if err != nil {
				t.Fatal(err)
			}
			var response struct {
				Imported []json.RawMessage     `json:"imported"`
				Omitted  []pluginAssetOmission `json:"omitted"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || len(response.Imported) != len(photos)-1 || len(response.Omitted) != 1 || response.Omitted[0].AssetID != "failed" || response.Omitted[0].Reason == "" {
				t.Fatalf("unexpected response: HTTP %d %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAssetImportResponseIncludesPhotosNotAttemptedAfterFailure(t *testing.T) {
	response := assetPluginImportResponse(pluginAssetLibraryOutput{
		Photos:        []pluginsystem.Photo{{ExternalID: "good"}, {ExternalID: "failed"}, {ExternalID: "not-attempted"}},
		OmittedAssets: []pluginAssetOmission{{AssetID: "provider-omission", Reason: "unavailable"}},
	}, []pluginAssetImportResult{{AssetID: "good"}}, []pluginAssetOmission{{AssetID: "failed", Reason: "photo could not be stored"}})
	if len(response.Imported) != 1 || len(response.Omitted) != 3 {
		t.Fatalf("unaccounted photos: %#v", response)
	}
	for _, omitted := range response.Omitted {
		if omitted.Reason == "" {
			t.Fatalf("missing reason: %#v", omitted)
		}
	}
	if response.Omitted[2].AssetID != "not-attempted" {
		t.Fatalf("missing unattempted photo: %#v", response.Omitted)
	}
}

func TestAssetImportCleansEmptyWaypointAndPreservesPartialSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"elements":[{"type":"node","id":1,"lat":46,"lon":8,"tags":{"name":"Test viewpoint","tourism":"viewpoint"}}]}`))
	}))
	defer server.Close()
	t.Setenv("OVERPASS_API_URL", server.URL)
	t.Setenv("NOMINATIM_URL", server.URL)
	withWaypointNameHTTPClient(t, server.Client())

	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty download", true: "storage failure after success"}[partial], func(t *testing.T) {
			app, auth := newAssetImportRouteTestApp(t)
			lat, lon := 46.0, 8.0
			photos := []pluginsystem.Photo{{ExternalID: "failed", Lat: &lat, Lon: &lon, Source: pluginsystem.MediaSource{Type: "unsupported"}}}
			if partial {
				photos = []pluginsystem.Photo{
					{ExternalID: "good", Lat: &lat, Lon: &lon, Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/good.jpg"}},
					{ExternalID: "failed", Lat: &lat, Lon: &lon, Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/failed.jpg"}},
				}
				app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
					if e.Record.GetString("external_id") == "failed" {
						return errors.New("storage unavailable")
					}
					return e.Next()
				})
			}
			results, omitted, err := importAssetPluginPhotosForTrail(context.Background(), app, auth.Id,
				pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "immich"}}, nil,
				map[string]any{"host": map[string]any{"photoMode": "link_private"}}, pluginAssetLibraryOutput{Photos: photos}, pluginAssetLibraryRequest{}, "", true)
			if (err != nil) != partial || len(results) != len(photos)-1 || len(omitted) != 1 {
				t.Fatalf("unexpected import result: %d results, %#v, %v", len(results), omitted, err)
			}
			waypoints, err := app.FindAllRecords("waypoints")
			if err != nil || len(waypoints) != len(results) {
				t.Fatalf("incorrect waypoint cleanup: %d waypoints for %d results, %v", len(waypoints), len(results), err)
			}
			links, err := app.FindAllRecords("waypoint_assets")
			if err != nil || len(links) != len(results) {
				t.Fatalf("incorrect photo links: %d links for %d results, %v", len(links), len(results), err)
			}
		})
	}
}

func newAssetImportRouteTestApp(t *testing.T) (*pbtests.TestApp, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	auth := core.NewRecord(core.NewAuthCollection("test_users"))
	auth.Id = "test-user"
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "user"})
	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(
		&core.TextField{Name: "type"}, &core.TextField{Name: "storage_mode"},
		&core.TextField{Name: "remote_status"}, &core.TextField{Name: "author"},
		&core.TextField{Name: "external_provider"}, &core.TextField{Name: "external_id"},
		&core.JSONField{Name: "metadata"}, &core.FileField{Name: "file", MaxSelect: 1},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	waypoints := core.NewBaseCollection("waypoints")
	waypoints.Fields.Add(
		&core.TextField{Name: "name"}, &core.TextField{Name: "author"},
		&core.TextField{Name: "trail"}, &core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"}, &core.NumberField{Name: "distance_from_start"},
	)
	links := core.NewBaseCollection("waypoint_assets")
	links.Fields.Add(&core.TextField{Name: "waypoint"}, &core.TextField{Name: "asset"})
	for _, collection := range []*core.Collection{actors, assets, waypoints, links} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	actor := core.NewRecord(actors)
	actor.Set("user", auth.Id)
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	return app, auth
}
