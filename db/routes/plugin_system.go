package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

// PluginSystemPluginsList refreshes the installed plugin cache and returns the
// plugins that are available from the local runtime directory.
func PluginSystemPluginsList(e *core.RequestEvent) error {
	if e.Auth == nil && !e.HasSuperuserAuth() {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	manager := pluginsystem.NewManager(e.App, "")
	if err := manager.SyncInstalledPlugins(e.Request.Context()); err != nil {
		return err
	}
	plugins, err := manager.ListLocalPlugins(e.Request.Context())
	if err != nil {
		return err
	}
	if !e.HasSuperuserAuth() {
		for i := range plugins {
			plugins[i].Path = ""
		}
	}

	return e.JSON(http.StatusOK, map[string]any{"items": plugins})
}

// localPlugin resolves an installed plugin from the cached installed_plugins
// record, with disk manifest fallback handled inside pluginsystem.
func localPlugin(app core.App, pluginID string) (pluginsystem.LocalPlugin, error) {
	plugin, err := pluginsystem.LoadInstalledPlugin(app, "", pluginID)
	if err != nil {
		return pluginsystem.LocalPlugin{}, apis.NewBadRequestError("unknown plugin", err)
	}
	return plugin, nil
}

// pluginCapability returns the manifest entry for a concrete capability/version
// pair so the host can call the export declared by the plugin.
func pluginCapability(plugin pluginsystem.LocalPlugin, name string, version string) (pluginsystem.CapabilityManifest, error) {
	for _, capability := range plugin.Manifest.Capabilities {
		if capability.Name == name && capability.Version == version {
			return capability, nil
		}
	}
	return pluginsystem.CapabilityManifest{}, apis.NewBadRequestError("plugin capability is not available", map[string]string{
		"name":    name,
		"version": version,
	})
}

// localPluginCapability resolves an installed plugin and verifies that it
// declares the requested capability.
func localPluginCapability(app core.App, pluginID string, name string, version string) (pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, error) {
	plugin, err := localPlugin(app, pluginID)
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, err
	}
	capability, err := pluginCapability(plugin, name, version)
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, err
	}
	return plugin, capability, nil
}
