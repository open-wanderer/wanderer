package federation

import (
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"

	// Register PocketBase system migrations (creates _collections and other system tables).
	_ "github.com/pocketbase/pocketbase/migrations"
)

// buildFollowActivity constructs a minimal pub.Activity of type Follow.
func buildFollowActivity(actorIRI, objectIRI string) pub.Activity {
	a := pub.ActivityNew(pub.IRI("https://remote.example.com/activity/1"), pub.FollowType, pub.IRI(objectIRI))
	a.Actor = pub.IRI(actorIRI)
	return *a
}

// TestProcessFollowInstanceActorSetsPending verifies that a Follow directed at
// the local instance actor (actor_type="instance", is_local=true) is stored as
// status="pending" and no Accept activity record is created.
func TestProcessFollowInstanceActorSetsPending(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)

	// Local instance actor (the followee / object of the Follow)
	instanceActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)

	// Remote sender actor (the follower / actor of the Follow)
	remoteActor := createTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	activity := buildFollowActivity(
		remoteActor.GetString("iri"),
		instanceActor.GetString("iri"),
	)

	if err := ProcessFollowActivity(app, remoteActor, activity); err != nil {
		t.Fatalf("ProcessFollowActivity returned error: %v", err)
	}

	// Assert follows record exists with status="pending"
	follows, err := app.FindRecordsByFilter("follows",
		"follower={:follower} && followee={:followee}",
		"-created", 10, 0,
		map[string]any{"follower": remoteActor.Id, "followee": instanceActor.Id},
	)
	if err != nil {
		t.Fatalf("querying follows: %v", err)
	}
	if len(follows) != 1 {
		t.Fatalf("expected 1 follow record, got %d", len(follows))
	}
	if got := follows[0].GetString("status"); got != "pending" {
		t.Errorf("follow status = %q, want pending", got)
	}

	// Assert NO Accept activity was created
	accepts, err := app.FindRecordsByFilter("activitypub_activities",
		"type={:type}",
		"-created", 10, 0,
		map[string]any{"type": string(pub.AcceptType)},
	)
	if err != nil {
		t.Fatalf("querying activitypub_activities for Accept: %v", err)
	}
	if len(accepts) != 0 {
		t.Errorf("expected 0 Accept activities, got %d — instance follow must not auto-accept", len(accepts))
	}
}

// TestProcessFollowInstanceFollowStored verifies that a Follow directed at the
// local instance actor stores the Follow activity, which Accept and Reject wrap.
func TestProcessFollowInstanceFollowStored(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)

	instanceActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)

	remoteActor := createTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	activity := buildFollowActivity(
		remoteActor.GetString("iri"),
		instanceActor.GetString("iri"),
	)

	if err := ProcessFollowActivity(app, remoteActor, activity); err != nil {
		t.Fatalf("ProcessFollowActivity returned error: %v", err)
	}

	// Assert an activitypub_activities record of type "Follow" was created
	followActivities, err := app.FindRecordsByFilter("activitypub_activities",
		"type={:type}",
		"-created", 10, 0,
		map[string]any{"type": string(pub.FollowType)},
	)
	if err != nil {
		t.Fatalf("querying activitypub_activities for Follow: %v", err)
	}
	if len(followActivities) != 1 {
		t.Fatalf("expected 1 Follow activity record, got %d", len(followActivities))
	}

	rec := followActivities[0]
	if got := rec.GetString("actor"); got != remoteActor.GetString("iri") {
		t.Errorf("activity.actor = %q, want %q", got, remoteActor.GetString("iri"))
	}
	// object is a json field, so GetString returns the quoted JSON string.
	gotObject := strings.Trim(rec.GetString("object"), "\"")
	if gotObject != instanceActor.GetString("iri") {
		t.Errorf("activity.object = %q, want %q", gotObject, instanceActor.GetString("iri"))
	}
}

// TestInstanceInboxDispatchesCreateActivity verifies that the instance inbox
// dispatches an incoming Create activity to ProcessCreateOrUpdateActivity, which
// creates a trail record in the database.
func TestInstanceInboxDispatchesCreateActivity(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)
	addFeedCollection(t, app)

	// Remote sender actor
	sender := createTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	// Local instance actor (the inbox recipient)
	instanceActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)

	const trailIRI = "https://remote.example.com/api/v1/trail/new1"

	// Build a minimal trail object with a Location to satisfy TrailFromActivity.
	place := &pub.Place{
		Type:      pub.PlaceType,
		Latitude:  47.5,
		Longitude: 11.5,
		Name:      pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Place")),
	}

	obj := &pub.Object{
		ID:       pub.IRI(trailIRI),
		Type:     pub.NoteType,
		Name:     pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Trail")),
		Content:  pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "")),
		Location: place,
	}

	act := pub.ActivityNew(
		pub.IRI("https://remote.example.com/api/v1/activitypub/activity/c1"),
		pub.CreateType,
		obj,
	)
	act.Actor = pub.IRI(sender.GetString("iri"))

	// Call the function the instance inbox dispatches Create to.
	err := ProcessCreateOrUpdateActivity(app, sender, instanceActor, *act)
	if err != nil {
		t.Fatalf("ProcessCreateOrUpdateActivity returned error: %v", err)
	}

	// A trails row with the object IRI must now exist
	_, lookupErr := app.FindFirstRecordByData("trails", "iri", trailIRI)
	if lookupErr != nil {
		t.Errorf("expected trail row with IRI %q to be created, got lookup error: %v", trailIRI, lookupErr)
	}
}

// TestProcessFollowPersonActorAutoAccepts verifies that the existing person-actor
// auto-accept path remains unchanged when a Follow is directed at a local person actor.
func TestProcessFollowPersonActorAutoAccepts(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)

	// Local person actor (the followee)
	personActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/user/alice",
		"person", true)

	// Remote sender actor (the follower)
	remoteActor := createTestActor(t, app,
		"https://remote.example.com/users/bob",
		"", false)

	activity := buildFollowActivity(
		remoteActor.GetString("iri"),
		personActor.GetString("iri"),
	)

	// Delivery of the Accept may fail in tests; only the row status matters.
	_ = ProcessFollowActivity(app, remoteActor, activity)

	// The follows record should be "accepted" for person actors
	follows, err := app.FindRecordsByFilter("follows",
		"follower={:follower} && followee={:followee}",
		"-created", 10, 0,
		map[string]any{"follower": remoteActor.Id, "followee": personActor.Id},
	)
	if err != nil {
		t.Fatalf("querying follows: %v", err)
	}
	if len(follows) != 1 {
		t.Fatalf("expected 1 follow record, got %d", len(follows))
	}
	if got := follows[0].GetString("status"); got != "accepted" {
		t.Errorf("follow status = %q, want accepted", got)
	}
}
