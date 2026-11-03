package routes

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/pocketbase/pocketbase/core"
)

// TestPeerUserDeleteAcceptedAtInstanceInbox covers the receiving end of the
// notice an account deletion sends to every accepted peer: a Delete naming
// a user, signed by that user, arrives at the instance inbox of an instance
// that has a peer relationship with the sender's host, and the local copy of
// the user is removed.
func TestPeerUserDeleteAcceptedAtInstanceInbox(t *testing.T) {
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	seedDisconnectMutualPeer(t, app, local, inbox.server.URL)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = federation.WaitForDeliveries(ctx)
	})

	// util.DeleteRemoteActor reads these before it removes the actor.
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"trails", "lists", "comments", "summit_logs", "waypoints"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.TextField{Name: "iri"},
			&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		)
		if err := app.Save(c); err != nil {
			t.Fatalf("create %s collection: %v", name, err)
		}
	}

	// The peer's user shares the peer instance's host and port. Only the
	// .../activitypub/user/<name> path classifies as an actor IRI.
	bobIRI := inbox.server.URL + "/api/v1/activitypub/user/bob"
	bob := createFedAdminTestActor(t, app, bobIRI, "person", false)
	priv, pubKey, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	bob.Set("domain", mustURLHost(t, bobIRI))
	bob.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
	if err := app.Save(bob); err != nil {
		t.Fatalf("save bob key: %v", err)
	}

	activity := pub.DeleteNew(pub.IRI(inbox.server.URL+"/api/v1/activitypub/activity/d1"), pub.IRI(bobIRI))
	activity.Actor = pub.IRI(bobIRI)
	activity.To = pub.ItemCollection{pub.IRI(pub.PublicNS)}
	body, err := jsonld.WithContext(
		jsonld.IRI(pub.ActivityBaseURI),
		jsonld.IRI(pub.SecurityContextURI),
	).Marshal(activity)
	if err != nil {
		t.Fatalf("marshal activity: %v", err)
	}

	e := disconnectGateSignedEvent(t, app, priv, bobIRI+"#main-key", body)
	if err := instanceInboxHandler(e); err != nil {
		t.Fatalf("expected the instance inbox to accept the peer user's Delete, got %v", err)
	}

	_, err = app.FindRecordById("activitypub_actors", bob.Id)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected the local copy of the peer user to be removed, lookup returned %v", err)
	}
}
