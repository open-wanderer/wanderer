package routes

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

type pluginOAuthStartRequest struct {
	PluginID    string `json:"pluginId"`
	InstanceID  string `json:"instanceId"`
	AuthContext string `json:"authContext,omitempty"`
	RedirectURI string `json:"redirectUri"`
}

type pluginOAuthCallbackRequest struct {
	InstanceID string `json:"instanceId"`
	Code       string `json:"code"`
	State      string `json:"state"`
}

type pluginOAuthRevokeRequest struct {
	InstanceID string `json:"instanceId"`
}

func PluginSystemOAuthStart(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginOAuthStartRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("failed to read request data", err)
	}
	if data.PluginID == "" || data.RedirectURI == "" {
		return apis.NewBadRequestError("pluginId and redirectUri are required", nil)
	}
	if err := pluginsystem.ValidateOAuthRedirectURI(data.RedirectURI); err != nil {
		return apis.NewBadRequestError("redirectUri is not allowed", err)
	}

	plugin, err := localPlugin(e.App, data.PluginID)
	if err != nil {
		return err
	}
	contextName, authContext, err := pluginsystem.OAuthContext(plugin, data.AuthContext)
	if err != nil {
		return apis.NewBadRequestError("plugin has no oauth auth context", err)
	}

	instance, err := pluginAuthInstance(e.App, e.Auth.Id, data.PluginID, data.InstanceID)
	if err != nil {
		return err
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return err
	}
	clientID := pluginsystem.StringFromAny(auth["clientId"])
	if clientID == "" {
		return apis.NewBadRequestError("oauth clientId is required", nil)
	}

	state := pluginsystem.NewOAuthState(32)
	auth[pluginsystem.AuthFieldOAuthContext] = contextName
	auth[pluginsystem.AuthFieldOAuthState] = state
	auth[pluginsystem.AuthFieldOAuthRedirectURI] = data.RedirectURI

	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", clientID)
	values.Set("redirect_uri", data.RedirectURI)
	values.Set("state", state)
	if len(authContext.Scopes) > 0 {
		separator := authContext.ScopeSeparator
		if separator == "" {
			separator = " "
		}
		values.Set("scope", strings.Join(authContext.Scopes, separator))
	}
	for key, value := range authContext.AuthorizationParams {
		values.Set(key, value)
	}
	if authContext.PKCE {
		verifier := pluginsystem.NewOAuthCodeVerifier(64)
		auth[pluginsystem.AuthFieldOAuthCodeVerifier] = verifier
		values.Set("code_challenge_method", "S256")
		values.Set("code_challenge", pluginsystem.PKCEChallenge(verifier))
	}

	instance.Set("auth", auth)
	instance.Set("status", "needs_auth")
	if err := e.App.Save(instance); err != nil {
		return err
	}

	authURL, err := url.Parse(authContext.AuthorizationURL)
	if err != nil {
		return err
	}
	query := authURL.Query()
	for key, value := range values {
		query[key] = value
	}
	authURL.RawQuery = query.Encode()

	return e.JSON(http.StatusOK, map[string]any{
		"url":        authURL.String(),
		"state":      state,
		"instanceId": instance.Id,
	})
}

func PluginSystemOAuthCallback(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginOAuthCallbackRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("failed to read request data", err)
	}
	if data.InstanceID == "" || data.Code == "" || data.State == "" {
		return apis.NewBadRequestError("instanceId, code and state are required", nil)
	}

	instance, err := e.App.FindRecordById("plugin_instances", data.InstanceID)
	if err != nil || instance.GetString("user") != e.Auth.Id {
		return apis.NewNotFoundError("plugin instance not found", nil)
	}
	plugin, err := localPlugin(e.App, instance.GetString("plugin_id"))
	if err != nil {
		return err
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return err
	}
	if data.State != pluginsystem.StringFromAny(auth[pluginsystem.AuthFieldOAuthState]) {
		return apis.NewBadRequestError("invalid oauth state", nil)
	}
	contextName := pluginsystem.StringFromAny(auth[pluginsystem.AuthFieldOAuthContext])
	_, authContext, err := pluginsystem.OAuthContext(plugin, contextName)
	if err != nil {
		return apis.NewBadRequestError("plugin has no oauth auth context", err)
	}

	token, err := pluginsystem.ExchangeOAuthToken(e.Request.Context(), plugin.Manifest, authContext, auth, map[string]string{
		"grant_type":   "authorization_code",
		"code":         data.Code,
		"redirect_uri": pluginsystem.StringFromAny(auth[pluginsystem.AuthFieldOAuthRedirectURI]),
		"code_verifier": pluginsystem.StringFromAny(
			auth[pluginsystem.AuthFieldOAuthCodeVerifier],
		),
	})
	if err != nil {
		return apis.NewBadRequestError("oauth token exchange failed", err)
	}
	pluginsystem.StoreOAuthToken(auth, contextName, token)
	for _, field := range pluginsystem.InternalOAuthTransientFields() {
		delete(auth, field)
	}

	instance.Set("auth", auth)
	instance.Set("status", "configured")
	instance.Set("last_error", map[string]any{})
	if err := e.App.Save(instance); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{"ok": true})
}

func PluginSystemOAuthRevoke(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginOAuthRevokeRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("failed to read request data", err)
	}
	if data.InstanceID == "" {
		return apis.NewBadRequestError("instanceId is required", nil)
	}
	instance, err := e.App.FindRecordById("plugin_instances", data.InstanceID)
	if err != nil || instance.GetString("user") != e.Auth.Id {
		return apis.NewNotFoundError("plugin instance not found", nil)
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return err
	}
	pluginsystem.ClearOAuthToken(auth)
	instance.Set("auth", auth)
	instance.Set("status", "needs_auth")
	if err := e.App.Save(instance); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{"ok": true})
}

func pluginAuthInstance(app core.App, userID string, pluginID string, instanceID string) (*core.Record, error) {
	if instanceID != "" {
		instance, err := app.FindRecordById("plugin_instances", instanceID)
		if err != nil || instance.GetString("user") != userID || instance.GetString("plugin_id") != pluginID {
			return nil, apis.NewNotFoundError("plugin instance not found", nil)
		}
		return instance, nil
	}
	return app.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id}",
		dbx.Params{"user": userID, "plugin_id": pluginID},
	)
}
