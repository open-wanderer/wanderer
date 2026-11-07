package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"pocketbase/federation"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// Resolving a remote follower page stores every listed actor, so anonymous
// callers must be refused before anything is fetched.
func TestRemoteProfileFollowsListRefusesAnonymousRemoteHandles(t *testing.T) {
	f := setupHandleLookupTest(t, "https://wanderer.example", http.StatusNotFound)
	for _, tc := range []struct {
		handle       string
		unauthorized bool
	}{
		{handle: "@alice@remote.example", unauthorized: true},
		{handle: "@alice@wanderer.example"},
		{handle: "@alice"},
	} {
		t.Run(tc.handle, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/remote/profile/"+tc.handle+"/follows", nil)
			request.SetPathValue("handle", tc.handle)
			e := &core.RequestEvent{}
			e.App, e.Request, e.Response = f.app, request, httptest.NewRecorder()
			err := RemoteProfileFollowsList(e)
			var apiErr *router.ApiError
			unauthorized := errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
			if unauthorized != tc.unauthorized {
				t.Fatalf("err = %v, want unauthorized = %v", err, tc.unauthorized)
			}
			if tc.unauthorized && len(f.transport.seen()) != 0 {
				t.Fatalf("refused request still fetched %v", f.transport.seen())
			}
		})
	}
}

// The actor lookup stores a remote actor it fetches, so it is refused the same
// way; local actors stay public.
func TestActivitypubActorRefusesAnonymousRemoteLookups(t *testing.T) {
	f := setupHandleLookupTest(t, "https://wanderer.example", http.StatusNotFound)
	for _, tc := range []struct {
		query        string
		unauthorized bool
	}{
		{query: "resource=acct:alice@remote.example", unauthorized: true},
		{query: "iri=https://remote.example/users/alice", unauthorized: true},
		{query: "resource=acct:alice@wanderer.example"},
		{query: "resource=acct:alice"},
		{query: "iri=https://wanderer.example/api/v1/activitypub/user/alice"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			e := &core.RequestEvent{}
			e.App, e.Request, e.Response = f.app, httptest.NewRequest(http.MethodGet, "/activitypub/actor?"+tc.query, nil), httptest.NewRecorder()
			err := ActivitypubActor(e)
			var apiErr *router.ApiError
			unauthorized := errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
			if unauthorized != tc.unauthorized {
				t.Fatalf("err = %v, want unauthorized = %v", err, tc.unauthorized)
			}
			if tc.unauthorized && len(f.transport.seen()) != 0 {
				t.Fatalf("refused request still fetched %v", f.transport.seen())
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// A follower whose instance is down still counts, so it stays listed from the
// stored copy. Anything that speaks against the actor keeps it off the list.
func TestRemoteProfileFollowsListKeepsFollowersOfUnreachableInstances(t *testing.T) {
	f := setupHandleLookupTest(t, "https://wanderer.example", http.StatusNotFound)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	f.actors.Fields.Add(&core.TextField{Name: "followers"})
	if err := f.app.Save(f.actors); err != nil {
		t.Fatal(err)
	}

	stale := time.Now().Add(-3 * time.Hour)
	remote := func(host, key string) string {
		iri := "https://" + host + "/users/follower"
		f.record(t, "activitypub_actors", map[string]any{"preferred_username": "follower", "domain": host, "iri": iri, "public_key": key, "last_fetched": stale})
		return iri
	}
	down := remote("down.example", "test-key")
	refused := remote("refused.example", "test-key")
	gone := remote("gone.example", "test-key")
	spoofed := remote("spoofed.example", "test-key")
	unverified := remote("unverified.example", "")

	private := f.localActor(t, "private", "https://wanderer.example", "wanderer.example")
	settings, err := f.app.FindFirstRecordByData("settings", "user", private.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	settings.Set("privacy", `{"account":"private"}`)
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}

	owner := f.localActor(t, "alice", "https://wanderer.example", "wanderer.example")
	owner.Set("followers", owner.GetString("iri")+"/followers")
	if err := f.app.Save(owner); err != nil {
		t.Fatal(err)
	}

	listed := []string{down, refused, gone, spoofed, unverified, private.GetString("iri"), "https://unknown.example/users/follower"}
	page, err := json.Marshal(map[string]any{"type": "OrderedCollectionPage", "totalItems": len(listed), "orderedItems": listed})
	if err != nil {
		t.Fatal(err)
	}
	respond := func(r *http.Request, status int, body string) *http.Response {
		return &http.Response{StatusCode: status, Header: make(http.Header), Request: r, Body: io.NopCloser(strings.NewReader(body))}
	}
	t.Cleanup(federation.SetCollectionFetchClientForTesting(func() *http.Client {
		return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return respond(r, http.StatusOK, string(page)), nil
		})}
	}))
	t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client {
		return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Host {
			case "refused.example":
				return nil, errors.New("connection refused")
			case "gone.example":
				return respond(r, http.StatusGone, ""), nil
			case "spoofed.example":
				iri := "https://elsewhere.example/users/follower"
				return respond(r, http.StatusOK, fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"follower","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"other-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"#main-key", iri)), nil
			}
			return respond(r, http.StatusServiceUnavailable, "no available server"), nil
		})}
	}))

	request := httptest.NewRequest(http.MethodGet, "/remote/profile/@alice/follows?type=followers", nil)
	request.SetPathValue("handle", "@alice")
	recorder := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.App, e.Request, e.Response = f.app, request, recorder
	if err := RemoteProfileFollowsList(e); err != nil {
		t.Fatal(err)
	}

	var result struct {
		TotalItems int              `json:"totalItems"`
		Items      []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, item := range result.Items {
		got = append(got, item["iri"].(string))
	}
	slices.Sort(got)
	if want := []string{down, refused}; !slices.Equal(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
	if result.TotalItems != len(listed) {
		t.Fatalf("totalItems = %d, want %d", result.TotalItems, len(listed))
	}
}
