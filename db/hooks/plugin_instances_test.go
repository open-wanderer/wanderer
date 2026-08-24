package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestSelectedRoutingPluginCannotBeDisabled(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	routingSettings := core.NewBaseCollection("routing_settings")
	routingSettings.Fields.Add(
		&core.TextField{Name: "scope"},
		&core.TextField{Name: "user"},
		&core.JSONField{Name: "config"},
	)
	pluginInstances := core.NewBaseCollection("plugin_instances")
	pluginInstances.Fields.Add(
		&core.TextField{Name: "user"},
		&core.TextField{Name: "plugin_id"},
		&core.BoolField{Name: "enabled"},
	)
	for _, collection := range []*core.Collection{routingSettings, pluginInstances} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("save %s collection: %v", collection.Name, err)
		}
	}
	builtin := core.NewRecord(routingSettings)
	builtin.Set("scope", "builtin")
	builtin.Set("config", map[string]any{
		"primaryRoutePluginId": "valhalla",
		"elevationPluginId":    "valhalla",
		"maneuverPluginId":     "brouter",
	})
	if err := app.Save(builtin); err != nil {
		t.Fatalf("save routing settings: %v", err)
	}

	for pluginID, wantBlocked := range map[string]bool{
		"valhalla": true,
		"brouter":  false,
		"other":    false,
	} {
		instance := core.NewRecord(pluginInstances)
		instance.Set("user", "user-id")
		instance.Set("plugin_id", pluginID)
		instance.Set("enabled", true)
		if err := app.Save(instance); err != nil {
			t.Fatalf("save %s plugin instance: %v", pluginID, err)
		}
		instance, err := app.FindRecordById("plugin_instances", instance.Id)
		if err != nil {
			t.Fatalf("reload %s plugin instance: %v", pluginID, err)
		}
		instance.Set("enabled", false)
		err = preventSelectedRoutingPluginDisable(app, instance)
		if (err != nil) != wantBlocked {
			t.Fatalf("disable %s blocked=%t, want %t (error=%v)", pluginID, err != nil, wantBlocked, err)
		}
	}

	disabledUserSettings := core.NewRecord(routingSettings)
	disabledUserSettings.Set("scope", "user")
	disabledUserSettings.Set("user", "routing-disabled-user")
	disabledUserSettings.Set("config", map[string]any{
		"exposedFeatures": map[string]bool{"routing": false},
	})
	if err := app.Save(disabledUserSettings); err != nil {
		t.Fatalf("save disabled user routing settings: %v", err)
	}
	disabledUserInstance := core.NewRecord(pluginInstances)
	disabledUserInstance.Set("user", "routing-disabled-user")
	disabledUserInstance.Set("plugin_id", "valhalla")
	disabledUserInstance.Set("enabled", true)
	if err := app.Save(disabledUserInstance); err != nil {
		t.Fatalf("save disabled user plugin instance: %v", err)
	}
	disabledUserInstance, err := app.FindRecordById("plugin_instances", disabledUserInstance.Id)
	if err != nil {
		t.Fatalf("reload disabled user plugin instance: %v", err)
	}
	disabledUserInstance.Set("enabled", false)
	if err := preventSelectedRoutingPluginDisable(app, disabledUserInstance); err != nil {
		t.Fatalf("routing-disabled user could not disable selected plugin: %v", err)
	}
}
