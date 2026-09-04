package pluginsystem

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
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
		return nil, OAuthTokenEndpointError{StatusCode: resp.StatusCode, Status: resp.Status, Body: strings.TrimSpace(string(respBody))}
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

// OAuthTokenEndpointError is a non-2xx answer of the provider's token endpoint.
type OAuthTokenEndpointError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e OAuthTokenEndpointError) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.Body)
}

// ErrOAuthRefreshTokenMissing means no refresh token is stored, so only a new
// authorization can restore the connection.
var ErrOAuthRefreshTokenMissing = errors.New("refreshToken is missing")

// ErrPluginInstanceChanged means that the plugin instance no longer matches
// the snapshot which started an OAuth refresh or sync transition. The caller
// must not continue with or persist data derived from the stale snapshot.
var ErrPluginInstanceChanged = errors.New("plugin instance changed concurrently")

// OAuthError is the standardised error code from the endpoint's JSON body
// (RFC 6749 section 5.2), empty when the provider sent none.
func (e OAuthTokenEndpointError) OAuthError() string {
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(e.Body), &body); err != nil {
		return ""
	}
	return strings.TrimSpace(body.Error)
}

// OAuthRefreshFailureCode maps a failed token refresh to the plugin error
// codes the rest of the system understands: invalid_grant when the user has
// to reconnect, auth_failed when the client credentials are wrong,
// plugin_error when the plugin sent a request the provider does not accept,
// rate_limited and provider_unavailable for conditions that say nothing
// about the credentials. The provider's standardised error field decides
// where it exists; without one, a 401 points at the client and a 400 at
// the grant, as Strava reports an invalid refresh token.
func OAuthRefreshFailureCode(err error) string {
	if errors.Is(err, ErrOAuthRefreshTokenMissing) {
		return "invalid_grant"
	}
	var endpointErr OAuthTokenEndpointError
	if !errors.As(err, &endpointErr) {
		return "provider_unavailable"
	}
	if endpointErr.StatusCode == http.StatusTooManyRequests {
		return "rate_limited"
	}
	if endpointErr.StatusCode >= 500 {
		return "provider_unavailable"
	}
	switch endpointErr.OAuthError() {
	case "invalid_grant":
		return "invalid_grant"
	case "invalid_client", "unauthorized_client":
		return "auth_failed"
	case "invalid_request", "unsupported_grant_type", "invalid_scope":
		return "plugin_error"
	}
	switch endpointErr.StatusCode {
	case http.StatusUnauthorized:
		return "auth_failed"
	case http.StatusBadRequest, http.StatusForbidden:
		return "invalid_grant"
	}
	return "provider_unavailable"
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
		return auth, ErrOAuthRefreshTokenMissing
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
	if err := storeRefreshedOAuthAuth(app, instance, auth); err != nil {
		return auth, err
	}
	return auth, nil
}

// storeRefreshedOAuthAuth persists a refreshed auth map. The instance record
// may have been loaded long before the token round trip, so the write goes
// to a fresh copy and touches only the refreshed fields. A reconnect or revoke
// changes auth itself and wins completely. If only another field changed, the
// rotated token is still stored (the provider may already have invalidated the
// previous refresh token), but status/config/enabled are preserved and the
// caller receives ErrPluginInstanceChanged. Compare and write run in one transaction;
// PocketBase serialises every write through a single connection, so no other
// save can slip in between. On success the caller is rebased to the freshly
// stored record, including its Original snapshot, so later field-scoped saves
// cannot accidentally include the old auth value.
func storeRefreshedOAuthAuth(app core.App, instance *core.Record, auth map[string]any) error {
	var stored *core.Record
	changedMeanwhile := false
	err := app.RunInTransaction(func(txApp core.App) error {
		fresh, err := txApp.FindRecordById("plugin_instances", instance.Id)
		if err != nil {
			return err
		}
		if fresh.GetString("auth") != instance.GetString("auth") {
			txApp.Logger().Info("plugin instance credentials changed during token refresh, refreshed token not stored", "instance", instance.Id)
			return ErrPluginInstanceChanged
		}
		changedMeanwhile = !reflect.DeepEqual(fresh.FieldsData(), instance.FieldsData())
		fresh.Set("auth", auth)
		if !changedMeanwhile {
			fresh.Set("status", "configured")
		}
		fresh.IgnoreUnchangedFields(true)
		if err := txApp.Save(fresh); err != nil {
			return err
		}
		// Read back what the database holds: autodate fields cannot be set
		// through Set, and the caller compares against stored values later.
		stored, err = txApp.FindRecordById("plugin_instances", instance.Id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if changedMeanwhile {
		app.Logger().Info("plugin instance changed during token refresh, stored rotated credentials but stopped stale operation", "instance", instance.Id)
		return ErrPluginInstanceChanged
	}
	// Only these fields changed after the full-record comparison above. Reset
	// the caller's baseline so later field-scoped status saves don't include
	// the old auth value.
	instance.SetRaw("auth", stored.GetRaw("auth"))
	instance.SetRaw("status", stored.GetRaw("status"))
	instance.SetRaw("updated", stored.GetRaw("updated"))
	return instance.PostScan()
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
