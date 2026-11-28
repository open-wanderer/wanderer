package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/routes"
)

func TestAssetTargetPermissionsRequireStoredActorUserOwnership(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, editor := f.localActor("owner"), f.localActor("editor")
	collision := f.save("users", map[string]any{"id": owner.Id, "username": "collision", "email": "collision@a.example"})
	f.save("activitypub_actors", map[string]any{"user": collision.Id, "preferred_username": "collision", "iri": "https://a.example/api/v1/activitypub/user/collision", "is_local": true})
	raw, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	api.DELETE("/assets/{id}", routes.AssetDelete)
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(user *core.Record, path string) *httptest.ResponseRecorder {
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodDelete, path, nil)
		req.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	user, err := f.app.FindRecordById("users", owner.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	editorUser, err := f.app.FindRecordById("users", editor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"trail", "waypoint", "summit_log"} {
		t.Run(kind, func(t *testing.T) {
			trail := f.save("trails", map[string]any{"name": "Private", "author": owner.Id})
			parent := trail
			if kind != "trail" {
				parent = f.save(map[string]string{"waypoint": "waypoints", "summit_log": "summit_logs"}[kind], map[string]any{"author": owner.Id, "trail": trail.Id})
			}
			file, err := filesystem.NewFileFromBytes(raw, "photo.png")
			if err != nil {
				t.Fatal(err)
			}
			photo := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": file})
			link := f.save(kind+"_assets", map[string]any{kind: parent.Id, "asset": photo.Id})
			path := "/assets/" + photo.Id + "?" + kind + "=" + parent.Id
			if response := request(collision, path); response.Code != http.StatusForbidden {
				t.Fatalf("unrelated user ID was treated as actor ownership: %d %s", response.Code, response.Body.String())
			}
			if !f.exists(link) || !f.exists(photo) {
				t.Fatal("rejected request changed photo linkage")
			}
			share := f.save("trail_share", map[string]any{"trail": trail.Id, "actor": editor.Id, "permission": "edit"})
			if kind == "summit_log" {
				if response := request(editorUser, path); response.Code != http.StatusForbidden || !f.exists(link) {
					t.Fatalf("trail edit share changed another entry's photo: %d %s", response.Code, response.Body.String())
				}
			} else {
				if response := request(editorUser, path); response.Code != http.StatusOK {
					t.Fatalf("legitimate editor rejected: %d %s", response.Code, response.Body.String())
				}
				link = f.save(kind+"_assets", map[string]any{kind: parent.Id, "asset": photo.Id})
			}
			if err := f.app.Delete(share); err != nil {
				t.Fatal(err)
			}
			if response := request(editorUser, path); response.Code != http.StatusForbidden || !f.exists(link) {
				t.Fatalf("revoked edit share retained permission: %d", response.Code)
			}
			if response := request(user, path); response.Code != http.StatusOK {
				t.Fatalf("owner rejected: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
