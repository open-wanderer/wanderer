package hooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestUpdateTrailLinkShareTargetThroughRecordsAPI(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1})
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1, Required: true},
		&core.BoolField{Name: "public"},
	)
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
	)
	links := core.NewBaseCollection("trail_link_share")
	links.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1, Required: true},
		&core.TextField{Name: "token", Min: 32, Max: 32, Required: true, AutogeneratePattern: "[a-z0-9]{32}"},
		&core.SelectField{Name: "permission", Values: []string{"view", "edit"}, MaxSelect: 1, Required: true},
	)
	// Production link-share ownership rules and one-link-per-trail constraint.
	links.CreateRule = types.Pointer("trail.author.user = @request.auth.id")
	links.UpdateRule = types.Pointer("trail.author.user = @request.auth.id")
	links.ViewRule = types.Pointer("trail.author.user = @request.auth.id")
	links.Indexes = []string{"CREATE UNIQUE INDEX idx_test_link_share_trail ON trail_link_share (trail)"}
	for _, collection := range []*core.Collection{actors, trails, shares, links} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	// Keep all production trail-view branches, including anonymous token access.
	trails.ViewRule = types.Pointer(`author.user = @request.auth.id || public = true || (@request.auth.id != "" && trail_share_via_trail.actor.user ?= @request.auth.id) || (trail_link_share_via_trail.token != "" && trail_link_share_via_trail.token = @request.query.share)`)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	save := func(collection *core.Collection, data map[string]any) *core.Record {
		t.Helper()
		record := core.NewRecord(collection)
		record.Load(data)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	owner := save(users, map[string]any{"email": "owner@example.com", "password": "test-password"})
	other := save(users, map[string]any{"email": "other@example.com", "password": "test-password"})
	ownerToken, err := owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	otherToken, err := other.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	ownerActor := save(actors, map[string]any{"user": owner.Id})
	otherActor := save(actors, map[string]any{"user": other.Id})
	ownTrail := save(trails, map[string]any{"author": ownerActor.Id, "public": false})
	anotherOwnTrail := save(trails, map[string]any{"author": ownerActor.Id, "public": true})
	otherPrivateTrail := save(trails, map[string]any{"author": otherActor.Id, "public": false})
	linkToken := strings.Repeat("a", 32)
	rotatedToken := strings.Repeat("b", 32)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(t *testing.T, method, path string, body map[string]string, auth string, wantStatus int) *httptest.ResponseRecorder {
		t.Helper()
		var data []byte
		if body != nil {
			var err error
			data, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != wantStatus {
			t.Fatalf("%s %s: status %d; want %d: %s", method, path, response.Code, wantStatus, response.Body.String())
		}
		return response
	}
	anonymousGet := func(t *testing.T, trailID, token string, wantStatus int) {
		t.Helper()
		request(t, http.MethodGet, "/api/collections/trails/records/"+trailID+"?share="+token, nil, "", wantStatus)
	}
	createLink := func(t *testing.T) *core.Record {
		t.Helper()
		response := request(t, http.MethodPost, "/api/collections/trail_link_share/records", map[string]string{
			"trail": ownTrail.Id, "token": linkToken, "permission": "view",
		}, ownerToken, http.StatusOK)
		var data struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		record, err := app.FindRecordById(links.Name, data.ID)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := app.Delete(record); err != nil {
				t.Error(err)
			}
		})
		return record
	}

	t.Run("unguarded update redirects anonymous token access", func(t *testing.T) {
		link := createLink(t)
		anonymousGet(t, ownTrail.Id, linkToken, http.StatusOK)
		anonymousGet(t, otherPrivateTrail.Id, linkToken, http.StatusNotFound)
		request(t, http.MethodPatch, "/api/collections/trail_link_share/records/"+link.Id,
			map[string]string{"trail": otherPrivateTrail.Id}, ownerToken, http.StatusOK)
		anonymousGet(t, otherPrivateTrail.Id, linkToken, http.StatusOK)
		anonymousGet(t, otherPrivateTrail.Id, "", http.StatusNotFound)
		anonymousGet(t, otherPrivateTrail.Id, rotatedToken, http.StatusNotFound)
	})

	app.OnRecordUpdateRequest(links.Name).BindFunc(UpdateShareTargetHandler("trail"))
	for _, test := range []struct {
		name   string
		body   map[string]string
		auth   string
		status int
	}{
		{"foreign target", map[string]string{"trail": otherPrivateTrail.Id}, ownerToken, http.StatusBadRequest},
		{"foreign target and new token", map[string]string{"trail": otherPrivateTrail.Id, "token": rotatedToken}, ownerToken, http.StatusBadRequest},
		{"different owned target", map[string]string{"trail": anotherOwnTrail.Id}, ownerToken, http.StatusBadRequest},
		{"empty target", map[string]string{"trail": ""}, ownerToken, http.StatusBadRequest},
		{"permission only", map[string]string{"permission": "edit"}, ownerToken, http.StatusOK},
		{"explicit unchanged target", map[string]string{"trail": ownTrail.Id, "permission": "edit"}, ownerToken, http.StatusOK},
		{"rotate token", map[string]string{"token": rotatedToken}, ownerToken, http.StatusOK},
		{"invalid token", map[string]string{"token": "short"}, ownerToken, http.StatusBadRequest},
		{"anonymous update", map[string]string{"permission": "edit"}, "", http.StatusNotFound},
		{"other user update", map[string]string{"permission": "edit"}, otherToken, http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			link := createLink(t)
			anonymousGet(t, otherPrivateTrail.Id, linkToken, http.StatusNotFound)
			request(t, http.MethodPatch, "/api/collections/trail_link_share/records/"+link.Id,
				test.body, test.auth, test.status)
			stored, err := app.FindRecordById(links.Name, link.Id)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"trail": ownTrail.Id, "token": linkToken, "permission": "view"}
			if test.status == http.StatusOK {
				for field, value := range test.body {
					want[field] = value
				}
			}
			for field, value := range want {
				if got := stored.GetString(field); got != value {
					t.Errorf("stored %s = %q; want %q", field, got, value)
				}
			}
			anonymousGet(t, otherPrivateTrail.Id, linkToken, http.StatusNotFound)
			anonymousGet(t, otherPrivateTrail.Id, rotatedToken, http.StatusNotFound)
			anonymousGet(t, ownTrail.Id, want["token"], http.StatusOK)
			if want["token"] != linkToken {
				anonymousGet(t, ownTrail.Id, linkToken, http.StatusNotFound)
			}
		})
	}
}
