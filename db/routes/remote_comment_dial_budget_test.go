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
	"pocketbase/util"
)

func TestRemoteRepliesDoNotRememberViewerDialRefusals(t *testing.T) {
	for _, kind := range []string{"comment", "note"} {
		for _, phase := range []string{"confirmation", "author"} {
			t.Run(kind+"/"+phase, func(t *testing.T) {
				f := newRemoteCommentFixture(t)
				t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
				origin := f.save("activitypub_actors", map[string]any{"preferred_username": "origin", "iri": "https://origin.example/users/origin", "domain": "origin.example", "public_key": "key", "last_fetched": time.Now()})
				trail := f.save("trails", map[string]any{"iri": "https://origin.example/api/v1/trail/parent", "name": "Remote trail", "public": true, "author": origin.Id})
				authorIRI := "https://reply.example/users/contributor"
				saveAuthor := func() {
					f.save("activitypub_actors", map[string]any{"preferred_username": "contributor", "iri": authorIRI, "domain": "reply.example", "public_key": "key", "last_fetched": time.Now()})
				}
				if phase == "confirmation" {
					saveAuthor()
				}
				iri := "https://reply.example/api/v1/comment/reply"
				if kind == "note" {
					iri = "https://reply.example/users/contributor/statuses/reply"
				}
				oldBudget := util.ActivityPubRateLimiter
				util.ActivityPubRateLimiter = util.NewRateLimiter(1, time.Minute)
				t.Cleanup(func() { util.ActivityPubRateLimiter = oldBudget })
				if err := util.ActivityPubRateLimiter.CheckRateLimit("anon:192.0.2.34", "reply.example"); err != nil {
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
				useRemoteCommentClient(t, &http.Client{Transport: remoteSyncTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
					if req.URL.Host == "origin.example" {
						return richTextSyncTransport{map[string]any{"items": []any{map[string]any{"iri": iri, "text": "Listed copy", "updated": "current"}}, "totalItems": 1, "totalPages": 1}}.RoundTrip(req)
					}
					if phase == "confirmation" && req.Context().Value("actor") == "anon:192.0.2.34" {
						response, err := realTransport.RoundTrip(req)
						if errors.Is(err, util.ErrRateLimited) {
							refusals.Add(1)
						}
						return response, err
					}
					served.Add(1)
					payload := map[string]any{"id": "reply", "iri": iri, "text": "Confirmed reply", "expand": map[string]any{"author": map[string]any{"iri": authorIRI}, "trail": map[string]any{"iri": trail.GetString("iri")}}}
					if kind == "note" {
						payload = map[string]any{"id": iri, "type": "Note", "content": "Confirmed reply", "attributedTo": authorIRI, "inReplyTo": trail.GetString("iri"), "to": []any{"https://www.w3.org/ns/activitystreams#Public"}}
					}
					return richTextSyncTransport{payload}.RoundTrip(req)
				}}})
				sync := func(addr string) {
					t.Helper()
					event := &core.RequestEvent{App: f.app}
					event.Request = httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
					event.Request.RemoteAddr = addr
					event.Response = httptest.NewRecorder()
					if err := syncRemoteComments(event, trail); err != nil {
						t.Fatal(err)
					}
				}
				sync("192.0.2.34:1234")
				if _, err := f.app.FindFirstRecordByData("comments", "iri", iri); err == nil {
					t.Fatal("reply with an unconfirmed object or author was imported")
				}
				key := canonicalLogCacheKey(f.app, trail, iri)
				canonicalRemoteLogCache.Lock()
				_, remembered := canonicalRemoteLogCache.entries[key]
				canonicalRemoteLogCache.Unlock()
				if remembered {
					t.Fatal("viewer dial refusal was remembered for other viewers")
				}
				if dns.Load() != 0 {
					t.Fatalf("refused budget reached DNS %d times", dns.Load())
				}
				if phase == "confirmation" {
					if refusals.Load() != 1 || served.Load() != 0 {
						t.Fatalf("refused confirmation: refusals%d served%d", refusals.Load(), served.Load())
					}
				} else {
					saveAuthor()
					if served.Load() != 1 {
						t.Fatalf("object confirmation count%d", served.Load())
					}
				}
				before := served.Load()
				sync("198.51.100.34:1234")
				saved, err := f.app.FindFirstRecordByData("comments", "iri", iri)
				if err != nil || saved.GetString("text") != "Confirmed reply" || served.Load() != before+1 {
					t.Fatalf("free viewer did not import reply: record%v err%v served%d", saved, err, served.Load())
				}
			})
		}
	}
}
