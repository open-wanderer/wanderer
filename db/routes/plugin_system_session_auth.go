package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

type pluginSessionAuthValidateRequest struct {
	PluginID    string         `json:"pluginId"`
	InstanceID  string         `json:"instanceId,omitempty"`
	AuthContext string         `json:"authContext,omitempty"`
	Auth        map[string]any `json:"auth,omitempty"`
}

type pluginSessionAuthRefreshInput struct {
	Instance pluginsystem.InstanceRef `json:"instance"`
	Auth     map[string]any           `json:"auth,omitempty"`
}

func PluginSystemSessionAuthValidate(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginSessionAuthValidateRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("failed to read request data", err)
	}
	if data.PluginID == "" {
		return apis.NewBadRequestError("pluginId is required", nil)
	}

	plugin, err := localPlugin(e.App, data.PluginID)
	if err != nil {
		return err
	}
	contextName, authContext, err := sessionAuthContext(plugin, data.AuthContext)
	if err != nil {
		return apis.NewBadRequestError("plugin has no session auth context", err)
	}
	if authContext.Refresh == nil || authContext.Refresh.Function == "" {
		return apis.NewBadRequestError("plugin session auth context has no refresh function", nil)
	}

	auth := map[string]any{}
	instanceID := data.InstanceID
	if instanceID != "" {
		instance, err := pluginAuthInstance(e.App, e.Auth.Id, data.PluginID, instanceID)
		if err != nil {
			return err
		}
		instanceID = instance.Id
		auth, err = decryptedInstanceAuth(instance)
		if err != nil {
			return err
		}
	}
	for key, value := range data.Auth {
		if value == "" {
			continue
		}
		auth[key] = value
	}

	inputBytes, err := json.Marshal(pluginSessionAuthRefreshInput{
		Instance: pluginsystem.InstanceRef{
			ID:       instanceID,
			PluginID: plugin.Manifest.ID,
		},
		Auth: pluginsystem.AuthForPluginRefresh(auth, authContext),
	})
	if err != nil {
		return err
	}

	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
	if err != nil {
		return err
	}
	// TODO: accept and merge plugin instance config here before supporting
	// session-auth plugins with configured connectors. The current validation
	// path is sufficient for public_api session plugins such as komoot and
	// hammerhead, but configured connectors need host config for policy
	// resolution and refresh input parity with production auth injection.
	policy := pluginInstancePolicy(plugin, map[string]any{}).WithHostAuth(auth)
	output, err := runtime.Call(e.Request.Context(), plugin, authContext.Refresh.Function, inputBytes, policy)
	if err != nil {
		return apis.NewBadRequestError("plugin credentials validation failed", err)
	}
	if err := pluginsystem.ValidatePluginSessionRefreshOutput(output); err != nil {
		return apis.NewBadRequestError("plugin credentials validation failed", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"ok":          true,
		"authContext": contextName,
	})
}

func sessionAuthContext(plugin pluginsystem.LocalPlugin, requested string) (string, pluginsystem.AuthContext, error) {
	if requested != "" {
		authContext, ok := plugin.Manifest.Auth.Contexts[requested]
		if !ok || authContext.Type != pluginsystem.AuthTypeSession {
			return "", pluginsystem.AuthContext{}, apis.NewBadRequestError("unknown session auth context", nil)
		}
		return requested, authContext, nil
	}
	for name, authContext := range plugin.Manifest.Auth.Contexts {
		if authContext.Type == pluginsystem.AuthTypeSession {
			return name, authContext, nil
		}
	}
	return "", pluginsystem.AuthContext{}, apis.NewBadRequestError("session auth context not found", nil)
}
