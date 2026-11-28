package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAssetDownloadsProjectRowsWhileKeepingPocketBaseGuardsAndServing(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("download-owner")
	other := f.localActor("download-other")
	trail := f.save("trails", map[string]any{"name": "Public", "author": owner.Id, "public": true})
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	file, err := filesystem.NewFileFromBytes(png, "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": file})
	f.save("trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id})
	metadata, _ := json.Marshal(map[string]any{"unrelated": strings.Repeat("EXIF", 250000)})
	source, _ := json.Marshal(map[string]any{"merged_assets": []any{map[string]any{"asset_id": "previous", "metadata": map[string]any{"notes": strings.Repeat("HISTORY", 800000)}}}, "source_collection": "trails", "source_record": trail.Id, "source_file": "old.png"})
	if _, err := f.app.DB().NewQuery("UPDATE assets SET metadata={:metadata},source_metadata={:source} WHERE id={:id}").Bind(dbx.Params{"metadata": string(metadata), "source": string(source), "id": asset.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.DB().NewQuery("INSERT INTO _asset_legacy_photo_aliases(asset,source_collection,source_record,source_file) VALUES({:asset},'trails',{:trail},'old.png')").Bind(dbx.Params{"asset": asset.Id, "trail": trail.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	api, err := newAssetDownloadRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	registerRoutes(&core.ServeEvent{App: f.app, Router: api}, nil)
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	queries := []string{}
	for _, database := range []*dbx.DB{f.app.ConcurrentDB().(*dbx.DB), f.app.NonconcurrentDB().(*dbx.DB)} {
		db := database
		previous := db.QueryLogFunc
		db.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
			if strings.Contains(query, "FROM `assets`") {
				mu.Lock()
				queries = append(queries, query)
				mu.Unlock()
			}
		}
		defer func() { db.QueryLogFunc = previous }()
	}
	auth := func(actor *core.Record) string {
		user, err := f.app.FindRecordById("users", actor.GetString("user"))
		if err != nil {
			t.Fatal(err)
		}
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	ownerAuth, otherAuth := auth(owner), auth(other)
	request := func(method, path, auth string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		wantCache := "no-store"
		if response.Code == 200 || response.Code == 206 || response.Code == 304 || response.Code == 302 || response.Code == 307 {
			wantCache = "private, max-age=3600"
		}
		if got := response.Header().Get("Cache-Control"); got != wantCache {
			t.Fatalf("asset response%d Cache-Control=%q want=%q", response.Code, got, wantCache)
		}
		return response
	}
	native := "/api/files/assets/" + asset.Id + "/" + asset.GetString("file")
	for _, path := range []string{native, "/assets/" + asset.Id + "/file", "/api/files/trails/" + trail.Id + "/old.png"} {
		response := request(http.MethodGet, path, "", nil)
		if response.Header().Get("Cache-Control") != "private, max-age=3600" {
			t.Fatalf("photo response cache header: %s", response.Header().Get("Cache-Control"))
		}
		if path == native {
			if response.Code != 200 || response.Body.String() != string(png) {
				t.Fatal("native bytes changed", response.Code, response.Body.String())
			}
		} else if response.Code != 302 && response.Code != 307 {
			t.Fatal("redirect changed", path, response.Code, response.Body.String())
		}
		if path == "/assets/"+asset.Id+"/file" && (response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Content-Security-Policy") != "default-src 'none'; sandbox") {
			t.Fatal("asset redirect response headers changed", response.Header())
		}
	}
	request(http.MethodGet, "/assets/nonexistent0000/file", "", nil)
	missingFile := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy"})
	if response := request(http.MethodGet, "/assets/"+missingFile.Id+"/file", ownerAuth, nil); response.Code != 404 {
		t.Fatal("missing asset file redirect", response.Code)
	}
	invalid := httptest.NewRequest(http.MethodGet, "/assets/"+asset.Id+"/file", strings.NewReader("{"))
	invalid.Header.Set("Content-Type", "application/json")
	failed := httptest.NewRecorder()
	mux.ServeHTTP(failed, invalid)
	if failed.Code != 500 || failed.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("asset redirect failure cache/status=%d %q", failed.Code, failed.Header().Get("Cache-Control"))
	}
	ranged := request(http.MethodGet, native, "", map[string]string{"Range": "bytes=0-7"})
	if ranged.Code != 206 || ranged.Body.String() != string(png[:8]) || ranged.Header().Get("Content-Range") == "" {
		t.Fatal("PB Range semantics lost", ranged.Code, ranged.Header())
	}
	head := request(http.MethodHead, native, "", nil)
	if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") == "" {
		t.Fatal("PB HEAD semantics lost", head.Code, head.Header())
	}
	modified := head.Header().Get("Last-Modified")
	if modified == "" {
		t.Fatal("Last-Modified lost")
	}
	if response := request(http.MethodGet, native, "", map[string]string{"If-Modified-Since": modified}); response.Code != 304 {
		t.Fatal("conditional file read changed", response.Code)
	}
	thumb := request(http.MethodGet, native+"?thumb=100x100", "", nil)
	if thumb.Code != 200 || thumb.Body.Len() == 0 {
		t.Fatal("native PB thumbnail generation lost", thumb.Code, thumb.Body.String())
	}
	mu.Lock()
	observed := append([]string(nil), queries...)
	mu.Unlock()
	if len(observed) < 5 {
		t.Fatal("actual media row queries missing", observed)
	}
	for _, query := range observed {
		if strings.Contains(query, "`assets`.*") || strings.Contains(query, "source_metadata") {
			t.Fatal("download loaded full private asset history", query)
		}
	}
	trail.Set("public", false)
	if err := f.app.Save(trail); err != nil {
		t.Fatal(err)
	}
	if response := request(http.MethodGet, "/assets/"+asset.Id+"/file", "", nil); response.Code != 404 {
		t.Fatal("private asset redirect readable", response.Code)
	}
	if response := request(http.MethodGet, native, "", nil); response.Code != 404 {
		t.Fatal("revoked public file still readable", response.Code)
	}
	if response := request(http.MethodGet, native, otherAuth, nil); response.Code != 404 {
		t.Fatal("other user read private bytes", response.Code)
	}
	if response := request(http.MethodGet, native, ownerAuth, nil); response.Code != 200 {
		t.Fatal("owner denied private bytes", response.Code)
	}
	collection, err := f.app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	field := collection.Fields.GetByName("file").(*core.FileField)
	field.Protected = true
	if err := f.app.Save(collection); err != nil {
		t.Fatal(err)
	}
	user, err := f.app.FindRecordById("users", owner.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	fileToken, err := user.NewFileToken()
	if err != nil {
		t.Fatal(err)
	}
	if response := request(http.MethodGet, native, ownerAuth, nil); response.Code != 404 {
		t.Fatal("protected file ignored file-token guard", response.Code)
	}
	if response := request(http.MethodGet, native+"?token=invalid", ownerAuth, nil); response.Code != 404 {
		t.Fatal("invalid file token accepted", response.Code)
	}
	if response := request(http.MethodGet, native+"?token="+fileToken, ownerAuth, nil); response.Code != 200 {
		t.Fatal("valid PB file token failed", response.Code, response.Body.String())
	}
	projected, err := util.FindAssetMediaRecord(f.app, asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(projected.GetString("metadata")) > 100 || (projected.GetString("source_metadata") != "" && projected.GetString("source_metadata") != "null") {
		t.Fatal("media projection retained unrelated payload")
	}
}
