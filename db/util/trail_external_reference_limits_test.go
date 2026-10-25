package util

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestExistingExternalReferencesAboveFilterLimit(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	for name, fields := range map[string][]string{
		"activitypub_actors":       {"user"},
		"trails":                   {"name", "author"},
		"trail_external_reference": {"user", "provider", "external_id", "trail", "plugin_id", "provider_category"},
	} {
		collection := core.NewBaseCollection(name)
		for _, field := range fields {
			collection.Fields.Add(&core.TextField{Name: field})
		}
		if name == "trail_external_reference" {
			collection.Fields.Add(&core.DateField{Name: "provider_category_checked_at"})
		}
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	save := func(app core.App, collection string, values map[string]any) *core.Record {
		t.Helper()
		schema, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatal(err)
		}
		record := core.NewRecord(schema)
		record.Load(values)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	actor := save(app, "activitypub_actors", map[string]any{"user": "references-user"})
	const count = 225
	opaqueIDs := []string{"abc", " abc", "abc ", " ", "  "}
	ids := make([]string, 0, count+len(opaqueIDs)+7)
	for i := range count {
		if i == 149 {
			// Put distinct opaque IDs on both sides of the 150-value chunk boundary.
			ids = append(ids, opaqueIDs...)
		}
		ids = append(ids, fmt.Sprintf("remote-%d", i))
	}
	if err := app.RunInTransaction(func(tx core.App) error {
		for i, id := range ids {
			trail := save(tx, "trails", map[string]any{"name": fmt.Sprintf("Trail %d", i), "author": actor.Id})
			if err := EnsureTrailExternalReference(tx, trail.Id, "test-provider", id, "plugin", ""); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	aliasTrail := save(app, "trails", map[string]any{"name": "Unrequested alias", "author": actor.Id})
	if err := EnsureTrailExternalReference(app, aliasTrail.Id, "test-provider", "alias", "plugin", ""); err != nil {
		t.Fatal(err)
	}
	stale := save(app, "trail_external_reference", map[string]any{
		"user": "references-user", "provider": "test-provider", "external_id": "stale", "trail": "missing-trail",
	})
	foreignUser := save(app, "trail_external_reference", map[string]any{
		"user": "other-user", "provider": "test-provider", "external_id": "missing", "trail": "missing-trail",
	})
	foreignProvider := save(app, "trail_external_reference", map[string]any{
		"user": "references-user", "provider": "other-provider", "external_id": "missing", "trail": "missing-trail",
	})
	ids = append(ids, ids[0], " abc", " ", "", "missing", "stale", " alias ")
	existing, err := FindExistingExternalReferenceIDsForUser(app, "references-user", "test-provider", ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range opaqueIDs {
		if !existing[id] {
			t.Errorf("opaque external reference %q was not found exactly", id)
		}
	}
	if existing["alias"] || existing[" alias "] {
		t.Error("lookup invented a match by trimming an external ID")
	}
	if len(existing) != count+len(opaqueIDs) || existing["missing"] || existing["stale"] {
		t.Fatalf("lookup above expression limit: %d IDs, error=%v", len(existing), err)
	}
	for i := range count {
		if !existing[fmt.Sprintf("remote-%d", i)] {
			t.Fatalf("missing external reference %d", i)
		}
	}
	if _, err := app.FindRecordById("trail_external_reference", stale.Id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("stale reference was not removed: %v", err)
	}
	for _, reference := range []*core.Record{foreignUser, foreignProvider} {
		if _, err := app.FindRecordById("trail_external_reference", reference.Id); err != nil {
			t.Fatalf("reference outside user/provider scope changed: %v", err)
		}
	}
}
