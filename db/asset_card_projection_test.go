package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
)

func TestFeedCardAssetExpandsKeepPublicScalarsAndBoundMetadata(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("card-owner")
	trail := f.save("trails", map[string]any{"name": "Card", "author": owner.Id, "public": true})
	values := map[string]any{
		"author": owner.Id, "type": "photo", "file": "clip.mp4", "storage_mode": "copy",
		"lat": 47.37, "lon": 8.54, "external_provider": "library", "external_id": "scalar-card",
		"remote_status": "missing", "remote_checked_at": "2026-11-08 12:00:00.000Z",
		"remote_missing_since": "2026-11-07 12:00:00.000Z", "remote_error": "Unavailable",
		"taken_at": "2026-11-06 12:00:00.000Z",
		"metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}, "caption": strings.Repeat("unrelated", 50000)},
	}
	asset := f.save("assets", values)
	f.save("trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id})
	f.save("feed", map[string]any{"actor": owner.Id, "author": owner.Id, "type": "trail", "item": trail.Id})
	f.bindHooks()
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.app.FindRecordById("users", owner.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/collections/feed/records", nil)
	request.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("feed status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []struct {
			Expand struct {
				Item struct {
					Expand struct {
						Links []struct {
							Expand struct {
								Asset map[string]any `json:"asset"`
							} `json:"expand"`
						} `json:"trail_assets_via_trail"`
					} `json:"expand"`
				} `json:"item"`
			} `json:"expand"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || len(body.Items[0].Expand.Item.Expand.Links) != 1 {
		t.Fatalf("missing card asset: %s", response.Body.String())
	}
	card := body.Items[0].Expand.Item.Expand.Links[0].Expand.Asset
	for _, field := range []string{"lat", "lon"} {
		if card[field] != asset.GetFloat(field) {
			t.Fatalf("%s changed in card projection: %v", field, card[field])
		}
	}
	for _, field := range []string{"external_id", "remote_status", "remote_checked_at", "remote_missing_since", "remote_error", "taken_at"} {
		if card[field] != asset.GetString(field) {
			t.Fatalf("%s changed in card projection: got=%v want=%s", field, card[field], asset.GetString(field))
		}
	}
	metadata, ok := card["metadata"].(map[string]any)
	if !ok || len(metadata) != 1 || metadata["generated"] == nil || strings.Contains(response.Body.String(), "unrelated") {
		t.Fatalf("card metadata was not reduced to the generated marker: %v", metadata)
	}
	if _, found := card["source_metadata"]; found {
		t.Fatal("card exposed hidden provenance")
	}
}
