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

func TestRemoteSummitAuthorDialRefusalIsNotRemembered(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	f := newRemoteChildFixture(t, "summit_log")
	authorIRI := "https://reply.example/users/contributor"
	iri := "https://reply.example/api/v1/summit-log/reply"
	oldLimiter := util.ActivityPubRateLimiter
	util.ActivityPubRateLimiter = util.NewRateLimiter(1, time.Minute)
	t.Cleanup(func() { util.ActivityPubRateLimiter = oldLimiter })
	if err := util.ActivityPubRateLimiter.CheckRateLimit("anon:192.0.2.33", "reply.example"); err != nil {
		t.Fatal(err)
	}
	oldResolver := net.DefaultResolver
	var dns atomic.Int32
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
		dns.Add(1)
		return nil, errors.New("unexpected name lookup")
	}}
	t.Cleanup(func() { net.DefaultResolver = oldResolver })
	request := httptest.NewRequest(http.MethodGet, "/remote/trail", nil)
	request.RemoteAddr = "192.0.2.33:1234"
	ctx, err := util.GetSafeActorContext(request, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveRemoteActorForSync(f.app, ctx, authorIRI); !errors.Is(err, util.ErrRateLimited) {
		t.Fatalf("actor lookup did not reach the real dial refusal: %v", err)
	}
	oldClient := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = oldClient })
	var confirmations atomic.Int32
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
			confirmations.Add(1)
			return richTextSyncTransport{map[string]any{"id": "reply", "iri": iri, "text": "Confirmed reply", "expand": map[string]any{"author": map[string]any{"iri": authorIRI}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}}.RoundTrip(req)
		}}}
	}
	data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"id": "reply", "iri": iri}}}}
	if got := fetchCanonicalRemoteSummitLogs(f.app, ctx, data, f.trail, "https://remote.example"); len(got) != 0 {
		t.Fatalf("unresolved author accepted: %v", got)
	}
	key := canonicalLogCacheKey(f.app, f.trail, iri)
	canonicalRemoteLogCache.Lock()
	_, remembered := canonicalRemoteLogCache.entries[key]
	canonicalRemoteLogCache.Unlock()
	if remembered {
		t.Fatal("author dial refusal was remembered for other viewers")
	}
	if dns.Load() != 0 || confirmations.Load() != 1 {
		t.Fatalf("DNS%d confirmations%d", dns.Load(), confirmations.Load())
	}
}
