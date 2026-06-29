package pluginsystem

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	AuthFieldOAuthContext = "oauthContext"
	AuthFieldTokenType    = "tokenType"
	AuthFieldExpiresAt    = "expiresAt"
	AuthFieldScope        = "scope"
)

type OAuthTokenResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token,omitempty"`
	TokenType    string          `json:"token_type,omitempty"`
	ExpiresIn    int             `json:"expires_in,omitempty"`
	Scope        string          `json:"scope,omitempty"`
	Raw          json.RawMessage `json:"-"`
}

// OAuthContext selects the OAuth auth context declared by a plugin. When the UI
// does not request a specific context, the first context by name is used.
func OAuthContext(plugin LocalPlugin, requested string) (string, AuthContext, error) {
	names := make([]string, 0, len(plugin.Manifest.Auth.Contexts))
	for name := range plugin.Manifest.Auth.Contexts {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		context := plugin.Manifest.Auth.Contexts[name]
		if requested != "" && requested != name {
			continue
		}
		if context.Type == AuthTypeOAuth2 {
			return name, context, nil
		}
	}
	return "", AuthContext{}, fmt.Errorf("plugin has no oauth auth context")
}

// ValidateOAuthRedirectURI accepts only the frontend plugin OAuth callback and,
// when ORIGIN is configured, requires the same external origin.
func ValidateOAuthRedirectURI(raw string) error {
	redirectURL, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if redirectURL.Scheme != "http" && redirectURL.Scheme != "https" {
		return fmt.Errorf("redirect uri scheme must be http or https")
	}
	if redirectURL.Host == "" {
		return fmt.Errorf("redirect uri must be absolute")
	}
	if redirectURL.Path != "/settings/plugins/oauth/callback" {
		return fmt.Errorf("redirect uri path is not allowed")
	}
	if origin := strings.TrimRight(os.Getenv("ORIGIN"), "/"); origin != "" {
		originURL, err := url.Parse(origin)
		if err != nil {
			return err
		}
		if !strings.EqualFold(redirectURL.Scheme, originURL.Scheme) || !strings.EqualFold(redirectURL.Host, originURL.Host) {
			return fmt.Errorf("redirect uri origin does not match ORIGIN")
		}
	}
	return nil
}

func NewOAuthState(size int) string {
	return randomURLToken(size)
}

func NewOAuthCodeVerifier(size int) string {
	return randomURLToken(size)
}

func PKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ExchangeOAuthToken performs the host-owned OAuth token exchange or refresh.
// The token endpoint must be allowed by the plugin manifest network policy.
func ExchangeOAuthToken(ctx context.Context, manifest Manifest, authContext AuthContext, auth map[string]any, values map[string]string) (*OAuthTokenResponse, error) {
	tokenURL, err := url.Parse(authContext.TokenURL)
	if err != nil {
		return nil, err
	}
	if tokenURL.Scheme != "http" && tokenURL.Scheme != "https" {
		return nil, fmt.Errorf("oauth token url scheme must be http or https")
	}
	if !OAuthTokenURLAllowed(manifest, tokenURL) {
		return nil, fmt.Errorf("oauth token host %q is not allowed by manifest permissions", tokenURL.Hostname())
	}

	clientID := StringFromAny(auth["clientId"])
	clientSecret := StringFromAny(auth[AuthFieldClientSecret])
	if clientID == "" {
		return nil, fmt.Errorf("clientId is required")
	}

	bodyValues := url.Values{}
	for key, value := range values {
		if value != "" {
			bodyValues.Set(key, value)
		}
	}
	bodyValues.Set("client_id", clientID)
	if authContext.TokenAuth == "" || authContext.TokenAuth == TokenAuthClientSecretPost {
		if clientSecret != "" {
			bodyValues.Set("client_secret", clientSecret)
		}
	}

	var body []byte
	contentType := "application/x-www-form-urlencoded"
	if authContext.TokenRequestFormat == TokenRequestFormatJSON {
		jsonBody := map[string]string{}
		for key, value := range bodyValues {
			if len(value) > 0 {
				jsonBody[key] = value[0]
			}
		}
		var err error
		body, err = json.Marshal(jsonBody)
		if err != nil {
			return nil, err
		}
		contentType = "application/json"
	} else {
		body = []byte(bodyValues.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	if authContext.TokenAuth == TokenAuthClientSecretBasic && clientSecret != "" {
		req.SetBasicAuth(clientID, clientSecret)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	var token OAuthTokenResponse
	token.Raw = append([]byte{}, respBody...)
	if err := json.Unmarshal(respBody, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("oauth token response has no access_token")
	}
	return &token, nil
}

func OAuthTokenURLAllowed(manifest Manifest, tokenURL *url.URL) bool {
	for _, connector := range manifest.Permissions.Network.Connectors {
		if connector.Type != ConnectorTypePublicAPI {
			continue
		}
		baseURL, basePath, err := NormalizeConnectorBase(connector.FixedBaseURL, "")
		if err != nil {
			continue
		}
		target := ResolvedConnectorTarget{
			Name:                connector.Name,
			Type:                connector.Type,
			BaseURL:             baseURL,
			BasePath:            basePath,
			AllowedPathPrefixes: connector.AllowedPathPrefixes,
		}
		if err := ValidateConnectorURL(target, tokenURL); err == nil {
			return true
		}
	}
	return false
}

// RefreshOAuthToken uses the stored refresh token, persists the refreshed auth
// map, and keeps the plugin instance configured when refresh succeeds.
func RefreshOAuthToken(ctx context.Context, app core.App, plugin LocalPlugin, instance *core.Record, auth map[string]any, contextName string) (map[string]any, error) {
	_, authContext, err := OAuthContext(plugin, contextName)
	if err != nil {
		return auth, err
	}
	grantType := "refresh_token"
	if authContext.Refresh != nil && authContext.Refresh.GrantType != "" {
		grantType = authContext.Refresh.GrantType
	}
	refreshToken := StringFromAny(auth[AuthFieldRefreshToken])
	if refreshToken == "" {
		return auth, fmt.Errorf("refreshToken is missing")
	}
	token, err := ExchangeOAuthToken(ctx, plugin.Manifest, authContext, auth, map[string]string{
		"grant_type":    grantType,
		"refresh_token": refreshToken,
	})
	if err != nil {
		return auth, err
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	StoreOAuthToken(auth, contextName, token)
	instance.Set("auth", auth)
	instance.Set("status", "configured")
	if err := app.Save(instance); err != nil {
		return auth, err
	}
	return auth, nil
}

// StoreOAuthToken normalizes provider token responses into the plugin instance
// auth map used by host injection and future refreshes.
func StoreOAuthToken(auth map[string]any, contextName string, token *OAuthTokenResponse) {
	auth[AuthFieldOAuthContext] = contextName
	auth[AuthFieldAccessToken] = token.AccessToken
	if token.RefreshToken != "" {
		auth[AuthFieldRefreshToken] = token.RefreshToken
	}
	if token.TokenType != "" {
		auth[AuthFieldTokenType] = token.TokenType
	}
	if token.Scope != "" {
		auth[AuthFieldScope] = token.Scope
	}
	if token.ExpiresIn > 0 {
		auth[AuthFieldExpiresAt] = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	}
}

// ClearOAuthToken removes persisted OAuth token material and transient OAuth
// flow fields from an auth map.
func ClearOAuthToken(auth map[string]any) {
	for _, key := range []string{
		AuthFieldAccessToken,
		AuthFieldRefreshToken,
		AuthFieldTokenType,
		AuthFieldExpiresAt,
		AuthFieldScope,
		AuthFieldOAuthState,
		AuthFieldOAuthCodeVerifier,
		AuthFieldOAuthRedirectURI,
	} {
		delete(auth, key)
	}
}

// PluginInputAuth returns the auth payload visible to plugin exports. OAuth
// token material is intentionally removed because provider requests should go
// through host auth injection instead.
func PluginInputAuth(plugin LocalPlugin, auth map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range auth {
		out[key] = value
	}
	for _, context := range plugin.Manifest.Auth.Contexts {
		if context.Type == AuthTypeOAuth2 {
			for _, key := range PluginInputAuthBlockedFields() {
				delete(out, key)
			}
		}
	}
	return out
}

// RefreshOAuthAuthIfNeeded refreshes host-managed OAuth before a sync run if no
// access token exists or the current token is close to expiry.
func RefreshOAuthAuthIfNeeded(ctx context.Context, app core.App, plugin LocalPlugin, instance *core.Record, auth map[string]any) (map[string]any, error) {
	for name, authContext := range plugin.Manifest.Auth.Contexts {
		if authContext.Type != AuthTypeOAuth2 {
			continue
		}
		if StringFromAny(auth[AuthFieldAccessToken]) == "" || OAuthNeedsRefresh(auth) {
			return RefreshOAuthToken(ctx, app, plugin, instance, auth, name)
		}
	}
	return auth, nil
}

func OAuthNeedsRefresh(auth map[string]any) bool {
	expiresAt := StringFromAny(auth[AuthFieldExpiresAt])
	if expiresAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return false
	}
	return time.Until(parsed) < time.Minute
}

func StringFromAny(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func randomURLToken(size int) string {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}
