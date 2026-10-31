package routes

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const disconnectGateInboxPath = "/api/v1/activitypub/instance/inbox"

// disconnectGateSignedEvent builds a request signed like PostActivity, as
// forwarded by the SvelteKit proxy. VerifySignature mutates the request, so a
// fresh one is built per call.
func disconnectGateSignedEvent(t *testing.T, app core.App, priv *rsa.PrivateKey, keyID string, body []byte) *core.RequestEvent {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://local.example.com"+disconnectGateInboxPath, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Add("Content-Type", "application/activity+json")
	req.Header.Add("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
	req.Header.Add("Host", "local.example.com")

	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		[]string{"(request-target)", "Date", "Digest", "Content-Type", "Host"},
		httpsig.Signature, 60)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	if err := signer.SignRequest(priv, keyID, req, body); err != nil {
		t.Fatalf("sign request: %v", err)
	}

	req.Header.Set("X-Internal-Secret", "s3cret")
	req.Header.Set("X-Forwarded-Path", disconnectGateInboxPath)

	e := &core.RequestEvent{}
	e.App = app
	e.Request = req
	e.Response = httptest.NewRecorder()
	return e
}

func disconnectGateCreateBody(t *testing.T, actorIRI, baseURL string) []byte {
	t.Helper()
	obj := &pub.Object{
		ID:      pub.IRI(baseURL + "/api/v1/trail/t1"),
		Type:    pub.NoteType,
		Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Trail")),
		Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "")),
	}
	act := pub.ActivityNew(pub.IRI(baseURL+"/activities/c1"), pub.CreateType, obj)
	act.Actor = pub.IRI(actorIRI)
	body, err := jsonld.WithContext(
		jsonld.IRI(pub.ActivityBaseURI),
		jsonld.IRI(pub.SecurityContextURI),
	).Marshal(act)
	if err != nil {
		t.Fatalf("marshal activity: %v", err)
	}
	return body
}

// TestDisconnectedPeerContentRefusedAtInstanceInbox checks that a signed Create
// from a peer's user passes the instance inbox while connected and gets 403
// after disconnectPeer.
func TestDisconnectedPeerContentRefusedAtInstanceInbox(t *testing.T) {
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	_, out, in := seedDisconnectMutualPeer(t, app, local, inbox.server.URL)

	// The peer's user actor shares the peer instance's host and port.
	bobIRI := inbox.server.URL + "/users/bob"
	bob := createFedAdminTestActor(t, app, bobIRI, "person", false)
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
		t.Fatalf("save bob key: %v", err)
	}

	body := disconnectGateCreateBody(t, bobIRI, inbox.server.URL)
	send := func() error {
		e := disconnectGateSignedEvent(t, app, priv, bobIRI+"#main-key", body)
		return federation.InstanceInboxHandler(e)
	}

	// Wait for the Undo and Reject deliveries before the app is torn down.
	t.Cleanup(func() { inbox.waitForDeliveries(2, 5*time.Second) })

	t.Run("mutual peer passes the gate", func(t *testing.T) {
		ok, err := federation.IsAcceptedPeerHost(app, bobIRI)
		if err != nil || !ok {
			t.Fatalf("IsAcceptedPeerHost = %v, %v; want true", ok, err)
		}
		err = send()
		if err == nil {
			return
		}
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatalf("unexpected non-API error: %v", err)
		}
		// The fixture has no trails collection, so a 400 from dispatch is fine; only 401
		// or 403 means the request was refused earlier.
		if apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden {
			t.Fatalf("request refused before dispatch: status %d (%v)", apiErr.Status, err)
		}
	})

	t.Run("disconnected peer is refused", func(t *testing.T) {
		if err := disconnectPeer(app, local.Id, out); err != nil {
			t.Fatalf("disconnectPeer: %v", err)
		}
		assertFollowStatus(t, app, "outbound", out.Id, "deleted")
		assertFollowStatus(t, app, "inbound", in.Id, "rejected")

		err := send()
		if err == nil {
			t.Fatal("expected 403 after disconnect, got nil error")
		}
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *router.ApiError, got %T: %v", err, err)
		}
		if apiErr.Status != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (%v)", apiErr.Status, err)
		}
		if !strings.Contains(apiErr.Message, "Not an accepted peer") {
			t.Fatalf("message = %q, want it to contain \"Not an accepted peer\"", apiErr.Message)
		}
	})
}
