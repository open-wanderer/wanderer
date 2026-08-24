package routes

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"pocketbase/pluginsystem"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestRoutingSettingsMergeSemantics(t *testing.T) {
	config := pluginsystem.MergeRoutingSettingsLayers(pluginsystem.RoutingSettingsLayers{
		Builtin: map[string]any{
			"primaryRoutePluginId": "valhalla",
			"maneuverPluginId":     "valhalla",
			"defaultVariantCount":  1,
			"defaultRoutingMode":   "segment",
			"defaultPreferences": map[string]any{
				"hillPreference": 0.5,
				"nested": map[string]any{
					"a": 1.0,
				},
			},
		},
		Admin: map[string]any{
			"primaryRoutePluginId": "brouter",
			"maneuverPluginId":     "graphhopper",
			"defaultVariantCount":  2,
			"defaultRoutingMode":   "via",
			"defaultPreferences": map[string]any{
				"speedPreference": 0.6,
				"nested": map[string]any{
					"b": 2.0,
				},
			},
		},
		User: map[string]any{
			"defaultPreferences": map[string]any{
				"hillPreference": 0.8,
			},
		},
	})
	resolved := routingSettingsFromConfig(config)

	if resolved.PrimaryRoutePluginID != "brouter" {
		t.Fatalf("expected admin scalar to replace builtin, got %q", resolved.PrimaryRoutePluginID)
	}
	if resolved.ManeuverPluginID != "graphhopper" {
		t.Fatalf("expected maneuver scalar to replace builtin, got %q", resolved.ManeuverPluginID)
	}
	if resolved.DefaultVariantCount != 2 {
		t.Fatalf("expected admin scalar to replace builtin, got %d", resolved.DefaultVariantCount)
	}
	if resolved.DefaultRoutingMode != "via" {
		t.Fatalf("expected routing mode override, got %q", resolved.DefaultRoutingMode)
	}
	if resolved.DefaultPreferences["hillPreference"] != 0.8 || resolved.DefaultPreferences["speedPreference"] != 0.6 {
		t.Fatalf("expected preference map deep merge, got %#v", resolved.DefaultPreferences)
	}
	nested, ok := resolved.DefaultPreferences["nested"].(map[string]any)
	if !ok || nested["a"] != 1.0 || nested["b"] != 2.0 {
		t.Fatalf("expected nested preference merge, got %#v", resolved.DefaultPreferences["nested"])
	}
}

func TestRoutingSettingsFromConfigPreservesAbsentCollectionFields(t *testing.T) {
	settings := routingSettingsFromConfig(map[string]any{})
	if settings.DefaultPreferences != nil {
		t.Fatalf("absent defaultPreferences became an explicit empty override: %#v", settings.DefaultPreferences)
	}
	if settings.ExposedFeatures != nil {
		t.Fatalf("absent exposedFeatures became an explicit empty override: %#v", settings.ExposedFeatures)
	}
}

func TestApplyRoutingEngineDefaultsUsesEnabledCapabilitiesAndPreservesOverrides(t *testing.T) {
	settings := routingSettings{ElevationPluginID: "configured-elevation"}
	engines := []routingEngineView{
		{PluginID: "multi-role", Enabled: true, Roles: []string{"route", "elevation", "maneuvers"}, discoveryRank: 2},
		{PluginID: "disabled-first", Enabled: false, Roles: []string{"route", "elevation", "maneuvers"}, discoveryRank: 0},
		{PluginID: "maneuver-later", Enabled: true, Roles: []string{"maneuvers"}, discoveryRank: 3},
		{PluginID: "route-first", Enabled: true, Roles: []string{"route"}, discoveryRank: 1},
	}

	applyRoutingEngineDefaults(&settings, engines)

	if settings.PrimaryRoutePluginID != "route-first" {
		t.Fatalf("primary route default = %q, want oldest enabled route engine", settings.PrimaryRoutePluginID)
	}
	if settings.ElevationPluginID != "configured-elevation" {
		t.Fatalf("explicit elevation selection was replaced: %q", settings.ElevationPluginID)
	}
	if settings.ManeuverPluginID != "multi-role" {
		t.Fatalf("maneuver default = %q, want oldest enabled capable engine", settings.ManeuverPluginID)
	}

	explicit := routingSettings{
		PrimaryRoutePluginID: "configured-route",
		ElevationPluginID:    "configured-elevation",
		ManeuverPluginID:     "configured-maneuvers",
	}
	applyRoutingEngineDefaults(&explicit, engines)
	if explicit.PrimaryRoutePluginID != "configured-route" || explicit.ElevationPluginID != "configured-elevation" || explicit.ManeuverPluginID != "configured-maneuvers" {
		t.Fatalf("explicit engine selections were replaced: %#v", explicit)
	}
}

func TestApplyRoutingEngineDefaultsPrefersCapablePrimaryEngine(t *testing.T) {
	settings := routingSettings{}
	engines := []routingEngineView{
		{PluginID: "primary", Enabled: true, Roles: []string{"route", "elevation", "maneuvers"}, discoveryRank: 0},
		{PluginID: "other-elevation", Enabled: true, Roles: []string{"elevation"}, discoveryRank: 1},
		{PluginID: "other-maneuvers", Enabled: true, Roles: []string{"maneuvers"}, discoveryRank: 2},
	}

	applyRoutingEngineDefaults(&settings, engines)

	if settings.PrimaryRoutePluginID != "primary" || settings.ElevationPluginID != "primary" || settings.ManeuverPluginID != "primary" {
		t.Fatalf("capable primary engine was not reused across roles: %#v", settings)
	}
}

func TestResolveRoutingSettingsDiscoversProviderNeutralEngineDefaults(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"capable-engine": true}, "segment")
	addRoutingTestCapability(t, app, "capable-engine", pluginsystem.CapabilityManifest{Name: "elevation", Version: "v1", Export: "elevation_v1"})
	addRoutingTestCapability(t, app, "capable-engine", pluginsystem.CapabilityManifest{Name: "maneuvers", Version: "v1", Export: "maneuvers_v1"})

	builtin, err := pluginsystem.RoutingSettingsRecordForScope(app, routingScopeBuiltin, "")
	if err != nil || builtin == nil {
		t.Fatalf("find builtin routing settings: record=%v err=%v", builtin, err)
	}
	builtin.Set("config", map[string]any{
		"defaultVariantCount": 3,
		"defaultRoutingMode":  "segment",
	})
	if err := app.Save(builtin); err != nil {
		t.Fatalf("remove persisted provider selections: %v", err)
	}

	settings, err := ResolveRoutingSettings(app, auth.Id)
	if err != nil {
		t.Fatalf("resolve provider-neutral routing defaults: %v", err)
	}
	if settings.PrimaryRoutePluginID != "capable-engine" || settings.ElevationPluginID != "capable-engine" || settings.ManeuverPluginID != "capable-engine" {
		t.Fatalf("enabled capable engine was not selected by role: %#v", settings)
	}

	instance, err := app.FindFirstRecordByData("plugin_instances", "plugin_id", "capable-engine")
	if err != nil {
		t.Fatalf("find capable engine instance: %v", err)
	}
	instance.Set("enabled", false)
	if err := app.Save(instance); err != nil {
		t.Fatalf("disable capable engine: %v", err)
	}
	settings, err = ResolveRoutingSettings(app, auth.Id)
	if err != nil {
		t.Fatalf("resolve settings without enabled engines: %v", err)
	}
	if settings.PrimaryRoutePluginID != "" || settings.ElevationPluginID != "" || settings.ManeuverPluginID != "" {
		t.Fatalf("disabled engine remained an implicit default: %#v", settings)
	}
}

func TestInitRoutingDefaultsPersistsNoProviderSelections(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}

	if err := InitRoutingDefaults(app); err != nil {
		t.Fatalf("initialize provider-neutral routing defaults: %v", err)
	}
	record, err := pluginsystem.RoutingSettingsRecordForScope(app, routingScopeBuiltin, "")
	if err != nil || record == nil {
		t.Fatalf("find initialized builtin settings: record=%v err=%v", record, err)
	}
	config := pluginsystem.JSONMapFromRecord(record, "config")
	for _, key := range []string{"primaryRoutePluginId", "elevationPluginId", "maneuverPluginId"} {
		if value, found := config[key]; found {
			t.Fatalf("builtin settings persisted provider selection %s=%v", key, value)
		}
	}
}

func TestRoutingCandidateSetCanBeExposedByAdmin(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"exposedFeatures": map[string]bool{"routeCandidates": false},
	}); err != nil {
		t.Fatalf("seed builtin routing settings: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeAdmin, "", map[string]any{
		"exposedFeatures": map[string]bool{"routeCandidates": true},
	}); err != nil {
		t.Fatalf("seed admin routing settings: %v", err)
	}
	if !routingCandidateSetExposed(app) {
		t.Fatal("expected admin scope to expose the candidate-set endpoint")
	}
}

func TestRoutingCandidateSetIgnoresUserFeatureOverride(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeUser, "user-id", map[string]any{
		"exposedFeatures": map[string]bool{"routeCandidates": true},
	}); err != nil {
		t.Fatalf("save user routing settings: %v", err)
	}
	if routingCandidateSetExposed(app) {
		t.Fatal("user scope must not expose the candidate-set endpoint")
	}
}

func TestNormalizeRoutingSettingsConfigClampsVariantDefaults(t *testing.T) {
	config := map[string]any{
		"primaryRoutePluginId": "primary",
		"defaultVariantCount":  float64(99),
		"defaultRoutingMode":   "segment",
	}
	if err := normalizeRoutingSettingsConfig(config); err != nil {
		t.Fatalf("normalize settings: %v", err)
	}
	if got := intValue(config["defaultVariantCount"]); got != routingMaxVariants {
		t.Fatalf("variant count = %d, want %d", got, routingMaxVariants)
	}
	config["defaultVariantCount"] = float64(-1)
	if err := normalizeRoutingSettingsConfig(config); err != nil {
		t.Fatalf("normalize minimum variant count: %v", err)
	}
	if got := intValue(config["defaultVariantCount"]); got != 1 {
		t.Fatalf("minimum variant count = %d, want 1", got)
	}

	config["defaultRoutingMode"] = "unknown"
	if err := normalizeRoutingSettingsConfig(config); err == nil {
		t.Fatal("expected invalid default routing mode to be rejected")
	}
}

func TestRoutingProfileRevisionTracksProfileContent(t *testing.T) {
	profile := pluginRoutingProfile{
		ID: "profile-id", PluginID: "brouter", Key: "custom", Kind: "custom_file", Mode: "bike",
		ContentBase64: "Zmlyc3Q=", ContentType: "text/plain",
		Metadata: map[string]any{"b": 2, "a": 1},
	}
	first := routingResolvedProfileRevision(profile)
	profile.Metadata = map[string]any{"a": 1, "b": 2}
	if second := routingResolvedProfileRevision(profile); second != first {
		t.Fatalf("equivalent profile metadata produced a different revision: %q != %q", first, second)
	}
	profile.ContentBase64 = "c2Vjb25k"
	if second := routingResolvedProfileRevision(profile); second == first {
		t.Fatal("changed profile content must change the profile revision")
	}
}

func TestRoutingFeatureGatesAreUpperBounds(t *testing.T) {
	resolved := pluginsystem.MergeRoutingSettingsLayers(pluginsystem.RoutingSettingsLayers{
		Builtin: map[string]any{
			"exposedFeatures": map[string]any{
				"standardControls":       true,
				"nativeAdvancedControls": true,
				"profileUpload":          false,
			},
		},
		Admin: map[string]any{
			"exposedFeatures": map[string]any{
				"nativeAdvancedControls": false,
				"profileUpload":          true,
			},
		},
		User: map[string]any{
			"exposedFeatures": map[string]any{
				"nativeAdvancedControls": true,
				"profileUpload":          true,
			},
		},
	})
	features := routingSettingsFromConfig(resolved).ExposedFeatures

	if !features["standardControls"] {
		t.Fatalf("expected untouched builtin feature to remain enabled: %#v", features)
	}
	if features["nativeAdvancedControls"] {
		t.Fatalf("expected admin false to cap user true: %#v", features)
	}
	if features["profileUpload"] {
		t.Fatalf("expected builtin false to cap admin/user true: %#v", features)
	}

	resolved = pluginsystem.MergeRoutingSettingsLayers(pluginsystem.RoutingSettingsLayers{
		Builtin: map[string]any{"exposedFeatures": map[string]any{"variants": true}},
		Admin:   map[string]any{"exposedFeatures": map[string]any{"variants": true}},
		User:    map[string]any{"exposedFeatures": map[string]any{"variants": false}},
	})
	features = routingSettingsFromConfig(resolved).ExposedFeatures
	if features["variants"] {
		t.Fatalf("expected user scope to disable an available feature: %#v", features)
	}
}

func TestPatchRoutingSettingsConfigPreservesUnrelatedValues(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: t.TempDir(),
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeAdmin, "", map[string]any{
		"defaultVariantCount": 2,
		"exposedFeatures": map[string]any{
			"profileUpload": true,
			"variants":      true,
		},
		"defaultPreferences": map[string]any{
			"hillPreference": 0.75,
		},
	}); err != nil {
		t.Fatalf("seed routing settings: %v", err)
	}
	if err := patchRoutingSettingsConfig(app, routingScopeAdmin, "", map[string]any{
		"primaryRoutePluginId": "brouter",
		"exposedFeatures": map[string]any{
			"variants": false,
		},
	}); err != nil {
		t.Fatalf("patch routing settings: %v", err)
	}

	settings, err := routingSettingsForScope(app, routingScopeAdmin, "")
	if err != nil {
		t.Fatalf("read patched routing settings: %v", err)
	}
	if settings.PrimaryRoutePluginID != "brouter" || settings.DefaultVariantCount != 2 || settings.DefaultPreferences["hillPreference"] != 0.75 {
		t.Fatalf("patch replaced unrelated settings: %#v", settings)
	}
	if settings.ExposedFeatures["variants"] || !settings.ExposedFeatures["profileUpload"] {
		t.Fatalf("feature patch replaced unrelated feature gates: %#v", settings.ExposedFeatures)
	}
}

func TestResolveRoutingSettingsSanitizesLegacyPreferencesByLayer(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"defaultPreferences": map[string]any{
			"speedPreference": 18.0,
			"hillPreference":  0.25,
		},
	}); err != nil {
		t.Fatalf("seed builtin routing settings: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeUser, "legacy-user", map[string]any{
		"defaultPreferences": map[string]any{
			"speedPreference": "fast",
			"hillPreference":  0.75,
			"providerMode":    "quiet",
			"legacyNested":    map[string]any{"enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed legacy user routing settings: %v", err)
	}

	settings, err := ResolveRoutingSettings(app, "legacy-user")
	if err != nil {
		t.Fatalf("legacy preferences locked settings reads: %v", err)
	}
	if settings.DefaultPreferences["speedPreference"] != 18.0 {
		t.Fatalf("invalid user override hid the valid builtin fallback: %#v", settings.DefaultPreferences)
	}
	if settings.DefaultPreferences["hillPreference"] != 0.75 || settings.DefaultPreferences["providerMode"] != "quiet" {
		t.Fatalf("valid legacy preferences were discarded: %#v", settings.DefaultPreferences)
	}
	if _, exists := settings.DefaultPreferences["legacyNested"]; exists {
		t.Fatalf("invalid legacy preference survived sanitization: %#v", settings.DefaultPreferences)
	}

	if err := patchRoutingSettingsConfig(app, routingScopeUser, "legacy-user", map[string]any{
		"defaultVariantCount": 2,
	}); err != nil {
		t.Fatalf("unrelated patch remained blocked by legacy preferences: %v", err)
	}
	record, err := pluginsystem.RoutingSettingsRecordForScope(app, routingScopeUser, "legacy-user")
	if err != nil || record == nil {
		t.Fatalf("read repaired user settings record: record=%v err=%v", record, err)
	}
	persisted := mapValue(pluginsystem.JSONMapFromRecord(record, "config")["defaultPreferences"])
	if _, exists := persisted["speedPreference"]; exists {
		t.Fatalf("invalid canonical preference was not repaired on update: %#v", persisted)
	}
	if _, exists := persisted["legacyNested"]; exists {
		t.Fatalf("invalid nested preference was not repaired on update: %#v", persisted)
	}
	if persisted["hillPreference"] != 0.75 || persisted["providerMode"] != "quiet" {
		t.Fatalf("repair removed valid persisted preferences: %#v", persisted)
	}
}

func TestResolveRoutingSettingsPreferenceLimitPreservesNearestScope(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}
	builtin := make(map[string]any, routingPreferenceMaxEntries)
	for index := range routingPreferenceMaxEntries {
		builtin[fmt.Sprintf("builtin%02d", index)] = float64(index)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{"defaultPreferences": builtin}); err != nil {
		t.Fatalf("seed builtin routing settings: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeUser, "priority-user", map[string]any{
		"defaultPreferences": map[string]any{"zzUserPreference": "keep-me"},
	}); err != nil {
		t.Fatalf("seed user routing settings: %v", err)
	}

	settings, err := ResolveRoutingSettings(app, "priority-user")
	if err != nil {
		t.Fatalf("resolve layered routing settings: %v", err)
	}
	if len(settings.DefaultPreferences) != routingPreferenceMaxEntries || settings.DefaultPreferences["zzUserPreference"] != "keep-me" {
		t.Fatalf("effective preference cap discarded the nearest scope: %#v", settings.DefaultPreferences)
	}
}

func TestPatchRoutingSettingsConfigStrictlyRejectsNewInvalidPreferences(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}

	for name, preferences := range map[string]map[string]any{
		"canonical type":  {"speedPreference": "fast"},
		"canonical range": {"hillPreference": 1.5},
		"nested provider": {"providerMode": map[string]any{"value": true}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := patchRoutingSettingsConfig(app, routingScopeUser, "user-id", map[string]any{
				"defaultPreferences": preferences,
			}); err == nil {
				t.Fatalf("accepted invalid submitted defaultPreferences: %#v", preferences)
			}
		})
	}

	if err := patchRoutingSettingsConfig(app, routingScopeUser, "user-id", map[string]any{
		"defaultPreferences": map[string]any{"speedPreference": 18.0, "providerMode": "quiet"},
	}); err != nil {
		t.Fatalf("rejected valid submitted defaultPreferences: %v", err)
	}
}

func TestRoutingSettingsRejectUnknownOrNonBooleanFeatures(t *testing.T) {
	for name, features := range map[string]map[string]any{
		"unknown":     {"futureTypo": true},
		"non-boolean": {"variants": "yes"},
	} {
		t.Run(name, func(t *testing.T) {
			config := map[string]any{"exposedFeatures": features}
			if err := normalizeRoutingSettingsConfig(config); err == nil {
				t.Fatalf("accepted invalid routing features: %#v", features)
			}
		})
	}
}

func TestRoutingSettingsKeepNavigationAdminControlledAndRejectUnknownSettings(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingSettingsCollectionJSON); err != nil {
		t.Fatalf("create routing settings collection: %v", err)
	}
	if err := patchRoutingSettingsConfig(app, routingScopeUser, "user-id", map[string]any{
		"exposedFeatures": map[string]any{"navigation": false},
	}); err == nil {
		t.Fatal("user scope changed the admin navigation gate")
	}
	if err := patchRoutingSettingsConfig(app, routingScopeAdmin, "", map[string]any{
		"exposedFeatures": map[string]any{"navigation": false},
	}); err != nil {
		t.Fatalf("admin scope could not change navigation gate: %v", err)
	}
	if err := patchRoutingSettingsConfig(app, routingScopeUser, "user-id", map[string]any{
		"unknownSetting": true,
	}); err == nil {
		t.Fatal("unknown routing setting remained writable")
	}
}

func TestRoutingScopeRankGivesUserOverAdminOverBuiltin(t *testing.T) {
	if !(routingScopeRank(routingScopeBuiltin) < routingScopeRank(routingScopeAdmin) &&
		routingScopeRank(routingScopeAdmin) < routingScopeRank(routingScopeUser)) {
		t.Fatalf("unexpected scope order")
	}
}

func TestRoutingProfileMappingKeysSeparatePluginAndInstanceMappings(t *testing.T) {
	pluginMapping := routingProfileMappingView{
		Scope:            routingScopeAdmin,
		Category:         "Hiking",
		PluginID:         "valhalla",
		NativeProfileKey: "pedestrian",
	}
	instanceMapping := routingProfileMappingView{
		Scope:            routingScopeAdmin,
		Category:         "Hiking",
		PluginID:         "valhalla",
		InstanceID:       "europe",
		NativeProfileKey: "pedestrian_eu",
	}

	if routingProfileMappingResolutionKey(pluginMapping) == routingProfileMappingResolutionKey(instanceMapping) {
		t.Fatalf("expected plugin and instance mappings to have distinct resolution keys")
	}
	if !routingProfileMappingPrecedenceLess(pluginMapping, instanceMapping) {
		t.Fatalf("expected plugin-level mapping to sort before instance-specific mapping in the same scope")
	}
	if routingProfileMappingPrecedenceLess(instanceMapping, pluginMapping) {
		t.Fatalf("expected instance-specific mapping to sort after plugin-level mapping in the same scope")
	}
}

func TestSelectRoutingProfileMappingUsesScopeInstanceAndSubcategoryPrecedence(t *testing.T) {
	selected := selectRoutingProfileMapping([]routingProfileMappingView{
		{
			Scope:            routingScopeBuiltin,
			Category:         "Biking",
			Subcategory:      "Gravel",
			PluginID:         "valhalla",
			NativeProfileKey: "builtin-subcategory",
		},
		{
			Scope:            routingScopeAdmin,
			Category:         "Biking",
			PluginID:         "valhalla",
			InstanceID:       "europe",
			NativeProfileKey: "admin-instance-category",
		},
		{
			Scope:            routingScopeUser,
			Category:         "Biking",
			PluginID:         "valhalla",
			NativeProfileKey: "user-plugin-category",
		},
		{
			Scope:            routingScopeUser,
			Category:         "Biking",
			Subcategory:      "Gravel",
			PluginID:         "valhalla",
			InstanceID:       "europe",
			NativeProfileKey: "user-instance-subcategory",
		},
	})

	if selected == nil || selected.NativeProfileKey != "user-instance-subcategory" {
		t.Fatalf("expected user instance mapping to win, got %#v", selected)
	}
}

func TestRoutingProfileInUseChecksAllMappings(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingProfileMappingsCollectionJSON); err != nil {
		t.Fatalf("create mappings collection: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("routing_profile_mappings")
	if err != nil {
		t.Fatalf("find mappings collection: %v", err)
	}
	mapping := core.NewRecord(collection)
	mapping.Set("scope", routingScopeUser)
	mapping.Set("category", "Hidden category")
	mapping.Set("plugin_id", "brouter")
	mapping.Set("profile", "profile-in-use")
	if err := app.Save(mapping); err != nil {
		t.Fatalf("save mapping: %v", err)
	}

	inUse, err := routingProfileInUse(app, "profile-in-use")
	if err != nil {
		t.Fatalf("check referenced profile: %v", err)
	}
	if !inUse {
		t.Fatal("expected referenced profile to be in use")
	}
	inUse, err = routingProfileInUse(app, "unused-profile")
	if err != nil {
		t.Fatalf("check unreferenced profile: %v", err)
	}
	if inUse {
		t.Fatal("expected unreferenced profile not to be in use")
	}
}

func TestCustomRoutingProfileRestrictsStandardPreferences(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingProfilesCollectionJSON); err != nil {
		t.Fatalf("create profiles collection: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("routing_profiles")
	if err != nil {
		t.Fatalf("find profiles collection: %v", err)
	}
	profile := core.NewRecord(collection)
	profile.Set("scope", routingScopeAdmin)
	profile.Set("kind", "custom_file")
	profile.Set("content_base64", "cHJvZmlsZQ==")
	profile.Set("metadata", map[string]any{
		"supportedPreferences": []string{"hillPreference", "maxHikingDifficulty"},
	})
	if err := app.Save(profile); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	supported, restricted, err := routingProfileMappingSupportedPreferences(app, "", routingProfileMappingView{ProfileID: profile.Id})
	if err != nil {
		t.Fatalf("resolve supported preferences: %v", err)
	}
	if !restricted || !supported["hillPreference"] || !supported["maxHikingDifficulty"] || supported["speedPreference"] {
		t.Fatalf("unexpected custom profile preference support: restricted=%v supported=%#v", restricted, supported)
	}
	requiresUpload, err := routingProfileMappingRequiresUpload(app, routingProfileMappingView{ProfileID: profile.Id})
	if err != nil {
		t.Fatalf("resolve profile upload requirement: %v", err)
	}
	if !requiresUpload {
		t.Fatal("expected custom profile content to require one upload per routed segment")
	}
}

func TestApplyStoredRoutingProfileKeepsRevisionMode(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	for _, raw := range []string{
		testRoutingSettingsCollectionJSON,
		testRoutingProfilesCollectionJSON,
		testRoutingProfileMappingsCollectionJSON,
	} {
		if err := createTestRoutingCollection(app, raw); err != nil {
			t.Fatalf("create routing collection: %v", err)
		}
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{}); err != nil {
		t.Fatalf("seed builtin routing settings: %v", err)
	}

	profiles, err := app.FindCollectionByNameOrId("routing_profiles")
	if err != nil {
		t.Fatalf("find profiles collection: %v", err)
	}
	profile := core.NewRecord(profiles)
	profile.Set("scope", routingScopeAdmin)
	profile.Set("plugin_id", "brouter")
	profile.Set("key", "custom-bike")
	profile.Set("name", "Custom bike")
	profile.Set("kind", "custom_file")
	profile.Set("mode", "bike")
	profile.Set("content_base64", "cHJvZmlsZQ==")
	profile.Set("content_type", "text/plain")
	profile.Set("metadata", map[string]any{"supportedPreferences": []string{}})
	profile.Set("enabled", true)
	if err := app.Save(profile); err != nil {
		t.Fatalf("save custom routing profile: %v", err)
	}

	mappings, err := app.FindCollectionByNameOrId("routing_profile_mappings")
	if err != nil {
		t.Fatalf("find mappings collection: %v", err)
	}
	mappingRecord := core.NewRecord(mappings)
	mappingRecord.Set("scope", routingScopeAdmin)
	setRoutingProfileMappingRecord(mappingRecord, routingProfileMappingView{
		Scope:     routingScopeAdmin,
		Category:  "Biking",
		PluginID:  "brouter",
		ProfileID: profile.Id,
	})
	if err := app.Save(mappingRecord); err != nil {
		t.Fatalf("save custom profile mapping: %v", err)
	}

	request := pluginRoutingRouteRequest{Category: "Biking"}
	if err := applyRoutingCategoryMapping(app, "", "brouter", "default", &request); err != nil {
		t.Fatalf("apply custom profile mapping: %v", err)
	}
	if request.Mode != "bike" || request.Profile.Mode != "bike" {
		t.Fatalf("applied route/profile modes = %q/%q, want bike/bike", request.Mode, request.Profile.Mode)
	}
	mapping := routingProfileMappingFromRecord(mappingRecord)
	expectedRevision, err := routingProfileRevisionForMapping(app, mapping)
	if err != nil {
		t.Fatalf("resolve custom profile revision: %v", err)
	}
	if revision := routingResolvedProfileRevision(request.Profile); revision != expectedRevision {
		t.Fatalf("applied custom profile revision = %q, effective-controls revision = %q", revision, expectedRevision)
	}
}

func TestRestrictRoutingPreferencesKeepsSupportedAndProviderSpecificValues(t *testing.T) {
	preferences := map[string]any{
		"speedPreference": 20.0,
		"hillPreference":  0.75,
		"providerKnob":    true,
	}
	restrictRoutingPreferences(preferences, map[string]bool{"hillPreference": true})

	if _, found := preferences["speedPreference"]; found {
		t.Fatalf("expected unsupported canonical preference to be removed: %#v", preferences)
	}
	if preferences["hillPreference"] != 0.75 || preferences["providerKnob"] != true {
		t.Fatalf("expected supported and provider-specific preferences to remain: %#v", preferences)
	}
}

func TestRestrictRoutingPluginPreferencesUsesManifestAllowlist(t *testing.T) {
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{Metadata: map[string]any{
		"routing": map[string]any{"standardPreferences": []any{
			map[string]any{"key": "hillPreference", "modes": []any{"foot"}},
			map[string]any{"key": "roadPreference", "modes": []any{"bike"}},
		}, "providerPreferences": []any{"providerKnob"}},
	}}}
	request := pluginRoutingRouteRequest{
		Mode: "foot",
		Preferences: map[string]any{
			"hillPreference":  0.7,
			"roadPreference":  0.4,
			"speedPreference": 0.5,
			"providerKnob":    true,
		},
	}
	restrictRoutingPluginPreferences(plugin, &request)
	if request.Preferences["hillPreference"] != 0.7 || request.Preferences["providerKnob"] != true {
		t.Fatalf("declared/provider preferences were removed: %#v", request.Preferences)
	}
	if _, exists := request.Preferences["roadPreference"]; exists {
		t.Fatalf("mode-incompatible canonical preference survived: %#v", request.Preferences)
	}
	if _, exists := request.Preferences["speedPreference"]; exists {
		t.Fatalf("undeclared canonical preference survived: %#v", request.Preferences)
	}

	request.Preferences = map[string]any{"hillPreference": 0.7, "providerKnob": true}
	restrictRoutingPluginPreferences(pluginsystem.LocalPlugin{}, &request)
	if len(request.Preferences) != 0 {
		t.Fatalf("missing manifest declaration was not fail-closed: %#v", request.Preferences)
	}
}

func TestValidateRoutingPreferencesRejectsUnsafeValues(t *testing.T) {
	if err := validateRoutingPreferences(map[string]any{"hillPreference": 1.1}, false); err == nil {
		t.Fatal("expected an out-of-range standard preference to be rejected")
	}
	if err := validateRoutingPreferences(map[string]any{"provider": map[string]any{"nested": true}}, false); err == nil {
		t.Fatal("expected a nested provider preference to be rejected")
	}
	if err := validateRoutingPreferences(map[string]any{"provider": true}, false); err != nil {
		t.Fatalf("expected a bounded scalar provider preference to be accepted: %v", err)
	}
	if err := validateRoutingPreferences(map[string]any{" hillPreference ": 0.5}, false); err == nil {
		t.Fatal("expected surrounding whitespace in a preference key to be rejected")
	}
}

func TestNormalizeRoutingEngineSelectionsBoundsAndDeduplicates(t *testing.T) {
	tooMany := make([]routingEngineSelection, routingMaxEngines+1)
	for index := range tooMany {
		tooMany[index].PluginID = fmt.Sprintf("plugin-%d", index)
	}
	if _, err := normalizeRoutingEngineSelections(tooMany, routingMaxEngines); err == nil || err.Code != "engine_limit_exceeded" {
		t.Fatalf("expected engine limit error, got %#v", err)
	}
	selections, err := normalizeRoutingEngineSelections([]routingEngineSelection{
		{PluginID: " valhalla ", InstanceID: " europe "},
		{PluginID: "valhalla", InstanceID: "europe"},
		{PluginID: ""},
	}, routingMaxEngines)
	if err != nil || len(selections) != 1 || selections[0].PluginID != "valhalla" || selections[0].InstanceID != "europe" {
		t.Fatalf("unexpected normalized selections: %#v err=%v", selections, err)
	}
}

func TestRoutingControlForPreferenceProvidesCompleteMetadata(t *testing.T) {
	control := routingControlForPreference("speedPreference", 20.0, "bike")
	if control.Label == "" || len(control.Labels) != 0 || control.Unit != "km/h" || control.Type != "number" || control.UI != "slider" ||
		control.ValueType != "number" || control.Min == nil || *control.Min != 5 || control.Max == nil || *control.Max != 50 ||
		control.Step == nil || *control.Step != 0.5 || control.Target != "preference" || len(control.Path) != 1 {
		t.Fatalf("incomplete effective-control metadata: %#v", control)
	}
	encoded, err := json.Marshal(control)
	if err != nil {
		t.Fatalf("marshal effective control: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(encoded, &wire); err != nil || wire["unit"] != "km/h" {
		t.Fatalf("effective-control wire unit missing: json=%s err=%v", encoded, err)
	}
}

func TestRoutingControlsFromMetadataPreservesOptionalUnit(t *testing.T) {
	controls := routingControlsFromMetadata([]any{map[string]any{
		"key": "bikerPower", "type": "number", "unit": " W ", "default": 100.0,
	}}, nil)
	if len(controls) != 1 || controls[0].Unit != "W" {
		t.Fatalf("native control unit was not preserved: %#v", controls)
	}
}

func TestAllowlistedRoutingNativeConfigFiltersAndValidatesControls(t *testing.T) {
	groups := []any{map[string]any{"key": "bike", "controls": []any{
		map[string]any{"key": "bike.use_roads", "type": "number", "min": 0.0, "max": 1.0, "path": []any{"bike", "use_roads"}},
	}}}
	filtered, err := allowlistedRoutingNativeConfig(map[string]any{
		"bike":   map[string]any{"use_roads": 0.7, "undeclared": true},
		"unsafe": true,
	}, groups)
	if err != nil {
		t.Fatal(err)
	}
	if value, found := nestedValue(filtered, []string{"bike", "use_roads"}); !found || value != 0.7 {
		t.Fatalf("declared value missing from filtered config: %#v", filtered)
	}
	if _, found := nestedValue(filtered, []string{"bike", "undeclared"}); found || len(filtered) != 1 {
		t.Fatalf("undeclared native config survived: %#v", filtered)
	}
	if _, err := allowlistedRoutingNativeConfig(map[string]any{"bike": map[string]any{"use_roads": 2.0}}, groups); err == nil {
		t.Fatal("expected an out-of-range native control to be rejected")
	}
}

func TestRoutingConfigLayersMergeProfileAndCategoryDefaults(t *testing.T) {
	profileConfig := map[string]any{
		"nativeConfig": map[string]any{
			"unused": true,
		},
		"bicycle_type": "Hybrid",
		"nested": map[string]any{
			"shared": "base",
			"keep":   true,
		},
	}
	categoryConfig := map[string]any{
		"bicycle_type": "Road",
		"nested": map[string]any{
			"shared": "category",
		},
	}

	nativeConfig := mergeRoutingConfigLayers(profileConfig, categoryConfig)
	if nativeConfig["bicycle_type"] != "Road" {
		t.Fatalf("expected category native config to override profile default, got %#v", nativeConfig)
	}
	nested := nativeConfig["nested"].(map[string]any)
	if nested["shared"] != "category" || nested["keep"] != true {
		t.Fatalf("expected nested native config merge, got %#v", nested)
	}
}

func TestRoutingDefaultsAndUpsertsHandleMissingRows(t *testing.T) {
	t.Chdir("..")
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: t.TempDir(),
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	for _, raw := range []string{
		testRoutingSettingsCollectionJSON,
		testRoutingProfileMappingsCollectionJSON,
	} {
		if err := createTestRoutingCollection(app, raw); err != nil {
			t.Fatalf("create test routing collection: %v", err)
		}
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"primaryRoutePluginId": "configured-routing",
		"defaultVariantCount":  1,
		"defaultPreferences": map[string]any{
			"vehicleWidth": 1.8,
		},
	}); err != nil {
		t.Fatalf("seed builtin settings: %v", err)
	}

	settings, err := ResolveRoutingSettings(app, "missing-user")
	if err != nil {
		t.Fatalf("resolve settings with missing admin/user scopes: %v", err)
	}
	if settings.DefaultVariantCount != 1 || settings.PrimaryRoutePluginID != "configured-routing" {
		t.Fatalf("unexpected builtin settings: %#v", settings)
	}

	collection, err := app.FindCollectionByNameOrId("routing_profile_mappings")
	if err != nil {
		t.Fatalf("find mappings collection: %v", err)
	}
	input := routingProfileMappingView{
		Scope:            routingScopeAdmin,
		Category:         "Hiking",
		PluginID:         "valhalla",
		InstanceID:       "europe",
		NativeProfileKey: "pedestrian",
		Preferences: map[string]any{
			"hillPreference":      0.9,
			"maxHikingDifficulty": 6.0,
		},
	}
	record, err := routingProfileMappingUpsertRecord(app, collection, routingScopeAdmin, "", input)
	if err != nil {
		t.Fatalf("create missing mapping upsert record: %v", err)
	}
	setRoutingProfileMappingRecord(record, input)
	if err := app.Save(record); err != nil {
		t.Fatalf("save first mapping: %v", err)
	}
	firstID := record.Id

	record, err = routingProfileMappingUpsertRecord(app, collection, routingScopeAdmin, "", input)
	if err != nil {
		t.Fatalf("find existing mapping upsert record: %v", err)
	}
	if record.Id != firstID {
		t.Fatalf("expected idempotent upsert to reuse %q, got %q", firstID, record.Id)
	}

	rows, err := app.FindRecordsByFilter(
		"routing_profile_mappings",
		"scope={:scope} && plugin_id={:plugin} && instance_id={:instance} && category={:category}",
		"",
		-1,
		0,
		map[string]any{
			"scope":    routingScopeAdmin,
			"plugin":   "valhalla",
			"instance": "europe",
			"category": "Hiking",
		},
	)
	if err != nil {
		t.Fatalf("count mapping rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one upserted mapping row, got %d", len(rows))
	}

	request := pluginRoutingRouteRequest{
		Category:    "Hiking",
		Subcategory: "Alpine",
		Profile: pluginRoutingProfile{
			Key: "client-supplied",
		},
		Preferences: map[string]any{
			"speedPreference": 6.2,
		},
	}
	if err := applyRoutingCategoryMapping(app, "", "valhalla", "europe", &request); err != nil {
		t.Fatalf("apply route category mapping: %v", err)
	}
	if request.Profile.Key != "pedestrian" {
		t.Fatalf("expected server-side mapping to override profile key, got %q", request.Profile.Key)
	}
	if request.Mode != "foot" {
		t.Fatalf("expected mapped profile to set mode, got %q", request.Mode)
	}
	if request.Profile.Mode != request.Mode {
		t.Fatalf("expected mapped profile mode %q in provenance revision input, got %q", request.Mode, request.Profile.Mode)
	}
	resolvedMapping, err := ResolveRoutingProfileMapping(app, "", "valhalla", "europe", "Hiking", "Alpine")
	if err != nil || resolvedMapping == nil {
		t.Fatalf("resolve applied routing profile mapping: mapping=%#v err=%v", resolvedMapping, err)
	}
	expectedRevision, err := routingProfileRevisionForMapping(app, *resolvedMapping)
	if err != nil {
		t.Fatalf("resolve mapped profile revision: %v", err)
	}
	if revision := routingResolvedProfileRevision(request.Profile); revision != expectedRevision {
		t.Fatalf("applied profile revision = %q, effective-controls revision = %q", revision, expectedRevision)
	}
	if numericPreference(request.Preferences["vehicleWidth"]) != 1.8 ||
		numericPreference(request.Preferences["maxHikingDifficulty"]) != 6 ||
		numericPreference(request.Preferences["hillPreference"]) != 0.9 ||
		numericPreference(request.Preferences["speedPreference"]) != 6.2 {
		t.Fatalf("expected settings, mapping, and request preferences to merge, got %#v", request.Preferences)
	}

	unmapped := pluginRoutingRouteRequest{Category: "Climbing"}
	err = applyRoutingCategoryMapping(app, "", "valhalla", "europe", &unmapped)
	var routingErr *routingError
	if !errors.As(err, &routingErr) || routingErr.Code != "mapping_missing" || routingErr.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("expected stable mapping_missing error for an unmapped category, got %#v", err)
	}
}

func TestManifestCategoryMappingsUseSubcategoryThenCategoryFallback(t *testing.T) {
	t.Chdir("..")
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	if err := createTestRoutingCollection(app, testRoutingProfileMappingsCollectionJSON); err != nil {
		t.Fatalf("create mappings collection: %v", err)
	}

	gravel, err := ResolveRoutingProfileMapping(app, "", "valhalla", "default", "Biking", "Gravel")
	if err != nil {
		t.Fatalf("resolve gravel mapping: %v", err)
	}
	if gravel == nil || gravel.NativeProfileKey != "bicycle" || gravel.NativeConfig["bicycle_type"] != "Cross" {
		t.Fatalf("expected exact Biking/Gravel manifest mapping, got %#v", gravel)
	}

	touring, err := ResolveRoutingProfileMapping(app, "", "valhalla", "default", "Biking", "Touring")
	if err != nil {
		t.Fatalf("resolve touring mapping: %v", err)
	}
	if touring == nil || touring.Subcategory != "" || touring.NativeConfig["bicycle_type"] != "Hybrid" {
		t.Fatalf("expected Biking category fallback, got %#v", touring)
	}
}

func TestEffectiveControlsReturnGenericStandardAndNativeMetadata(t *testing.T) {
	t.Chdir("..")
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	for _, raw := range []string{testRoutingSettingsCollectionJSON, testRoutingProfileMappingsCollectionJSON} {
		if err := createTestRoutingCollection(app, raw); err != nil {
			t.Fatalf("create routing collection: %v", err)
		}
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"primaryRoutePluginId": "valhalla",
		"defaultPreferences":   map[string]any{"speedPreference": 20.0},
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	input := routingEffectiveControlsInput{Category: "Biking", Subcategory: "Gravel"}
	input.Routing.Engines = []routingEngineSelection{{PluginID: "valhalla", InstanceID: "default"}}
	output, err := ResolveRoutingEffectiveControls(app, "", input)
	if err != nil {
		t.Fatalf("resolve effective controls: %v", err)
	}
	if output.Mode != "bike" || len(output.Controls) == 0 || len(output.NativeControlGroups) == 0 {
		t.Fatalf("effective metadata is incomplete: %#v", output)
	}
	foundTypedNativeControl := false
	for _, group := range output.NativeControlGroups {
		for _, control := range group.Controls {
			if control.Type == "boolean" || (control.Type == "number" && control.Min != nil && control.Max != nil && control.Step != nil) {
				foundTypedNativeControl = true
			}
		}
	}
	if !foundTypedNativeControl {
		t.Fatalf("typed native metadata missing: %#v", output.NativeControlGroups)
	}
}

func TestRoutingProfileValidationIsProviderNeutral(t *testing.T) {
	custom := routingProfileView{
		PluginID:      "custom-routing",
		Key:           "my-profile",
		Name:          "My profile",
		Kind:          "custom_file",
		Mode:          "bike",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("opaque-provider-payload")),
		ContentType:   "text/plain",
		Enabled:       true,
	}
	if err := validateRoutingProfileInput(custom); err != nil {
		t.Fatalf("expected opaque custom profile to validate, got %v", err)
	}

	oversized := custom
	oversized.ContentBase64 = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", routingProfileContentMaxBytes+1)))
	if err := validateRoutingProfileInput(oversized); err == nil {
		t.Fatal("expected oversized opaque custom profile to fail")
	}

	generated := custom
	generated.Kind = "generated"
	generated.ContentBase64 = ""
	generated.ContentType = ""
	generated.Metadata = map[string]any{"templateKey": "provider-owned-template"}
	generated.NativeConfig = map[string]any{"parameters": map[string]any{"providerKnob": 42.0}}
	if err := validateRoutingProfileInput(generated); err != nil {
		t.Fatalf("expected opaque generated profile metadata to validate, got %v", err)
	}

	invalid := custom
	invalid.Kind = "provider_specific_kind"
	if err := validateRoutingProfileInput(invalid); err == nil {
		t.Fatal("expected unsupported generic profile kind to fail")
	}
}

func TestRoutingNativeControlsFromMetadata(t *testing.T) {
	groups := routingNativeControlGroupsFromMetadata([]any{map[string]any{
		"key":   "template",
		"label": "Template",
		"labels": map[string]any{
			"de": "Vorlage",
			"en": "Template",
		},
		"controls": []any{
			map[string]any{
				"key":   "avoid_unsafe",
				"label": "Avoid unsafe roads",
				"labels": map[string]any{
					"de": "Unsichere Straßen meiden",
					"en": "Avoid unsafe roads",
				},
				"type":      "boolean",
				"valueType": "boolean",
				"default":   false,
				"path":      []any{"parameters", "avoid_unsafe"},
			},
			map[string]any{
				"key":       "uphillcost",
				"label":     "Uphill cost",
				"type":      "number",
				"ui":        "slider",
				"valueType": "number",
				"min":       0.0,
				"max":       120.0,
				"step":      5.0,
				"default":   15.0,
				"path":      []any{"parameters", "uphillcost"},
			},
		},
	}}, map[string]any{
		"parameters": map[string]any{
			"avoid_unsafe": true,
			"uphillcost":   25.0,
		},
	})
	if len(groups) != 1 || len(groups[0].Controls) != 2 {
		t.Fatalf("expected controls from generic metadata, got %#v", groups)
	}
	if groups[0].Controls[0].Current != true || groups[0].Controls[1].Current != 25.0 {
		t.Fatalf("expected current values from native config, got %#v", groups[0].Controls)
	}
	if groups[0].Labels["de"] != "Vorlage" || groups[0].Controls[0].Labels["de"] != "Unsichere Straßen meiden" {
		t.Fatalf("expected localized labels from metadata, got %#v", groups)
	}
}

func numericPreference(value any) float64 {
	switch value := value.(type) {
	case int:
		return float64(value)
	case float64:
		return value
	default:
		return 0
	}
}

func createTestRoutingCollection(app core.App, raw string) error {
	var collection core.Collection
	if err := json.Unmarshal([]byte(raw), &collection); err != nil {
		return err
	}
	return app.Save(&collection)
}

const testRoutingSettingsCollectionJSON = `{
	"id": "test_rt_settings",
	"name": "routing_settings",
	"type": "base",
	"fields": [
		{"id":"rts_id","name":"id","type":"text","system":true,"required":true,"primaryKey":true,"autogeneratePattern":"[a-z0-9]{15}","min":15,"max":15,"pattern":"^[a-z0-9]+$"},
		{"id":"rts_scope","name":"scope","type":"select","required":true,"maxSelect":1,"values":["builtin","admin","user"]},
		{"id":"rts_user","name":"user","type":"text","required":false},
		{"id":"rts_config","name":"config","type":"json","required":false,"maxSize":2000000}
	]
}`

const testRoutingProfileMappingsCollectionJSON = `{
	"id": "test_rt_mappings",
	"name": "routing_profile_mappings",
	"type": "base",
	"fields": [
		{"id":"rtm_id","name":"id","type":"text","system":true,"required":true,"primaryKey":true,"autogeneratePattern":"[a-z0-9]{15}","min":15,"max":15,"pattern":"^[a-z0-9]+$"},
		{"id":"rtm_scope","name":"scope","type":"select","required":true,"maxSelect":1,"values":["admin","user"]},
		{"id":"rtm_user","name":"user","type":"text","required":false},
		{"id":"rtm_category","name":"category","type":"text","required":true,"min":1,"max":256},
		{"id":"rtm_subcategory","name":"subcategory","type":"text","required":false,"max":256},
		{"id":"rtm_plugin","name":"plugin_id","type":"text","required":true,"min":1,"max":96},
		{"id":"rtm_instance","name":"instance_id","type":"text","required":false,"max":64},
		{"id":"rtm_pkey","name":"native_profile_key","type":"text","required":false,"max":128},
		{"id":"rtm_profile","name":"profile","type":"text","required":false},
		{"id":"rtm_prefs","name":"preferences","type":"json","required":false,"maxSize":2000000},
		{"id":"rtm_native","name":"native_config","type":"json","required":false,"maxSize":2000000}
	]
}`

const testRoutingProfilesCollectionJSON = `{
	"id": "test_rt_profiles",
	"name": "routing_profiles",
	"type": "base",
	"fields": [
		{"id":"rtp_id","name":"id","type":"text","system":true,"required":true,"primaryKey":true,"autogeneratePattern":"[a-z0-9]{15}","min":15,"max":15,"pattern":"^[a-z0-9]+$"},
		{"id":"rtp_scope","name":"scope","type":"select","required":true,"maxSelect":1,"values":["admin","user"]},
		{"id":"rtp_user","name":"user","type":"text","required":false},
		{"id":"rtp_plugin","name":"plugin_id","type":"text","required":false,"max":96},
		{"id":"rtp_key","name":"key","type":"text","required":false,"max":128},
		{"id":"rtp_name","name":"name","type":"text","required":false,"max":256},
		{"id":"rtp_kind","name":"kind","type":"select","required":true,"maxSelect":1,"values":["custom_file","generated","native_config"]},
		{"id":"rtp_mode","name":"mode","type":"select","required":false,"maxSelect":1,"values":["foot","bike","motor","mixed","other"]},
		{"id":"rtp_content","name":"content_base64","type":"text","required":false,"max":349528},
		{"id":"rtp_content_type","name":"content_type","type":"text","required":false,"max":128},
		{"id":"rtp_metadata","name":"metadata","type":"json","required":false,"maxSize":2000000},
		{"id":"rtp_native","name":"native_config","type":"json","required":false,"maxSize":2000000},
		{"id":"rtp_enabled","name":"enabled","type":"bool","required":false}
	]
}`

// The settings-wide default preferences are mode-agnostic, so a walking speed
// must not survive into a bike or car request.
func TestClampRoutingPreferencesToMode(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		preferences map[string]any
		want        map[string]any
	}{
		{
			name:        "a walking speed is lifted into the car range",
			mode:        "motor",
			preferences: map[string]any{"speedPreference": 5.1},
			want:        map[string]any{"speedPreference": 10.0},
		},
		{
			name:        "a car speed is capped for a cyclist",
			mode:        "bike",
			preferences: map[string]any{"speedPreference": 160.0},
			want:        map[string]any{"speedPreference": 50.0},
		},
		{
			name:        "values inside the range are untouched",
			mode:        "foot",
			preferences: map[string]any{"speedPreference": 5.1, "hillPreference": 0.5},
			want:        map[string]any{"speedPreference": 5.1, "hillPreference": 0.5},
		},
		{
			name:        "provider specific values are left alone",
			mode:        "bike",
			preferences: map[string]any{"customProviderKnob": 999.0},
			want:        map[string]any{"customProviderKnob": 999.0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clampRoutingPreferencesToMode(test.preferences, test.mode)
			for key, expected := range test.want {
				if test.preferences[key] != expected {
					t.Fatalf("%s = %#v, want %#v", key, test.preferences[key], expected)
				}
			}
		})
	}
}

func TestDefaultRoutingPreferenceValueFollowsTheMode(t *testing.T) {
	for mode, want := range map[string]any{"foot": 5.1, "bike": 20.0, "motor": 100.0} {
		if got := defaultRoutingPreferenceValue("speedPreference", mode); got != want {
			t.Fatalf("default speed for %s = %#v, want %#v", mode, got, want)
		}
	}
	if got := defaultRoutingPreferenceValue("hillPreference", "motor"); got != 0.5 {
		t.Fatalf("hill preference default = %#v, want 0.5", got)
	}
}

// The production merge path starts from the settings-wide defaults, which are
// seeded before any travel mode is known. A walking speed persisted there must
// not reach a motor profile, where it would overwrite the profile's own target
// speed.
func TestSettingsDefaultsDoNotCarryASpeedAcrossModes(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	for _, raw := range []string{
		testRoutingSettingsCollectionJSON,
		testRoutingProfilesCollectionJSON,
		testRoutingProfileMappingsCollectionJSON,
	} {
		if err := createTestRoutingCollection(app, raw); err != nil {
			t.Fatalf("create routing collection: %v", err)
		}
	}
	// An installation seeded before speeds became mode-specific.
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"defaultPreferences": map[string]any{"speedPreference": 5.1, "hillPreference": 0.5},
	}); err != nil {
		t.Fatalf("seed builtin routing settings: %v", err)
	}

	profiles, err := app.FindCollectionByNameOrId("routing_profiles")
	if err != nil {
		t.Fatalf("find profiles collection: %v", err)
	}
	profile := core.NewRecord(profiles)
	profile.Set("scope", routingScopeAdmin)
	profile.Set("plugin_id", "brouter")
	profile.Set("key", "custom-car")
	profile.Set("name", "Custom car")
	profile.Set("kind", "custom_file")
	profile.Set("mode", "motor")
	profile.Set("content_base64", "cHJvZmlsZQ==")
	profile.Set("content_type", "text/plain")
	profile.Set("metadata", map[string]any{"supportedPreferences": []string{"speedPreference"}})
	profile.Set("enabled", true)
	if err := app.Save(profile); err != nil {
		t.Fatalf("save motor profile: %v", err)
	}

	mappings, err := app.FindCollectionByNameOrId("routing_profile_mappings")
	if err != nil {
		t.Fatalf("find mappings collection: %v", err)
	}
	mappingRecord := core.NewRecord(mappings)
	mappingRecord.Set("scope", routingScopeAdmin)
	setRoutingProfileMappingRecord(mappingRecord, routingProfileMappingView{
		Scope:     routingScopeAdmin,
		Category:  "Other",
		PluginID:  "brouter",
		ProfileID: profile.Id,
	})
	if err := app.Save(mappingRecord); err != nil {
		t.Fatalf("save motor mapping: %v", err)
	}

	request := pluginRoutingRouteRequest{Category: "Other"}
	if err := applyRoutingCategoryMapping(app, "", "brouter", "default", &request); err != nil {
		t.Fatalf("apply motor mapping: %v", err)
	}
	if request.Mode != "motor" {
		t.Fatalf("resolved mode = %q, want motor", request.Mode)
	}
	if value, found := request.Preferences["speedPreference"]; found {
		t.Fatalf("a mode-agnostic default leaked a speed into a motor route: %#v", value)
	}
}

// A numeric control that carries options is rendered as a select, so its value
// comes back as the option's string form. It has to survive validation and
// reach the provider as the number the control declares.
func TestNumericSelectControlsSurviveValidationAsNumbers(t *testing.T) {
	metadata := map[string]any{
		"key":       "SAC_scale_limit",
		"valueType": "number",
		"options": []any{
			map[string]any{"value": "1", "label": "SAC T1"},
			map[string]any{"value": "3", "label": "SAC T3"},
		},
	}

	normalized, err := validateRoutingNativeControlValue(metadata, "3")
	if err != nil {
		t.Fatalf("selecting a declared option failed validation: %v", err)
	}
	if normalized != 3.0 {
		t.Fatalf("normalized value = %#v, want 3.0", normalized)
	}

	if normalized, err = validateRoutingNativeControlValue(metadata, 1.0); err != nil {
		t.Fatalf("a numeric option value failed validation: %v", err)
	}
	if normalized != 1.0 {
		t.Fatalf("normalized value = %#v, want 1.0", normalized)
	}

	if _, err := validateRoutingNativeControlValue(metadata, "5"); err == nil {
		t.Fatal("expected a value outside the declared options to be rejected")
	}
	if _, err := validateRoutingNativeControlValue(map[string]any{
		"valueType": "string",
		"options":   []any{map[string]any{"value": "01"}},
	}, "1"); err == nil {
		t.Fatal("numeric-looking string options must still match exactly")
	}
}

func TestAllowlistedNativeConfigNormalizesSelectedOptions(t *testing.T) {
	groups := []any{map[string]any{
		"key": "template",
		"controls": []any{map[string]any{
			"key":       "hills",
			"valueType": "number",
			"path":      []any{"parameters", "hills"},
			"options": []any{
				map[string]any{"value": "0", "label": "none"},
				map[string]any{"value": "2", "label": "avoid slopes"},
			},
		}},
	}}

	config, err := allowlistedRoutingNativeConfig(
		map[string]any{"parameters": map[string]any{"hills": "2"}}, groups,
	)
	if err != nil {
		t.Fatalf("allowlist rejected a selected option: %v", err)
	}
	parameters, ok := config["parameters"].(map[string]any)
	if !ok || parameters["hills"] != 2.0 {
		t.Fatalf("stored native config = %#v, want hills as the number 2", config)
	}
}

func TestNumericSelectPresentationUsesDeclaredOptionStrings(t *testing.T) {
	controls := routingControlsFromMetadata([]any{map[string]any{
		"key":       "SAC_scale_limit",
		"type":      "number",
		"valueType": "number",
		"default":   3.0,
		"path":      []any{"parameters", "SAC_scale_limit"},
		"options": []any{
			map[string]any{"value": "1", "label": "SAC T1"},
			map[string]any{"value": "3", "label": "SAC T3"},
		},
	}}, map[string]any{
		"parameters": map[string]any{"SAC_scale_limit": 1.0},
	})
	if len(controls) != 1 {
		t.Fatalf("controls = %#v, want one numeric select", controls)
	}
	control := controls[0]
	if control.Type != "string" || control.ValueType != "string" || control.UI != "select" {
		t.Fatalf("presentation contract = %#v, want a string select", control)
	}
	if control.Default != "3" || control.Current != "1" {
		t.Fatalf("default/current = %#v/%#v, want declared option strings 3/1", control.Default, control.Current)
	}
}

func TestEffectiveControlsUseProfileDefaultWithoutMaterializingIt(t *testing.T) {
	t.Chdir("..")
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()
	for _, raw := range []string{testRoutingSettingsCollectionJSON, testRoutingProfileMappingsCollectionJSON} {
		if err := createTestRoutingCollection(app, raw); err != nil {
			t.Fatalf("create routing collection: %v", err)
		}
	}
	// This legacy walking value also verifies that settings filtering happens
	// after the mapped profile has supplied its motor mode.
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"defaultPreferences": map[string]any{"speedPreference": 5.1},
	}); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	mappings, err := app.FindCollectionByNameOrId("routing_profile_mappings")
	if err != nil {
		t.Fatalf("find mappings collection: %v", err)
	}
	mappingRecord := core.NewRecord(mappings)
	mappingRecord.Set("scope", routingScopeAdmin)
	setRoutingProfileMappingRecord(mappingRecord, routingProfileMappingView{
		Scope:            routingScopeAdmin,
		Category:         "Driving",
		PluginID:         "brouter",
		NativeProfileKey: "car-vario",
	})
	if err := app.Save(mappingRecord); err != nil {
		t.Fatalf("save car mapping: %v", err)
	}

	input := routingEffectiveControlsInput{Category: "Driving"}
	input.Routing.Engines = []routingEngineSelection{{PluginID: "brouter", InstanceID: "default"}}
	output, err := ResolveRoutingEffectiveControls(app, "", input)
	if err != nil {
		t.Fatalf("resolve effective car controls: %v", err)
	}
	var speed *routingControlView
	for index := range output.Controls {
		if output.Controls[index].Key == "speedPreference" {
			speed = &output.Controls[index]
			break
		}
	}
	if speed == nil {
		t.Fatalf("speed control missing from %#v", output.Controls)
	}
	if speed.Default != 90.0 || speed.Current != nil {
		t.Fatalf("speed default/current = %#v/%#v, want profile fallback 90 and no explicit value", speed.Default, speed.Current)
	}

	request := pluginRoutingRouteRequest{Category: "Driving"}
	if err := applyRoutingCategoryMapping(app, "", "brouter", "default", &request); err != nil {
		t.Fatalf("apply car mapping: %v", err)
	}
	if _, explicit := request.Preferences["speedPreference"]; explicit {
		t.Fatalf("display fallback leaked into route preferences: %#v", request.Preferences)
	}

	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"defaultPreferences": map[string]any{"speedPreference": 70.0},
	}); err != nil {
		t.Fatalf("update settings default: %v", err)
	}
	mappingRecord.Set("preferences", map[string]any{"speedPreference": 80.0})
	if err := app.Save(mappingRecord); err != nil {
		t.Fatalf("update mapping preferences: %v", err)
	}
	output, err = ResolveRoutingEffectiveControls(app, "", input)
	if err != nil {
		t.Fatalf("resolve configured car controls: %v", err)
	}
	speed = nil
	for index := range output.Controls {
		if output.Controls[index].Key == "speedPreference" {
			speed = &output.Controls[index]
			break
		}
	}
	if speed == nil || speed.Default != 90.0 || speed.Current != 80.0 {
		t.Fatalf("configured speed default/current = %#v, want profile 90 < settings 70 < mapping 80", speed)
	}
}

func TestSharedProfilePreferenceDefaultRequiresConsensus(t *testing.T) {
	if value, ok := sharedRoutingProfilePreferenceDefault([]map[string]any{
		{"speedPreference": 90},
		{"speedPreference": 90.0},
	}, "speedPreference"); !ok || value != 90 {
		t.Fatalf("equal profile defaults did not reach consensus: %#v, %v", value, ok)
	}
	for _, defaults := range [][]map[string]any{
		{{"speedPreference": 90.0}, {"speedPreference": 100.0}},
		{{"speedPreference": 90.0}, nil},
	} {
		if value, ok := sharedRoutingProfilePreferenceDefault(defaults, "speedPreference"); ok {
			t.Fatalf("non-consensus profile defaults returned %#v", value)
		}
	}
}

func TestManifestProfilePreferenceDefaultsAreCanonicalAndSupported(t *testing.T) {
	defaults := routingManifestProfilePreferenceDefaults(map[string]any{
		"supportedPreferences": []any{"speedPreference"},
		"preferenceDefaults": map[string]any{
			"speedPreference":    90,
			"hillPreference":     0.5,
			"providerPreference": 1,
		},
	}, "motor")
	if defaults["speedPreference"] != 90.0 || len(defaults) != 1 {
		t.Fatalf("accepted profile defaults = %#v, want only canonical supported speed 90", defaults)
	}
}
