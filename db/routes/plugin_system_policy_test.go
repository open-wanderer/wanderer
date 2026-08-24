package routes

import (
	"testing"

	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"
)

func TestPluginInstancePolicyUsesHostConnectorConfig(t *testing.T) {
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
		Permissions: pluginsystem.PermissionManifest{
			Network: pluginsystem.NetworkPermissions{
				Connectors: []pluginsystem.ConnectorTargetPermission{{
					Name:              "media",
					Type:              pluginsystem.ConnectorTypeConfigured,
					ConfigKey:         "immich",
					SupportsCustomTLS: true,
				}},
			},
		},
	}}
	config := map[string]any{
		"plugin": map[string]any{
			"after": "2026-01-01",
		},
		"host": map[string]any{
			"connectors": map[string]any{
				"immich": map[string]any{
					"baseURL":      "https://photos.example.test",
					"basePath":     "/immich",
					"allowPrivate": true,
					"tls": map[string]any{
						"mode":     pluginsystem.TLSModeCustomCA,
						"caBundle": "test-ca",
					},
				},
			},
		},
	}

	policy := pluginhost.InstancePolicy(plugin, config)
	connector, ok := policy.Connectors["media"]
	if !ok {
		t.Fatal("expected configured connector to be resolved from host config")
	}
	if connector.BaseURL != "https://photos.example.test" || connector.BasePath != "/immich" {
		t.Fatalf("unexpected connector base: %#v", connector)
	}
	if !connector.AllowPrivate {
		t.Fatal("expected allowPrivate from host connector config")
	}
	if connector.TLS.Mode != pluginsystem.TLSModeCustomCA || string(connector.TLS.CABundle) != "test-ca" {
		t.Fatalf("unexpected TLS config: %#v", connector.TLS)
	}
}

func TestAssetPluginDraftConfigCannotOverrideConnectorPolicy(t *testing.T) {
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
		Permissions: pluginsystem.PermissionManifest{
			Network: pluginsystem.NetworkPermissions{
				Connectors: []pluginsystem.ConnectorTargetPermission{{
					Name:                     "media",
					Type:                     pluginsystem.ConnectorTypeConfigured,
					ConfigKey:                "immich",
					SupportsStorageRedirects: true,
				}},
			},
		},
	}}
	config := map[string]any{
		"host": map[string]any{
			"connectors": map[string]any{
				"immich": map[string]any{
					"baseURL": "https://admin.example.test",
					"storageOrigins": map[string]any{
						"storage": map[string]any{"baseURL": "https://storage.example.test"},
					},
				},
			},
		},
	}
	submitted := map[string]any{
		"host": map[string]any{
			"photoMode": "link_private",
			"connectors": map[string]any{
				"immich": map[string]any{
					"baseURL":      "http://127.0.0.1:8090",
					"allowPrivate": true,
					"storageOrigins": map[string]any{
						"internal": map[string]any{"baseURL": "http://169.254.169.254", "allowPrivate": true},
					},
				},
			},
		},
	}

	mergeAssetPluginDraftConfig(config, submitted)
	if pluginhost.HostConfig(config)["photoMode"] != "link_private" {
		t.Fatal("allowed draft host setting was not applied")
	}
	connector := pluginhost.InstancePolicy(plugin, config).Connectors["media"]
	if connector.BaseURL != "https://admin.example.test" || connector.AllowPrivate {
		t.Fatalf("draft replaced administrative connector: %#v", connector)
	}
	if len(connector.StorageOrigins) != 1 || connector.StorageOrigins["storage"].BaseURL != "https://storage.example.test" {
		t.Fatalf("draft replaced administrative storage origins: %#v", connector.StorageOrigins)
	}
}
