package federation

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type actorCacheCountingTransport struct {
	actorIdentityTransport
	calls int
}

func (t *actorCacheCountingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.calls++
	return t.actorIdentityTransport.RoundTrip(r)
}

func TestActorCacheRefreshKeepsExactHostAndPort(t *testing.T) {
	for _, tc := range []struct {
		name, domain, key string
		cached            bool
	}{
		{name: "verified", domain: "www.remote.example:8443", key: "cached-key", cached: true},
		{name: "host-case", domain: "WWW.REMOTE.EXAMPLE:8443", key: "cached-key", cached: true},
		{name: "empty-domain", key: "cached-key"},
		{name: "missing-key", domain: "www.remote.example:8443"},
		{name: "different-www", domain: "remote.example:8443", key: "cached-key"},
		{name: "normalized-host-only", domain: "remote.example", key: "cached-key"},
		{name: "missing-port", domain: "www.remote.example", key: "cached-key"},
		{name: "different-port", domain: "www.remote.example:9443", key: "cached-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
			app := setupActorDeleteTestApp(t)
			actors, err := app.FindCollectionByNameOrId("activitypub_actors")
			if err != nil {
				t.Fatal(err)
			}
			actors.Fields.Add(&core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.DateField{Name: "last_fetched"})
			if err := app.Save(actors); err != nil {
				t.Fatal(err)
			}
			iri := "https://www.remote.example:8443/users/alice"
			actor := core.NewRecord(actors)
			actor.Set("iri", iri)
			actor.Set("domain", tc.domain)
			actor.Set("public_key", tc.key)
			actor.Set("last_fetched", time.Now())
			if err := app.Save(actor); err != nil {
				t.Fatal(err)
			}
			transport := &actorCacheCountingTransport{actorIdentityTransport: actorIdentityTransport{document: fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"origin-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"/main-key", iri)}}
			original := newActorFetchHTTPClient
			newActorFetchHTTPClient = func() *http.Client { return &http.Client{Transport: transport} }
			t.Cleanup(func() { newActorFetchHTTPClient = original })
			got, err := GetActorByIRI(app, context.Background(), iri, false)
			if err != nil || got == nil {
				t.Fatalf("actor refresh failed: %v", err)
			}
			expectedCalls := 1
			expectedKey := "origin-key"
			if tc.cached {
				expectedCalls = 0
				expectedKey = tc.key
			}
			if transport.calls != expectedCalls || got.GetString("public_key") != expectedKey || !CachedActorIdentityMatchesOrigin(got) {
				t.Fatalf("cache boundary wrong: calls=%d domain=%q key=%q", transport.calls, got.GetString("domain"), got.GetString("public_key"))
			}
			if !tc.cached && got.GetString("domain") != "www.remote.example:8443" {
				t.Fatalf("refreshed host lost www or port: %q", got.GetString("domain"))
			}
			if _, err := GetActorByIRI(app, context.Background(), iri, false); err != nil || transport.calls != expectedCalls {
				t.Fatalf("verified refresh wasn't cached: calls=%d err=%v", transport.calls, err)
			}
		})
	}
}

func TestActorCacheRejectsMalformedOriginAndNil(t *testing.T) {
	if CachedActorIdentityMatchesOrigin(nil) {
		t.Fatal("nil cache trusted")
	}
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"})
	for _, iri := range []string{"https://user@remote.example/users/alice", "ftp://remote.example/alice", "/users/alice"} {
		actor := core.NewRecord(actors)
		actor.Set("iri", iri)
		actor.Set("domain", "remote.example")
		actor.Set("public_key", "key")
		if CachedActorIdentityMatchesOrigin(actor) {
			t.Fatalf("malformed cached origin trusted: %s", iri)
		}
	}
}
