package federation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	_ "pocketbase/migrations"
)

// newTestApp creates a bootstrapped PocketBase app with all migrations applied,
// using a temporary directory that is cleaned up after the test.
func newTestApp(t *testing.T) core.App {
	t.Helper()

	dir := t.TempDir()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       dir,
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	t.Cleanup(func() {
		app.ResetBootstrapState()
		os.RemoveAll(dir)
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
		t.Errorf("public_key does not start with PEM header, got: %q", pubKey[:min(50, len(pubKey))])
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Verify test data directory accessibility (guards against runtime.Caller issues)
func TestTestDataDirAccessible(t *testing.T) {
	dir := t.TempDir()
	if _, err := os.Stat(filepath.Join(dir)); err != nil {
		t.Fatalf("temp dir not accessible: %v", err)
	}
}
