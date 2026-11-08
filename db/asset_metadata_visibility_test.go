package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/hooks"
	"pocketbase/util"
)

func TestAssetSourceMetadataExportsOnlyToAuthorAndSuperuser(t *testing.T) {
	f := newRecordHooksFixture(t)
	f.allowAssetReads()
	author, trailOwner, editor := f.localActor("author"), f.localActor("owner"), f.localActor("editor")
	privateSource := f.save("trails", map[string]any{"name": "Private original", "author": author.Id})
	trail := f.save("trails", map[string]any{"name": "Public copy", "author": trailOwner.Id, "public": true})
	f.save("trail_share", map[string]any{"trail": trail.Id, "actor": editor.Id, "permission": "edit"})
	asset := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id,
		"metadata": map[string]any{"source_collection": "trails", "source_record": privateSource.Id, "source_file": "secret.jpg", "source_future": "private", "generated": "route-preview", "content_hash": "hash"}})
	f.save("trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id})
	admin := f.save("_superusers", map[string]any{"email": "admin@a.example", "password": "long-test-password"})
	f.bindHooks()
	if err := f.app.Save(asset); err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		auth  *core.Record
		allow bool
	}{
		{"anonymous", nil, false}, {"author", author, true}, {"trail owner", trailOwner, false}, {"shared editor", editor, false}, {"superuser", admin, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := ""
			if tc.auth != nil {
				user := tc.auth
				if user.Collection().Name == "activitypub_actors" {
					user, err = f.app.FindRecordById("users", user.GetString("user"))
					if err != nil {
						t.Fatal(err)
					}
				}
				token, err = user.NewAuthToken()
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, path := range []string{
				"/api/collections/assets/records/" + asset.Id,
				"/api/collections/trails/records/" + trail.Id + "?expand=trail_assets_via_trail.asset",
			} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				if token != "" {
					req.Header.Set("Authorization", token)
				}
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, req)
				if response.Code != http.StatusOK {
					t.Fatalf("GET %s: %d %s", path, response.Code, response.Body.String())
				}
				body := response.Body.String()
				if strings.Contains(body, `"source_record"`) != tc.allow || strings.Contains(body, `"source_future"`) != tc.allow {
					t.Fatalf("provenance visibility: %s", body)
				}
				if !strings.Contains(body, `"generated":"route-preview"`) || !strings.Contains(body, `"content_hash":"hash"`) {
					t.Fatalf("public metadata lost: %s", body)
				}
			}
		})
	}
	stored, err := f.app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	if metadata := util.AssetProvenanceMetadata(stored); metadata["source_record"] != privateSource.Id {
		t.Fatalf("export changed persisted provenance: %v", metadata)
	}
	// Internal enrichment without a request context must default to the
	// anonymous policy rather than expose migration source identities.
	e := &core.RecordEnrichEvent{App: f.app}
	e.Record = stored
	if err := hooks.HideAssetSourceMetadata()(e); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(stored)
	if err != nil || strings.Contains(string(data), "source_") {
		t.Fatalf("context-free export exposed provenance: %s, %v", data, err)
	}
}
