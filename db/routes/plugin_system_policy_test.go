package routes

import (
	"testing"

	"pocketbase/pluginsystem"
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

	policy := pluginInstancePolicy(plugin, config)
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
