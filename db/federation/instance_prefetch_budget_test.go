package federation

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

type prefetchBudgetFixture struct {
	app   core.App
	srv   *httptest.Server
	bob   *rsa.PrivateKey
	fetch *atomic.Int64
	// aliases maps a lower-case hostname to the host:port the dialer connects to
	// instead, on any port. The rate limit is still counted for the hostname.
	aliases sync.Map
}

// aliasHost makes host (for example "newpeer.example") reachable at srv.
func (f *prefetchBudgetFixture) aliasHost(host string, srv *httptest.Server) {
	f.aliases.Store(strings.ToLower(host), srv.Listener.Addr().String())
}

const prefetchLocalInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"

// newPrefetchBudgetFixture starts a peer server serving one actor
// (srv/actor/bob), makes its host an accepted peer, and installs a fresh rate
// limiter and an HTTP client that counts every dial like util.SafeHTTPClient but
// allows loopback.
func newPrefetchBudgetFixture(t *testing.T) *prefetchBudgetFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	origLimiter := util.ActivityPubRateLimiter
	util.ActivityPubRateLimiter = util.NewRateLimiter(30, time.Minute)
	f := &prefetchBudgetFixture{fetch: &atomic.Int64{}}
	origClient := newHTTPClient
	newHTTPClient = func() *http.Client {
		dialer := &net.Dialer{Timeout: 5 * time.Second}
		return &http.Client{Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, _, _ := net.SplitHostPort(addr)
				if err := util.CheckActivityPubRateLimit(ctx, host); err != nil {
					return nil, err
				}
				if target, ok := f.aliases.Load(strings.ToLower(host)); ok {
					addr = target.(string)
				}
				return dialer.DialContext(ctx, network, addr)
			},
		}}
	}
	t.Cleanup(func() {
		util.ActivityPubRateLimiter = origLimiter
		newHTTPClient = origClient
	})

	bobKey, bobPub, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(bobPub)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	bobPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	f.bob = bobKey
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.fetch.Add(1)
		if r.URL.Path != "/actor/bob" {
			http.NotFound(w, r)
			return
		}
		id := f.srv.URL + "/actor/bob"
		w.Header().Set("Content-Type", "application/activity+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@context":          "https://www.w3.org/ns/activitystreams",
			"type":              "Person",
			"id":                id,
			"preferredUsername": "bob",
			"inbox":             id + "/inbox",
			"outbox":            id + "/outbox",
			"publicKey": map[string]any{
				"id":           id + "#main-key",
				"owner":        id,
				"publicKeyPem": bobPEM,
			},
		})
	}))
	t.Cleanup(f.srv.Close)

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)
	addFeedCollection(t, app)
	f.app = app

	localInst := createTestActor(t, app, prefetchLocalInstanceIRI, "instance", true)
	peerInst := createTestActor(t, app, f.srv.URL+"/api/v1/activitypub/instance", "instance", false)
	follows, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows: %v", err)
	}
	fr := core.NewRecord(follows)
	fr.Set("follower", localInst.Id)
	fr.Set("followee", peerInst.Id)
	fr.Set("status", "accepted")
	if err := app.Save(fr); err != nil {
		t.Fatalf("save accepted follow: %v", err)
	}
	return f
}

func (f *prefetchBudgetFixture) unsignedEvent(body []byte) *core.RequestEvent {
	e := newInstanceInboxEvent("s3cret", instanceInboxPath, string(body))
	e.App = f.app
	return e
}

func TestInstanceInboxPrefetchBudgetNotExhaustedByStrangers(t *testing.T) {
	flood := func(t *testing.T, f *prefetchBudgetFixture, bodyFor func(i int) []byte) {
		t.Helper()
		for i := 1; i <= 30; i++ {
			_ = InstanceInboxHandler(f.unsignedEvent(bodyFor(i)))
		}
	}

	assertBobDelivered := func(t *testing.T, f *prefetchBudgetFixture) {
		t.Helper()
		bobIRI := f.srv.URL + "/actor/bob"
		trailIRI := f.srv.URL + "/api/v1/trail/bob1"
		body := marshalActivity(t, signedGateCreateTrail(bobIRI, trailIRI))
		e := newSignedInstanceInboxEvent(t, f.app, f.bob, bobIRI+"#main-key", body)
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 200)
		if _, err := f.app.FindFirstRecordByData("trails", "iri", trailIRI); err != nil {
			t.Fatalf("trail from the uncached peer user must be stored: %v", err)
		}
	}

	t.Run("lifecycle flood", func(t *testing.T) {
		f := newPrefetchBudgetFixture(t)
		flood(t, f, func(i int) []byte {
			actor := fmt.Sprintf("%s/x%d", f.srv.URL, i)
			follow := pub.FollowNew(pub.IRI(actor+"/activity/f"), pub.IRI(prefetchLocalInstanceIRI))
			follow.Actor = pub.IRI(actor)
			return marshalActivity(t, follow)
		})
		assertBobDelivered(t, f)
	})

	t.Run("content flood", func(t *testing.T) {
		f := newPrefetchBudgetFixture(t)
		flood(t, f, func(i int) []byte {
			actor := fmt.Sprintf("%s/y%d", f.srv.URL, i)
			return marshalActivity(t, signedGateCreateTrail(actor, fmt.Sprintf("%s/api/v1/trail/y%d", f.srv.URL, i)))
		})
		assertBobDelivered(t, f)
	})
}
