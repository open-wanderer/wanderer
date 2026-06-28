package routes

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	// Register PocketBase system migrations (creates _collections and other system tables).
	_ "github.com/pocketbase/pocketbase/migrations"
)

// newNodeInfoTestApp creates a bootstrapped PocketBase app with minimal
// collections needed to test the NodeInfo payload builders:
// activitypub_actors (for the local-trail JOIN), trails, and users.
func newNodeInfoTestApp(t *testing.T) core.App {
	t.Helper()

	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	collectionsJSON := `[
		{
			"id": "actors_coll_001",
			"name": "activitypub_actors",
			"type": "base",
			"system": false,
			"listRule": null, "viewRule": null, "createRule": null, "updateRule": null, "deleteRule": null,
			"fields": [
				{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
				{"hidden":false,"id":"bool_islocal_001","name":"is_local","presentable":false,"required":false,"system":false,"type":"bool"}
			],
			"indexes": []
		},
		{
			"id": "trails_coll_001",
			"name": "trails",
			"type": "base",
			"system": false,
			"listRule": null, "viewRule": null, "createRule": null, "updateRule": null, "deleteRule": null,
			"fields": [
				{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
				{"hidden":false,"id":"bool_public_001","name":"public","presentable":false,"required":false,"system":false,"type":"bool"},
				{"cascadeDelete":false,"collectionId":"actors_coll_001","hidden":false,"id":"rel_author_001","maxSelect":1,"minSelect":0,"name":"author","presentable":false,"required":false,"system":false,"type":"relation"}
			],
			"indexes": []
		}
	]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(collectionsJSON), false); err != nil {
		t.Fatalf("create collections: %v", err)
	}

	// Note: PocketBase bootstrap creates a "users" auth collection automatically.
	// We use it directly; no need to import it again.

	return app
}

// seedLocalActor inserts a local activitypub_actors record and returns its ID.
func seedLocalActor(t *testing.T, app core.App) string {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatalf("find activitypub_actors: %v", err)
	}
	r := core.NewRecord(coll)
	r.Set("is_local", true)
	if err := app.Save(r); err != nil {
		t.Fatalf("save local actor: %v", err)
	}
	return r.Id
}

// TestNodeInfoDiscoveryRel verifies the discovery document links[0].rel value.
func TestNodeInfoDiscoveryRel(t *testing.T) {
	doc := buildNodeInfoDiscovery("https://w.example.com")
	links, ok := doc["links"].([]map[string]string)
	if !ok {
		t.Fatalf("expected links to be []map[string]string, got %T", doc["links"])
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link entry, got %d", len(links))
	}
	const wantRel = "http://nodeinfo.diaspora.software/ns/schema/2.1"
	if links[0]["rel"] != wantRel {
		t.Errorf("links[0].rel = %q, want %q", links[0]["rel"], wantRel)
	}
}

// TestNodeInfoDiscoveryHref verifies the discovery document links[0].href value.
func TestNodeInfoDiscoveryHref(t *testing.T) {
	const origin = "https://w.example.com"
	doc := buildNodeInfoDiscovery(origin)
	links, ok := doc["links"].([]map[string]string)
	if !ok {
		t.Fatalf("expected links to be []map[string]string, got %T", doc["links"])
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link entry, got %d", len(links))
	}
	wantHref := origin + "/.well-known/nodeinfo/2.1"
	if links[0]["href"] != wantHref {
		t.Errorf("links[0].href = %q, want %q", links[0]["href"], wantHref)
	}
}

// TestNodeInfo21SoftwareName verifies software.name == "wanderer".
func TestNodeInfo21SoftwareName(t *testing.T) {
	app := newNodeInfoTestApp(t)
	defer app.ResetBootstrapState()

	payload, err := buildNodeInfo21(app)
	if err != nil {
		t.Fatalf("buildNodeInfo21: %v", err)
	}
	sw, ok := payload["software"].(map[string]any)
	if !ok {
		t.Fatalf("expected software to be map[string]any, got %T", payload["software"])
	}
	if sw["name"] != "wanderer" {
		t.Errorf("software.name = %q, want \"wanderer\"", sw["name"])
	}
	if _, exists := sw["version"]; !exists {
		t.Error("software.version key is missing")
	}
}

// TestNodeInfo21LocalPostsExcludesPrivate verifies that private trails are excluded from
// localPosts count (D-02 privacy hard constraint).
func TestNodeInfo21LocalPostsExcludesPrivate(t *testing.T) {
	app := newNodeInfoTestApp(t)
	defer app.ResetBootstrapState()

	actorID := seedLocalActor(t, app)

	// Seed 2 public + 1 private trail, all owned by the local actor.
	trailsColl, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("find trails collection: %v", err)
	}

	for i := 0; i < 2; i++ {
		r := core.NewRecord(trailsColl)
		r.Set("public", true)
		r.Set("author", actorID)
		if err := app.Save(r); err != nil {
			t.Fatalf("save public trail: %v", err)
		}
	}
	// private trail — must NOT be counted
	r := core.NewRecord(trailsColl)
	r.Set("public", false)
	r.Set("author", actorID)
	if err := app.Save(r); err != nil {
		t.Fatalf("save private trail: %v", err)
	}

	payload, err := buildNodeInfo21(app)
	if err != nil {
		t.Fatalf("buildNodeInfo21: %v", err)
	}
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected usage to be map[string]any, got %T", payload["usage"])
	}
	localPosts, ok := usage["localPosts"].(int64)
	if !ok {
		t.Fatalf("expected localPosts to be int64, got %T", usage["localPosts"])
	}
	if localPosts != 2 {
		t.Errorf("localPosts = %d, want 2 (private trail must be excluded)", localPosts)
	}
}

// TestNodeInfo21UsersTotal verifies usage.users.total equals the count of all users.
func TestNodeInfo21UsersTotal(t *testing.T) {
	app := newNodeInfoTestApp(t)
	defer app.ResetBootstrapState()

	usersColl, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}

	for i := 0; i < 3; i++ {
		r := core.NewRecord(usersColl)
		r.Set("email", "user"+string(rune('a'+i))+"@example.com")
		r.Set("password", "testpassword123")
		r.Set("passwordConfirm", "testpassword123")
		if err := app.Save(r); err != nil {
			t.Fatalf("save user %d: %v", i, err)
		}
	}

	payload, err := buildNodeInfo21(app)
	if err != nil {
		t.Fatalf("buildNodeInfo21: %v", err)
	}
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected usage to be map[string]any, got %T", payload["usage"])
	}
	users, ok := usage["users"].(map[string]any)
	if !ok {
		t.Fatalf("expected usage.users to be map[string]any, got %T", usage["users"])
	}
	total, ok := users["total"].(int64)
	if !ok {
		t.Fatalf("expected users.total to be int64, got %T", users["total"])
	}
	if total != 3 {
		t.Errorf("users.total = %d, want 3", total)
	}
}

// TestNodeInfo21RequiredKeys verifies all 7 top-level keys are present.
func TestNodeInfo21RequiredKeys(t *testing.T) {
	app := newNodeInfoTestApp(t)
	defer app.ResetBootstrapState()

	payload, err := buildNodeInfo21(app)
	if err != nil {
		t.Fatalf("buildNodeInfo21: %v", err)
	}

	requiredKeys := []string{"version", "software", "protocols", "services", "openRegistrations", "usage", "metadata"}
	for _, key := range requiredKeys {
		if _, exists := payload[key]; !exists {
			t.Errorf("missing required top-level key: %q", key)
		}
	}

	if payload["version"] != "2.1" {
		t.Errorf("version = %q, want \"2.1\"", payload["version"])
	}
}
