package pluginsystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// LoadInstalledPlugin resolves one plugin from the installed_plugins cache. If
// the cache record is missing or stale, it falls back to the local plugin
// directory so newly copied bundles can still be discovered.
func LoadInstalledPlugin(app core.App, dir string, pluginID string) (LocalPlugin, error) {
	if pluginID == "" {
		return LocalPlugin{}, fmt.Errorf("plugin id is required")
	}
	record, _ := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id}",
		dbx.Params{"plugin_id": pluginID},
	)
	if record != nil {
		plugin, err := localPluginFromRecord(record)
		if err == nil {
			return plugin, nil
		}
	}

	if dir == "" {
		dir = PluginDir()
	}
	plugins, err := LoadLocalPlugins(dir)
	if err != nil {
		return LocalPlugin{}, err
	}
	for _, plugin := range plugins {
		if plugin.Manifest.ID == pluginID {
			return plugin, nil
		}
	}
	return LocalPlugin{}, fmt.Errorf("unknown plugin")
}

// LoadInstalledPlugins returns the cached installed plugin manifests used by
// request hot paths, with disk discovery as a bootstrap fallback.
func LoadInstalledPlugins(app core.App, dir string) ([]LocalPlugin, error) {
	records, err := app.FindRecordsByFilter("installed_plugins", "", "", -1, 0)
	if err != nil {
		return nil, err
	}

	plugins := make([]LocalPlugin, 0, len(records))
	for _, record := range records {
		plugin, err := localPluginFromRecord(record)
		if err != nil {
			continue
		}
		plugins = append(plugins, plugin)
	}
	if len(plugins) > 0 {
		return plugins, nil
	}
	if dir == "" {
		dir = PluginDir()
	}
	return LoadLocalPlugins(dir)
}

func localPluginFromRecord(record *core.Record) (LocalPlugin, error) {
	var manifest Manifest
	if err := record.UnmarshalJSONField("manifest", &manifest); err != nil {
		return LocalPlugin{}, err
	}
	if err := ValidateManifest(manifest); err != nil {
		return LocalPlugin{}, err
	}

	dir := strings.TrimSpace(record.GetString("path"))
	if dir == "" {
		return LocalPlugin{}, fmt.Errorf("installed plugin path is empty")
	}
	entrypoint := filepath.Clean(manifest.Runtime.Entrypoint)
	if filepath.IsAbs(entrypoint) || entrypoint == ".." || strings.HasPrefix(entrypoint, ".."+string(filepath.Separator)) {
		return LocalPlugin{}, fmt.Errorf("runtime entrypoint must be relative to plugin directory")
	}
	wasmPath := filepath.Join(dir, entrypoint)
	if _, err := os.Stat(wasmPath); err != nil {
		return LocalPlugin{}, fmt.Errorf("runtime entrypoint: %w", err)
	}
	return LocalPlugin{
		Manifest: manifest,
		Dir:      dir,
		WASMPath: wasmPath,
	}, nil
}
