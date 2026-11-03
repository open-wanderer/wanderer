package hooks

import (
	"maps"
	"pocketbase/federation"
	"slices"
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
// deleted object's snapshots are stripped to its addressing, which the
// Deletes of the account and of the trail a reply was on read back later.

func setupSnapshotTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app := setupActorDeleteHooksTestApp(t)
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	lists.Fields.Add(&core.TextField{Name: "iri"}, &core.BoolField{Name: "public"})
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}
	// What the search document expands, so an index update gets as far as
	// the search client.
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tags", "categories"} {
		if err := app.Save(core.NewBaseCollection(name)); err != nil {
			t.Fatal(err)
		}
	}
	tags, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	categories, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		t.Fatal(err)
	}
	trails.Fields.Add(
		&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 99},
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	lists, err = app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	lists.Fields.Add(&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 99})
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
			share.Set("hidden", true)
			if err := app.Save(share); err != nil {
				t.Fatal(err)
			}
			reply := recordSnapshot(t, app, author, "Create", map[string]any{"id": "https://local.example/api/v1/comment/c", "inReplyTo": iri})
			deletedReply := recordSnapshot(t, app, author, "Create", map[string]any{"id": "https://local.example/api/v1/comment/gone", "inReplyTo": iri})
			if err := federation.StripObjectActivities(app, "https://local.example/api/v1/comment/gone"); err != nil {
				t.Fatal(err)
			}
			unrelated := recordSnapshot(t, app, author, "Create", map[string]any{"id": other.GetString("iri")})

			setPublic(t, app, record, false)
			assertHidden(t, app, true, own, edit, share, reply)
			assertHidden(t, app, false, unrelated)

			// A save that keeps the visibility leaves the rows as they are.
			setPublic(t, app, record, false)
			assertHidden(t, app, true, own, edit, share, reply)

			setPublic(t, app, record, true)
			assertHidden(t, app, false, own, edit, reply, unrelated)
			// A deleted reply's row has nothing left to list, and a share is
			// for its recipient alone.
			assertHidden(t, app, true, deletedReply, share)
		})
	}
}

// The comment's Delete goes to the actors its recorded Create was cc'd to,
// so the rows are only stripped once the Delete has been addressed.
func TestCommentDeleteStripsSnapshotsAfterAnnouncing(t *testing.T) {
	app := setupSnapshotTestApp(t)
	app.OnRecordAfterDeleteSuccess("comments").BindFunc(DeleteCommentHandler(nil))
	_, author := newDepartingActor(t, app, newInboxCounter(t))
	mentionedInbox := newInboxCounter(t)

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	comment := newSnapshotReply(t, app, "comments", "comment", author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())
	recordSnapshot(t, app, author, "Update", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())
	trailCreate := recordSnapshot(t, app, author, "Create", map[string]any{"id": trail.GetString("iri")})

	if err := app.Delete(comment); err != nil {
		t.Fatal(err)
	}

	if got := mentionedInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the mentioned actor should receive the comment's Delete, got %d", got)
	}
	assertStripped(t, app, comment.GetString("iri"), 2, trail.GetString("iri"), mentionedInbox.url())
	assertDeleteRecorded(t, app, comment.GetString("iri"))
	assertHidden(t, app, false, trailCreate)
}

// A trail deletion cascades its comments and summit logs: the trail's Delete
// uses the audience collected before the cascade, each reply's Delete reads
// its own rows, and afterwards none of them has content left.
func TestTrailDeleteStripsSnapshotsOfTrailAndReplies(t *testing.T) {
	app := setupSnapshotTestApp(t)
	// Nothing listens here; the handler logs the search failure and goes on.
	client := meilisearch.New("http://127.0.0.1:1")
	app.OnRecordDelete("trails").BindFunc(CollectTrailDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(DeleteTrailHandler(client))
	app.OnRecordAfterDeleteSuccess("comments").BindFunc(DeleteCommentHandler(nil))
	app.OnRecordAfterDeleteSuccess("summit_logs").BindFunc(DeleteSummitLogHandler(client))
	followerInbox := newInboxCounter(t)
	_, author := newDepartingActor(t, app, followerInbox)
	mentionedInbox := newInboxCounter(t)
	newRemoteActor(t, app, "carol", mentionedInbox)

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	comment := newSnapshotReply(t, app, "comments", "comment", author, trail)
	log := newSnapshotReply(t, app, "summit_logs", "summit-log", author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": trail.GetString("iri")})
	recordSnapshot(t, app, author, "Create", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())
	recordSnapshot(t, app, author, "Create", map[string]any{"id": log.GetString("iri"), "inReplyTo": trail.GetString("iri")})

	if err := app.Delete(trail); err != nil {
		t.Fatal(err)
	}

	// The trail's Delete and the summit log's.
	if got := followerInbox.waitForHits(2, 5*time.Second); got != 2 {
		t.Fatalf("the follower should receive the Deletes of the trail and the summit log, got %d", got)
	}
	// The comment's Delete, and the trail's: the comment's mention made the
	// actor a holder of the trail.
	if got := mentionedInbox.waitForHits(2, 5*time.Second); got != 2 {
		t.Fatalf("the mentioned actor should receive both Deletes, got %d", got)
	}
	assertStripped(t, app, trail.GetString("iri"), 1, "")
	assertStripped(t, app, comment.GetString("iri"), 1, trail.GetString("iri"), mentionedInbox.url())
	assertStripped(t, app, log.GetString("iri"), 1, trail.GetString("iri"))
	assertDeleteRecorded(t, app, trail.GetString("iri"))
	assertDeleteRecorded(t, app, comment.GetString("iri"))
	assertDeleteRecorded(t, app, log.GetString("iri"))
}

// A comment mentioning a remote actor who follows nobody here is deleted, and
// the trail after it. The trail's Delete still reaches that actor, whose
// instance fetched the trail the comment was on.
func TestTrailDeleteReachesActorMentionedInDeletedComment(t *testing.T) {
	app := setupSnapshotTestApp(t)
	client := meilisearch.New("http://127.0.0.1:1")
	app.OnRecordDelete("trails").BindFunc(CollectTrailDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(DeleteTrailHandler(client))
	app.OnRecordAfterDeleteSuccess("comments").BindFunc(DeleteCommentHandler(nil))
	_, author := newDepartingActor(t, app, newInboxCounter(t))
	mentionedInbox := newInboxCounter(t)
	newRemoteActor(t, app, "carol", mentionedInbox)

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	comment := newSnapshotReply(t, app, "comments", "comment", author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": trail.GetString("iri")})
	recordSnapshot(t, app, author, "Create", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())

	if err := app.Delete(comment); err != nil {
		t.Fatal(err)
	}
	if got := mentionedInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the mentioned actor should receive the comment's Delete, got %d", got)
	}
	if err := app.Delete(trail); err != nil {
		t.Fatal(err)
	}
	if got := mentionedInbox.waitForHits(2, 5*time.Second); got != 2 {
		t.Fatalf("the mentioned actor should receive the trail's Delete too, got %d", got)
	}
}

// The same for the account: its Delete(Actor) reaches an actor known only
// from the cc of a comment that was deleted before.
func TestAccountDeleteReachesActorMentionedInDeletedComment(t *testing.T) {
	app := setupSnapshotTestApp(t)
	app.OnRecordAfterDeleteSuccess("comments").BindFunc(DeleteCommentHandler(nil))
	owner, author := newDepartingActor(t, app, newInboxCounter(t))
	mentionedInbox := newInboxCounter(t)
	newRemoteActor(t, app, "carol", mentionedInbox)

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	comment := newSnapshotReply(t, app, "comments", "comment", author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": comment.GetString("iri"), "inReplyTo": trail.GetString("iri")}, mentionedInbox.url())

	if err := app.Delete(comment); err != nil {
		t.Fatal(err)
	}
	if got := mentionedInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the mentioned actor should receive the comment's Delete, got %d", got)
	}
	if err := app.Delete(owner); err != nil {
		t.Fatal(err)
	}
	if got := mentionedInbox.waitForHits(2, 5*time.Second); got != 2 {
		t.Fatalf("the mentioned actor should receive the account's Delete too, got %d", got)
	}
}

// A shared list's Delete reaches the share's recipient through the to of the
// Announce, which outlives the share; the list's rows are stripped after it.
func TestListDeleteStripsSnapshotsAfterAnnouncing(t *testing.T) {
	app := setupSnapshotTestApp(t)
	client := meilisearch.New("http://127.0.0.1:1")
	app.OnRecordDelete("lists").BindFunc(CollectListDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(DeleteListHandler(client))
	_, author := newDepartingActor(t, app, newInboxCounter(t))
	shareeInbox := newInboxCounter(t)
	sharee := newRemoteActor(t, app, "dave", shareeInbox)

	list := newSnapshotSubject(t, app, "lists", "list", author)
	announce := recordSnapshot(t, app, author, "Announce", map[string]any{"id": list.GetString("iri")})
	announce.Set("to", []string{sharee.GetString("iri")})
	if err := app.Save(announce); err != nil {
		t.Fatal(err)
	}
	recordSnapshot(t, app, author, "Create", map[string]any{"id": list.GetString("iri")})

	if err := app.Delete(list); err != nil {
		t.Fatal(err)
	}

	if got := shareeInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the share's recipient should receive the list's Delete, got %d", got)
	}
	assertStripped(t, app, list.GetString("iri"), 2, "")
	assertDeleteRecorded(t, app, list.GetString("iri"))
}

// A summit log on a trail that still exists is deleted while the search
// index is unreachable. The trail cannot be reindexed, but the log's Delete
// still reaches its author's followers and its rows are stripped.
func TestSummitLogDeleteSurvivesIndexFailure(t *testing.T) {
	app := setupSnapshotTestApp(t)
	client := meilisearch.New("http://127.0.0.1:1")
	app.OnRecordAfterDeleteSuccess("summit_logs").BindFunc(DeleteSummitLogHandler(client))
	followerInbox := newInboxCounter(t)
	_, author := newDepartingActor(t, app, followerInbox)

	trail := newSnapshotSubject(t, app, "trails", "trail", author)
	log := newSnapshotReply(t, app, "summit_logs", "summit-log", author, trail)
	recordSnapshot(t, app, author, "Create", map[string]any{"id": log.GetString("iri"), "inReplyTo": trail.GetString("iri")})

	if err := app.Delete(log); err != nil {
		t.Fatalf("the search failure must not fail the deletion: %v", err)
	}

	if got := followerInbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the follower should receive the summit log's Delete, got %d", got)
	}
	assertStripped(t, app, log.GetString("iri"), 1, trail.GetString("iri"))
	assertDeleteRecorded(t, app, log.GetString("iri"))
}

// The visibility follows the record even when the search index cannot be
// updated: the record is stored either way, and a later save would not
// notice that anything is left to do.
func TestVisibilityChangeSurvivesIndexFailure(t *testing.T) {
	client := meilisearch.New("http://127.0.0.1:1")
	for _, kind := range []struct {
		collection, path string
		handler          func(meilisearch.ServiceManager) func(*core.RecordEvent) error
	}{
		{"trails", "trail", UpdateTrailHandler},
		{"lists", "list", UpdateListHandler},
	} {
		t.Run(kind.collection, func(t *testing.T) {
			app := setupSnapshotTestApp(t)
			_, author := newDepartingActor(t, app, newInboxCounter(t))
			record := newSnapshotSubject(t, app, kind.collection, kind.path, author)
			create := recordSnapshot(t, app, author, "Create", map[string]any{"id": record.GetString("iri")})
			app.OnRecordAfterUpdateSuccess(kind.collection).BindFunc(kind.handler(client))

			fresh, err := app.FindRecordById(kind.collection, record.Id)
			if err != nil {
				t.Fatal(err)
			}
			fresh.Set("public", false)
			if err := app.Save(fresh); err == nil {
				t.Fatal("expected the unreachable search index to fail the update")
			}
			assertHidden(t, app, true, create)
		})
	}
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

func newSnapshotReply(t *testing.T, app *pbtests.TestApp, collection, path string, author, trail *core.Record) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(coll)
	r.Set("author", author.Id)
	r.Set("trail", trail.Id)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", "https://local.example/api/v1/"+path+"/"+r.Id)
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// newRemoteActor is a remote actor following nobody here, as a mentioned one
// or a share's recipient is.
func newRemoteActor(t *testing.T, app *pbtests.TestApp, name string, inbox *inboxCounter) *core.Record {
	t.Helper()

	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(actors)
	r.Set("iri", "https://remote.example/api/v1/activitypub/user/"+name)
	r.Set("inbox", inbox.url())
	r.Set("is_local", false)
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

// assertStripped checks that the want snapshots of objectIRI are all left
// hidden, with a Tombstone holding nothing but the id and inReplyTo, and
// still cc'd to the given inboxes.
func assertStripped(t *testing.T, app *pbtests.TestApp, objectIRI string, want int, inReplyTo string, cc ...string) {
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
	wantObject := map[string]any{"id": objectIRI, "type": "Tombstone"}
	if inReplyTo != "" {
		wantObject["inReplyTo"] = inReplyTo
	}
	for _, r := range records {
		var object map[string]any
		if err := r.UnmarshalJSONField("object", &object); err != nil {
			t.Fatal(err)
		}
		if !maps.Equal(object, wantObject) {
			t.Fatalf("%s of %s: object = %v, want %v", r.GetString("type"), objectIRI, object, wantObject)
		}
		if !r.GetBool("hidden") {
			t.Fatalf("%s of %s is not hidden", r.GetString("type"), objectIRI)
		}
		if got := r.GetStringSlice("cc"); len(cc) > 0 && !slices.Equal(got, cc) {
			t.Fatalf("%s of %s: cc = %v, want %v", r.GetString("type"), objectIRI, got, cc)
		}
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
