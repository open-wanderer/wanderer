package permissions_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"pocketbase/hooks"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestAssetFileDownloadRechecksAccessAfterShareRevocation(t *testing.T) {
	app := newRulesTestApp(t)
	app.OnFileDownloadRequest("assets").BindFunc(hooks.AuthorizeAssetFileDownload())
	owner := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "owner", "email": "owner@example.com", "password": "test-password",
	})
	stranger := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "stranger", "email": "stranger@example.com", "password": "test-password",
	})
	ownerToken, err := owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	strangerToken, err := stranger.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
		"username": "owner", "preferred_username": "owner", "domain": "example.com", "user": owner.Id,
		"public_key": "test-key", "is_local": true, "iri": "https://example.com/owner",
		"inbox": "https://example.com/owner/inbox",
	})
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{"name": "private", "author": actor.Id})
	var imageBytes bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 200, 150))
	picture.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&imageBytes, picture); err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(imageBytes.Bytes(), "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := saveRulesTestRecord(t, app, "assets", map[string]any{
		"author": actor.Id, "type": "photo", "storage_mode": "copy", "file": file,
	})
	saveRulesTestRecord(t, app, "trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id})

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	fileURL := "/api/files/" + asset.Collection().Id + "/" + asset.Id + "/" + asset.GetString("file")
	download := func(t *testing.T, method, share, thumb, auth, modifiedSince string, want int) *httptest.ResponseRecorder {
		t.Helper()
		query := url.Values{"share": {share}, "thumb": {thumb}}
		request := httptest.NewRequest(method, fileURL+"?"+query.Encode(), nil)
		if auth != "" {
			request.Header.Set("Authorization", "Bearer "+auth)
		}
		if modifiedSince != "" {
			request.Header.Set("If-Modified-Since", modifiedSince)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("%s file (share=%q, thumb=%q, auth=%t, conditional=%t): status %d, want %d; body %s",
				method, share, thumb, auth != "", modifiedSince != "", response.Code, want, response.Body.String())
		}
		if got := response.Header().Get("Cache-Control"); got != "private, no-cache, must-revalidate" {
			t.Errorf("Cache-Control = %q", got)
		}
		if !strings.Contains(strings.Join(response.Header().Values("Vary"), ","), "Authorization") {
			t.Error("missing Vary: Authorization")
		}
		if want == http.StatusOK && method == http.MethodGet {
			if _, err := png.Decode(bytes.NewReader(response.Body.Bytes())); err != nil {
				t.Errorf("file response is not a PNG: %v", err)
			}
		}
		return response
	}

	download(t, http.MethodGet, "", "", "", "", http.StatusNotFound)
	download(t, http.MethodGet, "", "", strangerToken, "", http.StatusNotFound)
	download(t, http.MethodGet, "", "", ownerToken, "", http.StatusOK)
	share := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
		"trail": trail.Id, "permission": "view",
	})
	shareToken := share.GetString("token")
	download(t, http.MethodGet, "", "", "", "", http.StatusNotFound)
	download(t, http.MethodGet, "wrong", "", "", "", http.StatusNotFound)

	lastModified := map[string]string{}
	for _, thumb := range []string{"", "100x100"} {
		response := download(t, http.MethodGet, shareToken, thumb, "", "", http.StatusOK)
		lastModified[thumb] = response.Header().Get("Last-Modified")
		if lastModified[thumb] == "" {
			t.Fatal("missing Last-Modified header for conditional download")
		}
		download(t, http.MethodGet, shareToken, thumb, "", lastModified[thumb], http.StatusNotModified)
	}
	if err := app.Delete(share); err != nil {
		t.Fatal(err)
	}
	for _, thumb := range []string{"", "100x100"} {
		t.Run("revoked/thumb="+thumb, func(t *testing.T) {
			download(t, http.MethodGet, shareToken, thumb, "", "", http.StatusNotFound)
			download(t, http.MethodGet, shareToken, thumb, "", lastModified[thumb], http.StatusNotFound)
			download(t, http.MethodHead, shareToken, thumb, "", lastModified[thumb], http.StatusNotFound)
			download(t, http.MethodGet, "", thumb, ownerToken, "", http.StatusOK)
		})
	}

	trail.Set("public", true)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	for _, thumb := range []string{"", "100x100"} {
		download(t, http.MethodGet, "", thumb, "", "", http.StatusOK)
	}
	trail.Set("public", false)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	for _, thumb := range []string{"", "100x100"} {
		download(t, http.MethodGet, "", thumb, "", lastModified[thumb], http.StatusNotFound)
	}
}
