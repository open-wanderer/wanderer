package pluginhost

import (
	"reflect"
	"testing"

	"pocketbase/pluginsystem"
)

func TestInstanceConfigOverridesUsesHostAllowlist(t *testing.T) {
	raw := map[string]any{
		"plugin": map[string]any{"url": "https://a.example", "custom": true},
		"host": map[string]any{
			"planned":                     true,
			"completed":                   false,
			"privacy":                     "settings",
			"createSummitLogForCompleted": false,
			"categoryMapping":             map[string]any{},
			"categoryMappingUpdatedAt":    "2026-08-05T00:00:00Z",
			"connectors":                  instanceConnectorConfig(),
			"futureHostOption":            true,
			"merge": map[string]any{
				"enabled":   true,
				"available": false,
				"future":    "blocked",
			},
		},
		"futureTopLevel": map[string]any{"blocked": true},
	}

	got := InstanceConfigOverrides(raw)
	want := map[string]any{
		"plugin": map[string]any{"url": "https://a.example", "custom": true},
		"host": map[string]any{
			"planned":                     true,
			"completed":                   false,
			"privacy":                     "settings",
			"createSummitLogForCompleted": false,
			"categoryMapping":             map[string]any{},
			"categoryMappingUpdatedAt":    "2026-08-05T00:00:00Z",
			"merge":                       map[string]any{"enabled": true},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projected config mismatch\ngot:  %#v\nwant: %#v", got, want)
	}

	rawPlugin := raw["plugin"].(map[string]any)
	rawPlugin["url"] = "https://b.example"
	if got["plugin"].(map[string]any)["url"] != "https://a.example" {
		t.Fatal("projected plugin config aliases caller-owned config")
	}
}

func TestEffectiveConfigKeepsConnectorPolicyAdministrative(t *testing.T) {
	installed := administrativeConfig("https://a.example", false)
	instance := map[string]any{
		"plugin": map[string]any{"url": "https://b.example"},
		"host": map[string]any{
			"connectors": instanceConnectorConfig(),
			"merge":      map[string]any{"enabled": true, "available": false},
		},
	}

	got := mergeEffectiveConfig(installed, instance)
	if RuntimeConfig(got)["url"] != "https://b.example" {
		t.Fatal("plugin-owned instance setting was not applied")
	}
	host := HostConfig(got)
	merge := host["merge"].(map[string]any)
	if merge["enabled"] != true || merge["available"] != true {
		t.Fatalf("nested merge policy mismatch: %#v", merge)
	}

	plugin := configuredTrailPlugin()
	connector := InstancePolicy(plugin, got).Connectors["api"]
	if connector.BaseURL != "https://a.example" || connector.AllowPrivate {
		t.Fatalf("instance replaced administrative connector target: %#v", connector)
	}
	if connector.TLS.Mode != pluginsystem.TLSModeCustomCA || string(connector.TLS.CABundle) != "admin-ca" {
		t.Fatalf("administrative TLS settings changed: %#v", connector.TLS)
	}
	if len(connector.StorageOrigins) != 1 || connector.StorageOrigins["admin-storage"].BaseURL != "https://a.example" {
		t.Fatalf("instance replaced administrative storage origins: %#v", connector.StorageOrigins)
	}
	if _, ok := connector.StorageOrigins["instance-storage"]; ok {
		t.Fatal("instance storage origin reached runtime policy")
	}
}

func TestStoredInstanceConfigDoesNotFreezeAdminConnector(t *testing.T) {
	oldAdmin := administrativeConfig("https://a.example", false)
	current := map[string]any{
		"plugin": map[string]any{"url": "https://b.example"},
		"host": map[string]any{
			"connectors": instanceConnectorConfig(),
		},
	}

	stored := MergeInstanceConfigDefaults(oldAdmin, current)
	if connectors := HostConfig(stored)["connectors"]; connectors != nil {
		t.Fatalf("connector snapshot persisted in instance config: %#v", connectors)
	}

	newAdmin := administrativeConfig("https://b.example", true)
	effective := mergeEffectiveConfig(newAdmin, stored)
	connector := InstancePolicy(configuredTrailPlugin(), effective).Connectors["api"]
	if connector.BaseURL != "https://b.example" || !connector.AllowPrivate {
		t.Fatalf("updated administrative connector did not take effect: %#v", connector)
	}
}

func administrativeConfig(baseURL string, allowPrivate bool) map[string]any {
	return map[string]any{
		"plugin": map[string]any{"url": "https://a.example"},
		"host": map[string]any{
			"merge": map[string]any{"available": true, "enabled": false},
			"connectors": map[string]any{
				"provider": map[string]any{
					"baseURL":      baseURL,
					"allowPrivate": allowPrivate,
					"tls": map[string]any{
						"mode":     pluginsystem.TLSModeCustomCA,
						"caBundle": "admin-ca",
					},
					"storageOrigins": map[string]any{
						"admin-storage": map[string]any{
							"baseURL": "https://a.example",
						},
					},
				},
			},
		},
	}
}

func instanceConnectorConfig() map[string]any {
	return map[string]any{
		"provider": map[string]any{
			"baseURL":      "https://b.example",
			"allowPrivate": true,
			"tls":          map[string]any{"mode": pluginsystem.TLSModeCustomCA, "caBundle": "instance-ca"},
			"storageOrigins": map[string]any{
				"instance-storage": map[string]any{
					"baseURL":      "https://b.example",
					"allowPrivate": true,
				},
			},
		},
	}
}

func configuredTrailPlugin() pluginsystem.LocalPlugin {
	return pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
		Permissions: pluginsystem.PermissionManifest{
			Network: pluginsystem.NetworkPermissions{
				Connectors: []pluginsystem.ConnectorTargetPermission{{
					Name:                     "api",
					Type:                     pluginsystem.ConnectorTypeConfigured,
					ConfigKey:                "provider",
					AllowedPathPrefixes:      []string{"/api"},
					SupportsStorageRedirects: true,
					SupportsCustomTLS:        true,
				}},
			},
		},
	}}
}
