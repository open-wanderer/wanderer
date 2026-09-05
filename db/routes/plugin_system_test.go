package routes

import (
	"encoding/json"
	"strings"
	"testing"

	"pocketbase/pluginsystem"
)

func TestPreparePluginInfoResponseRedactsUserDiagnostics(t *testing.T) {
	plugins := []pluginsystem.PluginInfo{{
		ID:             "broken-plugin",
		Path:           "/app/data/plugins/broken-plugin",
		Status:         "error",
		SetupErrorCode: pluginsystem.SetupErrorCodeManifestMissing,
		Error:          "open /app/data/plugins/broken-plugin/plugin.json: no such file or directory",
	}}

	preparePluginInfoResponse(plugins, false)

	if plugins[0].Path != "" {
		t.Fatalf("expected plugin path to be redacted, got %q", plugins[0].Path)
	}
	if plugins[0].Error != "" {
		t.Fatalf("expected plugin error to be redacted, got %q", plugins[0].Error)
	}
	if plugins[0].SetupErrorCode != pluginsystem.SetupErrorCodeManifestMissing {
		t.Fatalf("expected public setup error code to remain, got %q", plugins[0].SetupErrorCode)
	}

	payload, err := json.Marshal(plugins)
	if err != nil {
		t.Fatalf("marshal redacted plugins: %v", err)
	}
	for _, secret := range []string{"/app/data/plugins", "no such file or directory"} {
		if strings.Contains(string(payload), secret) {
			t.Fatalf("redacted payload contains private diagnostic %q: %s", secret, payload)
		}
	}
	var publicPlugins []map[string]any
	if err := json.Unmarshal(payload, &publicPlugins); err != nil {
		t.Fatalf("decode redacted plugins: %v", err)
	}
	if got := publicPlugins[0]["setupErrorCode"]; got != "manifest_missing" {
		t.Fatalf("public JSON setupErrorCode = %#v, want manifest_missing", got)
	}
}

func TestPreparePluginInfoResponsePreservesSuperuserDiagnostics(t *testing.T) {
	plugins := []pluginsystem.PluginInfo{{
		ID:             "broken-plugin",
		Path:           "/app/data/plugins/broken-plugin",
		Status:         "error",
		SetupErrorCode: pluginsystem.SetupErrorCodeManifestMissing,
		Error:          "private diagnostic",
	}}

	preparePluginInfoResponse(plugins, true)

	if plugins[0].Path != "/app/data/plugins/broken-plugin" || plugins[0].Error != "private diagnostic" {
		t.Fatalf("expected superuser diagnostics to remain unchanged: %#v", plugins[0])
	}
	if plugins[0].SetupErrorCode != pluginsystem.SetupErrorCodeManifestMissing {
		t.Fatalf("expected setup error code to remain unchanged, got %q", plugins[0].SetupErrorCode)
	}
}

func TestPreparePluginInfoResponseNormalizesGenericSetupErrorCode(t *testing.T) {
	for _, code := range []pluginsystem.PluginSetupErrorCode{"", "private:/server/path"} {
		t.Run(string(code), func(t *testing.T) {
			plugins := []pluginsystem.PluginInfo{{
				ID:             "broken-plugin",
				Path:           "/app/data/plugins/broken-plugin",
				Status:         "error",
				SetupErrorCode: code,
				Error:          "unexpected private diagnostic",
			}}

			preparePluginInfoResponse(plugins, false)

			if plugins[0].SetupErrorCode != pluginsystem.SetupErrorCodeFailed {
				t.Fatalf("expected generic setup error code, got %q", plugins[0].SetupErrorCode)
			}
			if plugins[0].Path != "" || plugins[0].Error != "" {
				t.Fatalf("expected private diagnostics to be redacted: %#v", plugins[0])
			}
		})
	}
}
