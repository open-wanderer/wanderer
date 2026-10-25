package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// TestProcessDeleteTrailActivityOwnershipCheck verifies that a Delete from an
// actor that is not the trail author is rejected and the trail is kept.
func TestProcessDeleteTrailActivityOwnershipCheck(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)

	// Two distinct remote actors
	actorA := createTestActor(t, app, "https://remote.example.com/users/a", "person", false)
	actorB := createTestActor(t, app, "https://remote.example.com/users/b", "person", false)

	// Seed a trail authored by actorA
	const trailIRI = "https://remote.example.com/api/v1/trail/xyz"
	trailsCol, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("find trails collection: %v", err)
	}
	trail := core.NewRecord(trailsCol)
	trail.Set("iri", trailIRI)
	trail.Set("author", actorA.Id)
	trail.Set("public", true)
	if err := app.Save(trail); err != nil {
		t.Fatalf("save trail record: %v", err)
	}

	// Build a Delete activity targeting the trail IRI
	deleteActivity := pub.DeleteNew(
		pub.IRI("https://remote.example.com/api/v1/activitypub/activity/del1"),
		pub.IRI(trailIRI),
	)

	// actorB attempts to delete actorA's trail — must be rejected
	err = processDeleteTrailActivity(app, actorB, *deleteActivity)
	if err == nil {
		t.Fatal("expected non-nil error for unauthorized delete, got nil")
	}

	// The trail record must still exist
	_, lookupErr := app.FindFirstRecordByData("trails", "iri", trailIRI)
	if lookupErr != nil {
		t.Errorf("trail was deleted despite unauthorized actor: lookup error = %v", lookupErr)
	}
}

// TestProcessDeleteTrailActivitySucceeds verifies that a Delete from the trail's
// author removes the trail.
func TestProcessDeleteTrailActivitySucceeds(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)
	addFeedCollection(t, app)

	// Single remote actor — both author and deleter
	actorA := createTestActor(t, app, "https://remote.example.com/users/a", "person", false)

	// Seed a trail authored by actorA
	const trailIRI = "https://remote.example.com/api/v1/trail/xyz"
	trailsCol, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("find trails collection: %v", err)
	}
	trail := core.NewRecord(trailsCol)
	trail.Set("iri", trailIRI)
	trail.Set("author", actorA.Id)
	trail.Set("public", true)
	if err := app.Save(trail); err != nil {
		t.Fatalf("save trail record: %v", err)
	}

	// Build a Delete activity targeting the trail IRI
	deleteActivity := pub.DeleteNew(
		pub.IRI("https://remote.example.com/api/v1/activitypub/activity/del1"),
		pub.IRI(trailIRI),
	)

	// Seed a feed entry for the trail so DeleteFromFeed succeeds (it calls
	// FindFirstRecordByData("feed","item",trailId) and propagates sql.ErrNoRows).
	feedCol, err := app.FindCollectionByNameOrId("feed")
	if err != nil {
		t.Fatalf("find feed collection: %v", err)
	}
	feedRecord := core.NewRecord(feedCol)
	feedRecord.Set("item", trail.Id)
	feedRecord.Set("actor", actorA.Id)
	feedRecord.Set("author", actorA.Id)
	feedRecord.Set("type", "trail")
	if err := app.Save(feedRecord); err != nil {
		t.Fatalf("save feed record: %v", err)
	}

	// actorA deletes their own trail — must succeed
	err = processDeleteTrailActivity(app, actorA, *deleteActivity)
	if err != nil {
		t.Fatalf("expected nil error for authorized delete, got: %v", err)
	}

	// The trail record must no longer exist
	_, lookupErr := app.FindFirstRecordByData("trails", "iri", trailIRI)
	if lookupErr == nil {
		t.Error("expected trail to be deleted, but record still exists")
	}
}
