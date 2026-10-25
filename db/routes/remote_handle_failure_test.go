package routes

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"pocketbase/federation"
	"pocketbase/util"
)

func TestUnknownRemoteHandleObjectsUsePublicFailureStatusAndShortNegativeCache(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, status := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
			t.Run(collection+"/"+http.StatusText(status), func(t *testing.T) {
				f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
				actor := f.remoteActor(t, "alice", "remote.example")
				actor.Set("public_key", "")
				if err := f.app.Save(actor); err != nil {
					t.Fatal(err)
				}
				original := remoteHandleGetActor
				t.Cleanup(func() { remoteHandleGetActor = original })
				calls := 0
				remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) {
					calls++
					if status == http.StatusNotFound {
						return actor, federation.ErrActorNotFound
					}
					return actor, errors.New("origin unavailable")
				}
				for i := 0; i < 2; i++ {
					code, _ := f.get(t, collection, "unknown00000001", "@alice@remote.example")
					if code != status {
						t.Fatalf("unknown object failed with%d, want%d", code, status)
					}
				}
				if calls != 1 {
					t.Fatalf("unavailable actor was fetched%d times, want1", calls)
				}
				cached := f.object(t, collection, actor.Id, "https://remote.example", "cached000000001", false, false)
				if code, _ := f.get(t, collection, cached.Id, "@alice@remote.example"); code != http.StatusOK {
					t.Fatalf("completed copy lost during cached actorfailure: %d", code)
				}
				if calls != 1 {
					t.Fatal("serving cached copy retried the known failed actor")
				}
			})
		}
	}
}

func TestHandleResolutionFailureCacheSeparatesCallersAndExpires(t *testing.T) {
	f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
	user := f.record(t, "users", map[string]any{"email": "handle-viewer@example.com", "password": "handle-test-password"})
	original := remoteHandleGetActor
	t.Cleanup(func() { remoteHandleGetActor = original })
	calls := 0
	remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) {
		calls++
		return nil, errors.New("origin unavailable")
	}
	request := func(auth *core.Record) {
		t.Helper()
		event := &core.RequestEvent{App: f.app}
		event.Auth = auth
		event.Request = httptest.NewRequest(http.MethodGet, "/remote/trail/unknown00000001?handle=%40alice%40remote.example", nil)
		event.Request.SetPathValue("id", "unknown00000001")
		event.Response = httptest.NewRecorder()
		var apiError *router.ApiError
		if err := RemoteTrailGet(event); !errors.As(err, &apiError) || apiError.Status != http.StatusServiceUnavailable {
			t.Fatalf("unavailable handle status=%v", err)
		}
	}
	request(nil)
	request(user)
	request(nil)
	if calls != 2 {
		t.Fatalf("callers shared negative actor outcome or missed the cache: calls%d", calls)
	}
	remoteHandleFailures.Lock()
	for key, value := range remoteHandleFailures.entries {
		value.until = time.Now().Add(-time.Second)
		remoteHandleFailures.entries[key] = value
	}
	remoteHandleFailures.Unlock()
	request(nil)
	if calls != 3 {
		t.Fatalf("expired failure never retried: calls%d", calls)
	}
}

func TestRateLimitedHandleLookupDoesNotHideNewStoredCopies(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		t.Run(collection, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
			originalLookup := remoteHandleGetActor
			lookupCalls := 0
			remoteHandleGetActor = func(app core.App, ctx context.Context, handle string, follows bool) (*core.Record, error) {
				lookupCalls++
				return originalLookup(app, ctx, handle, follows)
			}
			t.Cleanup(func() { remoteHandleGetActor = originalLookup })
			previousLimiter := util.ActivityPubRateLimiter
			util.ActivityPubRateLimiter = util.NewRateLimiter(30, time.Minute)
			t.Cleanup(func() { util.ActivityPubRateLimiter = previousLimiter })
			for range 30 {
				if err := util.ActivityPubRateLimiter.CheckRateLimit("anon:192.0.2.1", "budget.example"); err != nil {
					t.Fatal(err)
				}
			}
			previousResolver := net.DefaultResolver
			var lookups atomic.Int32
			net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
				lookups.Add(1)
				return nil, errors.New("unexpected name lookup")
			}}
			t.Cleanup(func() { net.DefaultResolver = previousResolver })
			request := func(remoteAddr string) int {
				t.Helper()
				event := &core.RequestEvent{App: f.app}
				event.Request = httptest.NewRequest(http.MethodGet, "/remote/object/remote000000001?handle=%40alice%40budget.example", nil)
				event.Request.RemoteAddr = remoteAddr
				event.Request.SetPathValue("id", "remote000000001")
				response := httptest.NewRecorder()
				event.Response = response
				handler := RemoteTrailGet
				if collection == "lists" {
					handler = RemoteListGet
				}
				var apiError *router.ApiError
				if err := handler(event); errors.As(err, &apiError) {
					return apiError.Status
				} else if err != nil {
					t.Fatal(err)
				}
				return response.Code
			}
			for range 2 {
				if code := request("192.0.2.1:1"); code != http.StatusTooManyRequests {
					t.Fatalf("exhausted dial budget answered %d", code)
				}
			}
			if lookupCalls != 2 {
				t.Fatalf("refused dial budget was remembered: lookup calls %d", lookupCalls)
			}
			alice := f.remoteActor(t, "alice", "budget.example")
			f.object(t, collection, alice.Id, "https://budget.example", "remote000000001", false, false)
			for _, remoteAddr := range []string{"192.0.2.1:1", "198.51.100.7:1"} {
				if code := request(remoteAddr); code != http.StatusOK {
					t.Fatalf("stored copy unavailable for %s: status %d", remoteAddr, code)
				}
			}
			if lookups.Load() != 0 || len(f.transport.seen()) != 0 {
				t.Fatalf("unexpected network activity: DNS %d, object requests %v", lookups.Load(), f.transport.seen())
			}
		})
	}
}

func TestRememberedHandleFailureUsesActorStoredLater(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		t.Run(collection, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
			original := remoteHandleGetActor
			t.Cleanup(func() { remoteHandleGetActor = original })
			calls := 0
			remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) {
				calls++
				return nil, errors.New("origin unavailable")
			}
			if code, _ := f.get(t, collection, "remote000000001", "@alice@remote.example"); code != http.StatusServiceUnavailable {
				t.Fatalf("unavailable handle answered %d", code)
			}
			alice := f.remoteActor(t, "alice", "remote.example")
			f.object(t, collection, alice.Id, "https://remote.example", "remote000000001", false, false)
			if code, _ := f.get(t, collection, "remote000000001", "@alice@remote.example"); code != http.StatusOK {
				t.Fatalf("copy stored after failed handle lookup answered %d", code)
			}
			if calls != 1 || len(f.transport.seen()) != 0 {
				t.Fatalf("stored copy retried its origin: lookups %d, object requests %v", calls, f.transport.seen())
			}
		})
	}
}
