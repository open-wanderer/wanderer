package routes

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pocketbase/util"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// TestUserInboxVerifiesDigest checks that a Mastodon-style signed delivery
// verifies at the user inbox and that the same headers with another body get
// 401.
func TestUserInboxVerifiesDigest(t *testing.T) {
	app := newFederationAdminTestApp(t)
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	const (
		inboxPath = "/api/v1/activitypub/user/alice/inbox"
		senderIRI = "https://remote.example.com/users/bob"
	)

	alice := createFedAdminTestActor(t, app, "https://local.example.com/api/v1/activitypub/user/alice", "person", true)
	alice.Set("inbox", "https://local.example.com"+inboxPath)
	if err := app.Save(alice); err != nil {
		t.Fatalf("save recipient: %v", err)
	}

	bob := createFedAdminTestActor(t, app, senderIRI, "person", false)
	priv, pubKey, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	bob.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
	if err := app.Save(bob); err != nil {
		t.Fatalf("save sender key: %v", err)
	}

	activity := func(id string) []byte {
		return []byte(`{"@context":"https://www.w3.org/ns/activitystreams","id":"` + senderIRI + `/activities/` + id +
			`","type":"Create","actor":"` + senderIRI + `","object":{"id":"` + senderIRI + `/notes/` + id +
			`","type":"Note","content":"hello"}}`)
	}

	// Mastodon signs (request-target) host date digest content-type.
	newEvent := func(signedBody, sentBody []byte) *core.RequestEvent {
		req, err := http.NewRequest(http.MethodPost, "https://local.example.com"+inboxPath, bytes.NewReader(sentBody))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/activity+json")
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		req.Header.Set("Host", "local.example.com")
		signer, _, err := httpsig.NewSigner(
			[]httpsig.Algorithm{httpsig.RSA_SHA256},
			httpsig.DigestSha256,
			[]string{"(request-target)", "host", "date", "digest", "content-type"},
			httpsig.Signature, 60)
		if err != nil {
			t.Fatalf("new signer: %v", err)
		}
		var key *rsa.PrivateKey = priv
		if err := signer.SignRequest(key, senderIRI+"#main-key", req, signedBody); err != nil {
			t.Fatalf("sign request: %v", err)
		}
		req.Header.Set("X-Internal-Secret", "s3cret")
		req.Header.Set("X-Forwarded-Path", inboxPath)

		e := &core.RequestEvent{}
		e.App = app
		e.Request = req
		e.Response = httptest.NewRecorder()
		return e
	}

	t.Run("mastodon-style signature passes verification", func(t *testing.T) {
		body := activity("one")
		err := ActivitypubActivityProcess(newEvent(body, body))
		var apiErr *router.ApiError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized {
			t.Fatalf("a correctly signed delivery was refused: %v", err)
		}
		// 200 means verification passed and dispatch ran.
		if err != nil {
			t.Fatalf("expected the request to reach dispatch, got %v", err)
		}
	})

	t.Run("tampered body is refused", func(t *testing.T) {
		err := ActivitypubActivityProcess(newEvent(activity("signed"), activity("swapped")))
		assertUnauthorized(t, err)
	})
}
