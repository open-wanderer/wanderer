package permissions_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestWaypointAccessUsesTrailOwnershipAndCurrentShares(t *testing.T) {
	app := newRulesTestApp(t)
	user := func(name string) *core.Record {
		return saveRulesTestRecord(t, app, "users", map[string]any{"username": name, "password": "test-password", "email": name + "@example.com"})
	}
	owner, editor := user("owner"), user("editor")
	actor := func(name string, u *core.Record) *core.Record {
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{"preferred_username": name, "username": name, "domain": "example.com", "is_local": true, "public_key": "test-key", "iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox", "user": u.Id})
	}
	ownerActor, editorActor := actor("owner", owner), actor("editor", editor)
	collision := saveRulesTestRecord(t, app, "users", map[string]any{"id": ownerActor.Id, "username": "collision", "password": "test-password", "email": "collision@example.com"})
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{"name": "Private", "author": ownerActor.Id})
	point := saveRulesTestRecord(t, app, "waypoints", map[string]any{"name": "Point", "author": ownerActor.Id, "trail": trail.Id})
	contributed := saveRulesTestRecord(t, app, "waypoints", map[string]any{"name": "Contribution", "author": editorActor.Id, "trail": trail.Id})
	api, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path string, u *core.Record, body any) *httptest.ResponseRecorder {
		var raw bytes.Buffer
		if body != nil {
			json.NewEncoder(&raw).Encode(body)
		}
		req := httptest.NewRequest(method, path, &raw)
		req.Header.Set("Content-Type", "application/json")
		if u != nil {
			token, err := u.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	path := "/api/collections/waypoints/records/" + point.Id
	if response := request("GET", path, collision, nil); response.Code != 404 {
		t.Fatalf("unrelated ID read private waypoint: %d %s", response.Code, response.Body.String())
	}
	if response := request("GET", "/api/collections/waypoints/records", collision, nil); response.Code != 200 || strings.Contains(response.Body.String(), point.Id) {
		t.Fatalf("list exposed private waypoint: %d %s", response.Code, response.Body.String())
	}
	for _, method := range []string{"PATCH", "DELETE"} {
		response := request(method, path, collision, map[string]any{"name": "Changed"})
		if response.Code != 404 {
			t.Fatalf("unrelated ID %s waypoint: %d %s", method, response.Code, response.Body.String())
		}
	}
	if response := request("PATCH", path, owner, map[string]any{"name": "Owner edit"}); response.Code != 200 {
		t.Fatalf("owner update rejected: %d %s", response.Code, response.Body.String())
	}
	share := saveRulesTestRecord(t, app, "trail_share", map[string]any{"trail": trail.Id, "actor": editorActor.Id, "permission": "edit"})
	if response := request("GET", path, editor, nil); response.Code != 200 {
		t.Fatal("active share lost read access")
	}
	if err := app.Delete(share); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*core.Record{point, contributed} {
		for _, method := range []string{"GET", "PATCH", "DELETE"} {
			response := request(method, "/api/collections/waypoints/records/"+p.Id, editor, map[string]any{"name": "Changed"})
			if response.Code != 404 {
				t.Fatalf("revoked share retained %s access to%s: %d %s", method, p.Id, response.Code, response.Body.String())
			}
		}
	}
	link := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{"trail": trail.Id, "token": strings.Repeat("s", 32), "permission": "view"})
	if response := request("GET", path+"?share="+link.GetString("token"), nil, nil); response.Code != 200 {
		t.Fatal("legitimate link share lost access")
	}
	trail.Set("public", true)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	if response := request("GET", path, nil, nil); response.Code != 200 {
		t.Fatal("public waypoint lost anonymous access")
	}
}
