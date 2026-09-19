package importer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestPhotoAssetCopyImportHonorsOnlyConfiguredItemLimits(t *testing.T) {
	originalFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = originalFetch })
	downloads := 0
	fetchPublicPluginMedia = func(_ context.Context, source string, _ int64) (*util.SafeFetchResult, error) {
		downloads++
		return &util.SafeFetchResult{Body: []byte("photo bytes"), ContentType: "image/jpeg", FinalURL: source}, nil
	}
	photos := make([]pluginsystem.Photo, 25)
	for i := range photos {
		id := fmt.Sprintf("photo-%d", i)
		photos[i] = pluginsystem.Photo{
			ExternalID: id, Filename: id + ".jpg",
			Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/" + id + ".jpg"},
		}
	}
	for _, test := range []struct {
		name   string
		limits *PhotoImportLimits
		want   int
	}{
		{name: "manual selection has no automatic item limit", want: 25},
		{name: "configured limit can exceed twenty", limits: &PhotoImportLimits{MaxPhotosPerTrail: 25}, want: 25},
		{name: "automatic import respects configured limit", limits: &PhotoImportLimits{MaxPhotosPerTrail: 3}, want: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newPhotoAssetImportTestApp(t)
			downloads = 0
			records, omitted, err := ImportPhotoAssetsWithOmissions(context.Background(), app, photos, Options{
				ActorID: "actor", PhotoLimits: test.limits, Manifest: pluginsystem.Manifest{ID: "immich"},
			}, PhotoAssetTarget{})
			if err != nil {
				t.Fatal(err)
			}
			if len(records) != test.want || downloads != test.want || len(omitted) != len(photos)-test.want {
				t.Fatalf("got %d records, %d downloads and %d omissions; want %d imports and %d omissions", len(records), downloads, len(omitted), test.want, len(photos)-test.want)
			}
			stored, err := app.FindAllRecords("assets")
			if err != nil || len(stored) != test.want {
				t.Fatalf("expected %d persisted assets, got %d: %v", test.want, len(stored), err)
			}
		})
	}
}

func TestPhotoAssetImportReportsFailedDownloadsAndKeepsSuccessfulPhotos(t *testing.T) {
	originalFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = originalFetch })
	fetchPublicPluginMedia = func(_ context.Context, source string, _ int64) (*util.SafeFetchResult, error) {
		if strings.Contains(source, "failed") {
			return nil, errors.New("HTTP 403: https://media.example/photo?api_key=secret")
		}
		return &util.SafeFetchResult{Body: []byte("photo bytes"), ContentType: "image/jpeg", FinalURL: source}, nil
	}

	for _, ids := range [][]string{{"failed"}, {"first", "failed", "last"}} {
		t.Run(strings.Join(ids, "-"), func(t *testing.T) {
			app := newPhotoAssetImportTestApp(t)
			photos := make([]pluginsystem.Photo, 0, len(ids))
			for _, id := range ids {
				photos = append(photos, pluginsystem.Photo{
					ExternalID: id, Filename: id + ".jpg",
					Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/" + id + ".jpg"},
				})
			}
			records, omitted, err := ImportPhotoAssetsWithOmissions(context.Background(), app, photos, Options{ActorID: "actor", Manifest: pluginsystem.Manifest{ID: "immich"}}, PhotoAssetTarget{})
			if err != nil {
				t.Fatal(err)
			}
			if len(records) != len(ids)-1 || len(omitted) != 1 || omitted[0].AssetID != "failed" {
				t.Fatalf("unexpected outcome: %d records, omissions %#v", len(records), omitted)
			}
			if !strings.Contains(omitted[0].Reason, "download failed") || strings.Contains(omitted[0].Reason, "secret") || strings.Contains(omitted[0].Reason, "https://") {
				t.Fatalf("unexpected public reason: %q", omitted[0].Reason)
			}
			stored, err := app.FindAllRecords("assets")
			if err != nil || len(stored) != len(records) {
				t.Fatalf("successful photos were not preserved: %d records, %v", len(stored), err)
			}
		})
	}
}

func TestPhotoAssetImportKeepsRecordsBeforeStorageFailure(t *testing.T) {
	app := newPhotoAssetImportTestApp(t)
	app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("external_id") == "failed" {
			return errors.New("storage unavailable")
		}
		return e.Next()
	})
	photos := []pluginsystem.Photo{
		{ExternalID: "first", Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/first.jpg"}},
		{ExternalID: "failed", Source: pluginsystem.MediaSource{Type: "url", URL: "https://media.example/failed.jpg"}},
	}
	records, omitted, err := ImportPhotoAssetsWithOmissions(context.Background(), app, photos, Options{ActorID: "actor", PhotoMode: "link_private", Manifest: pluginsystem.Manifest{ID: "immich"}}, PhotoAssetTarget{})
	if err == nil || len(records) != 1 || records[0].GetString("external_id") != "first" || len(omitted) != 1 || omitted[0].AssetID != "failed" {
		t.Fatalf("unexpected partial outcome: %d records, %#v, %v", len(records), omitted, err)
	}
	if _, err := app.FindRecordById("assets", records[0].Id); err != nil {
		t.Fatalf("successful photo was lost: %v", err)
	}
}

func newPhotoAssetImportTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(
		&core.TextField{Name: "type"},
		&core.TextField{Name: "storage_mode"},
		&core.TextField{Name: "remote_status"},
		&core.TextField{Name: "author"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.JSONField{Name: "metadata"},
		&core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1024},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	if err := app.Save(assets); err != nil {
		t.Fatal(err)
	}
	return app
}
