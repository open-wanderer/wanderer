package federation

import (
	"errors"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestActorAndWebfingerMissingResponsesKeepTheirNotFoundClassification(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	app := setupActorDeleteTestApp(t)
	for _, status := range []int{http.StatusNotFound, http.StatusGone, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			originalActor, originalWebfinger := newActorFetchHTTPClient, newWebfingerHTTPClient
			t.Cleanup(func() { newActorFetchHTTPClient, newWebfingerHTTPClient = originalActor, originalWebfinger })
			client := func() *http.Client {
				return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
				})}
			}
			newActorFetchHTTPClient, newWebfingerHTTPClient = client, client
			_, _, _, actorErr := fetchRemoteActor(app, t.Context(), "https://remote.example/users/alice", false)
			_, webfingerErr := iriFromHandle(t.Context(), "remote.example", "alice")
			missing := status == http.StatusNotFound || status == http.StatusGone
			if errors.Is(actorErr, ErrActorNotFound) != missing || errors.Is(webfingerErr, ErrActorNotFound) != missing {
				t.Fatalf("lost missing actor classification: actor=%v webfinger=%v", actorErr, webfingerErr)
			}
		})
	}
}

func TestUnknownLocalActorLookupDoesNotRequestWebfinger(t *testing.T) {
	app := setupActorDeleteTestApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(&core.TextField{Name: "preferred_username"}, &core.TextField{Name: "domain"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	original := newWebfingerHTTPClient
	t.Cleanup(func() { newWebfingerHTTPClient = original })
	requests := 0
	newWebfingerHTTPClient = func() *http.Client {
		requests++
		return &http.Client{Transport: identityLookupTransport(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected local Webfinger request")
		})}
	}
	actor, err := GetActorByHandle(app, t.Context(), "nobody", false)
	if actor != nil || !errors.Is(err, ErrActorNotFound) || requests != 0 {
		t.Fatalf("unknown local actor = %v, err = %v, requests = %d", actor, err, requests)
	}
}

func TestActorFetchPreservesTransportErrorCause(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	app := setupActorDeleteTestApp(t)
	original := newActorFetchHTTPClient
	t.Cleanup(func() { newActorFetchHTTPClient = original })
	cause := errors.New("connection unavailable")
	newActorFetchHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(*http.Request) (*http.Response, error) {
			return nil, cause
		})}
	}
	_, _, _, err := fetchRemoteActor(app, t.Context(), "https://remote.example/users/alice", false)
	if !errors.Is(err, cause) || errors.Is(err, ErrActorNotFound) {
		t.Fatalf("transport error cause changed: %v", err)
	}
}
