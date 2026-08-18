package routes

import (
	"testing"

	"pocketbase/pluginsystem"
)

func TestRedactPluginDiagnostics(t *testing.T) {
	plugins := []pluginsystem.PluginInfo{{
		ID:     "broken-plugin",
		Path:   "/app/data/plugins/broken-plugin",
		Status: "error",
		Error:  "open /app/data/plugins/broken-plugin/plugin.json: no such file or directory",
	}}

	redactPluginDiagnostics(plugins)

	if plugins[0].Path != "" {
		t.Fatalf("expected plugin path to be redacted, got %q", plugins[0].Path)
	}
	if plugins[0].Error != "" {
		t.Fatalf("expected plugin error to be redacted, got %q", plugins[0].Error)
	}
}
