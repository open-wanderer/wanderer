package hooks

import (
	"testing"
	"time"

	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
)

// Deleting is the one way to withdraw a record from other servers, and turning
// a record private sends nothing. So a trail that went private after it was
// handed out has to reach its followers and peers when it is deleted, while one
// that was never public has nothing to retract from them. These tests delete
// through the same hooks the server binds, rather than calling the recipient
// functions, because the hooks are where the record's current flag is read.

// bindTrailDeleteAnnouncement binds the two halves of a trail deletion as
// main.go does, minus the Meilisearch call, which needs a live index.
func bindTrailDeleteAnnouncement(app *pbtests.TestApp) {
	app.OnRecordDelete("trails").BindFunc(CollectTrailDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		audience, _ := e.Record.GetRaw(trailDeleteRecipientsKey).(federation.DeleteAudience)
		if err := federation.CreateTrailDeleteActivity(e.App, e.Record, audience); err != nil {
			return err
		}
		return e.Next()
	})
}

// bindListDeleteAnnouncement is bindTrailDeleteAnnouncement for a list.
func bindListDeleteAnnouncement(app *pbtests.TestApp) {
	app.OnRecordDelete("lists").BindFunc(CollectListDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(func(e *core.RecordEvent) error {
		audience, _ := e.Record.GetRaw(listDeleteRecipientsKey).(federation.DeleteAudience)
		if err := federation.CreateListDeleteActivity(e.App, e.Record, audience); err != nil {
			return err
		}
		return e.Next()
	})
}

// localTrailOf saves a trail of a local author and gives it the IRI this
// instance would.
func localTrailOf(t *testing.T, app *pbtests.TestApp, author *core.Record, public bool) *core.Record {
	t.Helper()

	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("author", author.Id)
	trail.Set("public", public)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	trail.Set("iri", "https://local.example/api/v1/trail/"+trail.Id)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	return trail
}

// localListOf is localTrailOf for a list.
func localListOf(t *testing.T, app *pbtests.TestApp, author *core.Record, public bool) *core.Record {
	t.Helper()

	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	list := core.NewRecord(lists)
	list.Set("author", author.Id)
	list.Set("public", public)
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	list.Set("iri", "https://local.example/api/v1/list/"+list.Id)
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	return list
}

// recordObjectActivity stores an outgoing activity of actorIRI about objectIRI
// the way create.go does, which is what proves the object was once handed out.
func recordObjectActivity(t *testing.T, app *pbtests.TestApp, actorIRI, typ, objectIRI string) {
	t.Helper()

	activities, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(activities)
	r.Set("iri", "https://local.example/api/v1/activitypub/activity/"+security.RandomString(8))
	r.Set("actor", actorIRI)
	r.Set("type", typ)
	r.Set("object", map[string]any{"id": objectIRI, "type": "Note"})
	r.Set("published", time.Now().Format(time.RFC3339))
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateTrailDeleteReachesItsPublicAudience(t *testing.T) {
	// Public when it went out, private when it is deleted: the follower holds a
	// copy and is told.
	t.Run("WentPrivateReachesFollower", func(t *testing.T) {
		app := setupActorDeleteHooksTestApp(t)
		bindTrailDeleteAnnouncement(app)
		followerInbox := newInboxCounter(t)

		_, author := newDepartingActor(t, app, followerInbox)
		trail := localTrailOf(t, app, author, true)
		recordObjectActivity(t, app, author.GetString("iri"), "Create", trail.GetString("iri"))

		trail.Set("public", false)
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
		if err := app.Delete(trail); err != nil {
			t.Fatal(err)
		}

		if got := followerInbox.waitForHits(1, 5*time.Second); got != 1 {
			t.Fatalf("expected exactly one delivery to the follower, got %d", got)
		}
		if got := followerInbox.waitForHits(2, 500*time.Millisecond); got != 1 {
			t.Fatalf("expected the follower to be told once, got %d deliveries", got)
		}
	})

	// A connected instance received the trail through instance federation, so
	// it is told alongside the follower.
	t.Run("WentPrivateReachesAcceptedPeer", func(t *testing.T) {
		app := setupActorDeleteHooksTestApp(t)
		bindTrailDeleteAnnouncement(app)
		followerInbox := newInboxCounter(t)
		peerInbox := newInboxCounter(t)

		_, author := newDepartingActor(t, app, followerInbox)
		seedAcceptedPeer(t, app, peerInbox.url(), false)
		trail := localTrailOf(t, app, author, true)
		recordObjectActivity(t, app, author.GetString("iri"), "Create", trail.GetString("iri"))

		trail.Set("public", false)
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
		if err := app.Delete(trail); err != nil {
			t.Fatal(err)
		}

		if got := peerInbox.waitForHits(1, 5*time.Second); got != 1 {
			t.Fatalf("expected exactly one delivery to the accepted peer, got %d", got)
		}
		if got := peerInbox.waitForHits(2, 500*time.Millisecond); got != 1 {
			t.Fatalf("expected the peer to be told once, got %d deliveries", got)
		}
	})

	// Never public, so no Create was recorded: telling the follower would
	// disclose the IRI of an unpublished trail.
	t.Run("NeverPublicStaysSilentForFollower", func(t *testing.T) {
		app := setupActorDeleteHooksTestApp(t)
		bindTrailDeleteAnnouncement(app)
		followerInbox := newInboxCounter(t)

		_, author := newDepartingActor(t, app, followerInbox)
		trail := localTrailOf(t, app, author, false)

		if err := app.Delete(trail); err != nil {
			t.Fatal(err)
		}

		if got := followerInbox.waitForHits(1, 500*time.Millisecond); got != 0 {
			t.Fatalf("a trail that was never public must not reach followers, got %d delivery(ies)", got)
		}
	})
}

func TestPrivateListDeleteReachesItsPublicAudience(t *testing.T) {
	t.Run("WentPrivateReachesFollower", func(t *testing.T) {
		app := setupActorDeleteHooksTestApp(t)
		bindListDeleteAnnouncement(app)
		followerInbox := newInboxCounter(t)

		_, author := newDepartingActor(t, app, followerInbox)
		list := localListOf(t, app, author, true)
		recordObjectActivity(t, app, author.GetString("iri"), "Create", list.GetString("iri"))

		list.Set("public", false)
		if err := app.Save(list); err != nil {
			t.Fatal(err)
		}
		if err := app.Delete(list); err != nil {
			t.Fatal(err)
		}

		if got := followerInbox.waitForHits(1, 5*time.Second); got != 1 {
			t.Fatalf("expected exactly one delivery to the follower, got %d", got)
		}
		if got := followerInbox.waitForHits(2, 500*time.Millisecond); got != 1 {
			t.Fatalf("expected the follower to be told once, got %d deliveries", got)
		}
	})

	t.Run("NeverPublicStaysSilentForFollower", func(t *testing.T) {
		app := setupActorDeleteHooksTestApp(t)
		bindListDeleteAnnouncement(app)
		followerInbox := newInboxCounter(t)

		_, author := newDepartingActor(t, app, followerInbox)
		list := localListOf(t, app, author, false)

		if err := app.Delete(list); err != nil {
			t.Fatal(err)
		}

		if got := followerInbox.waitForHits(1, 500*time.Millisecond); got != 0 {
			t.Fatalf("a list that was never public must not reach followers, got %d delivery(ies)", got)
		}
	})
}
