package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"pocketbase/util"
)

func TestAssetProvenanceCannotBeRecoveredByPublicOrRelationFilters(t *testing.T) {
	f := newRecordHooksFixture(t)
	f.allowAssetReads()
	owner := f.localActor("owner")
	caller := f.localActor("caller")
	private := f.save("trails", map[string]any{"author": owner.Id, "name": "Private origin"})
	public := f.save("trails", map[string]any{"author": owner.Id, "name": "Public copy", "public": true})
	f.bindHooks()
	source := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "metadata": map[string]any{"source_record": private.Id, "source_file": "private.jpg", "copied_from": "private-source-photo", "original_filename": "private-original.jpg", "merged_assets": []any{map[string]any{"metadata": map[string]any{"source_record": private.Id}}}, "label": "public"}})
	f.save("trail_assets", map[string]any{"trail": public.Id, "asset": source.Id})
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if user != "" {
			record, err := f.app.FindRecordById("users", user)
			if err != nil {
				t.Fatal(err)
			}
			token, err := record.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	for _, user := range []string{"", caller.GetString("user"), owner.GetString("user")} {
		for collection, prefix := range map[string]string{"assets": "", "trail_assets": "asset."} {
			for _, filter := range []string{prefix + "metadata.source_record ~ '" + private.Id[:5] + "'", prefix + "metadata.copied_from = 'private-source-photo'", prefix + "metadata.original_filename = 'private-original.jpg'", prefix + "metadata.merged_assets.metadata.source_record ~ '" + private.Id[:5] + "'"} {
				response := request("/api/collections/"+collection+"/records?filter="+url.QueryEscape(filter), user)
				if response.Code != 200 {
					t.Fatalf("public metadata query failed: %d %s", response.Code, response.Body.String())
				}
				var body struct{ TotalItems int }
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.TotalItems != 0 {
					t.Fatalf("provenance filter oracle exposed private source: %s", response.Body.String())
				}
			}
			response := request("/api/collections/"+collection+"/records?filter="+url.QueryEscape(prefix+"source_metadata.source_record = '"+private.Id+"'"), user)
			if response.Code != 400 {
				t.Fatalf("hidden provenance remained filterable: %d %s", response.Code, response.Body.String())
			}
		}
	}
	response := request("/api/collections/assets/records/"+source.Id, "")
	if response.Code != 200 || strings.Contains(response.Body.String(), private.Id) || strings.Contains(response.Body.String(), "private-original") || !strings.Contains(response.Body.String(), `"label":"public"`) {
		t.Fatalf("anonymous metadata response: %d %s", response.Code, response.Body.String())
	}
	stored, err := f.app.FindRecordById("assets", source.Id)
	if err != nil {
		t.Fatal(err)
	}
	publicMetadata := map[string]any{}
	if err := stored.UnmarshalJSONField("metadata", &publicMetadata); err != nil {
		t.Fatal(err)
	}
	if len(publicMetadata) != 1 || util.AssetProvenanceMetadata(stored)["source_record"] != private.Id {
		t.Fatalf("trusted model write did not move provenance: %v", stored.PublicExport())
	}
}
