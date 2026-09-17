package assets_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"pocketbase/routes"
	assetservice "pocketbase/services/assets"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestAssetFileRevalidationAndMaterializedRedirect(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, nil)
	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	// This test exercises file responses; the full sharing rules are covered
	// with the migrated schema in tests/permissions.
	collection.ViewRule = types.Pointer(`@request.query.share = "` + strings.Repeat("a", 32) + `"`)
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	asset := newPublishPhoto(t, app, owner.Id, "photo")
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/assets/{id}/file", routes.AssetFile)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	path := "/assets/" + asset.Id + "/file?share=" + strings.Repeat("a", 32)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/png" || response.Body.Len() == 0 {
		t.Fatalf("remote photo response = %d %v %s", response.Code, response.Header(), response.Body)
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-cache, must-revalidate" {
		t.Fatalf("remote photo cache policy = %q", got)
	}

	if err := assetservice.MaterializeRemotePluginAsset(t.Context(), app, asset); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path+"&thumb=100x100&unrelated=ignored", nil))
	if response.Code != http.StatusFound {
		t.Fatalf("materialized photo response = %d %s", response.Code, response.Body)
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != "/api/files/"+collection.Id+"/"+asset.Id+"/"+asset.GetString("file") {
		t.Fatalf("redirect location = %s", location)
	}
	if location.Query().Get("share") != strings.Repeat("a", 32) || location.Query().Get("thumb") != "100x100" || location.Query().Has("unrelated") {
		t.Fatalf("redirect lost access or thumbnail query: %s", location)
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-cache, must-revalidate" {
		t.Fatalf("redirect cache policy = %q", got)
	}
}
