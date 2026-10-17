package hooks

import (
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// Recorded Create, Update and Announce activities embed a snapshot of their
// object and are publicly listed. A trail or list going private hides its
// snapshots and those of its replies; going public shows them again. A
// deleted object's snapshots are removed, but only after its Delete has read
// its audience back from them.

func setupSnapshotTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app := setupActorDeleteHooksTestApp(t)
	activities, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	activities.Fields.Add(&core.BoolField{Name: "hidden"})
	if err := app.Save(activities); err != nil {
		t.Fatal(err)
	}
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	lists.Fields.Add(&core.TextField{Name: "iri"}, &core.BoolField{Name: "public"})
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}
	feed := core.NewBaseCollection("feed")
	feed.Fields.Add(&core.TextField{Name: "item"})
	if err := app.Save(feed); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestVisibilityChangeHidesSnapshots(t *testing.T) {
	for _, kind := range []struct{ collection, path string }{{"trails", "trail"}, {"lists", "list"}} {
		t.Run(kind.collection, func(t *testing.T) {
			app := setupSnapshotTestApp(t)
			app.OnRecordAfterUpdateSuccess(kind.collection).BindFunc(func(e *core.RecordEvent) error {
				if err := syncActivityVisibility(e); err != nil {
					return err
				}
				return e.Next()
			})
			_, author := newDepartingActor(t, app, newInboxCounter(t))

			record := newSnapshotSubject(t, app, kind.collection, kind.path, author)
			other := newSnapshotSubject(t, app, kind.collection, kind.path, author)
			iri := record.GetString("iri")
			own := recordSnapshot(t, app, author, "Create", map[string]any{"id": iri})
			edit := recordSnapshot(t, app, author, "Update", map[string]any{"id": iri})
			share := recordSnapshot(t, app, author, "Announce", map[string]any{"id": iri})
			reply := recordSnapshot(t, app, author, "Create", map[string]any{"id": "https://local.example/api/v1/comment/c", "inReplyTo": iri})
			unrelated := recordSnapshot(t, app, author, "Create", map[string]any{"id": other.GetString("iri")})

			setPublic(t, app, record, false)
			assertHidden(t, app, true, own, edit, share, reply)
			assertHidden(t, app, false, unrelated)

			// A save that keeps the visibility leaves the rows as they are.
			setPublic(t, app, record, false)
			assertHidden(t, app, true, own, edit, share, reply)

			setPublic(t, app, record, true)
			assertHidden(t, app, false, own, edit, share, reply, unrelated)
		})
	}
}

// The comment's Delete goes to the actors its recorded Create was cc'd to,
// so the rows can only go once the Delete has been addressed.
func TestCommentDeletePurgesSnapshotsAfterAnnouncing(t *testing.T) {
	app := setupSnapshotTestApp(t)
	app.OnRecordAfterDeleteSuccess("comments").BindFunc(DeleteCommentHandler(nil))
	_, author := newDepartingActor(t, app, newInboxCounter(t))
	mentionedInbox := newInboxCounter(t)

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	comment := newSnapshotReply(t, app, author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())
	recordSnapshot(t, app, author, "Update", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())
	trailCreate := recordSnapshot(t, app, author, "Create", map[string]any{"id": trail.GetString("iri")})

	if err := app.Delete(comment); err != nil {
		t.Fatal(err)
	}

	if got := mentionedInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the mentioned actor should receive the comment's Delete, got %d", got)
	}
	assertSnapshotsOf(t, app, comment.GetString("iri"), 0)
	assertDeleteRecorded(t, app, comment.GetString("iri"))
	assertHidden(t, app, false, trailCreate)
}

// A trail deletion cascades its comments: the trail's Delete uses the
// audience collected before the cascade, each comment's Delete reads its own
// rows, and afterwards no snapshot of either is left.
func TestTrailDeletePurgesSnapshotsOfTrailAndReplies(t *testing.T) {
	app := setupSnapshotTestApp(t)
	// Nothing listens here; the handler logs the search failure and goes on.
	client := meilisearch.New("http://127.0.0.1:1")
	app.OnRecordDelete("trails").BindFunc(CollectTrailDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(DeleteTrailHandler(client))
	app.OnRecordAfterDeleteSuccess("comments").BindFunc(DeleteCommentHandler(nil))
	followerInbox := newInboxCounter(t)
	_, author := newDepartingActor(t, app, followerInbox)
	mentionedInbox := newInboxCounter(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	mentioned := core.NewRecord(actors)
	mentioned.Set("iri", "https://remote.example/api/v1/activitypub/user/carol")
	mentioned.Set("inbox", mentionedInbox.url())
	mentioned.Set("is_local", false)
	if err := app.Save(mentioned); err != nil {
		t.Fatal(err)
	}

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	trail.Set("public", true)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	comment := newSnapshotReply(t, app, author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": trail.GetString("iri")})
	recordSnapshot(t, app, author, "Create", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentioned.GetString("inbox"))

	if err := app.Delete(trail); err != nil {
		t.Fatal(err)
	}

	if got := followerInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the follower should receive the trail's Delete, got %d", got)
	}
	// The comment's Delete, and the trail's: the comment's mention made the
	// actor a holder of the trail.
	if got := mentionedInbox.waitForHits(2, 5*time.Second); got != 2 {
		t.Fatalf("the mentioned actor should receive both Deletes, got %d", got)
	}
	assertSnapshotsOf(t, app, trail.GetString("iri"), 0)
	assertSnapshotsOf(t, app, comment.GetString("iri"), 0)
	assertDeleteRecorded(t, app, trail.GetString("iri"))
	assertDeleteRecorded(t, app, comment.GetString("iri"))
}

func newSnapshotSubject(t *testing.T, app *pbtests.TestApp, collection, path string, author *core.Record) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(coll)
	r.Set("author", author.Id)
	r.Set("public", true)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", "https://local.example/api/v1/"+path+"/"+r.Id)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func newSnapshotReply(t *testing.T, app *pbtests.TestApp, author, trail *core.Record) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(coll)
	r.Set("author", author.Id)
	r.Set("trail", trail.Id)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", "https://local.example/api/v1/comment/"+r.Id)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func recordSnapshot(t *testing.T, app *pbtests.TestApp, author *core.Record, typ string, object map[string]any, cc ...string) *core.Record {
	t.Helper()

	r := recordActivity(t, app, author.GetString("iri"), typ, cc...)
	r.Set("object", object)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// setPublic reloads the record first, so Original() is the stored row.
func setPublic(t *testing.T, app *pbtests.TestApp, record *core.Record, public bool) {
	t.Helper()

	fresh, err := app.FindRecordById(record.Collection().Name, record.Id)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Set("public", public)
	if err := app.Save(fresh); err != nil {
		t.Fatal(err)
	}
}

func assertHidden(t *testing.T, app *pbtests.TestApp, want bool, records ...*core.Record) {
	t.Helper()

	for _, r := range records {
		fresh, err := app.FindRecordById("activitypub_activities", r.Id)
		if err != nil {
			t.Fatal(err)
		}
		if got := fresh.GetBool("hidden"); got != want {
			t.Fatalf("%s %s: hidden = %v, want %v", fresh.GetString("type"), fresh.GetString("object"), got, want)
		}
	}
}

func assertSnapshotsOf(t *testing.T, app *pbtests.TestApp, objectIRI string, want int) {
	t.Helper()

	records, err := app.FindRecordsByFilter(
		"activitypub_activities",
		"type != 'Delete' && object.id = {:iri}",
		"", 0, 0,
		dbx.Params{"iri": objectIRI},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != want {
		t.Fatalf("%d snapshots of %s left, want %d", len(records), objectIRI, want)
	}
}

func assertDeleteRecorded(t *testing.T, app *pbtests.TestApp, objectIRI string) {
	t.Helper()

	if _, err := app.FindFirstRecordByFilter(
		"activitypub_activities",
		"type = 'Delete' && object = {:iri}",
		dbx.Params{"iri": objectIRI},
	); err != nil {
		t.Fatalf("expected a Delete of %s to be recorded: %v", objectIRI, err)
	}
}
