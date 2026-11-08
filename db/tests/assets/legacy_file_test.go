package assets_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/hooks"
	"pocketbase/routes"
)

func TestMigratedLegacyPhotoURLsRecheckCurrentAccess(t *testing.T) {
	for _, collection := range []string{"trails", "waypoints", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			app := newAssetFilterTestApp(t)
			newActor := func(name string) (*core.Record, *core.Record) {
				user := saveAssetFilterRecord(t, app, "users", map[string]any{"username": name, "email": name + "@a.example", "password": "test-password"})
				actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{"username": name, "preferred_username": name, "domain": "a.example", "public_key": "key", "is_local": true, "iri": "https://a.example/" + name, "inbox": "https://a.example/" + name + "/inbox", "user": user.Id})
				return user, actor
			}
			owner, ownerActor := newActor("owner")
			_, creator := newActor("creator")
			entryOwner, entryActor := newActor("entry")
			trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "Legacy", "author": ownerActor.Id, "public": true})
			target := trail
			linkCollection, targetField := "trail_assets", "trail"
			photoOwner := ownerActor
			if collection == "waypoints" {
				target = saveAssetFilterRecord(t, app, collection, map[string]any{"name": "Point", "trail": trail.Id, "author": creator.Id})
				linkCollection, targetField = "waypoint_assets", "waypoint"
			}
			if collection == "summit_logs" {
				target = saveAssetFilterRecord(t, app, collection, map[string]any{"trail": trail.Id, "author": entryActor.Id, "text": "Entry"})
				linkCollection, targetField = "summit_log_assets", "summit_log"
				owner, photoOwner = entryOwner, entryActor
			}
			photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
			if err != nil {
				t.Fatal(err)
			}
			file, err := filesystem.NewFileFromBytes(photo, "photo.png")
			if err != nil {
				t.Fatal(err)
			}
			asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"author": photoOwner.Id, "type": "photo", "storage_mode": "copy", "file": file, "metadata": map[string]any{"source_collection": collection, "source_record": target.Id, "source_file": "old_photo.png"}})
			saveAssetFilterRecord(t, app, linkCollection, map[string]any{targetField: target.Id, "asset": asset.Id})
			app.OnFileDownloadRequest("assets").BindFunc(hooks.AuthorizeAssetFileDownload())
			router, err := apis.NewRouter(app)
			if err != nil {
				t.Fatal(err)
			}
			router.BindFunc(routes.LegacyPhotoFileRedirect)
			mux, err := router.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			token, err := owner.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			path := "/api/files/" + collection + "/" + target.Id + "/old_photo.png"
			request := func(path, auth string) *httptest.ResponseRecorder {
				response := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, path, nil)
				if auth != "" {
					req.Header.Set("Authorization", auth)
				}
				mux.ServeHTTP(response, req)
				wantCache := "no-store"
				if response.Code == 200 || response.Code == 304 || response.Code == 307 {
					wantCache = "private, max-age=3600"
				}
				if got := response.Header().Get("Cache-Control"); got != wantCache {
					t.Fatalf("legacy photo response%d Cache-Control=%q want=%q", response.Code, got, wantCache)
				}
				return response
			}
			redirect := func(path, auth string) {
				t.Helper()
				response := request(path, auth)
				if response.Code != 307 {
					t.Fatalf("legacy redirect %d: %s", response.Code, response.Body.String())
				}
				if got := response.Header().Get("Cache-Control"); got != "private, max-age=3600" {
					t.Fatalf("legacy redirect Cache-Control = %q", got)
				}
				location := response.Header().Get("Location")
				if !strings.Contains(location, "/"+asset.Id+"/") {
					t.Fatalf("wrong legacy destination: %s", location)
				}
				if downloaded := request(location, auth); downloaded.Code != 200 || !bytes.Equal(downloaded.Body.Bytes(), photo) {
					t.Fatalf("legacy destination %d: %s", downloaded.Code, downloaded.Body.String())
				}
			}
			redirect(path, "")
			request("/api/files/"+collection+"/nonexistent0000/missing.png", "")
			request("/api/files/"+collection+"/"+target.Id+"/missing.png", "")
			bad := httptest.NewRequest(http.MethodGet, path, strings.NewReader("{"))
			bad.Header.Set("Content-Type", "application/json")
			failed := httptest.NewRecorder()
			mux.ServeHTTP(failed, bad)
			if failed.Code != 500 || failed.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("legacy photo failure cache/status=%d %q", failed.Code, failed.Header().Get("Cache-Control"))
			}
			trail.Set("public", false)
			if err := app.Save(trail); err != nil {
				t.Fatal(err)
			}
			if response := request(path, ""); response.Code != 404 {
				t.Fatalf("private legacy URL: %d", response.Code)
			}
			redirect(path, token)
			share := saveAssetFilterRecord(t, app, "trail_link_share", map[string]any{"trail": trail.Id, "permission": "view"})
			shared := path + "?share=" + share.GetString("token")
			response := request(shared, "")
			if response.Code != 307 || !strings.Contains(response.Header().Get("Location"), "share="+share.GetString("token")) {
				t.Fatalf("share redirect: %d %s", response.Code, response.Header().Get("Location"))
			}
			redirect(shared, "")
			if err := app.Delete(share); err != nil {
				t.Fatal(err)
			}
			if response := request(shared, ""); response.Code != 404 {
				t.Fatalf("revoked legacy share: %d", response.Code)
			}
		})
	}
}

func TestCurrentGPXDownloadDoesNotQueryMigratedPhotoLinks(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
		"username": "gpx", "preferred_username": "gpx", "domain": "a.example", "public_key": "key",
		"is_local": true, "iri": "https://a.example/gpx", "inbox": "https://a.example/gpx/inbox",
	})
	file, err := filesystem.NewFileFromBytes([]byte("<?xml version=\"1.0\"?><gpx></gpx>"), "track.gpx")
	if err != nil {
		t.Fatal(err)
	}
	trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "gpx", "author": actor.Id, "public": true, "gpx": file})
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.BindFunc(routes.LegacyPhotoFileRedirect)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, builder := range []dbx.Builder{app.ConcurrentDB(), app.NonconcurrentDB()} {
		db := builder.(*dbx.DB)
		old := db.QueryLogFunc
		t.Cleanup(func() { db.QueryLogFunc = old })
		db.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
			if strings.Contains(query, "trail_assets") || strings.Contains(query, "FROM `assets`") {
				t.Errorf("current GPX download queried photo inventory: %s", query)
			}
		}
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/files/trails/"+trail.Id+"/"+trail.GetString("gpx"), nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "<gpx>") {
		t.Fatalf("GPX download failed: %d %s", response.Code, response.Body)
	}
}

func TestLegacyPhotoURLCannotBeClaimedByUnrelatedAssets(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := func(name string) string {
		return saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "a.example", "public_key": "key",
			"is_local": true, "iri": "https://a.example/" + name, "inbox": "https://a.example/" + name + "/inbox",
		}).Id
	}
	owner, other := actor("owner"), actor("other")
	trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "target", "author": owner, "public": true})
	otherTrail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "unrelated", "author": other, "public": true})
	photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.BindFunc(routes.LegacyPhotoFileRedirect)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, author, linkedTrail string }{
		{"other public asset", other, otherTrail.Id},
		{"contribution by another author", other, trail.Id},
		{"owner asset linked elsewhere", owner, otherTrail.Id},
		{"owner asset without any link", owner, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := filesystem.NewFileFromBytes(photo, "other.png")
			if err != nil {
				t.Fatal(err)
			}
			asset := saveAssetFilterRecord(t, app, "assets", map[string]any{
				"author": test.author, "type": "photo", "storage_mode": "copy", "file": file,
				"metadata": map[string]any{"source_collection": "trails", "source_record": trail.Id, "source_file": "gone.png"},
			})
			if test.linkedTrail != "" {
				saveAssetFilterRecord(t, app, "trail_assets", map[string]any{"trail": test.linkedTrail, "asset": asset.Id})
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/files/trails/"+trail.Id+"/gone.png", nil))
			if response.Code != http.StatusNotFound || response.Header().Get("Location") != "" {
				t.Fatalf("unrelated asset claimed legacy URL: %d %s", response.Code, response.Header().Get("Location"))
			}
			if err := app.Delete(asset); err != nil {
				t.Fatal(err)
			}
		})
	}
}
