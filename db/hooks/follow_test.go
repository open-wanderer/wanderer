package hooks

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	// Register PocketBase system migrations (creates _collections and other system tables).
	_ "github.com/pocketbase/pocketbase/migrations"
)

// newFollowTestApp creates a bootstrapped PocketBase app with the activitypub_actors
// and follows collections for testing isInstanceFollow and related helpers.
func newFollowTestApp(t *testing.T) core.App {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// activitypub_actors collection
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
	// Status includes "rejected" per Plan 02-01 migration.
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
			{"hidden":false,"id":"select2063623452","maxSelect":1,"name":"status","presentable":false,"required":true,"system":false,"type":"select","values":["pending","accepted","rejected"]},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [],
		"system": false
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(followsJSON), false); err != nil {
		t.Fatalf("create follows collection: %v", err)
	}

	t.Cleanup(func() {
		app.ResetBootstrapState()
	})

	return app
}

// createFollowTestActor inserts an actor record into activitypub_actors and returns it.
func createFollowTestActor(t *testing.T, app core.App, iri, actorType string, isLocal bool) *core.Record {
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

// createFollowRecord creates a follows record between two actors.
func createFollowRecord(t *testing.T, app core.App, followerID, followeeID, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("follower", followerID)
	r.Set("followee", followeeID)
	r.Set("status", status)
	if err := app.Save(r); err != nil {
		t.Fatalf("save follow record: %v", err)
	}
	return r
}

// TestIsInstanceFollowTrueViaFollowee verifies that a follows record where the followee
// is the local instance actor (iri = ORIGIN + "/api/v1/activitypub/instance") returns true.
func TestIsInstanceFollowTrueViaFollowee(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newFollowTestApp(t)

	instanceActor := createFollowTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFollowTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"", false)

	follow := createFollowRecord(t, app, remoteActor.Id, instanceActor.Id, "pending")

	if !isInstanceFollow(app, follow) {
		t.Error("isInstanceFollow returned false — expected true when followee is the instance actor")
	}
}

// TestIsInstanceFollowTrueViaFollower verifies that a follows record where the follower
// is the local instance actor returns true.
func TestIsInstanceFollowTrueViaFollower(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newFollowTestApp(t)

	instanceActor := createFollowTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFollowTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"", false)

	follow := createFollowRecord(t, app, instanceActor.Id, remoteActor.Id, "pending")

	if !isInstanceFollow(app, follow) {
		t.Error("isInstanceFollow returned false — expected true when follower is the instance actor")
	}
}

// TestIsInstanceFollowFalse verifies that a follows record where neither follower nor followee
// is the instance actor (both are person actors) returns false.
func TestIsInstanceFollowFalse(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newFollowTestApp(t)

	alice := createFollowTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/user/alice",
		"person", true)
	bob := createFollowTestActor(t, app,
		"https://remote.example.com/users/bob",
		"person", false)

	follow := createFollowRecord(t, app, bob.Id, alice.Id, "accepted")

	if isInstanceFollow(app, follow) {
		t.Error("isInstanceFollow returned true — expected false when neither actor is the instance actor")
	}
}

// TestInstanceFollowActionNoOp verifies that instanceFollowAction returns "" when
// oldStatus == newStatus (no transition, no delivery action).
func TestInstanceFollowActionNoOp(t *testing.T) {
	cases := []struct {
		old string
		new string
	}{
		{"pending", "pending"},
		{"accepted", "accepted"},
		{"rejected", "rejected"},
	}
	for _, c := range cases {
		got := instanceFollowAction(c.old, c.new)
		if got != "" {
			t.Errorf("instanceFollowAction(%q, %q) = %q, want \"\"", c.old, c.new, got)
		}
	}
}

// TestInstanceFollowActionAccept verifies that instanceFollowAction returns "accept"
// when transitioning to "accepted" from a different status.
func TestInstanceFollowActionAccept(t *testing.T) {
	got := instanceFollowAction("pending", "accepted")
	if got != "accept" {
		t.Errorf("instanceFollowAction(\"pending\", \"accepted\") = %q, want \"accept\"", got)
	}
}

// TestInstanceFollowActionReject verifies that instanceFollowAction returns "reject"
// when transitioning to "rejected" from a different status.
func TestInstanceFollowActionReject(t *testing.T) {
	got := instanceFollowAction("pending", "rejected")
	if got != "reject" {
		t.Errorf("instanceFollowAction(\"pending\", \"rejected\") = %q, want \"reject\"", got)
	}
}

// TestIsOutboundInstanceFollowTrueWhenInstanceIsFollower verifies that a follows record
// where the FOLLOWER actor's iri equals the local instance IRI returns true.
// This is the admin-initiated outbound follow case (InstanceFollowCreateHandler must fire).
func TestIsOutboundInstanceFollowTrueWhenInstanceIsFollower(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newFollowTestApp(t)

	instanceActor := createFollowTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFollowTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"", false)

	// Instance is the follower (outbound follow initiated by admin).
	follow := createFollowRecord(t, app, instanceActor.Id, remoteActor.Id, "pending")

	if !isOutboundInstanceFollow(app, follow) {
		t.Error("isOutboundInstanceFollow returned false — expected true when instance actor is the follower")
	}
}

// TestIsOutboundInstanceFollowFalseWhenInstanceIsFollowee verifies that a follows record
// where the FOLLOWEE actor's iri equals the local instance IRI (the shape saved by
// ProcessFollowActivity for inbound follows) returns false.
// This is the CR-03 regression: InstanceFollowCreateHandler must NOT fire here.
func TestIsOutboundInstanceFollowFalseWhenInstanceIsFollowee(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newFollowTestApp(t)

	instanceActor := createFollowTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFollowTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"", false)

	// Remote is the follower, instance is the followee (inbound follow saved by ProcessFollowActivity).
	follow := createFollowRecord(t, app, remoteActor.Id, instanceActor.Id, "pending")

	if isOutboundInstanceFollow(app, follow) {
		t.Error("isOutboundInstanceFollow returned true — expected false when instance actor is the followee (inbound follow, CR-03 regression)")
	}
}

// TestIsOutboundInstanceFollowFalseWhenNeitherIsInstance verifies that a follows record
// between two non-instance actors (e.g., two person actors) returns false.
func TestIsOutboundInstanceFollowFalseWhenNeitherIsInstance(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newFollowTestApp(t)

	alice := createFollowTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/user/alice",
		"person", true)
	bob := createFollowTestActor(t, app,
		"https://remote.example.com/users/bob",
		"person", false)

	// Neither actor is the local instance actor.
	follow := createFollowRecord(t, app, bob.Id, alice.Id, "accepted")

	if isOutboundInstanceFollow(app, follow) {
		t.Error("isOutboundInstanceFollow returned true — expected false when neither actor is the instance actor")
	}
}
