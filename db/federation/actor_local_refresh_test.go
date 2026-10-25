package federation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestWebfingerLookupLeavesStoredLocalActorUnchanged(t *testing.T) {
	for _, scenario := range []struct {
		name, origin string
	}{
		{"current-origin", "https://local.example"},
		{"former-origin", "https://former.example"},
	} {
		for _, creation := range []string{"before-lookup", "during-fetch"} {
			t.Run(scenario.name+"/"+creation, func(t *testing.T) {
				app := setupActorLookupTestApp(t)
				collection, err := app.FindCollectionByNameOrId("activitypub_actors")
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"type", "user", "preferred_username", "username", "inbox", "outbox", "summary", "icon", "private_key"} {
					collection.Fields.Add(&core.TextField{Name: name})
				}
				collection.Fields.Add(&core.DateField{Name: "published"})
				if err := app.Save(collection); err != nil {
					t.Fatal(err)
				}

				iri := scenario.origin + "/api/v1/activitypub/user/local-profile"
				local := core.NewRecord(collection)
				for field, value := range map[string]any{
					"iri": iri, "is_local": true, "type": "person", "user": "local-user",
					"domain":   strings.TrimPrefix(scenario.origin, "https://"),
					"username": "Local Profile", "preferred_username": "local-profile",
					"inbox": iri + "/inbox", "outbox": iri + "/outbox",
					"followers": iri + "/followers", "following": iri + "/following",
					"summary": "Local profile summary", "icon": scenario.origin + "/profile.png",
					"public_key": "local-public-key", "private_key": "local-private-key",
					"published": "2024-01-02 03:04:05.000Z", "last_fetched": "2024-01-03 03:04:05.000Z",
					"follower_count": 23, "following_count": 41,
					"follower_count_pending": true, "following_count_pending": true,
				} {
					local.Set(field, value)
				}
				saveLocal := func() map[string]any {
					t.Helper()
					if err := app.Save(local); err != nil {
						t.Fatal(err)
					}
					saved, err := app.FindRecordById("activitypub_actors", local.Id)
					if err != nil {
						t.Fatal(err)
					}
					return saved.FieldsData()
				}
				var before map[string]any
				if creation == "before-lookup" {
					before = saveLocal()
				}
				originChangesDuringFetch := scenario.name == "current-origin" && creation == "during-fetch"
				if originChangesDuringFetch {
					// Start with a foreign IRI so the fetch reaches the late
					// local-record guard when the origin and record change.
					t.Setenv("ORIGIN", "https://initial.example")
				}

				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				started, resume := make(chan struct{}), make(chan struct{})
				var workers sync.WaitGroup
				var webfingerRequests, actorRequests, collectionRequests atomic.Int32
				previousActor, previousWebfinger, previousCollection := newActorFetchHTTPClient, newWebfingerHTTPClient, newHTTPClient
				defer func() {
					cancel()
					workers.Wait()
					newActorFetchHTTPClient, newWebfingerHTTPClient, newHTTPClient = previousActor, previousWebfinger, previousCollection
				}()
				response := func(request *http.Request, body string) *http.Response {
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}
				}
				newWebfingerHTTPClient = func() *http.Client {
					return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
						webfingerRequests.Add(1)
						if request.URL.Host != "directory.example" || request.URL.Path != "/.well-known/webfinger" || request.URL.Query().Get("resource") != "acct:remote-profile@directory.example" {
							return nil, fmt.Errorf("unexpected WebFinger request: %s", request.URL)
						}
						return response(request, fmt.Sprintf(`{"subject":"acct:remote-profile@directory.example","links":[{"rel":"self","href":%q}]}`, iri)), nil
					})}
				}
				newActorFetchHTTPClient = func() *http.Client {
					return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
						if actorRequests.Add(1) == 1 {
							close(started)
						}
						if request.URL.String() != iri {
							return nil, fmt.Errorf("unexpected actor request: %s", request.URL)
						}
						select {
						case <-resume:
						case <-ctx.Done():
							return nil, ctx.Err()
						}
						document := fmt.Sprintf(`{"id":%q,"type":"Person","name":"Remote Profile","preferredUsername":"remote-profile","summary":"Remote summary","inbox":%q,"outbox":%q,"followers":%q,"following":%q,"published":"2025-02-03T04:05:06Z","icon":{"type":"Image","url":"https://directory.example/remote.png"},"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"remote-public-key"}}`, iri, iri+"/new-inbox", iri+"/new-outbox", iri+"/new-followers", iri+"/new-following", iri+"#main-key", iri)
						return response(request, document), nil
					})}
				}
				newHTTPClient = func() *http.Client {
					return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
						collectionRequests.Add(1)
						if request.URL.String() != iri+"/new-followers" && request.URL.String() != iri+"/new-following" {
							return nil, fmt.Errorf("unexpected collection request: %s", request.URL)
						}
						return concurrentProfileCollectionResponse(request, 73), nil
					})}
				}

				type lookupResult struct {
					actor *core.Record
					err   error
				}
				completed := make(chan lookupResult, 1)
				workers.Add(1)
				go func() {
					defer workers.Done()
					actor, err := GetActorByHandle(app, ctx, "remote-profile@directory.example", false)
					completed <- lookupResult{actor, err}
				}()
				blockedBeforeFetch := scenario.name == "current-origin" && creation == "before-lookup"
				if !blockedBeforeFetch {
					awaitConcurrentProfileEvent(t, ctx, started, "actor fetch")
					if originChangesDuringFetch {
						t.Setenv("ORIGIN", scenario.origin)
					}
					if creation == "during-fetch" {
						before = saveLocal()
					}
					close(resume)
				}
				var result lookupResult
				select {
				case result = <-completed:
				case <-ctx.Done():
					t.Fatalf("waiting for profile lookup: %v", ctx.Err())
				}
				wantError := ErrInvalidActorResponse
				var wantActorRequests, wantCollectionRequests int32 = 1, 2
				if blockedBeforeFetch {
					wantError = ErrActorNotFound
					wantActorRequests, wantCollectionRequests = 0, 0
				}
				if result.actor != nil || !errors.Is(result.err, wantError) {
					t.Errorf("WebFinger lookup returned actor=%v err=%v, want no actor and %v", result.actor, result.err, wantError)
				}
				stored, err := app.FindRecordById("activitypub_actors", local.Id)
				if err != nil {
					t.Fatal(err)
				}
				for field, value := range before {
					if !reflect.DeepEqual(stored.FieldsData()[field], value) {
						t.Errorf("local actor field %q changed during remote profile lookup", field)
					}
				}
				if count, err := app.CountRecords("activitypub_actors"); err != nil || count != 1 {
					t.Errorf("actor records after lookup: count=%d err=%v", count, err)
				}
				if webfingerRequests.Load() != 1 || actorRequests.Load() != wantActorRequests || collectionRequests.Load() != wantCollectionRequests {
					t.Errorf("requests: WebFinger=%d actor=%d collections=%d", webfingerRequests.Load(), actorRequests.Load(), collectionRequests.Load())
				}
			})
		}
	}
}
