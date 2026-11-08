package permissions_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func assetRuleRequest(t *testing.T, handler http.Handler, auth *core.Record, method, path string, data map[string]any, status int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if data != nil {
		if err := json.NewEncoder(&body).Encode(data); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", "application/json")
	if auth != nil {
		token, err := auth.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != status {
		t.Fatalf("%s %s: status %d, want %d: %s", method, path, response.Code, status, response.Body.String())
	}
	return response
}

func TestWaypointPhotosUseTrailOwnershipAndCurrentShares(t *testing.T) {
	f := newOwnershipFixture(t)
	waypoint := saveRulesTestRecord(t, f.app, "waypoints", map[string]any{"name": "Contributed point", "trail": f.trail.Id, "author": f.editorActor.Id})
	ownerAsset := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": f.ownerActor.Id})
	editorAsset := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": f.editorActor.Id})
	ownerLink := saveRulesTestRecord(t, f.app, "waypoint_assets", map[string]any{"waypoint": waypoint.Id, "asset": ownerAsset.Id})
	editorLink := saveRulesTestRecord(t, f.app, "waypoint_assets", map[string]any{"waypoint": waypoint.Id, "asset": editorAsset.Id})
	unlinkedEditorAsset := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": f.editorActor.Id})
	assetRuleRequest(t, f.handler, f.editor, "GET", "/api/collections/assets/records/"+ownerAsset.Id, nil, 200)
	if err := f.app.Delete(f.trailShare); err != nil {
		t.Fatal(err)
	}
	assetRuleRequest(t, f.handler, f.editor, "GET", "/api/collections/waypoints/records/"+waypoint.Id, nil, 404)
	assetRuleRequest(t, f.handler, f.editor, "GET", "/api/collections/assets/records/"+ownerAsset.Id, nil, 404)
	assetRuleRequest(t, f.handler, f.editor, "GET", "/api/collections/waypoint_assets/records/"+ownerLink.Id, nil, 404)
	assetRuleRequest(t, f.handler, f.editor, "POST", "/api/collections/waypoint_assets/records", map[string]any{"waypoint": waypoint.Id, "asset": unlinkedEditorAsset.Id}, 400)
	assetRuleRequest(t, f.handler, f.editor, "DELETE", "/api/collections/waypoint_assets/records/"+ownerLink.Id, nil, 404)
	// Asset ownership still permits reading and withdrawing one's own photo.
	assetRuleRequest(t, f.handler, f.editor, "GET", "/api/collections/assets/records/"+editorAsset.Id, nil, 200)
	assetRuleRequest(t, f.handler, f.editor, "DELETE", "/api/collections/waypoint_assets/records/"+editorLink.Id, nil, 204)
	assetRuleRequest(t, f.handler, f.owner, "GET", "/api/collections/assets/records/"+ownerAsset.Id, nil, 200)
}

func TestSummitLogPhotoWritesFollowEntryAndTrailOwnership(t *testing.T) {
	f := newOwnershipFixture(t)
	entry := saveRulesTestRecord(t, f.app, "summit_logs", map[string]any{"trail": f.trail.Id, "author": f.strangerActor.Id, "text": "An entry"})
	entryAsset := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": f.strangerActor.Id})
	link := saveRulesTestRecord(t, f.app, "summit_log_assets", map[string]any{"summit_log": entry.Id, "asset": entryAsset.Id})
	editorAsset := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": f.editorActor.Id})
	path := "/api/collections/summit_log_assets/records"
	assetRuleRequest(t, f.handler, f.editor, "POST", path, map[string]any{"summit_log": entry.Id, "asset": editorAsset.Id}, 400)
	assetRuleRequest(t, f.handler, f.editor, "PATCH", path+"/"+link.Id, map[string]any{}, 404)
	assetRuleRequest(t, f.handler, f.editor, "DELETE", path+"/"+link.Id, nil, 404)
	for _, role := range []struct{ user, actor *core.Record }{{f.stranger, f.strangerActor}, {f.owner, f.ownerActor}} {
		asset := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": role.actor.Id})
		response := assetRuleRequest(t, f.handler, role.user, "POST", path, map[string]any{"summit_log": entry.Id, "asset": asset.Id}, 200)
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		assetRuleRequest(t, f.handler, role.user, "PATCH", path+"/"+created.ID, map[string]any{}, 200)
		assetRuleRequest(t, f.handler, role.user, "DELETE", path+"/"+created.ID, nil, 204)
	}
	withdrawal := saveRulesTestRecord(t, f.app, "summit_log_assets", map[string]any{"summit_log": entry.Id, "asset": editorAsset.Id})
	assetRuleRequest(t, f.handler, f.editor, "DELETE", path+"/"+withdrawal.Id, nil, 204)
}

func TestPhotoLinksRequireOwnedAssetsAndImmutableRelations(t *testing.T) {
	for _, kind := range []string{"trail", "waypoint", "summit_log"} {
		t.Run(kind, func(t *testing.T) {
			f := newOwnershipFixture(t)
			otherOwnerTrail := saveRulesTestRecord(t, f.app, "trails", map[string]any{"name": "Another owner trail", "author": f.ownerActor.Id})
			editorTrail := saveRulesTestRecord(t, f.app, "trails", map[string]any{"name": "Editor trail", "author": f.editorActor.Id})
			target := f.trail
			otherOwnerTarget := otherOwnerTrail
			editorTarget := editorTrail
			if kind != "trail" {
				author := f.ownerActor.Id
				if kind == "summit_log" {
					author = f.editorActor.Id
				}
				target = saveRulesTestRecord(t, f.app, kind+"s", map[string]any{"name": "Target", "trail": f.trail.Id, "author": author})
				otherOwnerTarget = saveRulesTestRecord(t, f.app, kind+"s", map[string]any{"name": "Another target", "trail": otherOwnerTrail.Id, "author": f.ownerActor.Id})
				editorTarget = saveRulesTestRecord(t, f.app, kind+"s", map[string]any{"name": "Editor target", "trail": editorTrail.Id, "author": f.editorActor.Id})
			}
			foreign := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": f.strangerActor.Id})
			path := "/api/collections/" + kind + "_assets/records"
			for _, role := range []struct{ user, actor, otherTarget *core.Record }{{f.owner, f.ownerActor, otherOwnerTarget}, {f.editor, f.editorActor, editorTarget}} {
				assetRuleRequest(t, f.handler, role.user, "GET", "/api/collections/assets/records/"+foreign.Id, nil, 404)
				assetRuleRequest(t, f.handler, role.user, "POST", path, map[string]any{kind: target.Id, "asset": foreign.Id}, 400)
				assetRuleRequest(t, f.handler, role.user, "GET", "/api/collections/assets/records/"+foreign.Id, nil, 404)
				first := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": role.actor.Id})
				second := saveRulesTestRecord(t, f.app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": role.actor.Id})
				link := saveRulesTestRecord(t, f.app, kind+"_assets", map[string]any{kind: target.Id, "asset": first.Id})
				assetRuleRequest(t, f.handler, role.user, "PATCH", path+"/"+link.Id, map[string]any{"asset": second.Id}, 404)
				assetRuleRequest(t, f.handler, role.user, "PATCH", path+"/"+link.Id, map[string]any{kind: role.otherTarget.Id}, 404)
			}
		})
	}
}
