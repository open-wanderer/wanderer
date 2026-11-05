package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
