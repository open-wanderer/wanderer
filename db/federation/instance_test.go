package federation

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	// Register PocketBase system migrations (creates _collections and other system tables).
	// These are required for Bootstrap() to succeed in tests.
	_ "github.com/pocketbase/pocketbase/migrations"
)

// newTestApp creates a bootstrapped PocketBase app with an activitypub_actors
// collection created programmatically. This avoids running the full migration
// chain (which requires an external Meilisearch service) while still providing
// a real database for testing InitInstanceActor.
//
// The collection schema matches the final state after all activitypub_actors
// migrations (including the actor_type field added by this phase).
func newTestApp(t *testing.T) core.App {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Create activitypub_actors collection with all fields needed by InitInstanceActor.
	// Uses the stable collection ID pbc_1295301207 to match production schema.
	jsonData := `[{
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
			{"autogeneratePattern":"","hidden":false,"id":"text_actor_type_001","max":0,"min":0,"name":"actor_type","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
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

	if err := app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false); err != nil {
		t.Fatalf("create activitypub_actors collection: %v", err)
	}

	t.Cleanup(func() {
		app.ResetBootstrapState()
	})

	return app
}

func TestInitInstanceActorCreatesApplicationActor(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newTestApp(t)

	if err := InitInstanceActor(app); err != nil {
		t.Fatalf("InitInstanceActor returned error: %v", err)
	}

	// Verify exactly one Application actor was created
	record, err := app.FindFirstRecordByData("activitypub_actors", "iri", "https://trails.example.com/api/v1/activitypub/instance")
	if err != nil {
		t.Fatalf("actor record not found: %v", err)
	}

	if got := record.GetString("actor_type"); got != "Application" {
		t.Errorf("actor_type = %q, want Application", got)
	}

	if got := record.GetString("preferred_username"); got != "instance" {
		t.Errorf("preferred_username = %q, want instance", got)
	}

	if got := record.GetBool("is_local"); !got {
		t.Errorf("is_local = false, want true")
	}

	if got := record.GetString("iri"); got != "https://trails.example.com/api/v1/activitypub/instance" {
		t.Errorf("iri = %q, want https://trails.example.com/api/v1/activitypub/instance", got)
	}

	pubKey := record.GetString("public_key")
	if pubKey == "" {
		t.Error("public_key is empty")
	}
	if !strings.HasPrefix(pubKey, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("public_key does not start with PEM header, got: %q", pubKey[:minLen(50, len(pubKey))])
	}
}

func TestInitInstanceActorIsIdempotent(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newTestApp(t)

	if err := InitInstanceActor(app); err != nil {
		t.Fatalf("first InitInstanceActor call returned error: %v", err)
	}

	// Record public key after first call
	record1, err := app.FindFirstRecordByData("activitypub_actors", "iri", "https://trails.example.com/api/v1/activitypub/instance")
	if err != nil {
		t.Fatalf("actor record not found after first call: %v", err)
	}
	pubKey1 := record1.GetString("public_key")

	// Second call must be idempotent
	if err := InitInstanceActor(app); err != nil {
		t.Fatalf("second InitInstanceActor call returned error: %v", err)
	}

	// Must still be exactly one record
	records, err := app.FindRecordsByFilter("activitypub_actors", "iri={:iri}", "-created", 10, 0, map[string]any{
		"iri": "https://trails.example.com/api/v1/activitypub/instance",
	})
	if err != nil {
		t.Fatalf("listing records: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected exactly 1 instance actor record, got %d", len(records))
	}

	// Public key must be identical (keypair never regenerated)
	record2, err := app.FindFirstRecordByData("activitypub_actors", "iri", "https://trails.example.com/api/v1/activitypub/instance")
	if err != nil {
		t.Fatalf("actor record not found after second call: %v", err)
	}
	pubKey2 := record2.GetString("public_key")

	if pubKey1 != pubKey2 {
		t.Error("public_key changed after second InitInstanceActor call — keypair was regenerated")
	}
}

func TestInitInstanceActorNameFromOrigin(t *testing.T) {
	t.Setenv("ORIGIN", "https://www.trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newTestApp(t)

	if err := InitInstanceActor(app); err != nil {
		t.Fatalf("InitInstanceActor returned error: %v", err)
	}

	record, err := app.FindFirstRecordByData("activitypub_actors", "iri", "https://www.trails.example.com/api/v1/activitypub/instance")
	if err != nil {
		t.Fatalf("actor record not found: %v", err)
	}

	// www. must be stripped from the hostname
	if got := record.GetString("username"); got != "Wanderer at trails.example.com" {
		t.Errorf("username = %q, want %q", got, "Wanderer at trails.example.com")
	}
}

func TestInstanceActorJSONShape(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newTestApp(t)

	if err := InitInstanceActor(app); err != nil {
		t.Fatalf("InitInstanceActor returned error: %v", err)
	}

	iri := "https://trails.example.com/api/v1/activitypub/instance"
	record, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
	if err != nil {
		t.Fatalf("actor record not found: %v", err)
	}

	actorJSON := buildInstanceActorJSON(record, iri)

	// @context
	if _, ok := actorJSON["@context"]; !ok {
		t.Error("missing @context")
	}

	// id
	if got, _ := actorJSON["id"].(string); got != iri {
		t.Errorf("id = %q, want %q", got, iri)
	}

	// type
	if got, _ := actorJSON["type"].(string); got != "Application" {
		t.Errorf("type = %q, want Application", got)
	}

	// preferredUsername
	if got, _ := actorJSON["preferredUsername"].(string); got != "instance" {
		t.Errorf("preferredUsername = %q, want instance", got)
	}

	// name must be set
	if got, _ := actorJSON["name"].(string); got == "" {
		t.Error("name is empty")
	}

	// inbox
	if got, _ := actorJSON["inbox"].(string); got == "" {
		t.Error("inbox is empty")
	}

	// outbox
	if got, _ := actorJSON["outbox"].(string); got == "" {
		t.Error("outbox is empty")
	}

	// manuallyApprovesFollowers must be true bool
	maf, ok := actorJSON["manuallyApprovesFollowers"]
	if !ok {
		t.Error("missing manuallyApprovesFollowers")
	} else if got, ok := maf.(bool); !ok || !got {
		t.Errorf("manuallyApprovesFollowers = %v, want true", maf)
	}

	// publicKey
	pkRaw, ok := actorJSON["publicKey"]
	if !ok {
		t.Fatal("missing publicKey")
	}
	pk, ok := pkRaw.(map[string]any)
	if !ok {
		t.Fatalf("publicKey is not map[string]any, got %T", pkRaw)
	}

	// publicKey.id must end with #main-key
	pkID, _ := pk["id"].(string)
	if !strings.HasSuffix(pkID, "#main-key") {
		t.Errorf("publicKey.id = %q, want suffix #main-key", pkID)
	}

	// publicKey.owner
	pkOwner, _ := pk["owner"].(string)
	if pkOwner == "" {
		t.Error("publicKey.owner is empty")
	}

	// publicKey.publicKeyPem
	pkPem, _ := pk["publicKeyPem"].(string)
	if pkPem == "" {
		t.Error("publicKey.publicKeyPem is empty")
	}
}

// minLen returns the smaller of a and b (used for truncated error messages).
func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
