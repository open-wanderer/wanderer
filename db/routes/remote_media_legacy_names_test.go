package routes

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestRemoteMediaRecognizesEarlierCopiesOfCameraFiles(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(content.Bytes())
	}))
	defer server.Close()
	restore := util.SetRemoteAttachmentClientForTesting(server.Client())
	defer restore()
	previous := fetchRemoteSyncMedia
	defer func() { fetchRemoteSyncMedia = previous }()
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		for _, uploadName := range []string{
			"PXL_20261018_090000000.MP.jpg",
			"IMG_20261018_090000000.PORTRAIT.jpg",
			"2026.10.18.09.10.11.jpg",
			"WhatsApp Image 2026-10-18 at 09.10.11.jpeg",
			"Screenshot 2026-10-18 at 09.10.11.png",
			"mountain.jpg",
		} {
			for _, copyKind := range []string{"legacy", "inbound_create"} {
				t.Run(target+"/"+uploadName+"/"+copyKind, func(t *testing.T) {
					f := newRemoteSyncFixture(t)
					f.remoteActor.Set("iri", "https://b.example/api/v1/actor/riley")
					f.remoteActor.Set("domain", "b.example")
					if err := f.app.Save(f.remoteActor); err != nil {
						t.Fatal(err)
					}
					collection := map[string]*core.Collection{"trail": f.trails, "waypoint": f.waypoints, "summit_log": f.logs}[target]
					source, err := filesystem.NewFileFromBytes(content.Bytes(), uploadName)
					if err != nil {
						t.Fatal(err)
					}
					var copied *filesystem.File
					if copyKind == "legacy" {
						copied, err = filesystem.NewFileFromBytes(content.Bytes(), source.Name)
					} else {
						var cleanup func()
						copied, cleanup, err = util.DownloadRemoteFile(t.Context(), server.URL+"/"+source.Name, 1<<20)
						if cleanup != nil {
							defer cleanup()
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					origin := "https://b.example"
					iri := origin + "/api/v1/" + target + "/remote"
					record := f.save(collection, map[string]any{"iri": iri, "author": f.remoteActor.Id})
					identity := origin + "/api/v1/files/" + collection.Name + "/remote/" + source.Name
					if copyKind == "legacy" {
						identity = ""
					}
					asset := testAddPhoto(t, f.app, record, copied, identity)
					original := asset.GetString("file")
					requests := 0
					fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
						requests++
						file, err := filesystem.NewFileFromBytes(content.Bytes(), name)
						return file, func() {}, err
					}
					data := map[string]any{"photos": []any{source.Name}}
					for attempt := 0; attempt < 2; attempt++ {
						collector := remoteMediaCollector{seen: map[string]bool{}}
						collector.record(f.app, target, "remote", origin, iri, data)
						ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
						if err != nil {
							t.Fatal(err)
						}
						if err := syncRecordPhotos(f.app, ctx, target, record.Id, "remote", f.remoteActor.Id, origin, data, true); err != nil {
							t.Fatal(err)
						}
						if err := f.app.Save(record); err != nil {
							cleanup()
							t.Fatal(err)
						}
						cleanup()
						record, err = f.app.FindRecordById(collection, record.Id)
						if err != nil {
							t.Fatal(err)
						}
					}
					stored := testPhotoFiles(t, f.app, record)
					t.Logf("source=%q copy=%q requests=%d stored=%q", source.Name, original, requests, stored)
					if requests != 0 || len(stored) != 1 || stored[0] != original {
						t.Fatalf("existing copy was downloaded again: requests=%d stored=%q", requests, stored)
					}
				})
			}
		}
	}
}

func TestModernPhotoAdoptsExistingOriginFileWithoutDownload(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	for _, kind := range []string{"migration", "legacy_identity"} {
		t.Run(kind, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			f.remoteActor.Set("iri", "https://b.example/api/v1/actor/riley")
			f.remoteActor.Set("domain", "b.example")
			if err := f.app.Save(f.remoteActor); err != nil {
				t.Fatal(err)
			}
			origin := "https://b.example"
			record := f.save(f.trails, map[string]any{"iri": origin + "/api/v1/trail/remote", "author": f.remoteActor.Id})
			file, err := filesystem.NewFileFromBytes([]byte("photo bytes"), "camera.jpg")
			if err != nil {
				t.Fatal(err)
			}
			asset := testAddPhoto(t, f.app, record, file, "")
			name := asset.GetString("file")
			if kind == "legacy_identity" {
				asset.Set("external_provider", util.FederationAssetProvider)
				asset.Set("external_id", origin+"/api/v1/files/trails/remote/"+name)
				if err := f.app.Save(asset); err != nil {
					t.Fatal(err)
				}
			}
			data := map[string]any{"expand": map[string]any{"trail_assets_via_trail": []any{map[string]any{"expand": map[string]any{"asset": map[string]any{"id": "modernphoto", "type": "photo", "file": name, "collectionId": "assets"}}}}}}
			collector := remoteMediaCollector{seen: map[string]bool{}}
			collector.record(f.app, "trail", "remote", origin, record.GetString("iri"), data)
			if len(collector.requests) != 0 {
				t.Fatalf("existing photo requested again: %v", collector.requests)
			}
			ctx := util.WithPreparedFederatedFiles(t.Context(), nil)
			if err := syncRecordPhotos(f.app, ctx, "trail", record.Id, "remote", f.remoteActor.Id, origin, data, true); err != nil {
				t.Fatal(err)
			}
			photos := testPhotoAssets(t, f.app, record)
			if len(photos) != 1 || photos[0].Id != asset.Id || photos[0].GetString("external_id") != origin+"/api/v1/assets/modernphoto" {
				t.Fatalf("existing photo not adopted: %v", photos)
			}
		})
	}
}
