package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

type deliveryPolicyTransport func(*http.Request) (*http.Response, error)

func (f deliveryPolicyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type deliveryErrorBody struct {
	read   int64
	closed bool
}

func (b *deliveryErrorBody) Read(p []byte) (int, error) {
	if b.read >= 64<<20 {
		return 0, io.EOF
	}
	for i := range p {
		p[i] = 'x'
	}
	b.read += int64(len(p))
	return len(p), nil
}

func (b *deliveryErrorBody) Close() error { b.closed = true; return nil }

func deliverySigningActor(t *testing.T) (core.App, *core.Record, *rsa.PrivateKey) {
	t.Helper()
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	app := setupActorDeleteTestApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(&core.TextField{Name: "private_key"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(key), summitLogTestKey)
	if err != nil {
		t.Fatal(err)
	}
	actor := newTestActor(t, app, "https://local.example/api/v1/activitypub/user/alice", true)
	actor.Set("private_key", encrypted)
	return app, actor, key
}

func waitPolicyDelivery(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestPostActivitySignsPublicDeliveryAndBoundsErrorBody(t *testing.T) {
	app, actor, key := deliverySigningActor(t)
	body := &deliveryErrorBody{}
	var requests atomic.Int32
	copyClient := *httpClient.Load()
	copyClient.Transport = deliveryPolicyTransport(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		data, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
		}
		verifier, err := httpsig.NewVerifier(req)
		if err != nil {
			t.Error(err)
		} else if err := verifier.Verify(&key.PublicKey, httpsig.RSA_SHA256); err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(data), "Like") || req.Header.Get("Content-Type") != "application/activity+json" {
			t.Errorf("invalid activity delivery: %s", data)
		}
		return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: body}, nil
	})
	t.Cleanup(SetDeliveryClientForTesting(&copyClient))
	activity := pub.ActivityNew(pub.IRI("https://local.example/activity"), pub.LikeType, pub.IRI("https://remote.example/trail"))
	if err := PostActivity(app, actor, activity, []string{"https://remote.example/inbox", "ftp://remote.example/inbox", "https://user:pass@remote.example/inbox", "http://:443/inbox", "https://remote.example/inbox"}); err != nil {
		t.Fatal(err)
	}
	waitPolicyDelivery(t)
	if requests.Load() != 1 {
		t.Fatalf("sent %d requests, expected one valid deduplicated inbox", requests.Load())
	}
	if body.read != deliveryErrorBodyLimit || !body.closed {
		t.Fatalf("error body read=%d closed=%v, want %d and closed", body.read, body.closed, deliveryErrorBodyLimit)
	}
}

func TestPostActivityDoesNotForwardSignedRedirect(t *testing.T) {
	app, actor, _ := deliverySigningActor(t)
	copyClient := *httpClient.Load()
	var requests atomic.Int32
	copyClient.Transport = deliveryPolicyTransport(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {"https://another.example/inbox"}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: req}, nil
	})
	t.Cleanup(SetDeliveryClientForTesting(&copyClient))
	if err := PostActivity(app, actor, pub.ActivityNew(pub.IRI("https://local.example/activity"), pub.LikeType, nil), []string{"https://remote.example/inbox"}); err != nil {
		t.Fatal(err)
	}
	waitPolicyDelivery(t)
	if requests.Load() != 1 {
		t.Fatalf("signed request followed redirect (%d requests)", requests.Load())
	}
}
