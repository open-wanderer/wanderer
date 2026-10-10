package federation

import (
	"context"
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// An Update of a stored list persists needs_full_sync.
func TestUpdateListPersistsNeedsFullSync(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	list := authorGuardSeedList(t, app, feedDedupListIRI, a.author.Id)
	if list.GetBool("needs_full_sync") {
		t.Fatal("seeded list already needs a full sync")
	}

	act := authorGuardAsUpdate(feedDedupListCreate(feedDedupAuthorIRI, feedDedupListIRI))
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act); err != nil {
		t.Fatalf("Update from the stored author: %v", err)
	}

	reloaded, err := app.FindRecordById("lists", list.Id)
	if err != nil {
		t.Fatalf("reload list: %v", err)
	}
	if !reloaded.GetBool("needs_full_sync") {
		t.Error("needs_full_sync = false after a list Update, want true")
	}
}

// failUpdates makes every update of the collection fail. It is bound after the
// fixture is seeded, so seeding is unaffected.
func failUpdates(app core.App, collection string, failure error) {
	app.OnRecordUpdate(collection).BindFunc(func(e *core.RecordEvent) error {
		return failure
	})
}

// A failing Update of an existing row is returned and adds no feed row.
func TestUpdateTrailFailureIsReturned(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, a.author.Id)
	failure := errors.New("update refused by test hook")
	failUpdates(app, "trails", failure)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI))
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act)

	if err == nil {
		t.Fatal("a failed Update of a stored trail returned nil")
	}
	if got := feedDedupCount(t, app, a.alice.Id, trail.Id); got != 0 {
		t.Errorf("feed rows for alice = %d, want 0", got)
	}
}

func TestUpdateListFailureIsReturned(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	list := authorGuardSeedList(t, app, feedDedupListIRI, a.author.Id)
	failure := errors.New("update refused by test hook")
	failUpdates(app, "lists", failure)

	act := authorGuardAsUpdate(feedDedupListCreate(feedDedupAuthorIRI, feedDedupListIRI))
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act)

	if err == nil {
		t.Fatal("a failed Update of a stored list returned nil")
	}
	if got := feedDedupCount(t, app, a.alice.Id, list.Id); got != 0 {
		t.Errorf("feed rows for alice = %d, want 0", got)
	}
}
