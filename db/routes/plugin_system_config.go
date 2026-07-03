package routes

import (
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/pluginhost"
)

func effectivePluginConfig(app core.App, pluginID string, instance *core.Record) map[string]any {
	return pluginhost.EffectiveConfig(app, pluginID, instance)
}

func pluginRuntimeConfig(config map[string]any) map[string]any {
	return pluginhost.RuntimeConfig(config)
}

func pluginHostConfig(config map[string]any) map[string]any {
	return pluginhost.HostConfig(config)
}

func installedPluginConfig(app core.App, pluginID string) map[string]any {
	return pluginhost.InstalledConfig(app, pluginID)
}
