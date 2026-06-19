package routes

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

func effectivePluginConfig(app core.App, pluginID string, instance *core.Record) map[string]any {
	config := installedPluginConfig(app, pluginID)
	pluginsystem.MergePluginConfig(config, pluginsystem.JSONMapFromRecord(instance, "config"))
	return config
}

func pluginRuntimeConfig(config map[string]any) map[string]any {
	return configSection(config, "plugin")
}

func pluginHostConfig(config map[string]any) map[string]any {
	return configSection(config, "host")
}

func configSection(config map[string]any, key string) map[string]any {
	raw, ok := config[key].(map[string]any)
	if !ok || raw == nil {
		return map[string]any{}
	}
	return raw
}

func installedPluginConfig(app core.App, pluginID string) map[string]any {
	record, _ := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id}",
		dbx.Params{"plugin_id": pluginID},
	)
	if record == nil {
		return map[string]any{}
	}
	return pluginsystem.JSONMapFromRecord(record, "config")
}
