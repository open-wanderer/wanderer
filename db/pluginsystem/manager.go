package pluginsystem

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Manager coordinates local plugin discovery with the installed_plugins cache.
// It is intentionally small: request hot paths should read cached manifests,
// while list/cron entrypoints refresh the cache from data/plugins first.
type Manager struct {
	App core.App
	Dir string
}

// PluginInfo is the UI-facing view of an installed plugin. It combines the
// static manifest with runtime availability and embedded icon data.
type PluginInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	IconDark     string   `json:"iconDark,omitempty"`
	Version      string   `json:"version"`
	Runtime      string   `json:"runtime"`
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
	Status       string   `json:"status"`
	Error        string   `json:"error,omitempty"`
	Manifest     Manifest `json:"manifest"`
}

// NewManager creates a manager for the configured plugin directory. Tests can
// pass a custom dir; production callers use the resolved runtime plugin
// directory.
func NewManager(app core.App, dir string) *Manager {
	if dir == "" {
		dir = PluginDir()
	}
	return &Manager{App: app, Dir: dir}
}

// ListLocalPlugins returns installed plugins in the shape consumed by the
// settings UI. It reads from installed_plugins first so listing does not need to
// parse every manifest from disk after the cache has been refreshed.
func (m *Manager) ListLocalPlugins(context.Context) ([]PluginInfo, error) {
	plugins, err := LoadInstalledPlugins(m.App, m.Dir)
	if err != nil {
		return nil, err
	}
	infos := make([]PluginInfo, 0, len(plugins))
	for _, plugin := range plugins {
		status := "available"
		record, _ := m.App.FindFirstRecordByFilter(
			"installed_plugins",
			"plugin_id={:plugin_id}",
			dbx.Params{"plugin_id": plugin.Manifest.ID},
		)
		if record != nil && record.GetString("status") != "" {
			status = record.GetString("status")
		}
		icon, iconDark := pluginIcons(plugin)
		infos = append(infos, PluginInfo{
			ID:           plugin.Manifest.ID,
			Name:         plugin.Manifest.Name,
			Description:  plugin.Manifest.Description,
			Icon:         icon,
			IconDark:     iconDark,
			Version:      plugin.Manifest.Version,
			Runtime:      plugin.Manifest.Runtime.Type,
			Path:         plugin.Dir,
			Capabilities: capabilityNames(plugin.Manifest.Capabilities),
			Status:       status,
			Manifest:     plugin.Manifest,
		})
	}
	return infos, nil
}

// pluginIcons embeds optional light/dark icon files from the plugin bundle as
// data URLs so the frontend does not need direct filesystem access.
func pluginIcons(plugin LocalPlugin) (string, string) {
	icons, _ := plugin.Manifest.Metadata["icons"].(map[string]any)
	return pluginIcon(plugin.Dir, stringMetadata(icons, "light")), pluginIcon(plugin.Dir, stringMetadata(icons, "dark"))
}

func stringMetadata(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func pluginIcon(pluginDir string, iconPath string) string {
	iconPath = strings.TrimSpace(iconPath)
	if iconPath == "" {
		return ""
	}
	cleanPath := filepath.Clean(iconPath)
	if filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return ""
	}
	fullPath := filepath.Join(pluginDir, cleanPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	contentType := "image/svg+xml"
	switch strings.ToLower(filepath.Ext(fullPath)) {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".webp":
		contentType = "image/webp"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// SyncInstalledPlugins scans the runtime plugin directory and upserts
// installed_plugins records.
// This keeps the manifest snapshot available even when later code paths should
// avoid repeated disk IO.
func (m *Manager) SyncInstalledPlugins(ctx context.Context) error {
	plugins, err := LoadLocalPlugins(m.Dir)
	if err != nil {
		return err
	}
	collection, err := m.App.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		return err
	}
	for _, plugin := range plugins {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, _ := m.App.FindFirstRecordByFilter(
			"installed_plugins",
			"plugin_id={:plugin_id}",
			dbx.Params{"plugin_id": plugin.Manifest.ID},
		)
		if record == nil {
			record = core.NewRecord(collection)
			record.Set("plugin_id", plugin.Manifest.ID)
			record.Set("status", "available")
		}
		record.Set("name", plugin.Manifest.Name)
		record.Set("version", plugin.Manifest.Version)
		record.Set("runtime", plugin.Manifest.Runtime.Type)
		record.Set("path", plugin.Dir)
		manifestJSON, err := marshalManifest(plugin.Manifest)
		if err != nil {
			return fmt.Errorf("encode installed plugin %s manifest: %w", plugin.Manifest.ID, err)
		}
		record.Set("manifest", manifestJSON)
		if record.GetString("status") == "" {
			record.Set("status", "available")
		}
		if err := m.App.Save(record); err != nil {
			return fmt.Errorf("save installed plugin %s: %w", plugin.Manifest.ID, err)
		}
	}
	return nil
}

func marshalManifest(manifest Manifest) (map[string]any, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func capabilityNames(capabilities []CapabilityManifest) []string {
	names := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		names = append(names, capability.Name+"."+capability.Version)
	}
	return names
}
