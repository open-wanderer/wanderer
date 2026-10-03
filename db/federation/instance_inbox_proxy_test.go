package federation

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"strings"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// newInstanceInboxEvent builds a minimal RequestEvent for the early reject
// paths of InstanceInboxHandler, which return before any App access.
func newInstanceInboxEvent(secretHeader, forwardedPath, body string) *core.RequestEvent {
	req := httptest.NewRequest("POST", "/activitypub/instance/inbox", strings.NewReader(body))
	if secretHeader != "" {
		req.Header.Set("X-Internal-Secret", secretHeader)
	}
	if forwardedPath != "" {
		req.Header.Set("X-Forwarded-Path", forwardedPath)
	}
	e := &core.RequestEvent{}
	e.Request = req
	e.Response = httptest.NewRecorder()
	return e
}

func assertInstanceInboxStatus(t *testing.T, err error, want int) {
	t.Helper()
	if want == 200 {
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected status %d, got nil error", want)
	}
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *router.ApiError, got %T: %v", err, err)
	}
	if apiErr.Status != want {
		t.Fatalf("expected status %d, got %d (%v)", want, apiErr.Status, err)
	}
}

func TestInstanceInboxRejectsUntrustedHop(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")

	t.Run("secret not configured", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "")
		e := newInstanceInboxEvent("anything", instanceInboxPath, "{}")
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
	})

	t.Run("missing secret header", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newInstanceInboxEvent("", instanceInboxPath, "{}")
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
	})

	t.Run("wrong secret header", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newInstanceInboxEvent("nope", instanceInboxPath, "{}")
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
	})

	t.Run("user inbox path with valid secret", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newInstanceInboxEvent("s3cret", "/api/v1/activitypub/user/alice/inbox", "{}")
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
	})

	t.Run("trailing slash path with valid secret", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newInstanceInboxEvent("s3cret", instanceInboxPath+"/", "{}")
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
	})

	t.Run("missing forwarded path with valid secret", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newInstanceInboxEvent("s3cret", "", "{}")
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
	})

	t.Run("activity without actor", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newInstanceInboxEvent("s3cret", instanceInboxPath, `{"type":"Create"}`)
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 400)
	})
}

// newSignedInstanceInboxEvent builds a request signed like PostActivity, as
// forwarded by the SvelteKit proxy. VerifySignature mutates the request, so a
// fresh one is built per call.
func newSignedInstanceInboxEvent(t *testing.T, app core.App, priv *rsa.PrivateKey, keyID string, body []byte) *core.RequestEvent {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://trails.example.com"+instanceInboxPath, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Add("Content-Type", "application/activity+json")
	req.Header.Add("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
	req.Header.Add("Host", "trails.example.com")

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
	req.Header.Set("X-Forwarded-Path", instanceInboxPath)

	e := &core.RequestEvent{}
	e.App = app
	e.Request = req
	e.Response = httptest.NewRecorder()
	return e
}

func marshalActivity(t *testing.T, act *pub.Activity) []byte {
	t.Helper()
	body, err := jsonld.WithContext(
		jsonld.IRI(pub.ActivityBaseURI),
		jsonld.IRI(pub.SecurityContextURI),
	).Marshal(act)
	if err != nil {
		t.Fatalf("marshal activity: %v", err)
	}
	return body
}

func signedGateCreateTrail(actorIRI, trailIRI string) *pub.Activity {
	place := &pub.Place{
		Type:      pub.PlaceType,
		Latitude:  47.5,
		Longitude: 11.5,
		Name:      pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Place")),
	}
	obj := &pub.Object{
		ID:       pub.IRI(trailIRI),
		Type:     pub.NoteType,
		Name:     pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Trail")),
		Content:  pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "")),
		Location: place,
	}
	act := pub.ActivityNew(pub.IRI(actorIRI+"/activity/c1"), pub.CreateType, obj)
	act.Actor = pub.IRI(actorIRI)
	return act
}

func TestInstanceInboxSignedGate(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	const (
		localInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"
		localUserIRI     = "https://trails.example.com/api/v1/activitypub/user/alice"
		peerInstanceIRI  = "https://peer.example.com/api/v1/activitypub/instance"
		peerUserIRI      = "https://peer.example.com/api/v1/activitypub/user/bob"
		strangerIRI      = "https://stranger.example.com/api/v1/activitypub/user/eve"
		strangerInstIRI  = "https://stranger.example.com/api/v1/activitypub/instance"
	)

	type fixture struct {
		app       core.App
		localInst *core.Record
		localUser *core.Record
		peerUser  *core.Record
		stranger  *core.Record
		strangerI *core.Record
		keys      map[string]*rsa.PrivateKey
	}

	setup := func(t *testing.T) *fixture {
		t.Helper()
		app := newInboxTestApp(t)
		addTrailsCollection(t, app)
		addFeedCollection(t, app)

		f := &fixture{app: app, keys: map[string]*rsa.PrivateKey{}}
		f.localInst = createTestActor(t, app, localInstanceIRI, "instance", true)
		f.localUser = createTestActor(t, app, localUserIRI, "person", true)
		peerInst := createTestActor(t, app, peerInstanceIRI, "instance", false)
		f.peerUser = createTestActor(t, app, peerUserIRI, "person", false)
		f.stranger = createTestActor(t, app, strangerIRI, "person", false)
		f.strangerI = createTestActor(t, app, strangerInstIRI, "instance", false)

		for _, r := range []*core.Record{peerInst, f.peerUser, f.stranger, f.strangerI} {
			priv, pubKey, err := util.GenerateRSAKeyPair()
			if err != nil {
				t.Fatalf("generate key: %v", err)
			}
			der, err := x509.MarshalPKIXPublicKey(pubKey)
			if err != nil {
				t.Fatalf("marshal public key: %v", err)
			}
			r.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
			if err := app.Save(r); err != nil {
				t.Fatalf("save actor key: %v", err)
			}
			f.keys[r.GetString("iri")] = priv
		}

		// Accepted peer relationship: peer instance -> local instance.
		follows, err := app.FindCollectionByNameOrId("follows")
		if err != nil {
			t.Fatalf("find follows: %v", err)
		}
		fr := core.NewRecord(follows)
		fr.Set("follower", peerInst.Id)
		fr.Set("followee", f.localInst.Id)
		fr.Set("status", "accepted")
		if err := app.Save(fr); err != nil {
			t.Fatalf("save accepted follow: %v", err)
		}
		return f
	}

	trailExists := func(f *fixture, iri string) bool {
		_, err := f.app.FindFirstRecordByData("trails", "iri", iri)
		return err == nil
	}
	followCount := func(t *testing.T, f *fixture, follower, followee string) int {
		t.Helper()
		rows, err := f.app.FindRecordsByFilter("follows",
			"follower={:a} && followee={:b}", "", 10, 0,
			map[string]any{"a": follower, "b": followee})
		if err != nil {
			t.Fatalf("query follows: %v", err)
		}
		return len(rows)
	}

	t.Run("stranger content is refused", func(t *testing.T) {
		f := setup(t)
		const trailIRI = "https://stranger.example.com/api/v1/trail/t1"
		body := marshalActivity(t, signedGateCreateTrail(strangerIRI, trailIRI))
		e := newSignedInstanceInboxEvent(t, f.app, f.keys[strangerIRI], strangerIRI+"#main-key", body)
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 403)
		if trailExists(f, trailIRI) {
			t.Fatal("trail from a non-peer must not be stored")
		}
	})

	t.Run("peer host user content is processed", func(t *testing.T) {
		f := setup(t)
		const trailIRI = "https://peer.example.com/api/v1/trail/t1"
		body := marshalActivity(t, signedGateCreateTrail(peerUserIRI, trailIRI))
		e := newSignedInstanceInboxEvent(t, f.app, f.keys[peerUserIRI], peerUserIRI+"#main-key", body)
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 200)
		if !trailExists(f, trailIRI) {
			t.Fatal("trail from a peer-host user must be stored")
		}
	})

	t.Run("stranger Follow of the instance actor is accepted as pending", func(t *testing.T) {
		f := setup(t)
		follow := pub.FollowNew(pub.IRI(strangerInstIRI+"/activity/f1"), pub.IRI(localInstanceIRI))
		follow.Actor = pub.IRI(strangerInstIRI)
		e := newSignedInstanceInboxEvent(t, f.app, f.keys[strangerInstIRI], strangerInstIRI+"#main-key", marshalActivity(t, follow))
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 200)
		rows, err := f.app.FindRecordsByFilter("follows",
			"follower={:a} && followee={:b}", "", 10, 0,
			map[string]any{"a": f.strangerI.Id, "b": f.localInst.Id})
		if err != nil || len(rows) != 1 {
			t.Fatalf("expected one follows row, got %d (err %v)", len(rows), err)
		}
		if got := rows[0].GetString("status"); got != "pending" {
			t.Fatalf("status = %q, want pending", got)
		}
	})

	t.Run("person Follow of the instance actor is refused", func(t *testing.T) {
		f := setup(t)
		follow := pub.FollowNew(pub.IRI(strangerIRI+"/activity/f3"), pub.IRI(localInstanceIRI))
		follow.Actor = pub.IRI(strangerIRI)
		e := newSignedInstanceInboxEvent(t, f.app, f.keys[strangerIRI], strangerIRI+"#main-key", marshalActivity(t, follow))
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 400)
		if n := followCount(t, f, f.stranger.Id, f.localInst.Id); n != 0 {
			t.Fatalf("expected no follows row, got %d", n)
		}
	})

	t.Run("stranger Follow of a user via the instance inbox is refused", func(t *testing.T) {
		f := setup(t)
		follow := pub.FollowNew(pub.IRI(strangerIRI+"/activity/f2"), pub.IRI(localUserIRI))
		follow.Actor = pub.IRI(strangerIRI)
		e := newSignedInstanceInboxEvent(t, f.app, f.keys[strangerIRI], strangerIRI+"#main-key", marshalActivity(t, follow))
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 403)
		if n := followCount(t, f, f.stranger.Id, f.localUser.Id); n != 0 {
			t.Fatalf("expected no follows row, got %d", n)
		}
	})

	t.Run("signature from a different key is rejected", func(t *testing.T) {
		f := setup(t)
		const trailIRI = "https://peer.example.com/api/v1/trail/t2"
		body := marshalActivity(t, signedGateCreateTrail(peerUserIRI, trailIRI))
		// Sign with the stranger's key while claiming to be the peer user.
		e := newSignedInstanceInboxEvent(t, f.app, f.keys[strangerIRI], peerUserIRI+"#main-key", body)
		assertInstanceInboxStatus(t, InstanceInboxHandler(e), 401)
		if trailExists(f, trailIRI) {
			t.Fatal("trail with a bad signature must not be stored")
		}
	})
}
