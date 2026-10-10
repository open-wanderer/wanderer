package federation

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"pocketbase/util"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

const (
	likeTestKey = "0123456789abcdef0123456789abcdef"

	likeSenderOrigin   = "https://peer.example.com"
	likeReceiverOrigin = "https://trails.example.com"

	likeSenderInstanceIRI   = likeSenderOrigin + "/api/v1/activitypub/instance"
	likeReceiverInstanceIRI = likeReceiverOrigin + "/api/v1/activitypub/instance"

	likeBobIRI   = likeSenderOrigin + "/api/v1/activitypub/user/bob"
	likeCarolIRI = "https://third.example.com/api/v1/activitypub/user/carol"
	likeTrailIRI = "https://third.example.com/api/v1/trail/t1"
)

// newLikeTestApp is the inbox test app plus trails and a trail_like collection
// with the production unique index on (trail, actor).
func newLikeTestApp(t *testing.T) core.App {
	t.Helper()
	app := newInboxTestApp(t)
	addTrailsCollection(t, app)

	col := core.NewBaseCollection("trail_like")
	col.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "actor"})
	col.AddIndex("idx_trail_like_unique", true, "trail,actor", "")
	if err := app.Save(col); err != nil {
		t.Fatalf("create trail_like collection: %v", err)
	}
	return app
}

// setActorPublicKey stores pubKey as PEM on the actor.
func setActorPublicKey(t *testing.T, app core.App, actor *core.Record, pubKey *rsa.PublicKey) {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	actor.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
	if err := app.Save(actor); err != nil {
		t.Fatalf("save public key: %v", err)
	}
}

// createSigningActor creates a local person that can sign outgoing activities.
func createSigningActor(t *testing.T, app core.App, iri string) (*core.Record, *rsa.PrivateKey) {
	t.Helper()
	actor := createTestActor(t, app, iri, "person", true)
	priv, pubKey, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	enc, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(priv), likeTestKey)
	if err != nil {
		t.Fatalf("encrypt private key: %v", err)
	}
	actor.Set("private_key", enc)
	setActorPublicKey(t, app, actor, pubKey)
	return actor, priv
}

// setActorInbox points the actor's inbox at url.
func setActorInbox(t *testing.T, app core.App, actor *core.Record, url string) {
	t.Helper()
	actor.Set("inbox", url)
	if err := app.Save(actor); err != nil {
		t.Fatalf("save inbox %s: %v", url, err)
	}
}

func saveTrail(t *testing.T, app core.App, iri string, author *core.Record, public bool) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("find trails: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("author", author.Id)
	r.Set("name", "Test Trail")
	r.Set("public", public)
	if err := app.Save(r); err != nil {
		t.Fatalf("save trail: %v", err)
	}
	return r
}

func saveTrailLike(t *testing.T, app core.App, trail, actor *core.Record) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("trail_like")
	if err != nil {
		t.Fatalf("find trail_like: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("trail", trail.Id)
	r.Set("actor", actor.Id)
	if err := app.Save(r); err != nil {
		t.Fatalf("save trail_like: %v", err)
	}
	return r
}

func countTrailLikes(t *testing.T, app core.App, trail, actor *core.Record) int {
	t.Helper()
	rows, err := app.FindAllRecords("trail_like")
	if err != nil {
		t.Fatalf("list trail_like: %v", err)
	}
	n := 0
	for _, r := range rows {
		if r.GetString("trail") == trail.Id && r.GetString("actor") == actor.Id {
			n++
		}
	}
	return n
}

// likeSender is instance A: local liker bob, a remote trail author carol and a
// copy of carol's trail, with one accepted peer instance.
type likeSender struct {
	app     core.App
	in      *inboxes
	bob     *core.Record
	bobPriv *rsa.PrivateKey
	trail   *core.Record
}

func newLikeSender(t *testing.T, in *inboxes, public bool) *likeSender {
	t.Helper()
	t.Setenv("ORIGIN", likeSenderOrigin)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", likeTestKey)
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	app := newLikeTestApp(t)

	localInst := createTestActor(t, app, likeSenderInstanceIRI, "instance", true)
	peerInst := createTestActor(t, app, likeReceiverInstanceIRI, "instance", false)
	setActorInbox(t, app, peerInst, in.url("trails-instance"))
	gateSeedFollow(t, app, localInst.Id, peerInst.Id, "accepted")

	carol := createTestActor(t, app, likeCarolIRI, "person", false)
	setActorInbox(t, app, carol, in.url("carol"))

	bob, bobPriv := createSigningActor(t, app, likeBobIRI)

	return &likeSender{
		app:     app,
		in:      in,
		bob:     bob,
		bobPriv: bobPriv,
		trail:   saveTrail(t, app, likeTrailIRI, carol, public),
	}
}

// likeReceiver is instance B: it holds the same trail copy and has accepted A
// as a peer.
type likeReceiver struct {
	app   core.App
	bob   *core.Record
	carol *core.Record
	trail *core.Record
}

// newLikeReceiver must be called after the sender's deliveries have arrived,
// because it switches ORIGIN.
func newLikeReceiver(t *testing.T, bobPub *rsa.PublicKey) *likeReceiver {
	t.Helper()
	t.Setenv("ORIGIN", likeReceiverOrigin)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", likeTestKey)
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	app := newLikeTestApp(t)

	localInst := createTestActor(t, app, likeReceiverInstanceIRI, "instance", true)
	peerInst := createTestActor(t, app, likeSenderInstanceIRI, "instance", false)
	gateSeedFollow(t, app, peerInst.Id, localInst.Id, "accepted")

	bob := createTestActor(t, app, likeBobIRI, "person", false)
	setActorPublicKey(t, app, bob, bobPub)

	carol := createTestActor(t, app, likeCarolIRI, "person", false)

	return &likeReceiver{
		app:   app,
		bob:   bob,
		carol: carol,
		trail: saveTrail(t, app, likeTrailIRI, carol, true),
	}
}

// deliver feeds a signed activity body into the receiver's instance inbox.
func (r *likeReceiver) deliver(t *testing.T, signer *rsa.PrivateKey, keyID string, body []byte, wantStatus int) {
	t.Helper()
	e := signedInstanceInboxEventAt(t, r.app, signer, keyID, body, body, time.Now())
	assertInstanceInboxStatus(t, InstanceInboxHandler(e), wantStatus)
}

func TestLikeReachesPeerInstanceRoundTrip(t *testing.T) {
	in := newInboxes(t)
	a := newLikeSender(t, in, true)

	like := saveTrailLike(t, a.app, a.trail, a.bob)
	if err := CreateLikeActivity(a.app, like); err != nil {
		t.Fatalf("CreateLikeActivity: %v", err)
	}

	if got := in.wait("carol", 1, 5*time.Second); got != 1 {
		t.Fatalf("trail author inbox deliveries = %d, want 1", got)
	}
	if got := in.wait("trails-instance", 1, 5*time.Second); got != 1 {
		t.Fatalf("peer instance inbox deliveries = %d, want 1", got)
	}
	body := in.body("trails-instance")

	b := newLikeReceiver(t, &a.bobPriv.PublicKey)
	b.deliver(t, a.bobPriv, likeBobIRI+"#main-key", body, 200)

	if got := countTrailLikes(t, b.app, b.trail, b.bob); got != 1 {
		t.Fatalf("trail_like rows on receiver = %d, want 1", got)
	}
}

func TestUnlikeReachesPeerInstanceRoundTrip(t *testing.T) {
	in := newInboxes(t)
	a := newLikeSender(t, in, true)

	like := saveTrailLike(t, a.app, a.trail, a.bob)
	if err := CreateLikeActivity(a.app, like); err != nil {
		t.Fatalf("CreateLikeActivity: %v", err)
	}
	if got := in.wait("trails-instance", 1, 5*time.Second); got != 1 {
		t.Fatalf("peer instance Like deliveries = %d, want 1", got)
	}
	likeBody := in.body("trails-instance")

	if err := CreateUnlikeActivity(a.app, like); err != nil {
		t.Fatalf("CreateUnlikeActivity: %v", err)
	}
	if got := in.wait("trails-instance", 2, 5*time.Second); got != 2 {
		t.Fatalf("peer instance deliveries = %d, want 2", got)
	}
	if got := in.wait("carol", 2, 5*time.Second); got != 2 {
		t.Fatalf("trail author deliveries = %d, want 2", got)
	}
	undoBody := in.body("trails-instance")

	b := newLikeReceiver(t, &a.bobPriv.PublicKey)
	keyID := likeBobIRI + "#main-key"

	b.deliver(t, a.bobPriv, keyID, likeBody, 200)
	if got := countTrailLikes(t, b.app, b.trail, b.bob); got != 1 {
		t.Fatalf("trail_like rows after Like = %d, want 1", got)
	}

	b.deliver(t, a.bobPriv, keyID, undoBody, 200)
	if got := countTrailLikes(t, b.app, b.trail, b.bob); got != 0 {
		t.Fatalf("trail_like rows after Undo = %d, want 0", got)
	}
}

func TestProcessUnlikeTolerance(t *testing.T) {
	priv, _, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	b := newLikeReceiver(t, &priv.PublicKey)

	undoOf := func(object pub.Item) pub.Activity {
		l := pub.LikeNew(pub.IRI(likeBobIRI+"/activity/l1"), object)
		l.Actor = pub.IRI(likeBobIRI)
		u := pub.UndoNew(pub.IRI(likeBobIRI+"/activity/u1"), l)
		u.Actor = pub.IRI(likeBobIRI)
		return *u
	}

	t.Run("duplicate delivery", func(t *testing.T) {
		saveTrailLike(t, b.app, b.trail, b.bob)
		undo := undoOf(pub.IRI(likeTrailIRI))
		if err := ProcessUndoActivity(b.app, b.bob, undo); err != nil {
			t.Fatalf("first Undo: %v", err)
		}
		if got := countTrailLikes(t, b.app, b.trail, b.bob); got != 0 {
			t.Fatalf("trail_like rows = %d, want 0", got)
		}
		if err := ProcessUndoActivity(b.app, b.bob, undo); err != nil {
			t.Fatalf("second Undo must succeed: %v", err)
		}
	})

	t.Run("trail not held", func(t *testing.T) {
		undo := undoOf(pub.IRI("https://third.example.com/api/v1/trail/unknown"))
		if err := ProcessUndoActivity(b.app, b.bob, undo); err != nil {
			t.Fatalf("Undo for a trail the receiver does not hold: %v", err)
		}
	})

	t.Run("only the signer's like is removed", func(t *testing.T) {
		other := createTestActor(t, b.app, "https://peer.example.com/api/v1/activitypub/user/zed", "person", false)
		saveTrailLike(t, b.app, b.trail, other)
		if err := ProcessUndoActivity(b.app, b.bob, undoOf(pub.IRI(likeTrailIRI))); err != nil {
			t.Fatalf("Undo: %v", err)
		}
		if got := countTrailLikes(t, b.app, b.trail, other); got != 1 {
			t.Fatalf("another actor's like was removed, rows = %d", got)
		}
	})

	t.Run("nil object", func(t *testing.T) {
		undo := undoOf(nil)
		if err := ProcessUndoActivity(b.app, b.bob, undo); err == nil {
			t.Fatal("expected an error for a Like without object")
		}
	})
}

func TestLikePrivateTrailStaysOffPeers(t *testing.T) {
	in := newInboxes(t)
	a := newLikeSender(t, in, false)

	like := saveTrailLike(t, a.app, a.trail, a.bob)
	if err := CreateLikeActivity(a.app, like); err != nil {
		t.Fatalf("CreateLikeActivity: %v", err)
	}
	if got := in.wait("carol", 1, 5*time.Second); got != 1 {
		t.Fatalf("trail author Like deliveries = %d, want 1", got)
	}

	if err := CreateUnlikeActivity(a.app, like); err != nil {
		t.Fatalf("CreateUnlikeActivity: %v", err)
	}
	if got := in.wait("carol", 2, 5*time.Second); got != 2 {
		t.Fatalf("trail author deliveries after Undo = %d, want 2", got)
	}

	if got := in.wait("trails-instance", 1, 500*time.Millisecond); got != 0 {
		t.Fatalf("a private trail's like reached a peer instance %d times", got)
	}
}

// addNotificationCollections adds the settings and notifications collections
// and a user field on the actors, which SendNotification needs.
func addNotificationCollections(t *testing.T, app core.App) {
	t.Helper()

	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatalf("find activitypub_actors: %v", err)
	}
	actors.Fields.Add(&core.TextField{Name: "user"})
	if err := app.Save(actors); err != nil {
		t.Fatalf("add user field: %v", err)
	}

	settings := core.NewBaseCollection("settings")
	settings.Fields.Add(&core.TextField{Name: "user"}, &core.JSONField{Name: "notifications"})
	if err := app.Save(settings); err != nil {
		t.Fatalf("create settings: %v", err)
	}

	notifications := core.NewBaseCollection("notifications")
	notifications.Fields.Add(
		&core.TextField{Name: "type"},
		&core.JSONField{Name: "metadata"},
		&core.BoolField{Name: "seen"},
		&core.TextField{Name: "recipient"},
		&core.TextField{Name: "author"},
	)
	if err := app.Save(notifications); err != nil {
		t.Fatalf("create notifications: %v", err)
	}
}

func countNotifications(t *testing.T, app core.App, recipient *core.Record) int {
	t.Helper()
	rows, err := app.FindAllRecords("notifications")
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	n := 0
	for _, r := range rows {
		if r.GetString("recipient") == recipient.Id {
			n++
		}
	}
	return n
}

func likeActivityFor(actorIRI, objectIRI string) pub.Activity {
	l := pub.LikeNew(pub.IRI(actorIRI+"/activity/like1"), pub.IRI(objectIRI))
	l.Actor = pub.IRI(actorIRI)
	return *l
}

func TestProcessLikeDuplicateDelivery(t *testing.T) {
	priv, _, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	b := newLikeReceiver(t, &priv.PublicKey)
	addNotificationCollections(t, b.app)

	alice := createTestActor(t, b.app, likeReceiverOrigin+"/api/v1/activitypub/user/alice", "person", true)
	alice.Set("user", "alice_user")
	if err := b.app.Save(alice); err != nil {
		t.Fatalf("save alice: %v", err)
	}
	settings, err := b.app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatalf("find settings: %v", err)
	}
	s := core.NewRecord(settings)
	s.Set("user", "alice_user")
	s.Set("notifications", `{"trail_like":{"web":true,"email":false}}`)
	if err := b.app.Save(s); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	trailIRI := likeReceiverOrigin + "/api/v1/trail/a1"
	trail := saveTrail(t, b.app, trailIRI, alice, true)

	t.Run("remote liker delivered twice", func(t *testing.T) {
		like := likeActivityFor(likeBobIRI, trailIRI)
		for i := 1; i <= 2; i++ {
			if err := ProcessLikeActivity(b.app, b.bob, like); err != nil {
				t.Fatalf("delivery %d: %v", i, err)
			}
		}
		if got := countTrailLikes(t, b.app, trail, b.bob); got != 1 {
			t.Fatalf("trail_like rows = %d, want 1", got)
		}
		if got := countNotifications(t, b.app, alice); got != 1 {
			t.Fatalf("notifications for alice = %d, want 1", got)
		}
	})

	t.Run("local liker still notifies the author", func(t *testing.T) {
		dave := createTestActor(t, b.app, likeReceiverOrigin+"/api/v1/activitypub/user/dave", "person", true)
		saveTrailLike(t, b.app, trail, dave)
		before := countNotifications(t, b.app, alice)

		if err := ProcessLikeActivity(b.app, dave, likeActivityFor(dave.GetString("iri"), trailIRI)); err != nil {
			t.Fatalf("ProcessLikeActivity: %v", err)
		}
		if got := countNotifications(t, b.app, alice); got != before+1 {
			t.Fatalf("notifications for alice = %d, want %d", got, before+1)
		}
		if got := countTrailLikes(t, b.app, trail, dave); got != 1 {
			t.Fatalf("trail_like rows for dave = %d, want 1", got)
		}
	})
}

func TestInstanceInboxLikeGuards(t *testing.T) {
	priv, _, err := util.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	b := newLikeReceiver(t, &priv.PublicKey)

	allLikes := func() int {
		rows, err := b.app.FindAllRecords("trail_like")
		if err != nil {
			t.Fatalf("list trail_like: %v", err)
		}
		return len(rows)
	}

	t.Run("trail not held", func(t *testing.T) {
		like := likeActivityFor(likeBobIRI, "https://third.example.com/api/v1/trail/unknown")
		b.deliver(t, priv, likeBobIRI+"#main-key", marshalActivity(t, &like), 200)
		if got := allLikes(); got != 0 {
			t.Fatalf("trail_like rows = %d, want 0", got)
		}
	})

	t.Run("signer is not an accepted peer", func(t *testing.T) {
		const eveIRI = "https://stranger.example.com/api/v1/activitypub/user/eve"
		evePriv, evePub, err := util.GenerateRSAKeyPair()
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		eve := createTestActor(t, b.app, eveIRI, "person", false)
		setActorPublicKey(t, b.app, eve, evePub)

		like := likeActivityFor(eveIRI, likeTrailIRI)
		b.deliver(t, evePriv, eveIRI+"#main-key", marshalActivity(t, &like), 403)
		if got := allLikes(); got != 0 {
			t.Fatalf("trail_like rows = %d, want 0", got)
		}
	})
}
