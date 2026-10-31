package routes

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// TestInstanceInboxReloadsRetainedKey covers an upgraded install: the remote
// actor cache refresh moved a peer user's key to legacy_public_key. The
// instance inbox must reload the key from the actor's origin, as the user
// inboxes do, instead of verifying against the emptied public_key.
func TestInstanceInboxReloadsRetainedKey(t *testing.T) {
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	seedDisconnectMutualPeer(t, app, local, inbox.server.URL)

	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	if actors.Fields.GetByName("legacy_public_key") == nil {
		actors.Fields.Add(&core.TextField{Name: "legacy_public_key", Hidden: true})
		if err := app.Save(actors); err != nil {
			t.Fatal(err)
		}
	}

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
	key := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	parsed, err := url.Parse(bobIRI)
	if err != nil {
		t.Fatal(err)
	}
	// The state the cache refresh migration leaves behind.
	bob.Set("public_key", "")
	bob.Set("legacy_public_key", key)
	bob.Set("domain", parsed.Hostname())
	if err := app.Save(bob); err != nil {
		t.Fatalf("save bob: %v", err)
	}

	// The origin serves the actor with its current key.
	fetches := 0
	previous := getInboxActorByIRI
	getInboxActorByIRI = func(app core.App, ctx context.Context, iri string, follows bool) (*core.Record, error) {
		fetches++
		stored, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
		if err != nil {
			return nil, err
		}
		stored.Set("public_key", key)
		stored.Set("domain", parsed.Host)
		return stored, app.Save(stored)
	}
	t.Cleanup(func() { getInboxActorByIRI = previous })

	body := disconnectGateCreateBody(t, bobIRI, inbox.server.URL)
	err = instanceInboxHandler(disconnectGateSignedEvent(t, app, priv, bobIRI+"#main-key", body))
	if err != nil {
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatalf("unexpected non-API error: %v", err)
		}
		// The fixture has no trails collection, so a 400 from dispatch is fine; only 401
		// or 403 means the request was refused earlier.
		if apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden {
			t.Fatalf("request refused before dispatch: status %d (%v)", apiErr.Status, err)
		}
	}
	if fetches != 1 {
		t.Fatalf("actor fetches = %d, want 1 key reload", fetches)
	}
}
