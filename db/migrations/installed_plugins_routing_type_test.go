package migrations

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestRoutingPluginTypeMigrationPreservesIndependentPluginTypes(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	installed := core.NewBaseCollection("installed_plugins")
	installed.Fields.Add(
		&core.TextField{Name: "plugin_id"},
		&core.SelectField{Name: "type", MaxSelect: 1, Values: []string{"trails", "assets", "separately_deployed"}},
	)
	instances := core.NewBaseCollection("plugin_instances")
	instances.Fields.Add(
		&core.TextField{Name: "user"},
		&core.TextField{Name: "plugin_id"},
	)
	for _, collection := range []*core.Collection{installed, instances} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("create %s collection: %v", collection.Name, err)
		}
	}

	if err := addInstalledRoutingPluginType(app); err != nil {
		t.Fatalf("add routing plugin type: %v", err)
	}
	if err := addInstalledRoutingPluginType(app); err != nil {
		t.Fatalf("idempotently add routing plugin type: %v", err)
	}
	assertInstalledPluginTypeValues(t, app, []string{"trails", "assets", "separately_deployed", "routing"})
	installed, err := app.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		t.Fatalf("reload installed plugins collection: %v", err)
	}

	for _, plugin := range []struct {
		id         string
		pluginType string
	}{
		{id: "valhalla", pluginType: "routing"},
		{id: "immich", pluginType: "assets"},
	} {
		record := core.NewRecord(installed)
		record.Set("plugin_id", plugin.id)
		record.Set("type", plugin.pluginType)
		if err := app.Save(record); err != nil {
			t.Fatalf("create installed %s plugin: %v", plugin.id, err)
		}
		instance := core.NewRecord(instances)
		instance.Set("user", "user-id")
		instance.Set("plugin_id", plugin.id)
		if err := app.Save(instance); err != nil {
			t.Fatalf("create %s plugin instance: %v", plugin.id, err)
		}
	}

	if err := removeInstalledRoutingPluginType(app); err != nil {
		t.Fatalf("remove routing plugin type: %v", err)
	}
	assertInstalledPluginTypeValues(t, app, []string{"trails", "assets", "separately_deployed"})
	if records, err := app.FindRecordsByFilter("installed_plugins", "plugin_id='valhalla'", "", -1, 0); err != nil || len(records) != 0 {
		t.Fatalf("routing plugin records after down migration = %d, err=%v", len(records), err)
	}
	if records, err := app.FindRecordsByFilter("plugin_instances", "plugin_id='valhalla'", "", -1, 0); err != nil || len(records) != 0 {
		t.Fatalf("routing plugin instances after down migration = %d, err=%v", len(records), err)
	}
	if records, err := app.FindRecordsByFilter("installed_plugins", "plugin_id='immich'", "", -1, 0); err != nil || len(records) != 1 {
		t.Fatalf("asset plugin records after routing down migration = %d, err=%v", len(records), err)
	}
	if records, err := app.FindRecordsByFilter("plugin_instances", "plugin_id='immich'", "", -1, 0); err != nil || len(records) != 1 {
		t.Fatalf("asset plugin instances after routing down migration = %d, err=%v", len(records), err)
	}
}

func TestAssetPluginTypeMigrationPreservesIndependentlyDeployedPluginTypes(t *testing.T) {
	app := newRoutingPluginTypeMigrationTestApp(t)
	installed := core.NewBaseCollection("installed_plugins")
	installed.Fields.Add(&core.SelectField{
		Name:      "type",
		MaxSelect: 1,
		Values:    []string{"trails", "routing", "separately_deployed"},
	})
	if err := app.Save(installed); err != nil {
		t.Fatalf("create installed_plugins collection: %v", err)
	}

	if err := setInstalledAssetPluginTypePresent(app, true); err != nil {
		t.Fatalf("add asset plugin type: %v", err)
	}
	assertInstalledPluginTypeValues(t, app, []string{"trails", "routing", "separately_deployed", "assets"})

	if err := setInstalledAssetPluginTypePresent(app, false); err != nil {
		t.Fatalf("remove asset plugin type: %v", err)
	}
	assertInstalledPluginTypeValues(t, app, []string{"trails", "routing", "separately_deployed"})
}

func TestRoutingPluginTypeMigrationRequiresExpectedSchema(t *testing.T) {
	t.Run("missing collection", func(t *testing.T) {
		app := newRoutingPluginTypeMigrationTestApp(t)
		if err := addInstalledRoutingPluginType(app); err == nil {
			t.Fatal("add routing plugin type succeeded without installed_plugins collection")
		}
	})

	t.Run("wrong field type", func(t *testing.T) {
		app := newRoutingPluginTypeMigrationTestApp(t)
		collection := core.NewBaseCollection("installed_plugins")
		collection.Fields.Add(&core.TextField{Name: "type"})
		if err := app.Save(collection); err != nil {
			t.Fatalf("create installed_plugins collection: %v", err)
		}
		if err := addInstalledRoutingPluginType(app); err == nil {
			t.Fatal("add routing plugin type succeeded with non-select type field")
		}
	})
}

func newRoutingPluginTypeMigrationTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}

func assertInstalledPluginTypeValues(t *testing.T, app core.App, want []string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		t.Fatalf("find installed plugins collection: %v", err)
	}
	field, ok := collection.Fields.GetByName("type").(*core.SelectField)
	if !ok {
		t.Fatalf("installed_plugins.type field type = %T", collection.Fields.GetByName("type"))
	}
	if !slices.Equal(field.Values, want) {
		t.Fatalf("installed_plugins.type values = %v, want %v", field.Values, want)
	}
}
