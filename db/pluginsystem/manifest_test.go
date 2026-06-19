package pluginsystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateManifestAcceptsHammerheadShape(t *testing.T) {
	manifest := hammerheadManifestForTest()
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateManifestRejectsUnknownAuthPermission(t *testing.T) {
	manifest := hammerheadManifestForTest()
	manifest.Permissions.Auth = []string{"missing"}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadLocalPluginRequiresRelativeEntrypoint(t *testing.T) {
	dir := t.TempDir()
	manifest := hammerheadManifestForTest()
	manifest.Runtime.Entrypoint = "/tmp/plugin.wasm"
	writeManifest(t, dir, manifest)

	if _, err := LoadLocalPlugin(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadLocalPluginsSkipsMissingPluginDir(t *testing.T) {
	plugins, err := LoadLocalPlugins(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("got %d plugins, want 0", len(plugins))
	}
}

func TestLoadLocalPluginsFindsDirectChildPlugins(t *testing.T) {
	root := t.TempDir()
	writePluginDir(t, root, "hammerhead")
	writePluginDir(t, root, "komoot")

	plugins, err := LoadLocalPlugins(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("got %d plugins, want 2", len(plugins))
	}
}

func TestDiscoverLocalPluginsReportsMissingManifest(t *testing.T) {
	root := t.TempDir()
	brokenDir := filepath.Join(root, "komoot")
	if err := os.MkdirAll(brokenDir, 0o700); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}

	plugins, issues, err := DiscoverLocalPlugins(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("got %d plugins, want 0", len(plugins))
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].ID != "komoot" || issues[0].Name != "komoot" || issues[0].Dir != brokenDir {
		t.Fatalf("unexpected issue: %#v", issues[0])
	}
	if issues[0].Error == "" || !strings.Contains(issues[0].Error, "plugin.json") {
		t.Fatalf("expected useful plugin.json error, got %#v", issues[0])
	}
}

func TestLoadLocalPluginsIgnoresMissingManifestForCompatibility(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "komoot"), 0o700); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}

	plugins, err := LoadLocalPlugins(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("got %d plugins, want 0", len(plugins))
	}
}

func TestLoadLocalPluginsSkipsUnsupportedPluginTypes(t *testing.T) {
	root := t.TempDir()
	writePluginDir(t, root, "hammerhead")

	assetsDir := filepath.Join(root, "immich")
	if err := os.MkdirAll(assetsDir, 0o700); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifest := hammerheadManifestForTest()
	manifest.ID = "immich"
	manifest.Name = "Immich"
	manifest.Type = "assets"
	writeManifest(t, assetsDir, manifest)
	if err := os.WriteFile(filepath.Join(assetsDir, "plugin.wasm"), []byte("wasm"), 0o600); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	plugins, err := LoadLocalPlugins(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("got %d plugins, want 1", len(plugins))
	}
	if plugins[0].Manifest.ID != "hammerhead" {
		t.Fatalf("got plugin %q, want hammerhead", plugins[0].Manifest.ID)
	}
}

func TestLoadLocalPluginsDoesNotSearchRecursively(t *testing.T) {
	root := t.TempDir()
	writePluginDir(t, filepath.Join(root, "nested"), "hammerhead")

	plugins, err := LoadLocalPlugins(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plugins) != 0 {
		t.Fatalf("got %d plugins, want 0", len(plugins))
	}
}

func hammerheadManifestForTest() Manifest {
	return Manifest{
		ManifestVersion: ManifestVersion,
		ID:              "hammerhead",
		Type:            PluginTypeTrails,
		Name:            "Hammerhead",
		Version:         "0.1.0",
		Runtime: RuntimeManifest{
			Type:       RuntimeWASM,
			Entrypoint: "plugin.wasm",
		},
		Capabilities: []CapabilityManifest{
			{Name: "prepare_trail_send", Version: "v1", Export: "prepare_trail_send_v1"},
		},
		Auth: AuthManifest{
			Contexts: map[string]AuthContext{
				"provider_session": {
					Type:         AuthTypeSession,
					SecretFields: []string{"email", "password"},
					Refresh: &AuthRefresh{
						Mode:     AuthRefreshModePlugin,
						Function: "refresh_session_v1",
					},
				},
			},
		},
		Permissions: PermissionManifest{
			Network: NetworkPermissions{
				Connectors: []ConnectorTargetPermission{{
					Name:                "api",
					Type:                ConnectorTypePublicAPI,
					FixedBaseURL:        "https://dashboard.hammerhead.io",
					AllowedPathPrefixes: []string{"/v1"},
					Auth:                []string{"provider_session"},
				}},
			},
			Auth: []string{"provider_session"},
			Uploads: UploadPermissions{
				MaxBytes:     10 << 20,
				ContentTypes: []string{"application/gpx+xml", "application/xml"},
			},
		},
	}
}

func writeManifest(t *testing.T, dir string, manifest Manifest) {
	t.Helper()
	data := []byte(`{
		"manifestVersion": "1.0",
		"id": "` + manifest.ID + `",
		"type": "` + manifest.Type + `",
		"name": "` + manifest.Name + `",
		"version": "` + manifest.Version + `",
		"runtime": {
			"type": "` + manifest.Runtime.Type + `",
			"entrypoint": "` + manifest.Runtime.Entrypoint + `"
		},
		"capabilities": [
			{"name": "prepare_trail_send", "version": "v1", "export": "prepare_trail_send_v1"}
		]
	}`)
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), data, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func writePluginDir(t *testing.T, root string, id string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifest := hammerheadManifestForTest()
	manifest.ID = id
	manifest.Name = id
	writeManifest(t, dir, manifest)
	if err := os.WriteFile(filepath.Join(dir, "plugin.wasm"), []byte("wasm"), 0o600); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
}
