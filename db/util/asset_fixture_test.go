package util

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func setupFederatedAssetsTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.BoolField{Name: "is_local"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"actor0000000001", "actor0000000002"} {
		actor := core.NewRecord(actors)
		actor.Id = id
		if err := app.Save(actor); err != nil {
			t.Fatal(err)
		}
	}

	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(
		&core.TextField{Name: "type"},
		&core.TextField{Name: "storage_mode"},
		&core.TextField{Name: "remote_status"},
		&core.TextField{Name: "author"},
		&core.TextField{Name: "file"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.DateField{Name: "taken_at"},
		&core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"},
		&core.JSONField{Name: "metadata"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(assets); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	for _, config := range []struct {
		name  string
		field string
	}{
		{name: "trail_assets", field: "trail"},
		{name: "waypoint_assets", field: "waypoint"},
		{name: "summit_log_assets", field: "summit_log"},
	} {
		collection := core.NewBaseCollection(config.name)
		collection.Fields.Add(
			&core.TextField{Name: "asset"},
			&core.TextField{Name: config.field},
		)
		if config.name == "trail_assets" {
			collection.Fields.Add(&core.BoolField{Name: "is_thumbnail"})
		}
		collection.Fields.Add(
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		if err := app.Save(collection); err != nil {
			app.Cleanup()
			t.Fatal(err)
		}
	}

	return app
}

func createFederatedTestAsset(t *testing.T, app core.App, author string, externalID string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("type", "photo")
	record.Set("storage_mode", "copy")
	record.Set("author", author)
	if externalID != "" {
		record.Set("external_provider", FederationAssetProvider)
		record.Set("external_id", externalID)
	}
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func createFederatedTestLink(t *testing.T, app core.App, collectionName string, targetField string, targetID string, assetID string) *core.Record {
	t.Helper()

	record, err := EnsureAssetLink(app, collectionName, targetField, targetID, assetID)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func assertFederatedAssetCount(t *testing.T, app core.App, want int) {
	t.Helper()

	records, err := app.FindRecordsByFilter("assets", "type='photo'", "", -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != want {
		t.Fatalf("got %d assets, want %d", len(records), want)
	}
}

func assertFederatedLinkCount(t *testing.T, app core.App, collectionName string, targetField string, targetID string, want int) {
	t.Helper()

	links, err := app.FindRecordsByFilter(collectionName, targetField+"={:target}", "", -1, 0, dbx.Params{"target": targetID})
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != want {
		t.Fatalf("got %d links for %s, want %d", len(links), targetID, want)
	}
}
