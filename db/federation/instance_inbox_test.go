package federation

import (
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"

	// Register PocketBase system migrations (creates _collections and other system tables).
	_ "github.com/pocketbase/pocketbase/migrations"
)

// newInboxTestApp creates a bootstrapped PocketBase app with the activitypub_actors,
// follows, and activitypub_activities collections created programmatically.
// This avoids running the full migration chain (which requires Meilisearch).
func newInboxTestApp(t *testing.T) core.App {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// activitypub_actors collection — same JSON as in instance_test.go
	actorsJSON := `[{
		"id": "pbc_1295301207",
		"name": "activitypub_actors",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text4166911607","max":0,"min":0,"name":"username","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text4002953752","max":0,"min":0,"name":"preferred_username","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text2812878347","max":0,"min":0,"name":"domain","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text3458754147","max":0,"min":0,"name":"summary","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text1727648867","max":0,"min":0,"name":"public_key","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":true,"id":"text4160324774","max":0,"min":0,"name":"private_key","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"select_actor_type_001","maxSelect":1,"name":"actor_type","presentable":false,"required":false,"system":false,"type":"select","values":["person","instance"]},
			{"hidden":false,"id":"bool2193750486","name":"is_local","presentable":false,"required":false,"system":false,"type":"bool"},
			{"exceptDomains":null,"hidden":false,"id":"url126331327","name":"iri","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"},
			{"exceptDomains":null,"hidden":false,"id":"url2115105593","name":"inbox","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"},
			{"exceptDomains":null,"hidden":false,"id":"url1793578352","name":"outbox","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"},
			{"hidden":false,"id":"date2062531289","max":"","min":"","name":"last_fetched","presentable":false,"required":false,"system":false,"type":"date"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`idx_rpT7QJwWTm`" + ` ON ` + "`activitypub_actors`" + ` (` + "`iri`" + `)"
		]
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(actorsJSON), false); err != nil {
		t.Fatalf("create activitypub_actors collection: %v", err)
	}

	// follows collection — follower/followee relate to activitypub_actors (pbc_1295301207)
	followsJSON := `[{
		"id": "8obn1ukumze565i",
		"name": "follows",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"cascadeDelete":true,"collectionId":"pbc_1295301207","hidden":false,"id":"relation3117812038","maxSelect":1,"minSelect":0,"name":"follower","presentable":false,"required":true,"system":false,"type":"relation"},
			{"cascadeDelete":true,"collectionId":"pbc_1295301207","hidden":false,"id":"relation973442177","maxSelect":1,"minSelect":0,"name":"followee","presentable":false,"required":true,"system":false,"type":"relation"},
			{"hidden":false,"id":"select2063623452","maxSelect":1,"name":"status","presentable":false,"required":true,"system":false,"type":"select","values":["pending","accepted"]},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [],
		"system": false
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(followsJSON), false); err != nil {
		t.Fatalf("create follows collection: %v", err)
	}

	// activitypub_activities collection
	activitiesJSON := `[{
		"id": "pbc_3752774184",
		"name": "activitypub_activities",
		"type": "base",
		"system": false,
		"listRule": "",
		"viewRule": "",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"exceptDomains":[],"hidden":false,"id":"url2434853685","name":"iri","onlyDomains":[],"presentable":false,"required":true,"system":false,"type":"url"},
			{"autogeneratePattern":"","hidden":false,"id":"text2363381545","max":0,"min":0,"name":"type","pattern":"","presentable":false,"primaryKey":false,"required":true,"system":false,"type":"text"},
			{"hidden":false,"id":"json3616002756","maxSize":0,"name":"to","presentable":false,"required":false,"system":false,"type":"json"},
			{"hidden":false,"id":"json3685882489","maxSize":0,"name":"cc","presentable":false,"required":false,"system":false,"type":"json"},
			{"hidden":false,"id":"json2893285722","maxSize":0,"name":"object","presentable":false,"required":true,"system":false,"type":"json"},
			{"exceptDomains":[],"hidden":false,"id":"url1148540665","name":"actor","onlyDomains":[],"presentable":false,"required":true,"system":false,"type":"url"},
			{"hidden":false,"id":"date1748787223","max":"","min":"","name":"published","presentable":false,"required":true,"system":false,"type":"date"},
			{"autogeneratePattern":"","hidden":false,"id":"text1653163849","max":15,"min":15,"name":"relation","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [],
		"system": false
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(activitiesJSON), false); err != nil {
		t.Fatalf("create activitypub_activities collection: %v", err)
	}

	t.Cleanup(func() {
		app.ResetBootstrapState()
	})

	return app
}

// createTestActor inserts an actor record into activitypub_actors and returns it.
func createTestActor(t *testing.T, app core.App, iri, actorType string, isLocal bool) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatalf("find activitypub_actors: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("actor_type", actorType)
	r.Set("is_local", isLocal)
	r.Set("preferred_username", "testactor")
	r.Set("domain", "example.com")
	r.Set("inbox", iri+"/inbox")
	r.Set("outbox", iri+"/outbox")
	r.Set("last_fetched", time.Now())
	if err := app.Save(r); err != nil {
		t.Fatalf("save actor %s: %v", iri, err)
	}
	return r
}

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
		"", false)

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
// local instance actor persists an activitypub_activities record of type "Follow"
// so Plan 03 can reconstruct the original Follow for Accept/Reject.
func TestProcessFollowInstanceFollowStored(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)

	instanceActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)

	remoteActor := createTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"", false)

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
	if got := rec.GetString("object"); got != instanceActor.GetString("iri") {
		t.Errorf("activity.object = %q, want %q", got, instanceActor.GetString("iri"))
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

	// ProcessFollowActivity for a person actor will attempt to PostActivity
	// (fire-and-forget goroutine) which may fail in the test environment (no real HTTP).
	// We ignore the error here and only assert the follows record status.
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
