package migrations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestRoutingProfileContentFieldAcceptsBackendMaximum(t *testing.T) {
	const decodedMaxBytes = 256 * 1024
	wantBase64Length := 4 * ((decodedMaxBytes + 2) / 3)

	var collection core.Collection
	if err := json.Unmarshal([]byte(routingProfilesCollectionJSON), &collection); err != nil {
		t.Fatalf("decode routing profiles collection: %v", err)
	}
	field, ok := collection.Fields.GetByName("content_base64").(*core.TextField)
	if !ok {
		t.Fatal("routing_profiles.content_base64 is not a text field")
	}
	if field.Max != wantBase64Length {
		t.Fatalf("content_base64 max = %d, want %d", field.Max, wantBase64Length)
	}
}

func TestRoutingMappingsUseDirectCategorySchema(t *testing.T) {
	var collection core.Collection
	if err := json.Unmarshal([]byte(routingProfileMappingsCollectionJSON), &collection); err != nil {
		t.Fatalf("decode routing profile mappings collection: %v", err)
	}
	for _, name := range []string{"category", "subcategory", "plugin_id", "profile"} {
		if collection.Fields.GetByName(name) == nil {
			t.Fatalf("routing_profile_mappings.%s field is missing", name)
		}
	}
	if collection.Fields.GetByName("intent_key") != nil {
		t.Fatal("routing_profile_mappings still contains experimental intent_key field")
	}
}

func TestCreateRoutingCollectionRejectsExistingSchema(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := app.Save(core.NewBaseCollection("routing_settings")); err != nil {
		t.Fatalf("create conflicting routing settings collection: %v", err)
	}

	err := createRoutingCollection(app, routingSettingsCollectionJSON)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("createRoutingCollection() error = %v, want existing-schema error", err)
	}
}

func TestCreateRoutingCollectionsFromFreshSchema(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	for _, raw := range []string{
		routingSettingsCollectionJSON,
		routingProfilesCollectionJSON,
		routingProfileMappingsCollectionJSON,
	} {
		if err := createRoutingCollection(app, raw); err != nil {
			t.Fatalf("create routing collection: %v", err)
		}
	}
	for _, name := range []string{"routing_settings", "routing_profiles", "routing_profile_mappings"} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Fatalf("find %s collection: %v", name, err)
		}
	}
}
