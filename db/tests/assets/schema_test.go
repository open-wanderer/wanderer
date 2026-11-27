package assets_test

import (
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

func TestAssetSchemaKeepsHiddenProvenanceAndUniqueExternalIdentity(t *testing.T) {
	app := newAssetFilterTestApp(t)
	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	provenance, ok := collection.Fields.GetByName("source_metadata").(*core.JSONField)
	if !ok || !provenance.Hidden || provenance.MaxSize != 8<<20 {
		t.Fatalf("provenance field: %#v", provenance)
	}
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{"username": "photos", "preferred_username": "photos", "is_local": true, "public_key": "key", "iri": "https://a.example/photos", "inbox": "https://a.example/inbox"})
	values := map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id, "external_provider": "provider", "external_id": "photo"}
	saveAssetFilterRecord(t, app, "assets", values)
	duplicate := core.NewRecord(collection)
	duplicate.Load(values)
	if err := app.Save(duplicate); err == nil {
		t.Fatal("duplicate external identity was accepted")
	}
	values["external_id"] = "other-photo"
	saveAssetFilterRecord(t, app, "assets", values)
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		links, err := app.FindCollectionByNameOrId(target + "_assets")
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"asset", target} {
			relation, ok := links.Fields.GetByName(field).(*core.RelationField)
			if !ok || !relation.CascadeDelete || !relation.Required {
				t.Fatalf("%s.%s relation: %#v", links.Name, field, relation)
			}
		}
	}
}
