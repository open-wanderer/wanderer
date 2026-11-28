package routes

import (
	"context"
	"fmt"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"net/http"
	"pocketbase/util"
	"testing"
)

const (
	placeholderTestOrigin = "https://a.example"
	placeholderTestIRI    = placeholderTestOrigin + "/api/v1/trail/remote"
	originPlaceholder     = "wanderer_route_placeholder_abc1234567.svg"
)

func newRoutePlaceholderSyncFixture(t *testing.T) *remoteSyncFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteSyncFixture(t)
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
		file, err := filesystem.NewFileFromBytes([]byte(name), name)
		return file, func() {}, err
	}
	return f
}

func TestRemoteMediaCollectorSkipsOriginPlaceholder(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t)
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "trail", "remote", placeholderTestOrigin, placeholderTestIRI, map[string]any{"photos": []any{originPlaceholder, "summit.jpg"}})
	if len(collector.requests) != 1 || collector.requests[0].name != "summit.jpg" {
		t.Fatalf("requests=%v", collector.requests)
	}
}

func TestRemoteMediaCollectorPlaceholderLeavesFileBudget(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t)
	collector := remoteMediaCollector{seen: map[string]bool{}, requests: make([]remoteMediaRequest, remoteMediaMaxFiles-1)}
	collector.record(f.app, "trail", "remote", placeholderTestOrigin, placeholderTestIRI, map[string]any{"photos": []any{originPlaceholder, "summit.jpg"}})
	if len(collector.requests) != remoteMediaMaxFiles || collector.requests[len(collector.requests)-1].name != "summit.jpg" {
		t.Fatalf("requests=%v", collector.requests)
	}
}

// Prepared origin previews must still be skipped by photo reconciliation.
func TestSyncRecordPhotosSkipsOriginPlaceholder(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t)
	record := f.save(f.trails, map[string]any{"iri": placeholderTestIRI, "author": f.remoteActor.Id})
	requests := []remoteMediaRequest{}
	for _, name := range []string{originPlaceholder, "summit.jpg"} {
		requests = append(requests, remoteMediaRequest{url: placeholderTestOrigin + "/api/v1/files/trails/remote/" + name, origin: placeholderTestOrigin, name: name, maxBytes: 1 << 20})
	}
	ctx, cleanup, err := prepareRemoteMedia(t.Context(), requests)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if err := syncRecordPhotos(f.app, ctx, "trail", record.Id, "remote", f.remoteActor.Id, placeholderTestOrigin, map[string]any{"photos": []any{originPlaceholder, "summit.jpg"}}, true); err != nil {
		t.Fatal(err)
	}
	assets := testPhotoAssets(t, f.app, record)
	if len(assets) != 1 || assets[0].GetString("external_id") != placeholderTestOrigin+"/api/v1/files/trails/remote/summit.jpg" {
		t.Fatalf("assets=%v", assets)
	}
}

func TestSyncKeepsLocalPlaceholder(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t)
	local, err := filesystem.NewFileFromBytes([]byte("<svg/>"), "wanderer_route_placeholder.svg")
	if err != nil {
		t.Fatal(err)
	}
	record := f.save(f.trails, map[string]any{"iri": placeholderTestIRI, "author": f.remoteActor.Id})
	preview := testAddPhoto(t, f.app, record, local, "")
	preview.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
	if err := f.app.Save(preview); err != nil {
		t.Fatal(err)
	}
	if !util.IsGeneratedRoutePreviewAsset(preview) {
		t.Fatal("fixture is not generated")
	}
	for _, name := range []string{originPlaceholder, "wanderer_route_placeholder_def7654321.svg"} {
		data := map[string]any{"photos": []any{name}}
		collector := remoteMediaCollector{seen: map[string]bool{}}
		collector.record(f.app, "trail", "remote", placeholderTestOrigin, placeholderTestIRI, data)
		ctx, cleanup, err := prepareRemoteMedia(t.Context(), collector.requests)
		if err != nil {
			cleanup()
			t.Fatal(err)
		}
		err = syncRecordPhotos(f.app, ctx, "trail", record.Id, "remote", f.remoteActor.Id, placeholderTestOrigin, data, true)
		cleanup()
		if err != nil {
			t.Fatal(err)
		}
	}
	assets := testPhotoAssets(t, f.app, record)
	if len(assets) != 1 || assets[0].Id != preview.Id {
		t.Fatalf("local preview changed: %v", assets)
	}
}

func TestFullSyncKeepsOnlyItsLocalGeneratedThumbnail(t *testing.T) {
	f := newRoutePlaceholderSyncFixture(t)
	f.remoteActor.Set("iri", placeholderTestOrigin+"/api/v1/actor/riley")
	f.remoteActor.Set("domain", "a.example")
	if err := f.app.Save(f.remoteActor); err != nil {
		t.Fatal(err)
	}
	record := f.save(f.trails, map[string]any{"iri": placeholderTestIRI, "author": f.remoteActor.Id, "public": true})
	local, err := filesystem.NewFileFromBytes([]byte("<svg/>"), "local.svg")
	if err != nil {
		t.Fatal(err)
	}
	preview := testAddPhoto(t, f.app, record, local, "")
	preview.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
	if err := f.app.Save(preview); err != nil {
		t.Fatal(err)
	}
	if err := util.SetTrailThumbnailAsset(f.app, record.Id, preview.Id); err != nil {
		t.Fatal(err)
	}
	refreshTransport(t, func(*http.Request) map[string]any {
		return map[string]any{"id": "remote", "iri": placeholderTestIRI, "name": "Remote", "public": true, "expand": map[string]any{"trail_assets_via_trail": []any{map[string]any{"is_thumbnail": true, "expand": map[string]any{"asset": map[string]any{"id": "originpreview", "type": "photo", "file": "map.webp", "collectionId": "assets", "metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}}}}}}}}
	})
	downloads := 0
	fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
		downloads++
		return nil, func() {}, fmt.Errorf("unexpected download")
	}
	if _, err := performFullSync(f.app, t.Context(), nil, record); err != nil {
		t.Fatal(err)
	}
	photos := testPhotoAssets(t, f.app, record)
	thumbnail, err := f.app.FindFirstRecordByData("trail_assets", "asset", preview.Id)
	if downloads != 0 || len(photos) != 1 || photos[0].Id != preview.Id || err != nil || thumbnail == nil || !thumbnail.GetBool("is_thumbnail") {
		t.Fatalf("local preview changed: downloads=%d photos=%v thumbnail=%v error=%v", downloads, photos, thumbnail, err)
	}
}
