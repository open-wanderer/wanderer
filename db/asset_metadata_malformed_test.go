package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
)

func TestMalformedOptionalAssetMetadataKeepsRequestedChildExpands(t *testing.T) {
	f := newRecordHooksFixture(t)
	f.allowAssetReads()
	owner := f.localActor("owner")
	trail := f.save("trails", map[string]any{"name": "Public trail", "author": owner.Id, "public": true})
	waypoint := f.save("waypoints", map[string]any{"name": "Waypoint", "author": owner.Id, "trail": trail.Id})
	log := f.save("summit_logs", map[string]any{"text": "Contribution", "author": owner.Id, "trail": trail.Id})
	photo := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "metadata": []any{"malformed optional object"}})
	f.save("waypoint_assets", map[string]any{"waypoint": waypoint.Id, "asset": photo.Id})
	f.save("summit_log_assets", map[string]any{"summit_log": log.Id, "asset": photo.Id})
	f.bindHooks()
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/collections/trails/records/"+trail.Id+"?expand=waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail.summit_log_assets_via_summit_log.asset", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatalf("GET failed: %d %s", response.Code, response.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	expand, _ := data["expand"].(map[string]any)
	for parent, childLink := range map[string]string{"waypoints_via_trail": "waypoint_assets_via_waypoint", "summit_logs_via_trail": "summit_log_assets_via_summit_log"} {
		children, _ := expand[parent].([]any)
		if len(children) != 1 {
			t.Fatalf("requested children lost due to metadata: %s", response.Body.String())
		}
		child := children[0].(map[string]any)
		childExpand, _ := child["expand"].(map[string]any)
		links, _ := childExpand[childLink].([]any)
		if len(links) != 1 {
			t.Fatalf("photo relation lost: %s", response.Body.String())
		}
		linked := links[0].(map[string]any)["expand"].(map[string]any)["asset"].(map[string]any)
		metadata, ok := linked["metadata"].(map[string]any)
		if !ok || len(metadata) != 0 {
			t.Fatalf("malformed metadata did not fail closed: %v", linked)
		}
	}
}
