package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"pocketbase/federation"
	"pocketbase/routes"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestLegacyActorKeyStaysInternal(t *testing.T) {
	app := newRealSchemaApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	field, ok := actors.Fields.GetByName("legacy_public_key").(*core.TextField)
	if !ok || !field.Hidden || field.Required {
		t.Fatalf("legacy_public_key must be an optional hidden text field, got %#v", actors.Fields.GetByName("legacy_public_key"))
	}

	const marker = "legacy-key-marker"
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.Load(map[string]any{"email": "reader@example.com", "username": "reader"})
	user.SetPassword("long-test-password")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	// The state the cache refresh leaves behind: no current key, the
	// previous one in the hidden field.
	actor := core.NewRecord(actors)
	actor.Load(map[string]any{"username": "Alice", "preferred_username": "alice", "domain": "remote.example",
		"iri": "https://remote.example/api/v1/activitypub/user/alice", "inbox": "https://remote.example/api/v1/activitypub/user/alice/inbox", "legacy_public_key": marker})
	if err := app.SaveNoValidate(actor); err != nil {
		t.Fatal(err)
	}
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Load(map[string]any{"name": "Remote copy", "author": actor.Id, "public": true, "iri": "https://remote.example/api/v1/trail/remotetrail0001"})
	if err := app.SaveNoValidate(trail); err != nil {
		t.Fatal(err)
	}
	stored, err := app.FindRecordById(actors, actor.Id)
	if err != nil || stored.GetString("legacy_public_key") != marker {
		t.Fatalf("legacy key did not persist: %v", err)
	}

	exposed := func(t *testing.T, name string, body []byte) {
		t.Helper()
		if strings.Contains(string(body), "legacy_public_key") || strings.Contains(string(body), marker) {
			t.Fatalf("%s exposed the legacy key: %s", name, body)
		}
	}
	data, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	exposed(t, "record JSON", data)

	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, auth := range []string{"", token} {
		for _, path := range []string{
			"/api/collections/activitypub_actors/records/" + actor.Id,
			"/api/collections/activitypub_actors/records?filter=" + url.QueryEscape("iri='"+actor.GetString("iri")+"'"),
			"/api/collections/trails/records/" + trail.Id + "?expand=author",
		} {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			if auth != "" {
				request.Header.Set("Authorization", auth)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), actor.Id) {
				t.Fatalf("%s (auth=%t): %d %s", path, auth != "", response.Code, response.Body.String())
			}
			exposed(t, path, response.Body.Bytes())
		}
		// Nor can the field be probed through a filter.
		request := httptest.NewRequest(http.MethodGet, "/api/collections/activitypub_actors/records?filter="+url.QueryEscape("legacy_public_key='"+marker+"'"), nil)
		if auth != "" {
			request.Header.Set("Authorization", auth)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code == http.StatusOK && strings.Contains(response.Body.String(), actor.Id) {
			t.Fatalf("legacy key filter matched (auth=%t): %s", auth != "", response.Body.String())
		}
	}

	// The actor endpoint answers with this cached copy while its origin
	// cannot be reached.
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client {
		return &http.Client{Transport: legacyKeyRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody, Header: http.Header{}}, nil
		})}
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/activitypub/actor?iri="+url.QueryEscape(actor.GetString("iri")), nil)
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: app}
	event.Request, event.Response = request, response
	if err := routes.ActivitypubActor(event); err != nil || response.Code != http.StatusOK || !strings.Contains(response.Body.String(), actor.Id) {
		t.Fatalf("cached actor lookup: %v %d %s", err, response.Code, response.Body.String())
	}
	exposed(t, "actor endpoint", response.Body.Bytes())
}

type legacyKeyRoundTrip func(*http.Request) (*http.Response, error)

func (f legacyKeyRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestActorRefreshReplacesLegacyKey(t *testing.T) {
	app := newRealSchemaApp(t)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	const iri = "https://remote.example/api/v1/activitypub/user/alice"
	actor := core.NewRecord(collection)
	actor.Load(map[string]any{"username": "Alice", "preferred_username": "alice", "domain": "remote.example", "iri": iri, "inbox": iri + "/inbox", "legacy_public_key": "previous-key"})
	if err := app.SaveNoValidate(actor); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client {
		return &http.Client{Transport: legacyKeyRoundTrip(func(request *http.Request) (*http.Response, error) {
			body := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"confirmed-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"#main-key", iri)
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
		})}
	}))
	if _, err := federation.GetActorByIRI(app, context.Background(), iri, false); err != nil {
		t.Fatal(err)
	}
	stored, err := app.FindRecordById(collection, actor.Id)
	if err != nil || stored.GetString("public_key") != "confirmed-key" || stored.GetString("legacy_public_key") != "" {
		t.Fatalf("refreshed key did not replace the previous key: actor=%v err=%v", stored, err)
	}
}
