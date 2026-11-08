package assets_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/routes"
	"pocketbase/util"
)

func TestLegacyPhotoAliasLookupUsesIndex(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{"username": "aliases", "preferred_username": "aliases", "domain": "a.example", "public_key": "key", "is_local": true, "iri": "https://a.example/aliases", "inbox": "https://a.example/aliases/inbox"})
	trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "Aliases", "author": actor.Id, "public": true})
	photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	assetByFilename := map[string]*core.Record{}
	for index := 0; index <= 32; index++ {
		file, err := filesystem.NewFileFromBytes(photo, "photo.png")
		if err != nil {
			t.Fatal(err)
		}
		filename := fmt.Sprintf("legacy-%d.png", index)
		asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"author": actor.Id, "type": "photo", "storage_mode": "copy", "file": file,
			"metadata": map[string]any{"source_collection": "trails", "source_record": trail.Id, "source_file": filename}})
		saveAssetFilterRecord(t, app, "trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id})
		assetByFilename[filename] = asset
	}
	var count int
	if err := app.DB().NewQuery("SELECT count(*) FROM " + util.LegacyPhotoAliasTable).Row(&count); err != nil || count != 33 {
		t.Fatalf("alias rows were not indexed: count=%d error=%v", count, err)
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
	captured := ""
	database := app.ConcurrentDB().(*dbx.DB)
	previous := database.QueryLogFunc
	database.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
		if strings.HasPrefix(query, "SELECT") && (strings.Contains(query, util.LegacyPhotoAliasTable) || strings.Contains(query, "json_tree") || strings.Contains(query, "source_metadata")) {
			captured = query
		}
	}
	defer func() { database.QueryLogFunc = previous }()
	request := func(filename string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/files/trails/"+trail.Id+"/"+filename, nil))
		return response
	}
	for _, filename := range []string{"legacy-0.png", "legacy-16.png", "legacy-32.png"} {
		response := request(filename)
		if response.Code != http.StatusTemporaryRedirect || !strings.Contains(response.Header().Get("Location"), "/"+assetByFilename[filename].Id+"/") {
			t.Fatalf("lost legacy alias %s: %d %s", filename, response.Code, response.Body)
		}
	}
	if response := request("unknown.png"); response.Code != http.StatusNotFound {
		t.Fatalf("unknown filename: %d %s", response.Code, response.Body)
	}
	if captured == "" || strings.Contains(captured, "json_") {
		t.Fatalf("legacy lookup still parses history: %s", captured)
	}
	plan := []struct{ Detail string }{}
	if err := app.DB().NewQuery("EXPLAIN QUERY PLAN " + captured).All(&plan); err != nil {
		t.Fatal(err)
	}
	indexed := false
	for _, row := range plan {
		indexed = indexed || strings.Contains(row.Detail, "idx_asset_legacy_photo_alias_lookup")
	}
	if !indexed {
		t.Fatalf("exact filename lookup is not indexed: %v", plan)
	}

	trail.Set("public", false)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	captured = ""
	if response := request("unknown.png"); response.Code != http.StatusNotFound || captured != "" {
		t.Fatalf("private target reached alias lookup: %d query=%s", response.Code, captured)
	}
}

func TestLegacyAliasUpdatesRollBackWithTheirAsset(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{"username": "rollback", "preferred_username": "rollback", "domain": "a.example", "public_key": "key", "is_local": true, "iri": "https://a.example/rollback", "inbox": "https://a.example/rollback/inbox"})
	asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"author": actor.Id, "type": "photo", "storage_mode": "copy", "metadata": map[string]any{"source_collection": "trails", "source_record": "trail0000000001", "source_file": "old.png"}})
	failure := errors.New("cancel asset edit")
	if err := app.RunInTransaction(func(tx core.App) error {
		changed, err := tx.FindRecordById("assets", asset.Id)
		if err != nil {
			return err
		}
		changed.Set(util.AssetProvenanceField, map[string]any{"source_collection": "trails", "source_record": "trail0000000001", "source_file": "new.png"})
		if err := tx.Save(changed); err != nil {
			return err
		}
		return failure
	}); !errors.Is(err, failure) {
		t.Fatalf("rollback failed: %v", err)
	}
	var file string
	if err := app.DB().NewQuery("SELECT source_file FROM " + util.LegacyPhotoAliasTable + " WHERE asset = {:id}").Bind(dbx.Params{"id": asset.Id}).Row(&file); err != nil || file != "old.png" {
		t.Fatalf("alias changed independently of its asset: %s %v", file, err)
	}
}
