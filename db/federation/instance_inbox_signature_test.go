package federation

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
)

// signedInstanceInboxEventAt signs body at the given time and sends sentBody, so
// the same helper builds fresh, stale and tampered deliveries.
func signedInstanceInboxEventAt(t *testing.T, app core.App, priv *rsa.PrivateKey, keyID string, signedBody, sentBody []byte, date time.Time) *core.RequestEvent {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://trails.example.com"+instanceInboxPath, bytes.NewReader(sentBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Date", date.UTC().Format(http.TimeFormat))
	req.Header.Set("Host", "trails.example.com")

	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		[]string{"(request-target)", "Date", "Digest", "Content-Type", "Host"},
		httpsig.Signature, 60)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	if err := signer.SignRequest(priv, keyID, req, signedBody); err != nil {
		t.Fatalf("sign request: %v", err)
	}
	req.Header.Set("X-Internal-Secret", "s3cret")
	req.Header.Set("X-Forwarded-Path", instanceInboxPath)

	e := &core.RequestEvent{}
	e.App = app
	e.Request = req
	e.Response = httptest.NewRecorder()
	return e
}

// TestInstanceInboxRefusesTamperedOrStaleSignature checks that a valid
// signature cannot be reused with another body or outside the time window.
func TestInstanceInboxRefusesTamperedOrStaleSignature(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	const (
		localInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"
		peerInstanceIRI  = "https://peer.example.com/api/v1/activitypub/instance"
		peerUserIRI      = "https://peer.example.com/api/v1/activitypub/user/bob"
	)

	setup := func(t *testing.T) (core.App, *rsa.PrivateKey) {
		t.Helper()
		app := newInboxTestApp(t)
		addTrailsCollection(t, app)
		addFeedCollection(t, app)

		localInst := createTestActor(t, app, localInstanceIRI, "instance", true)
		peerInst := createTestActor(t, app, peerInstanceIRI, "instance", false)
		peerUser := createTestActor(t, app, peerUserIRI, "person", false)

		priv, pubKey, err := util.GenerateRSAKeyPair()
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		der, err := x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			t.Fatalf("marshal public key: %v", err)
		}
		peerUser.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
		if err := app.Save(peerUser); err != nil {
			t.Fatalf("save peer user key: %v", err)
		}

		follows, err := app.FindCollectionByNameOrId("follows")
		if err != nil {
			t.Fatalf("find follows: %v", err)
		}
		fr := core.NewRecord(follows)
		fr.Set("follower", peerInst.Id)
		fr.Set("followee", localInst.Id)
		fr.Set("status", "accepted")
		if err := app.Save(fr); err != nil {
			t.Fatalf("save accepted follow: %v", err)
		}
		return app, priv
	}

	trailExists := func(app core.App, iri string) bool {
		_, err := app.FindFirstRecordByData("trails", "iri", iri)
		return err == nil
	}

	t.Run("tampered body", func(t *testing.T) {
		app, priv := setup(t)
		const signedIRI = "https://peer.example.com/api/v1/trail/signed"
		const swappedIRI = "https://peer.example.com/api/v1/trail/swapped"
		signed := marshalActivity(t, signedGateCreateTrail(peerUserIRI, signedIRI))
		swapped := marshalActivity(t, signedGateCreateTrail(peerUserIRI, swappedIRI))
		e := signedInstanceInboxEventAt(t, app, priv, peerUserIRI+"#main-key", signed, swapped, time.Now())
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
		if trailExists(app, swappedIRI) || trailExists(app, signedIRI) {
			t.Fatal("a tampered delivery must not store anything")
		}
	})

	t.Run("stale date", func(t *testing.T) {
		app, priv := setup(t)
		const trailIRI = "https://peer.example.com/api/v1/trail/stale"
		body := marshalActivity(t, signedGateCreateTrail(peerUserIRI, trailIRI))
		e := signedInstanceInboxEventAt(t, app, priv, peerUserIRI+"#main-key", body, body, time.Now().Add(-2*time.Hour))
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
		if trailExists(app, trailIRI) {
			t.Fatal("a stale delivery must not store anything")
		}
	})

	t.Run("fresh request passes", func(t *testing.T) {
		app, priv := setup(t)
		const trailIRI = "https://peer.example.com/api/v1/trail/fresh"
		body := marshalActivity(t, signedGateCreateTrail(peerUserIRI, trailIRI))
		e := signedInstanceInboxEventAt(t, app, priv, peerUserIRI+"#main-key", body, body, time.Now())
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 200)
		if !trailExists(app, trailIRI) {
			t.Fatal("a fresh, untampered delivery must be stored")
		}
	})
}
