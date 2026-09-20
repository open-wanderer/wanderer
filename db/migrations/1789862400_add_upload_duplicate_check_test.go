package migrations

import (
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestUploadDuplicateCheckMigration(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })

	settings := core.NewBaseCollection("settings")
	settings.Fields.Add(&core.JSONField{Name: "behavior"})
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	legacy := core.NewRecord(settings)
	behavior := map[string]any{"allowAutoGeolocate": true, "mapClusteringMaxZoom": float64(9)}
	legacy.Set("behavior", behavior)
	if err := app.Save(legacy); err != nil {
		t.Fatal(err)
	}

	var up, down func(core.App) error
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1789862400_add_upload_duplicate_check.go" {
			up, down = migration.Up, migration.Down
			break
		}
	}
	if up == nil || down == nil {
		t.Fatal("upload duplicate check migration is not registered")
	}
	if err := app.RunInTransaction(up); err != nil {
		t.Fatal(err)
	}
	settings, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	field, ok := settings.Fields.GetByName("uploadDuplicateCheck").(*core.JSONField)
	if !ok || field.Required {
		t.Fatalf("uploadDuplicateCheck must be an optional JSON field, got %#v", field)
	}

	assertStored := func(wantScope map[string]bool) {
		t.Helper()
		record, err := app.FindRecordById("settings", legacy.Id)
		if err != nil {
			t.Fatal(err)
		}
		var gotScope map[string]bool
		if err := record.UnmarshalJSONField("uploadDuplicateCheck", &gotScope); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotScope, wantScope) {
			t.Fatalf("stored scope = %#v, want %#v", gotScope, wantScope)
		}
		var gotBehavior map[string]any
		if err := record.UnmarshalJSONField("behavior", &gotBehavior); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotBehavior, behavior) {
			t.Fatalf("existing map behavior changed: %#v", gotBehavior)
		}
	}
	assertStored(nil)

	for _, scope := range []map[string]bool{
		{"includePublic": true, "includeShared": false},
		{"includeShared": true},
		nil,
	} {
		record, err := app.FindRecordById("settings", legacy.Id)
		if err != nil {
			t.Fatal(err)
		}
		record.Set("uploadDuplicateCheck", scope)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		assertStored(scope)
	}

	if err := app.RunInTransaction(down); err != nil {
		t.Fatal(err)
	}
	settings, err = app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	if settings.Fields.GetByName("uploadDuplicateCheck") != nil {
		t.Fatal("rollback did not remove uploadDuplicateCheck")
	}
	if settings.Fields.GetByName("behavior") == nil {
		t.Fatal("rollback removed map behavior")
	}
	if _, err := app.FindRecordById("settings", legacy.Id); err != nil {
		t.Fatalf("rollback lost existing settings: %v", err)
	}
}
