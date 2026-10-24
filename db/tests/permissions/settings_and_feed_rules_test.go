package permissions_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func newRulesTestMux(t *testing.T, app core.App) http.Handler {
	t.Helper()
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return mux
}

func rulesTestRequest(t *testing.T, mux http.Handler, auth *core.Record, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		var err error
		if data, err = json.Marshal(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	if auth != nil {
		token, err := auth.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", token)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func TestSettingsCreateRule(t *testing.T) {
	app := newRulesTestApp(t)
	mux := newRulesTestMux(t, app)
	newUser := func(name string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
	}
	owner, stranger := newUser("owner"), newUser("stranger")
	body := map[string]any{
		"user": owner.Id, "language": "en", "unit": "metric", "mapFocus": "trails",
		"bio": "planted", "privacy": map[string]any{"account": "private"},
	}

	for name, auth := range map[string]*core.Record{"anonymous": nil, "another user": stranger} {
		response := rulesTestRequest(t, mux, auth, http.MethodPost, "/api/collections/settings/records", body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s creates settings for owner: status %d, want %d: %s", name, response.Code, http.StatusBadRequest, response.Body.String())
		}
	}

	response := rulesTestRequest(t, mux, owner, http.MethodPost, "/api/collections/settings/records", body)
	if response.Code != http.StatusOK {
		t.Fatalf("owner creates own settings: status %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	response = rulesTestRequest(t, mux, owner, http.MethodPost, "/api/collections/settings/records", body)
	if response.Code != http.StatusBadRequest {
		t.Errorf("owner creates a second settings record: status %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestProfileFeedHidesPrivateAccounts(t *testing.T) {
	app := newRulesTestApp(t)
	mux := newRulesTestMux(t, app)
	newAccount := func(name string, privacy map[string]any) (*core.Record, *core.Record, *core.Record) {
		t.Helper()
		user := saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
		if privacy != nil {
			saveRulesTestRecord(t, app, "settings", map[string]any{
				"user": user.Id, "language": "en", "unit": "metric", "mapFocus": "trails", "privacy": privacy,
			})
		}
		actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com", "user": user.Id,
			"is_local": true, "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
		trail := saveRulesTestRecord(t, app, "trails", map[string]any{
			"name": name, "author": actor.Id, "public": true,
		})
		return user, actor, trail
	}
	privateUser, privateActor, privateTrail := newAccount("private", map[string]any{"account": "private"})
	_, _, publicTrail := newAccount("public", map[string]any{"account": "public"})
	_, _, noSettingsTrail := newAccount("nosettings", nil)
	remoteActor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
		"username": "remote", "preferred_username": "remote", "domain": "remote.example", "public_key": "test-key",
		"iri": "https://remote.example/remote", "inbox": "https://remote.example/remote/inbox",
	})
	remoteTrail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "remote", "author": remoteActor.Id, "public": true,
	})

	feedItems := func(auth *core.Record) []string {
		t.Helper()
		response := rulesTestRequest(t, mux, auth, http.MethodGet, "/api/collections/profile_feed/records?perPage=100", nil)
		if response.Code != http.StatusOK {
			t.Fatalf("list profile_feed: status %d: %s", response.Code, response.Body.String())
		}
		var result struct {
			Items []struct {
				Actor string `json:"actor"`
				Item  string `json:"item"`
			} `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		var items []string
		for _, item := range result.Items {
			items = append(items, item.Item)
		}
		slices.Sort(items)
		return items
	}
	sorted := func(ids ...string) []string {
		slices.Sort(ids)
		return ids
	}

	if got, want := feedItems(nil), sorted(publicTrail.Id, noSettingsTrail.Id, remoteTrail.Id); !slices.Equal(got, want) {
		t.Errorf("anonymous feed = %v, want %v", got, want)
	}
	if got, want := feedItems(privateUser), sorted(privateTrail.Id, publicTrail.Id, noSettingsTrail.Id, remoteTrail.Id); !slices.Equal(got, want) {
		t.Errorf("private owner feed = %v, want %v", got, want)
	}

	// The web client filters the view by actor id.
	response := rulesTestRequest(t, mux, nil, http.MethodGet, "/api/collections/profile_feed/records?filter="+
		"actor%3D%27"+privateActor.Id+"%27", nil)
	if response.Code != http.StatusOK || bytes.Contains(response.Body.Bytes(), []byte(privateTrail.Id)) {
		t.Errorf("anonymous feed filtered to the private actor: status %d, body %s", response.Code, response.Body.String())
	}
}
