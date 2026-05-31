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
