package pluginsystem

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestRoutingPluginIsSelectedUsesEffectiveActiveSettings(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	settings := core.NewBaseCollection("routing_settings")
	settings.Fields.Add(
		&core.TextField{Name: "scope"},
		&core.TextField{Name: "user"},
		&core.JSONField{Name: "config"},
	)
	if err := app.Save(settings); err != nil {
		t.Fatalf("save routing settings collection: %v", err)
	}
	builtin := core.NewRecord(settings)
	builtin.Set("scope", "builtin")
	builtin.Set("config", map[string]any{
		"primaryRoutePluginId": "valhalla",
		"elevationPluginId":    "valhalla",
		"maneuverPluginId":     "navigation-only",
		"exposedFeatures":      map[string]any{"routing": true},
	})
	if err := app.Save(builtin); err != nil {
		t.Fatalf("save builtin routing settings: %v", err)
	}
	user := core.NewRecord(settings)
	user.Set("scope", "user")
	user.Set("user", "user-id")
	user.Set("config", map[string]any{"primaryRoutePluginId": "brouter"})
	if err := app.Save(user); err != nil {
		t.Fatalf("save user routing settings: %v", err)
	}

	for pluginID, want := range map[string]bool{
		"valhalla":        true,
		"brouter":         true,
		"comparison-only": false,
		"navigation-only": false,
		"other":           false,
	} {
		selected, err := RoutingPluginIsSelected(app, "user-id", pluginID)
		if err != nil {
			t.Fatalf("check selected plugin %s: %v", pluginID, err)
		}
		if selected != want {
			t.Fatalf("RoutingPluginIsSelected(%q) = %t, want %t", pluginID, selected, want)
		}
	}

	user.Set("config", map[string]any{
		"primaryRoutePluginId": "brouter",
		"exposedFeatures":      map[string]any{"routing": false},
	})
	if err := app.Save(user); err != nil {
		t.Fatalf("disable user routing: %v", err)
	}
	for _, pluginID := range []string{"valhalla", "brouter"} {
		selected, err := RoutingPluginIsSelected(app, "user-id", pluginID)
		if err != nil {
			t.Fatalf("check selected plugin %s with routing disabled: %v", pluginID, err)
		}
		if selected {
			t.Fatalf("RoutingPluginIsSelected(%q) = true with routing disabled", pluginID)
		}
	}
}

func TestRoutingSettingsLayersResolveDuplicateScopeDeterministically(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	settings := core.NewBaseCollection("routing_settings")
	settings.Fields.Add(
		&core.TextField{Name: "scope"},
		&core.TextField{Name: "user"},
		&core.JSONField{Name: "config"},
	)
	if err := app.Save(settings); err != nil {
		t.Fatalf("save routing settings collection: %v", err)
	}

	records := make([]*core.Record, 2)
	for index, pluginID := range []string{"route-a", "route-b"} {
		records[index] = core.NewRecord(settings)
		records[index].Set("scope", "admin")
		records[index].Set("config", map[string]any{
			"primaryRoutePluginId": pluginID,
		})
		if err := app.Save(records[index]); err != nil {
			t.Fatalf("save duplicate admin settings %d: %v", index, err)
		}
	}

	selectedRecord := records[0]
	if records[1].Id < records[0].Id {
		selectedRecord = records[1]
	}
	selectedPlugin := stringFromJSON(JSONMapFromRecord(selectedRecord, "config")["primaryRoutePluginId"])

	layers, err := ResolveRoutingSettingsLayers(app, "user-id")
	if err != nil {
		t.Fatalf("resolve routing settings layers: %v", err)
	}
	if got := stringFromJSON(layers.Admin["primaryRoutePluginId"]); got != selectedPlugin {
		t.Fatalf("resolved admin plugin = %q, want %q", got, selectedPlugin)
	}
	selected, err := RoutingPluginIsSelected(app, "user-id", selectedPlugin)
	if err != nil {
		t.Fatalf("resolve selected routing plugin: %v", err)
	}
	if !selected {
		t.Fatalf("shared selection policy did not use resolved plugin %q", selectedPlugin)
	}
}

func TestRoutingNavigationGateIgnoresUserScope(t *testing.T) {
	effective := MergeRoutingSettingsLayers(RoutingSettingsLayers{
		Builtin: map[string]any{"exposedFeatures": map[string]any{"navigation": true}},
		Admin:   map[string]any{"exposedFeatures": map[string]any{"navigation": true}},
		User:    map[string]any{"exposedFeatures": map[string]any{"navigation": false}},
	})
	features := effective["exposedFeatures"].(map[string]any)
	if features["navigation"] != true {
		t.Fatalf("user scope disabled admin navigation gate: %#v", features)
	}

	effective = MergeRoutingSettingsLayers(RoutingSettingsLayers{
		Builtin: map[string]any{"exposedFeatures": map[string]any{"navigation": true}},
		Admin:   map[string]any{"exposedFeatures": map[string]any{"navigation": false}},
		User:    map[string]any{"exposedFeatures": map[string]any{"navigation": true}},
	})
	features = effective["exposedFeatures"].(map[string]any)
	if features["navigation"] != false {
		t.Fatalf("user scope re-enabled admin navigation gate: %#v", features)
	}
}

func TestRoutingSettingsLayersKeepScopeAttachedToConfig(t *testing.T) {
	layers := RoutingSettingsLayers{
		Builtin: map[string]any{"scopeMarker": "builtin"},
		Admin:   map[string]any{"scopeMarker": "admin"},
		User:    map[string]any{"scopeMarker": "user"},
	}.Configs()
	wantScopes := []string{routingSettingsScopeBuiltin, routingSettingsScopeAdmin, routingSettingsScopeUser}
	for index, wantScope := range wantScopes {
		if layers[index].Scope != wantScope || layers[index].Config["scopeMarker"] != wantScope {
			t.Fatalf("layer %d lost its scope identity: %#v", index, layers[index])
		}
	}
}
