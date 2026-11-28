package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestFederationSaveBeginAndCompleteRequireTargetEditPermission(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	f := newRemoteSyncFixture(t)
	users, err := f.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := f.app.FindRecordById(users, f.localActor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	newUser := func(name string) (*core.Record, *core.Record) {
		t.Helper()
		user := core.NewRecord(users)
		user.SetEmail(name + "@a.example")
		user.SetPassword("save-batch-test-password")
		if err := f.app.Save(user); err != nil {
			t.Fatal(err)
		}
		return user, f.save(f.actors, map[string]any{"is_local": true, "user": user.Id})
	}
	editor, editorActor := newUser("editor")
	reader, readerActor := newUser("reader")
	stranger, _ := newUser("stranger")
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(&core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1}, &core.RelationField{Name: "actor", CollectionId: f.actors.Id, MaxSelect: 1}, &core.TextField{Name: "permission"})
	if err := f.app.Save(shares); err != nil {
		t.Fatal(err)
	}
	f.trails.UpdateRule = types.Pointer(`author.user = @request.auth.id || (trail_share_via_trail.actor.user ?= @request.auth.id && trail_share_via_trail.permission ?= "edit")`)
	f.logs.UpdateRule = types.Pointer(`author.user = @request.auth.id`)
	f.logs.Fields.Add(&core.BoolField{Name: "federation_suppressed"})
	for _, collection := range []*core.Collection{f.trails, f.logs} {
		if err := f.app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	trail := f.save(f.trails, map[string]any{"public": true, "author": f.localActor.Id, "iri": "https://a.example/api/v1/trail/local"})
	log := f.save(f.logs, map[string]any{"trail": trail.Id, "author": f.localActor.Id, "iri": "https://a.example/api/v1/summit-log/local"})
	f.save(shares, map[string]any{"trail": trail.Id, "actor": editorActor.Id, "permission": "edit"})
	f.save(shares, map[string]any{"trail": trail.Id, "actor": readerActor.Id, "permission": "view"})
	calls := 0
	operation := func(*core.Record, string) bool { calls++; return true }
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	router.POST("/federation/{kind}/{id}/save/{action}", FederationSaveBatch(operation, operation))
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(user *core.Record, kind, id, action, batch string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/federation/%s/%s/save/%s", kind, id, action), strings.NewReader(fmt.Sprintf(`{"batchId":%q}`, batch)))
		req.Header.Set("Content-Type", "application/json")
		if user != nil {
			token, err := user.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	for _, target := range []struct {
		kind   string
		record *core.Record
	}{
		{"trail", trail}, {"summit-log", log},
	} {
		for _, action := range []string{"begin", "complete"} {
			for _, tc := range []struct {
				name   string
				user   *core.Record
				status int
			}{
				{"anonymous", nil, http.StatusUnauthorized},
				{"owner", owner, http.StatusOK},
				{"editor", editor, map[bool]int{true: http.StatusOK, false: http.StatusForbidden}[target.kind == "trail"]},
				{"reader", reader, http.StatusForbidden},
				{"stranger", stranger, http.StatusForbidden},
			} {
				t.Run(target.kind+"/"+action+"/"+tc.name, func(t *testing.T) {
					before := calls
					response := request(tc.user, target.kind, target.record.Id, action, "authorized-editors-save-token")
					if response.Code != tc.status {
						t.Fatalf("status=%d want=%d: %s", response.Code, tc.status, response.Body.String())
					}
					if (calls > before) != (tc.status == http.StatusOK) {
						t.Fatal("save batch callback did not follow authorization")
					}
				})
			}
		}
	}
	privateTrail := f.save(f.trails, map[string]any{"author": f.localActor.Id, "iri": "https://a.example/api/v1/trail/private"})
	suppressedLog := f.save(f.logs, map[string]any{"author": f.localActor.Id, "trail": trail.Id, "iri": "https://a.example/api/v1/summit-log/suppressed", "federation_suppressed": true})
	for _, action := range []string{"begin", "complete"} {
		t.Run("suppressed/"+action, func(t *testing.T) {
			before := calls
			response := request(owner, "summit-log", suppressedLog.Id, action, "authorized-editors-save-token")
			if response.Code != http.StatusOK || calls != before || !strings.Contains(response.Body.String(), "not_federated") {
				t.Fatalf("suppressed entry did not return a no-op: %d %s callbacks=%d", response.Code, response.Body.String(), calls-before)
			}
			response = request(stranger, "summit-log", suppressedLog.Id, action, "authorized-editors-save-token")
			if response.Code != http.StatusForbidden || calls != before {
				t.Fatalf("suppressed entry skipped edit permission: %d %s", response.Code, response.Body.String())
			}
		})
		t.Run("private-trail/"+action, func(t *testing.T) {
			before := calls
			response := request(owner, "trail", privateTrail.Id, action, "authorized-editors-save-token")
			if response.Code != http.StatusOK || calls != before+1 || !strings.Contains(response.Body.String(), `"status":"ok"`) {
				t.Fatalf("prepublication trail lease was not available: %d %s callbacks=%d", response.Code, response.Body.String(), calls-before)
			}
		})
	}
	legacyTrail := f.save(f.trails, map[string]any{"author": f.localActor.Id, "public": true, "iri": "https://b.example/api/v1/trail/local"})
	legacyLog := f.save(f.logs, map[string]any{"author": f.localActor.Id, "trail": legacyTrail.Id, "iri": "https://b.example/api/v1/summit-log/local"})
	for _, action := range []string{"begin", "complete"} {
		before := calls
		for _, target := range []struct {
			kind   string
			record *core.Record
		}{{"trail", legacyTrail}, {"summit-log", legacyLog}} {
			response := request(owner, target.kind, target.record.Id, action, "authorized-editors-save-token")
			if response.Code != http.StatusOK || calls != before || !strings.Contains(response.Body.String(), "not_federated") {
				t.Fatalf("historical local content did not return a non-federating no-op: %d %s", response.Code, response.Body.String())
			}
		}
		response := request(owner, "trail", trail.Id, action, "short")
		if response.Code != http.StatusBadRequest || calls != before {
			t.Fatalf("invalid token accepted: %d %s", response.Code, response.Body.String())
		}
	}
}
