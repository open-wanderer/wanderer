package hooks

import (
	"testing"
	"time"

	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

// Whoever commented on, logged a summit on, liked or was handed a trail holds
// a copy of it and has to be told when it goes. Those rows are cascaded away
// with the trail, inside the deleting transaction, so the audience is read
// before the deletion and announced after the commit.

func TestTrailDeleteAudienceSurvivesCascade(t *testing.T) {
	app := setupActorDeleteHooksTestApp(t)
	inbox := newInboxCounter(t)

	app.OnRecordDelete("trails").BindFunc(CollectTrailDeleteRecipientsHandler())
	// DeleteTrailHandler minus the Meilisearch call, which needs a live
	// index; the part under test is that the collected audience reaches it.
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		audience, _ := e.Record.GetRaw(trailDeleteRecipientsKey).(federation.DeleteAudience)
		if err := federation.CreateTrailDeleteActivity(e.App, e.Record, audience); err != nil {
			return err
		}
		return e.Next()
	})

	_, author := newDepartingActor(t, app, newInboxCounter(t))

	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	commenter := core.NewRecord(actors)
	commenter.Set("iri", "https://remote.example/api/v1/activitypub/user/commenter")
	commenter.Set("inbox", inbox.url())
	commenter.Set("is_local", false)
	if err := app.Save(commenter); err != nil {
		t.Fatal(err)
	}

	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("author", author.Id)
	trail.Set("public", true)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	trail.Set("iri", "https://local.example/api/v1/trail/"+trail.Id)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}

	comments, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatal(err)
	}
	comment := core.NewRecord(comments)
	comment.Set("author", commenter.Id)
	comment.Set("trail", trail.Id)
	if err := app.Save(comment); err != nil {
		t.Fatal(err)
	}

	if err := app.Delete(trail); err != nil {
		t.Fatal(err)
	}

	if _, err := app.FindRecordById("comments", comment.Id); err == nil {
		t.Fatal("expected the comment to be cascaded away with its trail")
	}
	if got := inbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the commenter should receive the trail's Delete, got %d delivery(ies)", got)
	}
}
