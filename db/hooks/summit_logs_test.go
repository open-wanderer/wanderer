package hooks

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// A summit log removed by its trail's deletion still has an audience of its
// own: the log author's followers, who received it when it was created and
// need not follow the trail's author. The delete hook used to return early
// when the trail was gone, so they were never told.

func TestDeleteSummitLogHandlerCascadedByTrail(t *testing.T) {
	app := setupActorDeleteHooksTestApp(t)
	inbox := newInboxCounter(t)

	// No Meilisearch client: with the trail gone there is nothing to reindex,
	// and that branch is the one under test.
	app.OnRecordAfterDeleteSuccess("summit_logs").BindFunc(DeleteSummitLogHandler(nil))

	// A local author with a follower whose inbox is the counting server.
	_, author := newDepartingActor(t, app, inbox)

	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	trailAuthor := core.NewRecord(actors)
	trailAuthor.Set("iri", "https://remote.example/api/v1/activitypub/user/trailowner")
	trailAuthor.Set("inbox", "https://remote.example/api/v1/activitypub/user/trailowner/inbox")
	trailAuthor.Set("is_local", false)
	if err := app.Save(trailAuthor); err != nil {
		t.Fatal(err)
	}

	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("author", trailAuthor.Id)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}

	logs, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		t.Fatal(err)
	}
	log := core.NewRecord(logs)
	log.Set("iri", "https://local.example/api/v1/summit-log/abc")
	log.Set("author", author.Id)
	log.Set("trail", trail.Id)
	if err := app.Save(log); err != nil {
		t.Fatal(err)
	}

	// Deleting the trail cascades to the log, whose post-commit hook then runs
	// with no trail left to look up.
	if err := app.Delete(trail); err != nil {
		t.Fatal(err)
	}

	if _, err := app.FindRecordById("summit_logs", log.Id); err == nil {
		t.Fatal("expected the log to be cascaded away with its trail")
	}

	if got := inbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("the log author's follower should receive the log's Delete, got %d delivery(ies)", got)
	}
}
