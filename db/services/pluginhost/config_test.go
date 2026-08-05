package pluginhost

import (
	"reflect"
	"testing"

	"pocketbase/pluginsystem"
)

func TestInstanceConfigOverridesUsesHostAllowlist(t *testing.T) {
	raw := map[string]any{
		"plugin": map[string]any{"url": "https://photos.example.test", "custom": true},
		"host": map[string]any{
			"planned":                       true,
			"completed":                     false,
			"privacy":                       "settings",
			"createSummitLogForCompleted":   false,
			"categoryMapping":               map[string]any{},
			"categoryMappingUpdatedAt":      "2026-08-05T00:00:00Z",
			"photoMode":                     "link_private",
			"maxPhotosPerTrail":             8,
			"maxPhotosPerWaypoint":          4,
			"maxPhotosPerSummitLog":         6,
			"connectors":                    maliciousConnectorConfig(),
			"futureSecuritySensitiveOption": true,
			"merge": map[string]any{
				"enabled":   true,
				"available": false,
				"future":    "blocked",
			},
			"autoAttach": map[string]any{
				"trailPlugins": false,
				"upload":       true,
				"future":       "blocked",
			},
		},
		"futureTopLevel": map[string]any{"blocked": true},
	}

	got := InstanceConfigOverrides(raw)
	want := map[string]any{
		"plugin": map[string]any{"url": "https://photos.example.test", "custom": true},
		"host": map[string]any{
			"planned":                     true,
			"completed":                   false,
			"privacy":                     "settings",
			"createSummitLogForCompleted": false,
			"categoryMapping":             map[string]any{},
			"categoryMappingUpdatedAt":    "2026-08-05T00:00:00Z",
			"photoMode":                   "link_private",
			"maxPhotosPerTrail":           float64(8),
			"maxPhotosPerWaypoint":        float64(4),
			"maxPhotosPerSummitLog":       float64(6),
			"merge":                       map[string]any{"enabled": true},
			"autoAttach":                  map[string]any{"trailPlugins": false, "upload": true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projected config mismatch\ngot:  %#v\nwant: %#v", got, want)
	}

	rawPlugin := raw["plugin"].(map[string]any)
	rawPlugin["url"] = "https://changed.example.test"
	if got["plugin"].(map[string]any)["url"] != "https://photos.example.test" {
		t.Fatal("projected plugin config aliases caller-owned config")
	}
}

func TestEffectiveConfigKeepsConnectorPolicyAdministrative(t *testing.T) {
	installed := administrativeConfig("https://admin.example.test", false)
	instance := map[string]any{
		"plugin": map[string]any{"url": "https://user.example.test"},
		"host": map[string]any{
			"photoMode":  "link_private",
			"connectors": maliciousConnectorConfig(),
			"merge":      map[string]any{"enabled": true, "available": false},
		},
	}

	got := mergeEffectiveConfig(installed, instance)
	if RuntimeConfig(got)["url"] != "https://user.example.test" {
		t.Fatal("plugin-owned instance setting was not applied")
	}
	host := HostConfig(got)
	if host["photoMode"] != "link_private" {
		t.Fatal("allowed host instance setting was not applied")
	}
	merge := host["merge"].(map[string]any)
	if merge["enabled"] != true || merge["available"] != true {
		t.Fatalf("nested merge policy mismatch: %#v", merge)
	}

	plugin := configuredMediaPlugin()
	connector := InstancePolicy(plugin, got).Connectors["api"]
	if connector.BaseURL != "https://admin.example.test" || connector.AllowPrivate {
		t.Fatalf("instance replaced administrative connector target: %#v", connector)
	}
	if len(connector.StorageOrigins) != 1 || connector.StorageOrigins["admin-storage"].BaseURL != "https://storage.example.test" {
		t.Fatalf("instance replaced administrative storage origins: %#v", connector.StorageOrigins)
	}
	if _, ok := connector.StorageOrigins["internal"]; ok {
		t.Fatal("instance-injected internal storage origin reached runtime policy")
	}
}

func TestStoredInstanceConfigDoesNotFreezeAdminConnector(t *testing.T) {
	oldAdmin := administrativeConfig("https://old.example.test", false)
	current := map[string]any{
		"plugin": map[string]any{"url": "https://user.example.test"},
		"host": map[string]any{
			"photoMode":  "copy",
			"connectors": maliciousConnectorConfig(),
		},
	}

	stored := MergeInstanceConfigDefaults(oldAdmin, current)
	if connectors := HostConfig(stored)["connectors"]; connectors != nil {
		t.Fatalf("connector snapshot persisted in instance config: %#v", connectors)
	}

	newAdmin := administrativeConfig("https://new.example.test", true)
	effective := mergeEffectiveConfig(newAdmin, stored)
	connector := InstancePolicy(configuredMediaPlugin(), effective).Connectors["api"]
	if connector.BaseURL != "https://new.example.test" || !connector.AllowPrivate {
		t.Fatalf("updated administrative connector did not take effect: %#v", connector)
	}
}

func TestUserSelectedAssetConnectorDoesNotInheritAdminTrust(t *testing.T) {
	plugin := configuredMediaPlugin()
	config := AssetConnectorConfig(plugin, administrativeConfig("", true))
	connector := InstancePolicy(plugin, config).Connectors["api"]

	if connector.BaseURL != "https://default.example.test" {
		t.Fatalf("user-selected connector URL mismatch: %#v", connector)
	}
	if connector.AllowPrivate {
		t.Fatal("user-selected connector inherited private-network access")
	}
	if connector.TLS.Mode != pluginsystem.TLSModeSystem || len(connector.TLS.CABundle) != 0 {
		t.Fatalf("user-selected connector inherited custom TLS trust: %#v", connector.TLS)
	}
	if len(connector.StorageOrigins) != 0 {
		t.Fatalf("user-selected connector inherited storage origins: %#v", connector.StorageOrigins)
	}
}

func TestFixedAdminAssetConnectorKeepsAdminTrust(t *testing.T) {
	plugin := configuredMediaPlugin()
	config := AssetConnectorConfig(plugin, administrativeConfig("https://admin.example.test", true))
	connector := InstancePolicy(plugin, config).Connectors["api"]

	if connector.BaseURL != "https://admin.example.test" || !connector.AllowPrivate {
		t.Fatalf("fixed administrative connector trust mismatch: %#v", connector)
	}
	if connector.TLS.Mode != pluginsystem.TLSModeCustomCA || string(connector.TLS.CABundle) != "admin-ca" {
		t.Fatalf("fixed administrative connector lost custom TLS trust: %#v", connector.TLS)
	}
	if len(connector.StorageOrigins) != 1 || connector.StorageOrigins["admin-storage"].BaseURL != "https://storage.example.test" {
		t.Fatalf("fixed administrative connector lost storage origins: %#v", connector.StorageOrigins)
	}
}

func administrativeConfig(baseURL string, allowPrivate bool) map[string]any {
	return map[string]any{
		"plugin": map[string]any{"url": "https://default.example.test"},
		"host": map[string]any{
			"photoMode": "copy",
			"merge":     map[string]any{"available": true, "enabled": false},
			"connectors": map[string]any{
				"immich": map[string]any{
					"baseURL":      baseURL,
					"allowPrivate": allowPrivate,
					"tls": map[string]any{
						"mode":     pluginsystem.TLSModeCustomCA,
						"caBundle": "admin-ca",
					},
					"storageOrigins": map[string]any{
						"admin-storage": map[string]any{
							"baseURL": "https://storage.example.test",
						},
					},
				},
			},
		},
	}
}

func maliciousConnectorConfig() map[string]any {
	return map[string]any{
		"immich": map[string]any{
			"baseURL":      "http://127.0.0.1:8090",
			"allowPrivate": true,
			"storageOrigins": map[string]any{
				"internal": map[string]any{
					"baseURL":      "http://169.254.169.254",
					"allowPrivate": true,
				},
			},
		},
	}
}

func configuredMediaPlugin() pluginsystem.LocalPlugin {
	return pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
		Permissions: pluginsystem.PermissionManifest{
			Network: pluginsystem.NetworkPermissions{
				Connectors: []pluginsystem.ConnectorTargetPermission{{
					Name:                     "api",
					Type:                     pluginsystem.ConnectorTypeConfigured,
					ConfigKey:                "immich",
					AllowedPathPrefixes:      []string{"/api"},
					SupportsStorageRedirects: true,
					SupportsCustomTLS:        true,
				}},
			},
		},
	}}
}
