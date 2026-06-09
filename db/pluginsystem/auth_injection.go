package pluginsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type AuthInjectionInput struct {
	App      core.App
	Runtime  Runtime
	Session  RuntimeSession
	Plugin   LocalPlugin
	Instance *core.Record
	Auth     map[string]any
	Config   map[string]any
	Spec     *HostRequestSpec
	Policy   RequestPolicyContext
}

func InjectRequestAuthForContext(manifest Manifest, auth map[string]any, contextName string, req *http.Request) error {
	if contextName == "" {
		return nil
	}
	if err := ValidateAuthReference(manifest, contextName); err != nil {
		return err
	}
	authContext, ok := manifest.Auth.Contexts[contextName]
	if !ok {
		return fmt.Errorf("plugin requested unknown auth context")
	}
	switch authContext.Type {
	case AuthTypeOAuth2:
		token := StringFromAny(auth[AuthFieldAccessToken])
		if token == "" {
			return fmt.Errorf("oauth access token is missing")
		}
		scheme := StringFromAny(auth[AuthFieldTokenType])
		if scheme == "" {
			scheme = AuthSchemeBearer
		}
		req.Header.Set(AuthHeaderAuthorization, scheme+" "+token)
	case AuthTypeAPIKey:
		secret := StringFromAny(auth[authContext.SecretField])
		if secret == "" {
			return fmt.Errorf("api key is missing")
		}
		name := authContext.Name
		if name == "" {
			name = authContext.SecretField
		}
		if authContext.Placement == AuthPlacementQuery {
			req.URL.RawQuery = setRawQueryParamOrdered(req.URL.RawQuery, name, secret)
		} else {
			req.Header.Set(name, secret)
		}
	case AuthTypeBearer:
		secret := StringFromAny(auth[authContext.SecretField])
		if secret == "" {
			return fmt.Errorf("bearer token is missing")
		}
		req.Header.Set(AuthHeaderAuthorization, AuthSchemeBearer+" "+secret)
	default:
		return fmt.Errorf("auth context %q is not supported for media requests", contextName)
	}
	return nil
}

func InjectHostRequestAuthFromPolicy(manifest Manifest, auth map[string]any, spec *HostRequestSpec) error {
	if spec == nil || spec.Auth == "" {
		return nil
	}
	if err := ValidateAuthReference(manifest, spec.Auth); err != nil {
		return err
	}
	authContext, ok := manifest.Auth.Contexts[spec.Auth]
	if !ok {
		return fmt.Errorf("plugin requested unknown auth context")
	}
	switch authContext.Type {
	case AuthTypeOAuth2:
		token := StringFromAny(auth[AuthFieldAccessToken])
		if token == "" {
			return fmt.Errorf("oauth access token is missing")
		}
		scheme := StringFromAny(auth[AuthFieldTokenType])
		if scheme == "" {
			scheme = AuthSchemeBearer
		}
		setAuthHeader(spec, scheme+" "+token)
	case AuthTypeAPIKey:
		return injectAPIKeyAuth(authContext, auth, spec)
	case AuthTypeBearer:
		return injectBearerAuth(authContext, auth, spec)
	case AuthTypeSession:
		return fmt.Errorf("session auth requires handler-managed injection")
	default:
		return fmt.Errorf("auth context is not supported for host requests")
	}
	return nil
}

type pluginSessionResponse struct {
	Token   string `json:"token"`
	Scheme  string `json:"scheme,omitempty"`
	Expires string `json:"expiresAt,omitempty"`
}

// ValidateAuthContext checks that a manifest auth context contains enough data
// for the host to own OAuth/API key/session injection safely.
func ValidateAuthContext(name string, context AuthContext) error {
	switch context.Type {
	case AuthTypeOAuth2:
		if context.AuthorizationURL == "" || context.TokenURL == "" {
			return fmt.Errorf("oauth2 auth context %s requires authorizationUrl and tokenUrl", name)
		}
		if _, err := url.ParseRequestURI(context.AuthorizationURL); err != nil {
			return fmt.Errorf("auth context %s authorizationUrl: %w", name, err)
		}
		if _, err := url.ParseRequestURI(context.TokenURL); err != nil {
			return fmt.Errorf("auth context %s tokenUrl: %w", name, err)
		}
		if context.Refresh == nil || context.Refresh.Mode != AuthRefreshModeHost {
			return fmt.Errorf("oauth2 auth context %s must use host refresh", name)
		}
	case AuthTypeAPIKey, AuthTypeBearer:
		if context.SecretField == "" {
			return fmt.Errorf("%s auth context %s requires secretField", context.Type, name)
		}
	case AuthTypeSession:
		if context.Refresh == nil || context.Refresh.Mode != AuthRefreshModePlugin || context.Refresh.Function == "" {
			return fmt.Errorf("session auth context %s requires plugin refresh function", name)
		}
		if len(context.SecretFields) == 0 {
			return fmt.Errorf("session auth context %s requires secretFields", name)
		}
	default:
		return fmt.Errorf("auth context %s has unsupported type %q", name, context.Type)
	}
	return nil
}

// InjectHostRequestAuth resolves the auth reference from a HostRequestSpec and
// mutates the request with the provider-specific header/query/session token.
func InjectHostRequestAuth(ctx context.Context, input AuthInjectionInput) error {
	if input.Spec == nil {
		return fmt.Errorf("host request spec is required")
	}
	if input.Spec.Auth == "" {
		return nil
	}
	if err := ValidateAuthReference(input.Plugin.Manifest, input.Spec.Auth); err != nil {
		return err
	}
	authContext, ok := input.Plugin.Manifest.Auth.Contexts[input.Spec.Auth]
	if !ok {
		return fmt.Errorf("plugin requested unknown auth context")
	}

	switch authContext.Type {
	case AuthTypeOAuth2:
		return injectOAuthAuth(ctx, input, input.Spec.Auth)
	case AuthTypeAPIKey:
		return injectAPIKeyAuth(authContext, input.Auth, input.Spec)
	case AuthTypeBearer:
		return injectBearerAuth(authContext, input.Auth, input.Spec)
	case AuthTypeSession:
		return injectSessionAuth(ctx, input, authContext)
	default:
		return fmt.Errorf("auth context is not supported for route sending")
	}
}

func injectOAuthAuth(ctx context.Context, input AuthInjectionInput, contextName string) error {
	if input.Instance == nil {
		return fmt.Errorf("plugin instance is required")
	}
	auth := input.Auth
	if OAuthNeedsRefresh(auth) {
		refreshed, err := RefreshOAuthToken(ctx, input.App, input.Plugin, input.Instance, auth, contextName)
		if err != nil {
			return fmt.Errorf("oauth token refresh failed: %w", err)
		}
		auth = refreshed
	}
	token := StringFromAny(auth[AuthFieldAccessToken])
	if token == "" {
		return fmt.Errorf("oauth access token is missing")
	}
	scheme := StringFromAny(auth[AuthFieldTokenType])
	if scheme == "" {
		scheme = AuthSchemeBearer
	}
	setAuthHeader(input.Spec, scheme+" "+token)
	return nil
}

func injectAPIKeyAuth(authContext AuthContext, auth map[string]any, spec *HostRequestSpec) error {
	secret := StringFromAny(auth[authContext.SecretField])
	if secret == "" {
		return fmt.Errorf("api key is missing")
	}
	if authContext.Placement == AuthPlacementQuery {
		name := authContext.Name
		if name == "" {
			name = authContext.SecretField
		}
		query := make([]QueryParam, 0, len(spec.Target.Query)+1)
		for _, param := range spec.Target.Query {
			if param.Name != name {
				query = append(query, param)
			}
		}
		query = append(query, QueryParam{Name: name, Value: secret})
		spec.Target.Query = query
		return nil
	}
	name := authContext.Name
	if name == "" {
		name = AuthHeaderAuthorization
	}
	if spec.Headers == nil {
		spec.Headers = map[string]string{}
	}
	spec.Headers[name] = secret
	return nil
}

func injectBearerAuth(authContext AuthContext, auth map[string]any, spec *HostRequestSpec) error {
	secret := StringFromAny(auth[authContext.SecretField])
	if secret == "" {
		return fmt.Errorf("bearer token is missing")
	}
	setAuthHeader(spec, AuthSchemeBearer+" "+secret)
	return nil
}

func injectSessionAuth(ctx context.Context, input AuthInjectionInput, authContext AuthContext) error {
	if authContext.Refresh == nil || authContext.Refresh.Mode != AuthRefreshModePlugin {
		return fmt.Errorf("session auth context is not supported")
	}
	if input.Instance == nil {
		return fmt.Errorf("plugin instance is required")
	}

	pluginInput := map[string]any{
		"instance": InstanceRef{
			ID:       input.Instance.Id,
			PluginID: input.Instance.GetString("plugin_id"),
		},
		"auth":   AuthForPluginRefresh(input.Auth, authContext),
		"config": input.Config,
	}
	inputBytes, err := json.Marshal(pluginInput)
	if err != nil {
		return err
	}
	var output []byte
	if input.Session != nil {
		output, err = input.Session.Call(ctx, authContext.Refresh.Function, inputBytes)
	} else {
		output, err = input.Runtime.Call(ctx, input.Plugin, authContext.Refresh.Function, inputBytes, input.Policy)
	}
	if err != nil {
		return err
	}
	var session pluginSessionResponse
	if err := validatePluginSessionRefreshOutput(output, &session); err != nil {
		return err
	}
	scheme := session.Scheme
	if scheme == "" {
		scheme = AuthSchemeBearer
	}
	setAuthHeader(input.Spec, scheme+" "+session.Token)
	return nil
}

func ValidatePluginSessionRefreshOutput(output []byte) error {
	var session pluginSessionResponse
	return validatePluginSessionRefreshOutput(output, &session)
}

func validatePluginSessionRefreshOutput(output []byte, session *pluginSessionResponse) error {
	if err := json.Unmarshal(output, session); err != nil {
		return fmt.Errorf("plugin returned an invalid session: %w", err)
	}
	if session.Token == "" {
		return fmt.Errorf("plugin returned an empty session token")
	}
	return nil
}

func setAuthHeader(spec *HostRequestSpec, value string) {
	if spec.Headers == nil {
		spec.Headers = map[string]string{}
	}
	spec.Headers[AuthHeaderAuthorization] = value
}

func setRawQueryParamOrdered(rawQuery string, name string, value string) string {
	encoded := url.QueryEscape(name) + "=" + url.QueryEscape(value)
	if rawQuery == "" {
		return encoded
	}
	parts := strings.Split(rawQuery, "&")
	kept := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if part == "" {
			continue
		}
		rawName := part
		if idx := strings.Index(rawName, "="); idx >= 0 {
			rawName = rawName[:idx]
		}
		decodedName, err := url.QueryUnescape(rawName)
		if err == nil && decodedName == name {
			continue
		}
		kept = append(kept, part)
	}
	kept = append(kept, encoded)
	return strings.Join(kept, "&")
}

func AuthForPluginRefresh(auth map[string]any, authContext AuthContext) map[string]any {
	filtered := map[string]any{}
	for _, field := range authContext.Fields {
		if value, ok := auth[field]; ok {
			filtered[field] = value
		}
	}
	for _, field := range authContext.SecretFields {
		if value, ok := auth[field]; ok {
			filtered[field] = value
		}
	}
	if authContext.SecretField != "" {
		if value, ok := auth[authContext.SecretField]; ok {
			filtered[authContext.SecretField] = value
		}
	}
	return filtered
}
