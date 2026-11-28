package routes

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pocketbase/hooks"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func repairedSummitPhotoFixture(t *testing.T) (*directedShareFixture, *core.Record, *core.Record) {
	t.Helper()
	f := newDirectedShareFixture(t)
	f.trail.Set("public", true)
	if err := f.app.Save(f.trail); err != nil {
		t.Fatal(err)
	}
	contributor := saveRemoteChildRecord(t, f.app, "activitypub_actors", map[string]any{
		"username": "riley@b.example", "preferred_username": "riley", "domain": "b.example", "public_key": "key",
		"iri": "https://b.example/api/v1/activitypub/user/riley", "inbox": "https://b.example/inbox/riley", "last_fetched": time.Now(),
	})
	entry := saveRemoteChildRecord(t, f.app, "summit_logs", map[string]any{
		"author": f.remote.Id, "trail": f.trail.Id, "text": "Earlier entry", "iri": "https://b.example/api/v1/summit-log/entry0000000001",
	})
	f.app.OnRecordCreate("assets").BindFunc(hooks.StoreAssetSourceMetadata())
	f.app.OnRecordUpdate("assets").BindFunc(hooks.StoreAssetSourceMetadata())
	return f, contributor, entry
}

func TestRemoteSummitContributorPhotoRepairAdoptsOnceAndPreservesLegacyURL(t *testing.T) {
	f, contributor, entry := repairedSummitPhotoFixture(t)
	var imageBody bytes.Buffer
	if err := png.Encode(&imageBody, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	const sourceName = "summit.png"
	file, err := filesystem.NewFileFromBytes(imageBody.Bytes(), sourceName)
	if err != nil {
		t.Fatal(err)
	}
	file.Name = util.FederatedSourceFileName(sourceName)
	asset, err := util.CreatePhotoAsset(f.app, util.PhotoAssetInput{
		Author: f.remote.Id, SummitLog: entry.Id, File: file,
		Metadata: map[string]any{"source_collection": "summit_logs", "source_record": entry.Id, "source_file": file.Name},
	})
	if err != nil {
		t.Fatal(err)
	}
	originalFile := asset.GetString("file")
	originalLink, err := f.app.FindFirstRecordByData("summit_log_assets", "summit_log", entry.Id)
	if err != nil {
		t.Fatal(err)
	}
	originalOrder := originalLink.GetString("created")
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	router.BindFunc(LegacyPhotoFileRedirect)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	assertLegacyURL := func() {
		t.Helper()
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/files/summit_logs/"+entry.Id+"/"+originalFile, nil))
		location := response.Header().Get("Location")
		if response.Code != http.StatusTemporaryRedirect || !strings.Contains(location, "/"+asset.Id+"/") {
			t.Fatalf("repaired entry lost its legacy photo URL: status=%d location=%s body=%s", response.Code, location, response.Body.String())
		}
		download := httptest.NewRecorder()
		mux.ServeHTTP(download, httptest.NewRequest(http.MethodGet, location, nil))
		if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), imageBody.Bytes()) {
			t.Fatalf("legacy photo redirect lost its stored bytes: status=%d", download.Code)
		}
	}
	assertLegacyURL()
	photoLinks := []any{map[string]any{"expand": map[string]any{"asset": map[string]any{
		"id": "photo0000000001", "type": "photo", "file": sourceName, "collectionId": "assets",
	}}}}
	payload := f.payload()
	payload["public"] = true
	entryData := map[string]any{
		"id": "entry0000000001", "iri": entry.GetString("iri"), "text": "Updated contributor entry",
		"expand": map[string]any{"author": map[string]any{"iri": contributor.GetString("iri")}, "summit_log_assets_via_summit_log": photoLinks},
	}
	payload["expand"].(map[string]any)["summit_logs_via_trail"] = []any{entryData}
	f.client(t, func(*http.Request) (map[string]any, int) { return payload, http.StatusOK })
	downloads := 0
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
		downloads++
		return nil, func() {}, errors.New("existing entry photo should not be downloaded")
	}
	for pass := 0; pass < 2; pass++ {
		trail, err := f.app.FindRecordById("trails", f.trail.Id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := performFullSync(f.app, t.Context(), nil, trail); err != nil {
			t.Fatal(err)
		}
		stored, err := f.app.FindRecordById("summit_logs", entry.Id)
		if err != nil || stored.GetString("author") != contributor.Id {
			t.Fatalf("contributor entry author was not repaired: %v", err)
		}
		photos := testPhotoAssets(t, f.app, stored)
		if downloads != 0 || len(photos) != 1 || photos[0].Id != asset.Id || photos[0].GetString("author") != contributor.Id || photos[0].GetString("external_provider") != util.FederationAssetProvider || photos[0].GetString("external_id") != "https://b.example/api/v1/assets/photo0000000001" || photos[0].GetString("file") != originalFile {
			t.Fatalf("contributor repair did not adopt the single migrated photo: pass=%d downloads=%d photos=%v", pass+1, downloads, photos)
		}
		links, err := f.app.FindRecordsByFilter("summit_log_assets", "summit_log={:entry}", "", -1, 0, dbx.Params{"entry": entry.Id})
		if err != nil || len(links) != 1 || links[0].Id != originalLink.Id || links[0].GetString("created") != originalOrder {
			t.Fatalf("contributor repair changed the original photo link or order: %v", err)
		}
		assertLegacyURL()
	}
	entryData["expand"].(map[string]any)["summit_log_assets_via_summit_log"] = []any{}
	trail, err := f.app.FindRecordById("trails", f.trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := performFullSync(f.app, t.Context(), nil, trail); err != nil {
		t.Fatal(err)
	}
	if photos := testPhotoAssets(t, f.app, entry); len(photos) != 0 || downloads != 0 {
		t.Fatalf("origin photo removal did not remove the repaired entry link: photos=%d downloads=%d", len(photos), downloads)
	}
}

func TestRepairedSummitLogPhotosKeepIndependentOwnership(t *testing.T) {
	f, contributor, entry := repairedSummitPhotoFixture(t)
	otherEntry := saveRemoteChildRecord(t, f.app, "summit_logs", map[string]any{"trail": f.trail.Id, "author": f.remote.Id, "text": "Other entry"})
	waypoint := saveRemoteChildRecord(t, f.app, "waypoints", map[string]any{"trail": f.trail.Id, "author": f.remote.Id, "name": "Point", "lat": 47, "lon": 8})
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	assets := map[string]*core.Record{}
	for _, kind := range []string{"entry_only", "generated", "provider", "another_author", "trail_link", "waypoint_link", "entry_link"} {
		file, err := filesystem.NewFileFromBytes(content.Bytes(), kind+".png")
		if err != nil {
			t.Fatal(err)
		}
		asset := testAddPhoto(t, f.app, entry, file, "")
		switch kind {
		case "generated":
			asset.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
		case "provider":
			asset.Set("external_provider", "connector")
			asset.Set("external_id", "photo-source")
		case "another_author":
			asset.Set("author", f.bob.Id)
		case "trail_link":
			saveRemoteChildRecord(t, f.app, "trail_assets", map[string]any{"trail": f.trail.Id, "asset": asset.Id})
		case "waypoint_link":
			saveRemoteChildRecord(t, f.app, "waypoint_assets", map[string]any{"waypoint": waypoint.Id, "asset": asset.Id})
		case "entry_link":
			saveRemoteChildRecord(t, f.app, "summit_log_assets", map[string]any{"summit_log": otherEntry.Id, "asset": asset.Id})
		}
		if err := f.app.Save(asset); err != nil {
			t.Fatal(err)
		}
		assets[kind] = asset
	}
	entry.Set("author", contributor.Id)
	if err := f.app.Save(entry); err != nil {
		t.Fatal(err)
	}
	if err := util.HandOverRepairedSummitLogPhotos(f.app, entry.Id, f.remote.Id, contributor.Id); err != nil {
		t.Fatal(err)
	}
	for kind, asset := range assets {
		stored, err := f.app.FindRecordById("assets", asset.Id)
		expected := asset.GetString("author")
		if kind == "entry_only" {
			expected = contributor.Id
		}
		if err != nil || stored.GetString("author") != expected || stored.GetString("external_provider") != asset.GetString("external_provider") {
			t.Fatalf("entry photo handover changed independent ownership: kind=%s error=%v", kind, err)
		}
	}
}
