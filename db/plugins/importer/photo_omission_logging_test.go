package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/pluginsystem"
	"pocketbase/util"
)

type photoOmissionLogApp struct {
	core.App
	log *slog.Logger
}

func (app photoOmissionLogApp) Logger() *slog.Logger { return app.log }

func photoOmissionLogs(t *testing.T, output *bytes.Buffer) []map[string]any {
	t.Helper()
	result := []map[string]any{}
	decoder := json.NewDecoder(output)
	for decoder.More() {
		row := map[string]any{}
		if err := decoder.Decode(&row); err != nil {
			t.Fatal(err)
		}
		if row["msg"] == "plugin photo omitted" {
			result = append(result, row)
		}
	}
	return result
}

func TestPhotoOmissionsLogEveryReasonWithImportContext(t *testing.T) {
	oldFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = oldFetch })
	for _, test := range []struct {
		name, reason string
		limit        int
		budget       int64
		noFile       bool
		failure      error
	}{
		{name: "target limit", reason: "photo_limit_reached", limit: 1},
		{name: "aggregate budget", reason: "media_budget_exhausted", budget: util.DefaultPluginMaxImportMediaBytes},
		{name: "missing photo", reason: "not_found", failure: util.HTTPStatusError{StatusCode: 404}},
		{name: "denied photo", reason: "access_denied", failure: util.HTTPStatusError{StatusCode: 403}},
		{name: "large photo status", reason: "photo_too_large", failure: util.HTTPStatusError{StatusCode: 413}},
		{name: "large photo body", reason: "photo_too_large", failure: util.ErrPluginMediaTooLarge},
		{name: "unsupported media status", reason: "unsupported_media", failure: util.HTTPStatusError{StatusCode: 415}},
		{name: "unsupported media bytes", reason: "unsupported_media", failure: errPhotoUnsupported},
		{name: "other download failure", reason: "download_failed", failure: errors.New("photo unavailable")},
		{name: "no file", reason: "no_file", noFile: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := newPhotoAssetImportTestApp(t)
			var output bytes.Buffer
			app := photoOmissionLogApp{App: base, log: slog.New(slog.NewJSONHandler(&output, nil))}
			fetchPublicPluginMedia = func(_ context.Context, source string, _ int64) (*util.SafeFetchResult, error) {
				if test.noFile {
					return nil, nil
				}
				if test.limit > 0 {
					return &util.SafeFetchResult{Body: []byte("photo"), FinalURL: source}, nil
				}
				return nil, test.failure
			}
			photos := []pluginsystem.Photo{
				{ExternalID: "first", Filename: "first.jpg", Source: pluginsystem.MediaSource{Type: "url", URL: "https://b.example/first.jpg"}},
				{ExternalID: "second", Filename: "second.jpg", Source: pluginsystem.MediaSource{Type: "url", URL: "https://b.example/second.jpg"}},
			}
			_, omitted, err := preparePhotoAssets(context.Background(), app, photos, Options{maxImportedPhotosPerTarget: test.limit}, &pluginMediaBudget{bytes: test.budget}, photoImportContext{Provider: "komoot", TrailExternalID: "route-42", TrailName: "Mountain walk", WaypointName: "Summit"})
			if err != nil {
				t.Fatal(err)
			}
			rows := photoOmissionLogs(t, &output)
			want := 2
			if test.limit > 0 {
				want = 1
			}
			if len(omitted) != want || len(rows) != want {
				t.Fatalf("omitted=%v logs=%v want=%d", omitted, rows, want)
			}
			for _, row := range rows {
				if row["provider"] != "komoot" || row["trail_external_id"] != "route-42" || row["trail_name"] != "Mountain walk" || row["waypoint_name"] != "Summit" || row["reason"] != test.reason || row["external_id"] == "" {
					t.Fatalf("incomplete omission context: %v", row)
				}
			}
		})
	}
}

func TestImportTrailLogsPhotoOmissionsForTrailAndWaypoint(t *testing.T) {
	base, actor := newTrailPhotoLimitTestApp(t)
	var output bytes.Buffer
	app := photoOmissionLogApp{App: base, log: slog.New(slog.NewJSONHandler(&output, nil))}
	oldFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = oldFetch })
	fetchPublicPluginMedia = func(_ context.Context, source string, _ int64) (*util.SafeFetchResult, error) {
		status := 404
		if strings.Contains(source, "waypoint") {
			status = 403
		}
		return nil, util.HTTPStatusError{StatusCode: status}
	}
	photo := func(id string) pluginsystem.Photo {
		return pluginsystem.Photo{ExternalID: id, Source: pluginsystem.MediaSource{Type: "url", URL: "https://b.example/" + id + ".jpg"}}
	}
	item := pluginsystem.TrailImport{Source: pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "route-42"}, Name: "Mountain walk", Track: gpxTrack(), Photos: []pluginsystem.Photo{photo("trail-photo")}, Waypoints: []pluginsystem.Waypoint{{Name: "Summit", Lat: 46, Lon: 8, Photos: []pluginsystem.Photo{photo("waypoint-photo")}}}}
	if _, err := ImportTrail(context.Background(), app, item, Options{UserID: "owner", ActorID: actor.Id, Manifest: pluginsystem.Manifest{ID: "komoot", Type: pluginsystem.PluginTypeTrails}}); err != nil {
		t.Fatal(err)
	}
	rows := photoOmissionLogs(t, &output)
	if len(rows) != 2 {
		t.Fatalf("omission logs=%v", rows)
	}
	for i, row := range rows {
		waypoint, reason := "", "not_found"
		if i == 1 {
			waypoint, reason = "Summit", "access_denied"
		}
		if row["provider"] != "komoot" || row["trail_external_id"] != "route-42" || row["trail_name"] != "Mountain walk" || row["waypoint_name"] != waypoint || row["reason"] != reason {
			t.Fatalf("omission log=%v", row)
		}
	}
}
