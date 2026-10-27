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

	"pocketbase/util"
)

func TestRemoteSummitConfirmationDoesNotRememberViewerDialRefusal(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://reply.example/users/contributor", "last_fetched": time.Now()})
	iri := "https://reply.example/api/v1/summit-log/reply"
	oldBudget := util.ActivityPubRateLimiter
	util.ActivityPubRateLimiter = util.NewRateLimiter(1, time.Minute)
	t.Cleanup(func() { util.ActivityPubRateLimiter = oldBudget })
	if err := util.ActivityPubRateLimiter.CheckRateLimit("anon:192.0.2.30", "reply.example"); err != nil {
		t.Fatal(err)
	}
	oldResolver := net.DefaultResolver
	var dns, refusals, served atomic.Int32
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
		dns.Add(1)
		return nil, errors.New("unexpected name lookup")
	}}
	t.Cleanup(func() { net.DefaultResolver = oldResolver })
	realTransport := util.SafeHTTPClient().Transport.(*http.Transport)
	t.Cleanup(realTransport.CloseIdleConnections)
	oldClient := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = oldClient })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "reply.example" {
				if req.Context().Value("actor") == "anon:192.0.2.30" {
					response, err := realTransport.RoundTrip(req)
					if errors.Is(err, util.ErrRateLimited) {
						refusals.Add(1)
					}
					return response, err
				}
				served.Add(1)
				return richTextSyncTransport{map[string]any{"id": "reply", "iri": iri, "text": "Confirmed reply", "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}}.RoundTrip(req)
			}
			return richTextSyncTransport{map[string]any{"id": "trail", "name": "Synced", "public": true, "expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"id": "reply", "iri": iri, "text": "Listed copy"}}}}}.RoundTrip(req)
		}}}
	}
	sync := func(addr string) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/remote/trail", nil)
		request.RemoteAddr = addr
		ctx, err := util.GetSafeActorContext(request, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := performFullSync(f.app, ctx, nil, f.trail); err != nil {
			t.Fatal(err)
		}
	}
	sync("192.0.2.30:1234")
	key := canonicalLogCacheKey(f.app, f.trail, iri)
	canonicalRemoteLogCache.Lock()
	_, remembered := canonicalRemoteLogCache.entries[key]
	canonicalRemoteLogCache.Unlock()
	if remembered {
		t.Fatal("viewer dial refusal was remembered for other viewers")
	}
	if refusals.Load() != 1 || dns.Load() != 0 || served.Load() != 0 {
		t.Fatalf("refused request: refusals%d DNS%d served%d", refusals.Load(), dns.Load(), served.Load())
	}
	sync("198.51.100.30:1234")
	stored, err := f.app.FindFirstRecordByData("summit_logs", "iri", iri)
	if err != nil || stored.GetString("text") != "Confirmed reply" || served.Load() != 1 {
		t.Fatalf("next viewer did not import reply: stored%v err%v served%d", stored, err, served.Load())
	}
}
