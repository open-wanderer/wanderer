package federation

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// A handle must never resolve to an instance actor, including through
// the remote path: a stored row that the handle filter skipped is found again by
// IRI once WebFinger names it, and a fetched Application document is an instance
// actor too.
func TestGetActorByHandleNeverReturnsInstanceActorFromRemotePath(t *testing.T) {
	for _, scenario := range []string{"stored-instance-actor", "fetched-application-document"} {
		t.Run(scenario, func(t *testing.T) {
			app := setupActorLookupTestApp(t)
			collection, err := app.FindCollectionByNameOrId("activitypub_actors")
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"actor_type", "preferred_username", "username", "inbox", "outbox", "summary", "icon", "legacy_public_key"} {
				collection.Fields.Add(&core.TextField{Name: name})
			}
			collection.Fields.Add(&core.DateField{Name: "published"})
			if err := app.Save(collection); err != nil {
				t.Fatal(err)
			}

			iri := "https://peer.example/api/v1/activitypub/instance"
			var stored *core.Record
			if scenario == "stored-instance-actor" {
				stored = core.NewRecord(collection)
				for field, value := range map[string]any{
					"iri": iri, "is_local": false, "actor_type": "instance", "domain": "peer.example",
					"preferred_username": "-instance", "username": "Stored name", "summary": "Stored summary",
					"inbox": iri + "/inbox", "outbox": iri + "/outbox", "public_key": "stored-key",
					"last_fetched": "2024-01-03 03:04:05.000Z",
				} {
					stored.Set(field, value)
				}
				if err := app.Save(stored); err != nil {
					t.Fatal(err)
				}
			}

			var webfingerRequests, actorRequests atomic.Int32
			previousActor, previousWebfinger := newActorFetchHTTPClient, newWebfingerHTTPClient
			t.Cleanup(func() { newActorFetchHTTPClient, newWebfingerHTTPClient = previousActor, previousWebfinger })
			response := func(request *http.Request, body string) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}
			}
			newWebfingerHTTPClient = func() *http.Client {
				return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
					webfingerRequests.Add(1)
					return response(request, fmt.Sprintf(`{"subject":"acct:peer@peer.example","links":[{"rel":"self","href":%q}]}`, iri)), nil
				})}
			}
			newActorFetchHTTPClient = func() *http.Client {
				return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
					actorRequests.Add(1)
					document := fmt.Sprintf(`{"id":%q,"type":"Application","name":"Fetched name","preferredUsername":"-instance","summary":"Fetched summary","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"fetched-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"#main-key", iri)
					return response(request, document), nil
				})}
			}

			actor, err := GetActorByHandle(app, t.Context(), "peer@peer.example", false)
			if actor != nil || !errors.Is(err, ErrActorNotFound) {
				id := ""
				if actor != nil {
					id = actor.Id + " (" + actor.GetString("actor_type") + ")"
				}
				t.Fatalf("GetActorByHandle(peer@peer.example) = %q, %v; want no actor and ErrActorNotFound", id, err)
			}
			if webfingerRequests.Load() != 1 {
				t.Errorf("WebFinger requests = %d, want 1", webfingerRequests.Load())
			}
			if stored == nil {
				return
			}
			// A stored instance actor is excluded before any fetch, and its row stays as it was.
			if actorRequests.Load() != 0 {
				t.Errorf("actor fetches = %d, want 0 for a stored instance actor", actorRequests.Load())
			}
			after, err := app.FindRecordById("activitypub_actors", stored.Id)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"username", "summary", "public_key", "preferred_username", "actor_type"} {
				if after.GetString(field) != stored.GetString(field) {
					t.Errorf("stored instance actor field %q changed from %q to %q", field, stored.GetString(field), after.GetString(field))
				}
			}
		})
	}
}
