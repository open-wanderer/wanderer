package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// A federated Delete naming an actor is the only incoming activity that can
// remove a whole account's content, so the guard that keeps it to "an actor may
// only delete itself" is worth holding in place with tests.

func setupActorDeleteTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	t.Setenv("ORIGIN", "https://local.example")

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "is_local"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	return app
}

func newTestActor(t *testing.T, app *pbtests.TestApp, iri string, isLocal bool) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}

	record := core.NewRecord(collection)
	record.Set("iri", iri)
	record.Set("is_local", isLocal)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	return record
}

func newDeleteActivity(object string, claimedActor string) pub.Activity {
	activity := pub.Activity{Type: pub.DeleteType}
	activity.Object = pub.IRI(object)
	activity.Actor = pub.IRI(claimedActor)
	return activity
}

func assertStillExists(t *testing.T, app *pbtests.TestApp, actor *core.Record) {
	t.Helper()

	if _, err := app.FindRecordById("activitypub_actors", actor.Id); err != nil {
		t.Fatalf("actor %q was deleted but should have survived: %v", actor.GetString("iri"), err)
	}
}

func TestProcessDeleteActorActivity(t *testing.T) {
	// The attack this exists to stop: one instance asking us to delete an
	// account belonging to someone else.
	t.Run("RefusesDeletingAnotherAccount", func(t *testing.T) {
		app := setupActorDeleteTestApp(t)

		attacker := newTestActor(t, app, "https://evil.example/api/v1/activitypub/user/mallory", false)
		victim := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/alice", false)

		activity := newDeleteActivity(victim.GetString("iri"), attacker.GetString("iri"))

		if err := processDeleteActorActivity(app, attacker, activity); err == nil {
			t.Fatal("expected a signed Delete naming another account to be refused, got nil")
		}

		assertStillExists(t, app, victim)
		assertStillExists(t, app, attacker)
	})

	// The object matches the authenticated signer, but the Actor on the wire
	// names someone else. Refused rather than reconciled.
	t.Run("RefusesWhenWireActorDiffersFromSigner", func(t *testing.T) {
		app := setupActorDeleteTestApp(t)

		signer := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/alice", false)
		other := newTestActor(t, app, "https://evil.example/api/v1/activitypub/user/mallory", false)

		activity := newDeleteActivity(signer.GetString("iri"), other.GetString("iri"))

		if err := processDeleteActorActivity(app, signer, activity); err == nil {
			t.Fatal("expected a mismatched wire Actor to be refused, got nil")
		}

		assertStillExists(t, app, signer)
	})

	// No federated message may remove an account that lives on this instance.
	t.Run("RefusesLocalActor", func(t *testing.T) {
		app := setupActorDeleteTestApp(t)

		local := newTestActor(t, app, "https://local.example/api/v1/activitypub/user/alice", true)

		activity := newDeleteActivity(local.GetString("iri"), local.GetString("iri"))

		if err := processDeleteActorActivity(app, local, activity); err == nil {
			t.Fatal("expected deletion of a local actor to be refused, got nil")
		}

		assertStillExists(t, app, local)
	})

	// Even flagged remote, an IRI on our own origin is refused.
	t.Run("RefusesActorOnLocalOrigin", func(t *testing.T) {
		app := setupActorDeleteTestApp(t)

		impostor := newTestActor(t, app, "https://local.example/api/v1/activitypub/user/alice", false)

		activity := newDeleteActivity(impostor.GetString("iri"), impostor.GetString("iri"))

		if err := processDeleteActorActivity(app, impostor, activity); err == nil {
			t.Fatal("expected deletion of an actor on the local origin to be refused, got nil")
		}

		assertStillExists(t, app, impostor)
	})

	t.Run("RefusesEmptyObject", func(t *testing.T) {
		app := setupActorDeleteTestApp(t)

		signer := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/alice", false)

		activity := newDeleteActivity("", signer.GetString("iri"))

		if err := processDeleteActorActivity(app, signer, activity); err == nil {
			t.Fatal("expected an empty object to be refused, got nil")
		}

		assertStillExists(t, app, signer)
	})

	// The one case that is allowed to go through.
	t.Run("DeletesItself", func(t *testing.T) {
		app := setupActorDeleteTestApp(t)

		departing := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/alice", false)
		bystander := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/bob", false)

		activity := newDeleteActivity(departing.GetString("iri"), departing.GetString("iri"))

		if err := processDeleteActorActivity(app, departing, activity); err != nil {
			t.Fatalf("expected an actor to be able to delete itself, got %v", err)
		}

		if _, err := app.FindRecordById("activitypub_actors", departing.Id); err == nil {
			t.Fatal("expected the departing actor to be gone")
		}

		assertStillExists(t, app, bystander)
	})
}
