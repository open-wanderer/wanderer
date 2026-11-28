package assets_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/hooks"
	"pocketbase/routes"
)

func TestProfileThumbnailLegacyPeerMappingResolvesLinkedPhoto(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
		"username": "hiker", "preferred_username": "hiker", "domain": "a.example", "public_key": "key",
		"is_local": true, "iri": "https://a.example/hiker", "inbox": "https://a.example/hiker/inbox",
	})
	trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "Profile photo", "author": actor.Id, "public": true})
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(raw, "profile.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"author": actor.Id, "type": "photo", "storage_mode": "copy", "file": file})
	saveAssetFilterRecord(t, app, "trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id, "is_thumbnail": true})
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
	// v0.21.2 peers append the bare profile thumbnail to this public path.
	thumbnail := asset.GetString("file")
	peerURL, err := url.Parse("https://a.example/api/v1/files/trails/" + trail.Id + "/" + thumbnail)
	if err != nil {
		t.Fatal(err)
	}
	// The file proxy forwards the same collection, trail ID and filename.
	backendPath := strings.Replace(peerURL.Path, "/api/v1/files/", "/api/files/", 1)
	request := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	response := request(backendPath)
	if response.Code != http.StatusTemporaryRedirect {
		t.Fatalf("peer thumbnail mapping returned %d: %s", response.Code, response.Body)
	}
	downloaded := request(response.Header().Get("Location"))
	if downloaded.Code != http.StatusOK || !bytes.Equal(downloaded.Body.Bytes(), raw) {
		t.Fatalf("peer thumbnail download returned %d", downloaded.Code)
	}
	trail.Set("public", false)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	if private := request(backendPath); private.Code != http.StatusNotFound {
		t.Fatalf("private profile photo returned %d", private.Code)
	}
}
