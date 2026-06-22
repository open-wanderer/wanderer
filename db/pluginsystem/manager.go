package pluginsystem

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
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
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName,omitempty"`
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
	infoByPath := map[string]int{}
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
		errorMessage := ""
		if record != nil {
			errorMessage = record.GetString("error")
		}
		icon, iconDark := pluginIcons(plugin)
		infos = append(infos, PluginInfo{
			ID:           plugin.Manifest.ID,
			Type:         plugin.Manifest.Type,
			Name:         plugin.Manifest.Name,
			DisplayName:  stringMetadata(plugin.Manifest.Metadata, "displayName"),
			Description:  plugin.Manifest.Description,
			Icon:         icon,
			IconDark:     iconDark,
			Version:      plugin.Manifest.Version,
			Runtime:      plugin.Manifest.Runtime.Type,
			Path:         plugin.Dir,
			Capabilities: capabilityNames(plugin.Manifest.Capabilities),
			Status:       status,
			Error:        errorMessage,
			Manifest:     plugin.Manifest,
		})
		infoByPath[filepath.Clean(plugin.Dir)] = len(infos) - 1
	}

	_, issues, err := DiscoverLocalPlugins(m.Dir)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		if index, ok := infoByPath[filepath.Clean(issue.Dir)]; ok {
			infos[index].Status = "error"
			infos[index].Error = issue.Error
			continue
		}
		infos = append(infos, PluginInfo{
			ID:      issue.ID,
			Type:    PluginTypeTrails,
			Name:    issue.Name,
			Path:    issue.Dir,
			Status:  "error",
			Error:   issue.Error,
			Runtime: RuntimeWASM,
			Manifest: Manifest{
				ID:   issue.ID,
				Type: PluginTypeTrails,
				Name: issue.Name,
				Runtime: RuntimeManifest{
					Type: RuntimeWASM,
				},
			},
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
	plugins, issues, err := DiscoverLocalPlugins(m.Dir)
	if err != nil {
		return err
	}
	collection, err := m.App.FindCollectionByNameOrId("installed_plugins")
	if err != nil {
		return err
	}
	activePaths := activePluginPaths(plugins, issues)
	if err := m.deleteStaleInstalledPlugins(ctx, activePaths); err != nil {
		return err
	}
	for _, issue := range issues {
		if err := ctx.Err(); err != nil {
			return err
		}
		m.App.Logger().Warn("plugin setup error", "plugin", issue.ID, "path", issue.Dir, "error", issue.Error)
		if err := m.savePluginIssue(collection, issue); err != nil {
			return err
		}
	}
	for _, plugin := range plugins {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, err := m.findPluginRecord(collection, plugin)
		if err != nil {
			return err
		}
		record.Set("plugin_id", plugin.Manifest.ID)
		record.Set("name", plugin.Manifest.Name)
		record.Set("type", plugin.Manifest.Type)
		record.Set("version", plugin.Manifest.Version)
		record.Set("runtime", plugin.Manifest.Runtime.Type)
		record.Set("path", plugin.Dir)
		record.Set("status", "available")
		record.Set("error", "")
		manifestJSON, err := marshalManifest(plugin.Manifest)
		if err != nil {
			return fmt.Errorf("encode installed plugin %s manifest: %w", plugin.Manifest.ID, err)
		}
		record.Set("manifest", manifestJSON)
		record.Set("config", mergeDefaultConfig(defaultConfig(plugin.Manifest), JSONMapFromRecord(record, "config")))
		if err := m.App.Save(record); err != nil {
			return fmt.Errorf("save installed plugin %s: %w", plugin.Manifest.ID, err)
		}
	}
	return nil
}

func activePluginPaths(plugins []LocalPlugin, issues []LocalPluginIssue) map[string]bool {
	paths := make(map[string]bool, len(plugins)+len(issues))
	for _, plugin := range plugins {
		if plugin.Dir != "" {
			paths[filepath.Clean(plugin.Dir)] = true
		}
	}
	for _, issue := range issues {
		if issue.Dir != "" {
			paths[filepath.Clean(issue.Dir)] = true
		}
	}
	return paths
}

func (m *Manager) deleteStaleInstalledPlugins(ctx context.Context, activePaths map[string]bool) error {
	records, err := m.App.FindRecordsByFilter("installed_plugins", "", "", -1, 0)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := strings.TrimSpace(record.GetString("path"))
		if path != "" && activePaths[filepath.Clean(path)] {
			continue
		}
		if err := m.App.Delete(record); err != nil {
			return fmt.Errorf("delete stale installed plugin %s: %w", record.GetString("plugin_id"), err)
		}
	}
	return nil
}

func (m *Manager) findPluginRecord(collection *core.Collection, plugin LocalPlugin) (*core.Record, error) {
	recordByID, _ := m.App.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id}",
		dbx.Params{"plugin_id": plugin.Manifest.ID},
	)
	var recordByPath *core.Record
	if plugin.Dir != "" {
		recordByPath, _ = m.App.FindFirstRecordByFilter(
			"installed_plugins",
			"path={:path}",
			dbx.Params{"path": plugin.Dir},
		)
	}
	if recordByID != nil && recordByPath != nil && recordByID.Id != recordByPath.Id {
		if err := m.App.Delete(recordByPath); err != nil {
			return nil, fmt.Errorf("delete superseded installed plugin %s: %w", recordByPath.GetString("plugin_id"), err)
		}
	}
	if recordByID != nil {
		return recordByID, nil
	}
	if recordByPath != nil {
		return recordByPath, nil
	}
	return core.NewRecord(collection), nil
}

func (m *Manager) savePluginIssue(collection *core.Collection, issue LocalPluginIssue) error {
	recordID := pluginIssueRecordID(issue)
	record, _ := m.App.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id}",
		dbx.Params{"plugin_id": recordID},
	)
	if record == nil && issue.Dir != "" {
		record, _ = m.App.FindFirstRecordByFilter(
			"installed_plugins",
			"path={:path}",
			dbx.Params{"path": issue.Dir},
		)
	}
	if record == nil {
		record = core.NewRecord(collection)
		record.Set("plugin_id", recordID)
	}
	record.Set("name", issue.Name)
	record.Set("type", PluginTypeTrails)
	record.Set("version", "unknown")
	record.Set("runtime", RuntimeWASM)
	record.Set("path", issue.Dir)
	record.Set("manifest", map[string]any{
		"id":   record.GetString("plugin_id"),
		"type": PluginTypeTrails,
		"name": issue.Name,
	})
	record.Set("status", "error")
	record.Set("error", issue.Error)
	if err := m.App.Save(record); err != nil {
		return fmt.Errorf("save plugin setup error %s: %w", issue.ID, err)
	}
	return nil
}

func pluginIssueRecordID(issue LocalPluginIssue) string {
	originalID := strings.TrimSpace(issue.ID)
	id := strings.ToLower(originalID)
	var builder strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-_")
	if result == "" {
		result = "plugin-setup-error"
	}
	if originalID != result || !pluginIDPattern.MatchString(result) {
		result = strings.Trim(result, "-_")
		if result == "" {
			result = "plugin-setup-error"
		}
		result = result + "-" + pluginIssueHash(issue)
	}
	if len(result) > 128 {
		hash := pluginIssueHash(issue)
		prefixLength := 128 - len(hash) - 1
		result = strings.Trim(result[:prefixLength], "-_") + "-" + hash
	}
	if pluginIDPattern.MatchString(result) {
		return result
	}
	return "plugin-setup-error-" + pluginIssueHash(issue)
}

func pluginIssueHash(issue LocalPluginIssue) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(issue.Dir))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(issue.ID))
	return fmt.Sprintf("%08x", hash.Sum32())
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

func defaultConfig(manifest Manifest) map[string]any {
	hostConfig, _ := CloneJSONValue(manifest.HostConfig).(map[string]any)
	if hostConfig == nil {
		hostConfig = map[string]any{}
	}
	config := map[string]any{
		"host": hostConfig,
	}
	pluginConfig := map[string]any{}
	for _, field := range manifest.ConfigSchema {
		if field.Key == "" || field.Default == nil {
			continue
		}
		pluginConfig[field.Key] = CloneJSONValue(field.Default)
	}
	config["plugin"] = pluginConfig
	return config
}

func mergeDefaultConfig(defaults map[string]any, current map[string]any) map[string]any {
	if len(defaults) == 0 {
		return current
	}
	merged := CloneJSONMap(defaults)
	MergePluginConfig(merged, current)
	return merged
}

func MergePluginConfig(dst map[string]any, src map[string]any) {
	DeepMergeConfigWithReplaceKeys(dst, src, map[string]bool{
		"categoryMapping": true,
	})
}

func capabilityNames(capabilities []CapabilityManifest) []string {
	names := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		names = append(names, capability.Name+"."+capability.Version)
	}
	return names
}
