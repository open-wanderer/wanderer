package federation

import (
	"errors"
	"sort"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

const (
	gateOrigin      = "https://trails.example.com"
	gateInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"
	gateLocalUser   = "https://trails.example.com/api/v1/activitypub/user/alice"
	gatePeer1       = "https://peer1.example.com/api/v1/activitypub/instance"
	gatePeer2       = "https://peer2.example.com/api/v1/activitypub/instance"
	gatePeer3       = "https://peer3.example.com/api/v1/activitypub/instance"
	gatePeer4       = "https://peer4.example.com/api/v1/activitypub/instance"
	gateStranger    = "https://stranger.example.com/api/v1/activitypub/user/eve"
)

// newPeerGateTestApp extends the inbox fixture so follows.status also accepts "rejected".
func newPeerGateTestApp(t *testing.T) core.App {
	t.Helper()
	t.Setenv("ORIGIN", gateOrigin)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	col, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows: %v", err)
	}
	sel, ok := col.Fields.GetByName("status").(*core.SelectField)
	if !ok {
		t.Fatalf("status is not a select field")
	}
	sel.Values = append(sel.Values, "rejected")
	if err := app.Save(col); err != nil {
		t.Fatalf("save follows: %v", err)
	}
	return app
}

func gateSeedFollow(t *testing.T, app core.App, followerID, followeeID, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("follower", followerID)
	r.Set("followee", followeeID)
	r.Set("status", status)
	if err := app.Save(r); err != nil {
		t.Fatalf("save follow: %v", err)
	}
	return r
}

type gateFixture struct {
	app                              core.App
	instance, user                   *core.Record
	p1, p2, p3, p4, stranger         *core.Record
	outboundP1, inboundP2, inboundP4 *core.Record
}

// newGateFixture: P1 accepted outbound, P2 accepted inbound, P3 pending inbound only,
// P4 rejected inbound only.
func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	app := newPeerGateTestApp(t)
	f := &gateFixture{app: app}
	f.instance = createTestActor(t, app, gateInstanceIRI, "instance", true)
	f.user = createTestActor(t, app, gateLocalUser, "person", true)
	f.p1 = createTestActor(t, app, gatePeer1, "instance", false)
	f.p2 = createTestActor(t, app, gatePeer2, "instance", false)
	f.p3 = createTestActor(t, app, gatePeer3, "instance", false)
	f.p4 = createTestActor(t, app, gatePeer4, "instance", false)
	f.stranger = createTestActor(t, app, gateStranger, "person", false)
	f.outboundP1 = gateSeedFollow(t, app, f.instance.Id, f.p1.Id, "accepted")
	f.inboundP2 = gateSeedFollow(t, app, f.p2.Id, f.instance.Id, "accepted")
	gateSeedFollow(t, app, f.p3.Id, f.instance.Id, "pending")
	f.inboundP4 = gateSeedFollow(t, app, f.p4.Id, f.instance.Id, "rejected")
	return f
}

func gateFollow(id, actorIRI, objectIRI string) *pub.Activity {
	a := pub.ActivityNew(pub.IRI(id), pub.FollowType, pub.IRI(objectIRI))
	a.Actor = pub.IRI(actorIRI)
	return a
}

func gateWrap(typ pub.ActivityVocabularyType, id, actorIRI string, object pub.Item) pub.Activity {
	a := pub.ActivityNew(pub.IRI(id), typ, object)
	a.Actor = pub.IRI(actorIRI)
	return *a
}

func TestAcceptedPeerActorsBothDirections(t *testing.T) {
	f := newGateFixture(t)
	// P1 additionally follows us: mutual peer must appear once.
	gateSeedFollow(t, f.app, f.p1.Id, f.instance.Id, "accepted")

	peers, err := acceptedPeerActors(f.app)
	if err != nil {
		t.Fatalf("acceptedPeerActors: %v", err)
	}
	var got []string
	for _, p := range peers {
		got = append(got, p.IRI)
	}
	sort.Strings(got)
	want := []string{gatePeer1, gatePeer2}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("peers = %v, want %v", got, want)
	}

	t.Run("no instance actor", func(t *testing.T) {
		app := newPeerGateTestApp(t)
		peers, err := acceptedPeerActors(app)
		if err != nil || peers != nil {
			t.Fatalf("got (%v, %v), want (nil, nil)", peers, err)
		}
	})

	t.Run("ORIGIN unset", func(t *testing.T) {
		t.Setenv("ORIGIN", "")
		if _, err := acceptedPeerActors(f.app); err == nil {
			t.Fatal("expected error when ORIGIN is empty")
		}
	})
}

func TestIsAcceptedPeerHost(t *testing.T) {
	f := newGateFixture(t)
	cases := []struct {
		name string
		iri  string
		want bool
	}{
		{"peer1 instance", gatePeer1, true},
		{"user on peer1 host", "https://peer1.example.com/api/v1/activitypub/user/bob", true},
		{"peer2 inbound-only", gatePeer2, true},
		{"peer3 pending only", gatePeer3, false},
		{"peer4 rejected only", gatePeer4, false},
		{"unknown host", "https://unknown.example.com/x", false},
		{"look-alike suffix", "https://evil-peer1.example.com/x", false},
		{"different port", "https://peer1.example.com:8443/x", false},
		{"case-insensitive", "HTTPS://PEER1.EXAMPLE.COM/x", true},
		{"empty", "", false},
		{"not a url", "not a url", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IsAcceptedPeerHost(f.app, tc.iri)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("IsAcceptedPeerHost(%q) = %v, want %v", tc.iri, got, tc.want)
			}
		})
	}
}

func TestLifecycleTargetsInstance(t *testing.T) {
	embeddedFollowFromInstance := gateFollow("https://trails.example.com/activity/f1", gateInstanceIRI, gatePeer1)
	embeddedFollowFromUser := gateFollow("https://trails.example.com/activity/f2", gateLocalUser, gatePeer1)
	embeddedFollowToInstance := gateFollow("https://peer1.example.com/activity/f3", gatePeer1, gateInstanceIRI)
	embeddedFollowToUser := gateFollow("https://peer1.example.com/activity/f4", gatePeer1, gateLocalUser)
	like := pub.ActivityNew(pub.IRI("https://peer1.example.com/activity/l1"), pub.LikeType, pub.IRI("https://x.example.com/o"))

	noObject := gateWrap(pub.FollowType, "https://peer1.example.com/activity/n", gatePeer1, nil)
	noObject.Object = nil

	cases := []struct {
		name string
		act  pub.Activity
		want bool
	}{
		{"follow to instance", gateWrap(pub.FollowType, "https://peer1.example.com/a/1", gatePeer1, pub.IRI(gateInstanceIRI)), true},
		{"follow to local user", gateWrap(pub.FollowType, "https://peer1.example.com/a/2", gatePeer1, pub.IRI(gateLocalUser)), false},
		{"accept of instance follow", gateWrap(pub.AcceptType, "https://peer1.example.com/a/3", gatePeer1, embeddedFollowFromInstance), true},
		{"accept of user follow", gateWrap(pub.AcceptType, "https://peer1.example.com/a/4", gatePeer1, embeddedFollowFromUser), false},
		{"accept with IRI-only object", gateWrap(pub.AcceptType, "https://peer1.example.com/a/5", gatePeer1, pub.IRI("https://trails.example.com/activity/f1")), false},
		{"reject of instance follow", gateWrap(pub.RejectType, "https://peer1.example.com/a/6", gatePeer1, embeddedFollowFromInstance), true},
		{"undo follow to instance", gateWrap(pub.UndoType, "https://peer1.example.com/a/7", gatePeer1, embeddedFollowToInstance), true},
		{"undo follow to user", gateWrap(pub.UndoType, "https://peer1.example.com/a/8", gatePeer1, embeddedFollowToUser), false},
		{"undo like", gateWrap(pub.UndoType, "https://peer1.example.com/a/9", gatePeer1, like), false},
		{"create", gateWrap(pub.CreateType, "https://peer1.example.com/a/10", gatePeer1, pub.IRI(gateInstanceIRI)), false},
		{"nil object", noObject, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lifecycleTargetsInstance(tc.act, gateInstanceIRI); got != tc.want {
				t.Fatalf("lifecycleTargetsInstance = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAuthorizeInstanceActivity(t *testing.T) {
	f := newGateFixture(t)
	userOnP1 := createTestActor(t, f.app, "https://peer1.example.com/api/v1/activitypub/user/bob", "person", false)

	create := gateWrap(pub.CreateType, "https://x.example.com/a/1", gateStranger, pub.IRI("https://x.example.com/o/1"))
	del := gateWrap(pub.DeleteType, "https://x.example.com/a/2", gatePeer4, pub.IRI("https://x.example.com/o/2"))
	followInstance := gateWrap(pub.FollowType, "https://x.example.com/a/3", gateStranger, pub.IRI(gateInstanceIRI))
	followUser := gateWrap(pub.FollowType, "https://x.example.com/a/4", gateStranger, pub.IRI(gateLocalUser))
	undoLike := gateWrap(pub.UndoType, "https://x.example.com/a/5", gateStranger,
		pub.ActivityNew(pub.IRI("https://x.example.com/l"), pub.LikeType, pub.IRI("https://x.example.com/o")))

	cases := []struct {
		name   string
		signer *core.Record
		act    pub.Activity
		want   error
	}{
		{"create from stranger", f.stranger, create, errNotAcceptedPeer},
		{"create from user on accepted peer host", userOnP1, create, nil},
		{"delete from rejected-only host", f.p4, del, errNotAcceptedPeer},
		{"follow to instance from stranger", f.stranger, followInstance, nil},
		{"follow to local user from stranger", f.stranger, followUser, errLifecycleNotForInstance},
		{"undo like from stranger", f.stranger, undoLike, errNotAcceptedPeer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := authorizeInstanceActivity(f.app, tc.signer, tc.act, gateInstanceIRI)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestInstanceFollowerInboxesAfterDisconnectState(t *testing.T) {
	f := newGateFixture(t)

	// Disconnect P1 (outbound row deleted) and P2 (inbound row rejected).
	if err := f.app.Delete(f.outboundP1); err != nil {
		t.Fatalf("delete outbound: %v", err)
	}
	f.inboundP2.Set("status", "rejected")
	if err := f.app.Save(f.inboundP2); err != nil {
		t.Fatalf("reject inbound: %v", err)
	}

	inboxes, err := instanceFollowerInboxes(f.app)
	if err != nil {
		t.Fatalf("instanceFollowerInboxes: %v", err)
	}
	if len(inboxes) != 0 {
		t.Fatalf("inboxes after disconnect = %v, want none", inboxes)
	}
	for _, iri := range []string{gatePeer1, gatePeer2} {
		ok, err := IsAcceptedPeerHost(f.app, iri)
		if err != nil {
			t.Fatalf("IsAcceptedPeerHost: %v", err)
		}
		if ok {
			t.Fatalf("IsAcceptedPeerHost(%s) = true after disconnect", iri)
		}
	}
}
