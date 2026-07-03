package routes

import (
	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"
)

func pluginInstancePolicy(plugin pluginsystem.LocalPlugin, config map[string]any) pluginsystem.RequestPolicyContext {
	return pluginhost.InstancePolicy(plugin, config)
}
