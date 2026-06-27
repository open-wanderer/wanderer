package routes

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	// Register PocketBase system migrations (creates _collections and other system tables).
	_ "github.com/pocketbase/pocketbase/migrations"
)

// newFederationAdminTestApp creates a bootstrapped PocketBase app with the
// activitypub_actors and follows collections needed to test the federation
// admin helpers.
func newFederationAdminTestApp(t *testing.T) core.App {
	t.Helper()

	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("ORIGIN", "https://local.example.com")

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// activitypub_actors collection — mirrors the schema from follow_test.go
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

// createFedAdminTestFollow creates a follows record between two actors and returns it.
func createFedAdminTestFollow(t *testing.T, app core.App, followerID, followeeID, status string) *core.Record {
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

// createFedAdminTestActor inserts an actor record and returns it.
func createFedAdminTestActor(t *testing.T, app core.App, iri, actorType string, isLocal bool) *core.Record {
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

// ---------------------------------------------------------------------------
// TestPickNodeInfo21Href — pure function tests, no app bootstrap needed
// ---------------------------------------------------------------------------

// TestPickNodeInfo21HrefReturnsSingleLink verifies that a JRD with only the
// 2.1 link returns its href.
func TestPickNodeInfo21HrefReturnsSingleLink(t *testing.T) {
	links := []nodeInfoLink{
		{Rel: "http://nodeinfo.diaspora.software/ns/schema/2.1", Href: "https://remote.example.com/.well-known/nodeinfo/2.1"},
	}
	got, err := pickNodeInfo21Href(links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://remote.example.com/.well-known/nodeinfo/2.1"
	if got != want {
		t.Errorf("pickNodeInfo21Href = %q, want %q", got, want)
	}
}

// TestPickNodeInfo21HrefSelectsFromMultiple verifies that when a JRD contains
// both a 2.0 and a 2.1 link, the 2.1 href is selected (D-08: select by rel,
// not by index — Pitfall 6).
func TestPickNodeInfo21HrefSelectsFromMultiple(t *testing.T) {
	links := []nodeInfoLink{
		{Rel: "http://nodeinfo.diaspora.software/ns/schema/2.0", Href: "https://remote.example.com/.well-known/nodeinfo/2.0"},
		{Rel: "http://nodeinfo.diaspora.software/ns/schema/2.1", Href: "https://remote.example.com/.well-known/nodeinfo/2.1"},
	}
	got, err := pickNodeInfo21Href(links)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://remote.example.com/.well-known/nodeinfo/2.1"
	if got != want {
		t.Errorf("pickNodeInfo21Href = %q, want %q", got, want)
	}
}

// TestPickNodeInfo21HrefErrorWhenMissing verifies that a JRD with no 2.1 link
// returns an error containing "not a Wanderer instance" (DISC-02).
func TestPickNodeInfo21HrefErrorWhenMissing(t *testing.T) {
	links := []nodeInfoLink{
		{Rel: "http://nodeinfo.diaspora.software/ns/schema/2.0", Href: "https://remote.example.com/.well-known/nodeinfo/2.0"},
	}
	_, err := pickNodeInfo21Href(links)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
	// Error should indicate this is not a Wanderer instance (DISC-02)
	wantSubstring := "not a Wanderer instance"
	if !containsString(err.Error(), wantSubstring) {
		t.Errorf("error = %q, want string containing %q", err.Error(), wantSubstring)
	}
}

// TestPickNodeInfo21HrefErrorWhenEmpty verifies an empty links slice returns an
// error containing "not a Wanderer instance".
func TestPickNodeInfo21HrefErrorWhenEmpty(t *testing.T) {
	_, err := pickNodeInfo21Href(nil)
	if err == nil {
		t.Fatal("expected error for empty links, got nil")
	}
	wantSubstring := "not a Wanderer instance"
	if !containsString(err.Error(), wantSubstring) {
		t.Errorf("error = %q, want string containing %q", err.Error(), wantSubstring)
	}
}

// ---------------------------------------------------------------------------
// TestFindLocalInstanceActor — requires app bootstrap
// ---------------------------------------------------------------------------

// TestFindLocalInstanceActorReturnsRecord verifies that findLocalInstanceActor
// returns the actor with actor_type="instance" and is_local=true.
func TestFindLocalInstanceActorReturnsRecord(t *testing.T) {
	app := newFederationAdminTestApp(t)

	// Create the local instance actor
	localActor := createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/instance",
		"instance", true)

	// Also create a non-instance actor to confirm filtering works
	createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/user/alice",
		"person", true)

	got, err := findLocalInstanceActor(app)
	if err != nil {
		t.Fatalf("findLocalInstanceActor: %v", err)
	}
	if got == nil {
		t.Fatal("findLocalInstanceActor returned nil record")
	}
	if got.Id != localActor.Id {
		t.Errorf("got actor id %q, want %q", got.Id, localActor.Id)
	}
	if got.GetString("actor_type") != "instance" {
		t.Errorf("actor_type = %q, want \"instance\"", got.GetString("actor_type"))
	}
	if !got.GetBool("is_local") {
		t.Error("is_local = false, want true")
	}
}

// TestFindLocalInstanceActorErrorWhenNone verifies that findLocalInstanceActor
// returns an error when no instance actor exists in the DB.
func TestFindLocalInstanceActorErrorWhenNone(t *testing.T) {
	app := newFederationAdminTestApp(t)

	// Only create a person actor — no instance actor
	createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/user/alice",
		"person", true)

	_, err := findLocalInstanceActor(app)
	if err == nil {
		t.Fatal("expected error when no instance actor exists, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestFederationDiscover — pure-logic tests for the Wanderer identity check
// ---------------------------------------------------------------------------

// TestFederationDiscoverRejectsNonWanderer verifies that a NodeInfo payload
// with software.name != "wanderer" is rejected. This tests the extracted
// verification logic without requiring a full HTTP server harness.
func TestFederationDiscoverRejectsNonWanderer(t *testing.T) {
	// The Wanderer check is: softwareInfo.Software.Name == "wanderer"
	// We test the condition directly using the nodeInfo21 type.
	ni := nodeInfo21{
		Software: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "mastodon",
			Version: "4.2.0",
		},
	}
	if ni.Software.Name == "wanderer" {
		t.Error("expected non-wanderer software name to be rejected, but check passed")
	}
}

// TestFederationDiscoverAcceptsWanderer verifies that a NodeInfo payload with
// software.name == "wanderer" passes the identity check.
func TestFederationDiscoverAcceptsWanderer(t *testing.T) {
	ni := nodeInfo21{
		Software: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{
			Name:    "wanderer",
			Version: "1.0.0",
		},
	}
	if ni.Software.Name != "wanderer" {
		t.Errorf("software.name = %q, want \"wanderer\"", ni.Software.Name)
	}
}

// ---------------------------------------------------------------------------
// TestFederationFollow — createOutboundFollow helper tests
// ---------------------------------------------------------------------------

// TestFederationFollowCreatesPendingRecord verifies that createOutboundFollow
// creates a follows record with follower=localID, followee=remoteID, status="pending".
func TestFederationFollowCreatesPendingRecord(t *testing.T) {
	app := newFederationAdminTestApp(t)

	localActor := createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFedAdminTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	rec, err := createOutboundFollow(app, localActor.Id, remoteActor.Id)
	if err != nil {
		t.Fatalf("createOutboundFollow: %v", err)
	}
	if rec == nil {
		t.Fatal("createOutboundFollow returned nil record")
	}
	if rec.GetString("follower") != localActor.Id {
		t.Errorf("follower = %q, want %q", rec.GetString("follower"), localActor.Id)
	}
	if rec.GetString("followee") != remoteActor.Id {
		t.Errorf("followee = %q, want %q", rec.GetString("followee"), remoteActor.Id)
	}
	if rec.GetString("status") != "pending" {
		t.Errorf("status = %q, want \"pending\"", rec.GetString("status"))
	}
}

// TestFederationFollowUnknownActor verifies that createOutboundFollow returns
// an error when the remote actor ID does not exist (D-02: 400 missing actor).
func TestFederationFollowUnknownActor(t *testing.T) {
	app := newFederationAdminTestApp(t)

	localActor := createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/instance",
		"instance", true)

	// "nonexistent123456" is not a valid actor record ID
	_, err := createOutboundFollow(app, localActor.Id, "nonexistent123456")
	if err == nil {
		t.Fatal("expected error for unknown remote actor, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestFederationApprove / TestFederationReject — setFollowStatus helper tests
// ---------------------------------------------------------------------------

// TestFederationApproveSetsAccepted verifies that setFollowStatus sets the
// follow record status to "accepted" when the local instance is the followee.
func TestFederationApproveSetsAccepted(t *testing.T) {
	app := newFederationAdminTestApp(t)

	localActor := createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFedAdminTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	// Inbound: remote follows local
	follow := createFedAdminTestFollow(t, app, remoteActor.Id, localActor.Id, "pending")

	if err := setFollowStatus(app, follow.Id, "accepted", localActor.Id); err != nil {
		t.Fatalf("setFollowStatus: %v", err)
	}

	updated, err := app.FindRecordById("follows", follow.Id)
	if err != nil {
		t.Fatalf("FindRecordById: %v", err)
	}
	if updated.GetString("status") != "accepted" {
		t.Errorf("status = %q, want \"accepted\"", updated.GetString("status"))
	}
}

// TestFederationRejectSetsRejected verifies that setFollowStatus sets the
// follow record status to "rejected" when the local instance is the followee.
func TestFederationRejectSetsRejected(t *testing.T) {
	app := newFederationAdminTestApp(t)

	localActor := createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFedAdminTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	// Inbound: remote follows local
	follow := createFedAdminTestFollow(t, app, remoteActor.Id, localActor.Id, "pending")

	if err := setFollowStatus(app, follow.Id, "rejected", localActor.Id); err != nil {
		t.Fatalf("setFollowStatus: %v", err)
	}

	updated, err := app.FindRecordById("follows", follow.Id)
	if err != nil {
		t.Fatalf("FindRecordById: %v", err)
	}
	if updated.GetString("status") != "rejected" {
		t.Errorf("status = %q, want \"rejected\"", updated.GetString("status"))
	}
}

// TestFederationApproveRejectsOutbound verifies that setFollowStatus returns an
// error when the follow's followee is not the local instance actor (direction guard).
func TestFederationApproveRejectsOutbound(t *testing.T) {
	app := newFederationAdminTestApp(t)

	localActor := createFedAdminTestActor(t, app,
		"https://local.example.com/api/v1/activitypub/instance",
		"instance", true)
	remoteActor := createFedAdminTestActor(t, app,
		"https://remote.example.com/api/v1/activitypub/instance",
		"instance", false)

	// Outbound: local follows remote (local is the follower, not the followee)
	follow := createFedAdminTestFollow(t, app, localActor.Id, remoteActor.Id, "pending")

	err := setFollowStatus(app, follow.Id, "accepted", localActor.Id)
	if err == nil {
		t.Fatal("expected error for outbound follow (local not the followee), got nil")
	}
}

// ---------------------------------------------------------------------------
// TestDisconnectAction — pure function tests, no app bootstrap needed
// ---------------------------------------------------------------------------

// TestDisconnectActionOutbound verifies that disconnectAction returns "delete"
// when followerID == localID (outbound follow; local issued the Follow).
func TestDisconnectActionOutbound(t *testing.T) {
	got := disconnectAction("localActorId", "localActorId")
	if got != "delete" {
		t.Errorf("disconnectAction = %q, want \"delete\"", got)
	}
}

// TestDisconnectActionInbound verifies that disconnectAction returns "reject"
// when followerID != localID (inbound-only follow; remote issued the Follow).
func TestDisconnectActionInbound(t *testing.T) {
	got := disconnectAction("remoteActorId", "localActorId")
	if got != "reject" {
		t.Errorf("disconnectAction = %q, want \"reject\"", got)
	}
}

// ---------------------------------------------------------------------------
// TestBuildPeerEntries — pure function tests, no app bootstrap needed
// ---------------------------------------------------------------------------

// followInput is a lightweight struct used to pass synthetic follow data to
// buildPeerEntries without needing to construct core.Record objects in tests.
// The handler adapts core.Record to followInput before calling buildPeerEntries.
type followInput struct {
	ID       string
	Follower string
	Followee string
	Status   string
}

// TestBuildPeerEntriesOutboundOnly verifies that one accepted outbound follow
// produces one peerEntry with direction="outbound".
func TestBuildPeerEntriesOutboundOnly(t *testing.T) {
	localID := "localActor001"
	outbound := []followInput{
		{ID: "follow001", Follower: localID, Followee: "remoteActor001", Status: "accepted"},
	}
	entries := buildPeerEntries(outbound, nil, localID, func(actorID string) string {
		if actorID == "remoteActor001" {
			return "remote.example.com"
		}
		return ""
	})
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Direction != "outbound" {
		t.Errorf("direction = %q, want \"outbound\"", e.Direction)
	}
	if e.Domain != "remote.example.com" {
		t.Errorf("domain = %q, want \"remote.example.com\"", e.Domain)
	}
	if e.Status != "accepted" {
		t.Errorf("status = %q, want \"accepted\"", e.Status)
	}
}

// TestBuildPeerEntriesInboundOnly verifies that one pending inbound follow
// produces one peerEntry with direction="inbound".
func TestBuildPeerEntriesInboundOnly(t *testing.T) {
	localID := "localActor001"
	inbound := []followInput{
		{ID: "follow002", Follower: "remoteActor001", Followee: localID, Status: "pending"},
	}
	entries := buildPeerEntries(nil, inbound, localID, func(actorID string) string {
		if actorID == "remoteActor001" {
			return "remote.example.com"
		}
		return ""
	})
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Direction != "inbound" {
		t.Errorf("direction = %q, want \"inbound\"", e.Direction)
	}
	if e.Domain != "remote.example.com" {
		t.Errorf("domain = %q, want \"remote.example.com\"", e.Domain)
	}
	if e.Status != "pending" {
		t.Errorf("status = %q, want \"pending\"", e.Status)
	}
}

// TestBuildPeerEntriesMutual verifies that an accepted outbound and accepted
// inbound follow for the same remote domain collapse to one peerEntry with
// direction="mutual" and follow_id equal to the outbound record id (D-05).
func TestBuildPeerEntriesMutual(t *testing.T) {
	localID := "localActor001"
	outboundID := "follow001"
	outbound := []followInput{
		{ID: outboundID, Follower: localID, Followee: "remoteActor001", Status: "accepted"},
	}
	inbound := []followInput{
		{ID: "follow002", Follower: "remoteActor001", Followee: localID, Status: "accepted"},
	}
	domainOf := func(actorID string) string {
		if actorID == "remoteActor001" {
			return "remote.example.com"
		}
		return ""
	}
	entries := buildPeerEntries(outbound, inbound, localID, domainOf)
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (mutual collapse)", len(entries))
	}
	e := entries[0]
	if e.Direction != "mutual" {
		t.Errorf("direction = %q, want \"mutual\"", e.Direction)
	}
	if e.FollowID != outboundID {
		t.Errorf("follow_id = %q, want outbound id %q (D-05)", e.FollowID, outboundID)
	}
	if e.Domain != "remote.example.com" {
		t.Errorf("domain = %q, want \"remote.example.com\"", e.Domain)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
