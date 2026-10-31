package federation

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// Flood sizes, larger than the per-host fetch limits.
const (
	starvationContentFlood   = 301
	starvationLifecycleFlood = 31
	starvationVariantFlood   = 60
	// canonical path of a Wanderer instance actor
	starvationInstancePath = "/api/v1/activitypub/instance"
)

// newPeer is a Wanderer instance that is not a peer yet: it is reachable as
// http://newpeer.example on the default port and serves its instance actor only
// at the canonical path and only once ready is set.
type newPeer struct {
	iri      string
	key      *rsa.PrivateKey
	requests *atomic.Int64
	ready    *atomic.Bool
}

func (f *prefetchBudgetFixture) startNewPeer(t *testing.T) *newPeer {
	t.Helper()
	key, pubKey, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	np := &newPeer{
		iri:      "http://newpeer.example" + starvationInstancePath,
		key:      key,
		requests: &atomic.Int64{},
		ready:    &atomic.Bool{},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		np.requests.Add(1)
		if !np.ready.Load() || r.URL.Path != starvationInstancePath || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/activity+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@context":          "https://www.w3.org/ns/activitystreams",
			"type":              "Application",
			"id":                np.iri,
			"preferredUsername": "instance",
			"inbox":             np.iri + "/inbox",
			"outbox":            np.iri + "/outbox",
			"publicKey": map[string]any{
				"id":           np.iri + "#main-key",
				"owner":        np.iri,
				"publicKeyPem": pemKey,
			},
		})
	}))
	t.Cleanup(srv.Close)
	f.aliasHost("newpeer.example", srv)
	return np
}

func starvationFollow(t *testing.T, id, actorIRI string) []byte {
	t.Helper()
	follow := pub.FollowNew(pub.IRI(id), pub.IRI(prefetchLocalInstanceIRI))
	follow.Actor = pub.IRI(actorIRI)
	return marshalActivity(t, follow)
}

func (f *prefetchBudgetFixture) assertBobDelivered(t *testing.T) {
	t.Helper()
	bobIRI := f.srv.URL + "/actor/bob"
	trailIRI := f.srv.URL + "/api/v1/trail/bob-starvation"
	body := marshalActivity(t, signedGateCreateTrail(bobIRI, trailIRI))
	e := newSignedInstanceInboxEvent(t, f.app, f.bob, bobIRI+"#main-key", body)
	assertInstanceInboxStatus(t, InstanceInboxHandler(e), 200)
	if _, err := f.app.FindFirstRecordByData("trails", "iri", trailIRI); err != nil {
		t.Fatalf("trail from the uncached peer user must be stored: %v", err)
	}
}

func (f *prefetchBudgetFixture) assertNewPeerCanFollow(t *testing.T, np *newPeer) {
	t.Helper()
	body := starvationFollow(t, np.iri+"/activity/follow-1", np.iri)
	e := newSignedInstanceInboxEvent(t, f.app, np.key, np.iri+"#main-key", body)
	assertInstanceInboxStatus(t, InstanceInboxHandler(e), 200)

	actor, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", np.iri)
	if err != nil {
		t.Fatalf("new peer's instance actor must be stored: %v", err)
	}
	local, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", prefetchLocalInstanceIRI)
	if err != nil {
		t.Fatalf("find local instance actor: %v", err)
	}
	row, err := f.app.FindFirstRecordByFilter("follows", "follower={:a} && followee={:b} && status='pending'",
		map[string]any{"a": actor.Id, "b": local.Id})
	if err != nil || row == nil {
		t.Fatalf("a pending follows row from the new peer must exist: %v", err)
	}
}

func TestInstanceInboxUnsignedFloodCausesNoFetch(t *testing.T) {
	f := newPrefetchBudgetFixture(t)

	for i := 1; i <= starvationContentFlood; i++ {
		actor := fmt.Sprintf("%s/u%d", f.srv.URL, i)
		body := marshalActivity(t, signedGateCreateTrail(actor, fmt.Sprintf("%s/api/v1/trail/u%d", f.srv.URL, i)))
		if err := InstanceInboxHandler(f.unsignedEvent(body)); err == nil {
			t.Fatalf("request %d: unsigned activity must be refused", i)
		}
	}
	if got := f.fetch.Load(); got != 0 {
		t.Fatalf("unsigned requests caused %d fetches to the peer, want 0", got)
	}
	f.assertBobDelivered(t)
}

func TestInstanceInboxForgedFloodDoesNotStarvePeerUser(t *testing.T) {
	f := newPrefetchBudgetFixture(t)
	attacker, _, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	for i := 1; i <= starvationContentFlood; i++ {
		actor := fmt.Sprintf("%s/forged/%d", f.srv.URL, i)
		body := marshalActivity(t, signedGateCreateTrail(actor, fmt.Sprintf("%s/api/v1/trail/forged%d", f.srv.URL, i)))
		e := newSignedInstanceInboxEvent(t, f.app, attacker, actor+"#main-key", body)
		if err := InstanceInboxHandler(e); err == nil {
			t.Fatalf("request %d: forged activity must be refused", i)
		}
	}
	f.assertBobDelivered(t)
}

func TestInstanceInboxForgedFollowFloodDoesNotBlockNewPeer(t *testing.T) {
	f := newPrefetchBudgetFixture(t)
	np := f.startNewPeer(t)
	np.ready.Store(true)
	attacker, _, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	for i := 1; i <= starvationLifecycleFlood; i++ {
		actor := fmt.Sprintf("http://newpeer.example/x%d", i)
		body := starvationFollow(t, actor+"/activity/f", actor)
		e := newSignedInstanceInboxEvent(t, f.app, attacker, actor+"#main-key", body)
		if err := InstanceInboxHandler(e); err == nil {
			t.Fatalf("request %d: forged Follow must be refused", i)
		}
	}
	f.assertNewPeerCanFollow(t, np)
}

func TestInstanceInboxInstanceActorVariantFloodSharesLifecycleBudget(t *testing.T) {
	f := newPrefetchBudgetFixture(t)
	np := f.startNewPeer(t) // answers 404 to everything until ready is set
	attacker, _, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	var variants []string
	for i := 1; i <= 15; i++ {
		variants = append(variants,
			fmt.Sprintf("http://newpeer.example/x%d%s", i, starvationInstancePath),
			fmt.Sprintf("http://newpeer.example%s?n=%d", starvationInstancePath, i),
			fmt.Sprintf("http://newpeer.example%s#%d", starvationInstancePath, i),
			fmt.Sprintf("http://newpeer.example:%d%s", 8100+i, starvationInstancePath),
		)
	}
	if len(variants) != starvationVariantFlood {
		t.Fatalf("variants = %d, want %d", len(variants), starvationVariantFlood)
	}

	for i, iri := range variants {
		keyIRI := iri
		if idx := strings.IndexAny(keyIRI, "?#"); idx >= 0 {
			keyIRI = keyIRI[:idx]
		}
		body := starvationFollow(t, fmt.Sprintf("http://newpeer.example/activity/v%d", i), iri)
		e := newSignedInstanceInboxEvent(t, f.app, attacker, keyIRI+"#main-key", body)
		if err := InstanceInboxHandler(e); err == nil {
			t.Fatalf("variant %q: forged Follow must be refused", iri)
		}
	}
	if got := np.requests.Load(); got > 30 {
		t.Fatalf("variant flood caused %d requests to the host, want at most 30 (one shared per-host budget)", got)
	}

	np.ready.Store(true)
	f.assertNewPeerCanFollow(t, np)
}

func TestInstanceInboxPrecheckRefusesBeforeFetch(t *testing.T) {
	bodyFor := func(f *prefetchBudgetFixture) []byte {
		bobIRI := f.srv.URL + "/actor/bob"
		return marshalActivity(t, signedGateCreateTrail(bobIRI, f.srv.URL+"/api/v1/trail/precheck"))
	}

	tests := []struct {
		name  string
		build func(f *prefetchBudgetFixture, body []byte) *core.RequestEvent
	}{
		{"no Signature header", func(f *prefetchBudgetFixture, body []byte) *core.RequestEvent {
			return f.unsignedEvent(body)
		}},
		{"keyId on another host", func(f *prefetchBudgetFixture, body []byte) *core.RequestEvent {
			return newSignedInstanceInboxEvent(t, f.app, f.bob, "https://other.example/actor/bob#main-key", body)
		}},
		{"Date two hours old", func(f *prefetchBudgetFixture, body []byte) *core.RequestEvent {
			return signedInstanceInboxEventAt(t, f.app, f.bob, f.srv.URL+"/actor/bob#main-key", body, body, time.Now().Add(-2*time.Hour))
		}},
		{"body differs from the signed Digest", func(f *prefetchBudgetFixture, body []byte) *core.RequestEvent {
			other := []byte(strings.Replace(string(body), "Test Trail", "Other Trail", 1))
			return signedInstanceInboxEventAt(t, f.app, f.bob, f.srv.URL+"/actor/bob#main-key", body, other, time.Now())
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newPrefetchBudgetFixture(t)
			body := bodyFor(f)
			assertInstanceInboxStatus(t, InstanceInboxHandler(tc.build(f, body)), 401)
			if got := f.fetch.Load(); got != 0 {
				t.Fatalf("refused request caused %d fetches, want 0", got)
			}
		})
	}
}

func TestIsInstanceActorIRI(t *testing.T) {
	const p = "/api/v1/activitypub/instance"
	tests := []struct {
		iri  string
		want string // normalized identifier part, empty when not canonical
	}{
		{"https://a.example" + p, "https://a.example" + p},
		{"http://a.example" + p, "http://a.example" + p},
		{"https://A.Example" + p, "https://a.example" + p},
		{"https://a.example:443" + p, "https://a.example" + p},
		{"http://a.example:80" + p, "http://a.example" + p},
		{"https://a.example/prefix" + p, ""},
		{"https://a.example" + p + "/", ""},
		{"https://a.example" + p + "?n=1", ""},
		{"https://a.example" + p + "?", ""},
		{"https://a.example" + p + "#k", ""},
		{"https://a.example" + p + "#", ""},
		{"https://a.example:8443" + p, ""},
		{"http://a.example:443" + p, ""},
		{"https://user@a.example" + p, ""},
		{"https://a.example." + p, ""},
		{"https://a.example/api/v1/activitypub/%69nstance", ""},
		{"ftp://a.example" + p, ""},
		{"https://a.example/api/v1/activitypub/user/x", ""},
		{"https://a.example" + p + "/inbox", ""},
		{"", ""},
		{"https://a.example:bad" + p, ""},
		{"%zz", ""},
	}
	for _, tc := range tests {
		t.Run(tc.iri, func(t *testing.T) {
			got, ok := isInstanceActorIRI(tc.iri)
			if ok != (tc.want != "") || got != tc.want {
				t.Fatalf("isInstanceActorIRI(%q) = (%q, %v), want (%q, %v)", tc.iri, got, ok, tc.want, tc.want != "")
			}
		})
	}
}

func TestContentPrefetchSurgeWarnsOncePerHost(t *testing.T) {
	origCounter, origSampler := instanceContentFetchCounter, instanceContentFetchWarnSampler
	instanceContentFetchCounter = util.NewRateLimiter(3, time.Minute)
	instanceContentFetchWarnSampler = util.NewRateLimiter(1, time.Minute)
	t.Cleanup(func() {
		instanceContentFetchCounter, instanceContentFetchWarnSampler = origCounter, origSampler
	})

	app := newInboxTestApp(t)
	for i := 1; i <= 10; i++ {
		want := i == 4
		if got := noteContentPrefetch(app, "a.example"); got != want {
			t.Fatalf("a.example call %d: logged = %v, want %v", i, got, want)
		}
	}
	for i := 1; i <= 4; i++ {
		want := i == 4
		if got := noteContentPrefetch(app, "b.example"); got != want {
			t.Fatalf("b.example call %d: logged = %v, want %v", i, got, want)
		}
	}
}
