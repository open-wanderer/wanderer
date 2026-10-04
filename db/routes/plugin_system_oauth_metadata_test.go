package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"

	"pocketbase/hooks"
	"pocketbase/pluginsystem"
)

const oauthMetadataTestKey = "0123456789abcdef0123456789abcdef"

type oauthMetadataFixture struct {
	app           *pbtests.TestApp
	plugin        pluginsystem.LocalPlugin
	instanceID    string
	authToken     string
	handler       http.Handler
	tokenRequests atomic.Int32
}

// Use the shipped Strava v1 manifest and production record hooks. Only the
// token URL and its connector point at a local server; no WASM is executed.
func newOAuthMetadataFixture(t *testing.T, prepareAuth ...func(map[string]any)) *oauthMetadataFixture {
	t.Helper()
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", oauthMetadataTestKey)
	t.Setenv("ORIGIN", "https://wanderer.example")
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	f := &oauthMetadataFixture{app: app}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.tokenRequests.Add(1)
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode Strava token request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" ||
			body["client_id"] != "strava-test-client" || body["client_secret"] != "strava-test-client-secret" {
			t.Error("token request did not use the stored Strava client credentials and JSON format")
		}
		switch body["grant_type"] {
		case "refresh_token":
			if body["refresh_token"] != "strava-test-refresh" {
				t.Error("refresh did not use the original stored refresh token")
			}
		case "authorization_code":
			if body["code"] != "test-authorization-code" {
				t.Error("callback did not send the authorization code")
			}
		default:
			t.Errorf("unexpected token grant type %q", body["grant_type"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "strava-test-access-rotated", "refresh_token": "strava-test-refresh-rotated",
			"token_type": "Bearer", "expires_in": 21600, "scope": "read_all,activity:read_all,rotated",
		})
	}))
	t.Cleanup(server.Close)

	manifestData, err := os.ReadFile(filepath.Join("..", "..", "plugins", "strava", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest pluginsystem.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "strava" || manifest.ManifestVersion != "1.0" {
		t.Fatal("fixture must use the shipped Strava v1 manifest")
	}
	context := manifest.Auth.Contexts["oauth_access_token"]
	context.TokenURL = server.URL + "/oauth/token"
	manifest.Auth.Contexts["oauth_access_token"] = context
	for i := range manifest.Permissions.Network.Connectors {
		if manifest.Permissions.Network.Connectors[i].Name == "oauth" {
			manifest.Permissions.Network.Connectors[i].FixedBaseURL = server.URL + "/oauth"
		}
	}
	if err := pluginsystem.ValidateManifest(manifest); err != nil {
		t.Fatal(err)
	}

	users := core.NewAuthCollection("oauth_metadata_users")
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
	owner := core.NewRecord(users)
	owner.SetEmail("oauth-metadata@example.com")
	owner.SetPassword("test-oauth-password")
	if err := app.Save(owner); err != nil {
		t.Fatal(err)
	}
	f.authToken, err = owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}

	installed := core.NewBaseCollection("installed_plugins")
	installed.Fields.Add(&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "path"},
		&core.JSONField{Name: "manifest"}, &core.JSONField{Name: "config"})
	if err := app.Save(installed); err != nil {
		t.Fatal(err)
	}
	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, manifest.Runtime.Entrypoint), []byte("\x00asm\x01\x00\x00\x00"), 0600); err != nil {
		t.Fatal(err)
	}
	pluginRecord := core.NewRecord(installed)
	pluginRecord.Set("plugin_id", manifest.ID)
	pluginRecord.Set("path", pluginDir)
	pluginRecord.Set("manifest", manifest)
	if err := app.Save(pluginRecord); err != nil {
		t.Fatal(err)
	}
	f.plugin, err = pluginsystem.LoadInstalledPlugin(app, "", manifest.ID)
	if err != nil {
		t.Fatal(err)
	}

	instances := core.NewBaseCollection("plugin_instances")
	instances.ViewRule = types.Pointer("user = @request.auth.id")
	instances.UpdateRule = instances.ViewRule
	instances.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1},
		&core.TextField{Name: "plugin_id"}, &core.BoolField{Name: "enabled"},
		&core.JSONField{Name: "auth"}, &core.JSONField{Name: "config"},
		&core.TextField{Name: "status"}, &core.JSONField{Name: "last_error"},
	)
	if err := app.Save(instances); err != nil {
		t.Fatal(err)
	}
	app.OnRecordsListRequest("plugin_instances").BindFunc(hooks.ListPluginInstanceHandler())
	app.OnRecordViewRequest("plugin_instances").BindFunc(hooks.ViewPluginInstanceHandler())
	app.OnRecordCreate("plugin_instances").BindFunc(hooks.CreatePluginInstanceHandler())
	app.OnRecordAfterCreateSuccess("plugin_instances").BindFunc(hooks.CreateUpdatePluginInstanceSuccessHandler())
	app.OnRecordUpdate("plugin_instances").BindFunc(hooks.UpdatePluginInstanceHandler())
	app.OnRecordAfterUpdateSuccess("plugin_instances").BindFunc(hooks.CreateUpdatePluginInstanceSuccessHandler())
	instance := core.NewRecord(instances)
	instance.Set("user", owner.Id)
	instance.Set("plugin_id", manifest.ID)
	instance.Set("enabled", true)
	instance.Set("status", "configured")
	initialAuth := map[string]any{
		"clientId": "strava-test-client", "clientSecret": "strava-test-client-secret",
		"accessToken": "strava-test-access", "refreshToken": "strava-test-refresh",
		"oauthContext": "oauth_access_token", "tokenType": "Bearer",
		"expiresAt": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), "scope": "read_all,activity:read_all",
		"unusedAuthNote": "not-an-internal-metadata-field",
	}
	for _, prepare := range prepareAuth {
		prepare(initialAuth)
	}
	instance.Set("auth", initialAuth)
	if err := app.Save(instance); err != nil {
		t.Fatal(err)
	}
	f.instanceID = instance.Id
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.POST("/api/plugin-system/oauth/start", PluginSystemOAuthStart)
	router.POST("/api/plugin-system/oauth/callback", PluginSystemOAuthCallback)
	router.POST("/api/plugin-system/oauth/revoke", PluginSystemOAuthRevoke)
	f.handler, err = router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *oauthMetadataFixture) request(t *testing.T, method, path string, body map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Authorization", f.authToken)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s: status %d, %s", method, path, response.Code, response.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func (f *oauthMetadataFixture) storedAuth(t *testing.T) (*core.Record, map[string]any) {
	t.Helper()
	instance, err := f.app.FindRecordById("plugin_instances", f.instanceID)
	if err != nil {
		t.Fatal(err)
	}
	auth := pluginsystem.JSONMapFromRecord(instance, "auth")
	for _, field := range []string{"clientSecret", "accessToken", "refreshToken", "oauthState", "oauthCodeVerifier"} {
		stored := pluginsystem.StringFromAny(auth[field])
		if stored == "" {
			continue
		}
		decoded, err := security.Decrypt(stored, oauthMetadataTestKey)
		if err != nil {
			t.Fatalf("stored %s is not encrypted: %v", field, err)
		}
		auth[field] = string(decoded)
	}
	return instance, auth
}

// Match initialAuth/pluginInstanceFromForm: only manifest input fields leave
// the modal, and censored secret values are empty placeholders.
func (f *oauthMetadataFixture) modalPayload(t *testing.T) map[string]any {
	t.Helper()
	result := f.request(t, http.MethodGet, "/api/collections/plugin_instances/records/"+f.instanceID, nil)
	auth := result["auth"].(map[string]any)
	for _, field := range []string{"clientSecret", "accessToken", "refreshToken"} {
		if auth[field] != "" {
			t.Fatalf("GET disclosed %s", field)
		}
	}
	submitted := map[string]any{}
	for _, field := range f.plugin.Manifest.Auth.Contexts["oauth_access_token"].Fields {
		submitted[field] = auth[field]
	}
	return map[string]any{
		"auth": submitted, "enabled": result["enabled"],
		"config": map[string]any{"host": map[string]any{"completed": true, "planned": false}},
	}
}

func (f *oauthMetadataFixture) saveModal(t *testing.T, payload map[string]any) {
	t.Helper()
	result := f.request(t, http.MethodPatch, "/api/collections/plugin_instances/records/"+f.instanceID, payload)
	auth := result["auth"].(map[string]any)
	for _, field := range []string{"clientSecret", "accessToken", "refreshToken"} {
		if value, present := auth[field]; present && value != "" {
			t.Fatalf("settings response disclosed %s", field)
		}
	}
}

func TestOAuthMetadataSettingsSaveStillRefreshes(t *testing.T) {
	f := newOAuthMetadataFixture(t)
	_, before := f.storedAuth(t)
	payload := f.modalPayload(t)
	for range 2 {
		f.saveModal(t, payload)
		_, saved := f.storedAuth(t)
		for _, field := range []string{"expiresAt", "tokenType", "oauthContext", "scope"} {
			if saved[field] != before[field] {
				t.Errorf("settings save changed omitted %s: got %#v, want %#v", field, saved[field], before[field])
			}
		}
		if _, ok := saved["unusedAuthNote"]; ok {
			t.Error("settings save restored an unrelated omitted auth field")
		}
	}
	instance, auth := f.storedAuth(t)
	if !pluginsystem.OAuthNeedsRefresh(auth) {
		t.Error("settings save lost the expiry needed to trigger refresh")
	}
	if _, err := pluginsystem.RefreshOAuthAuthIfNeeded(context.Background(), f.app, f.plugin, instance, auth); err != nil {
		t.Fatal(err)
	}
	if got := f.tokenRequests.Load(); got != 1 {
		t.Errorf("token requests = %d, want one actual refresh", got)
	}
	_, refreshed := f.storedAuth(t)
	if refreshed["accessToken"] != "strava-test-access-rotated" || refreshed["refreshToken"] != "strava-test-refresh-rotated" {
		t.Error("refresh did not persist rotated, encrypted tokens")
	}
	if pluginsystem.OAuthNeedsRefresh(refreshed) {
		t.Error("refreshed grant is still expired")
	}
}

func TestOAuthMetadataOpenModalBeforeRefreshKeepsLatestGrant(t *testing.T) {
	f := newOAuthMetadataFixture(t)
	payload := f.modalPayload(t)
	instance, auth := f.storedAuth(t)
	if _, err := pluginsystem.RefreshOAuthAuthIfNeeded(context.Background(), f.app, f.plugin, instance, auth); err != nil {
		t.Fatal(err)
	}
	_, refreshed := f.storedAuth(t)
	f.saveModal(t, payload)
	_, saved := f.storedAuth(t)
	for _, field := range []string{"accessToken", "refreshToken", "expiresAt", "tokenType", "oauthContext", "scope"} {
		if saved[field] != refreshed[field] {
			t.Errorf("old modal save changed latest %s: got %#v, want %#v", field, saved[field], refreshed[field])
		}
	}
}

func TestOAuthMetadataOpenModalBeforeRevokeKeepsGrantRemoved(t *testing.T) {
	f := newOAuthMetadataFixture(t)
	payload := f.modalPayload(t)
	f.request(t, http.MethodPost, "/api/plugin-system/oauth/revoke", map[string]any{"instanceId": f.instanceID})
	for _, stage := range []string{"revoke", "old modal save", "repeated modal save"} {
		if stage != "revoke" {
			f.saveModal(t, payload)
		}
		_, auth := f.storedAuth(t)
		for _, field := range []string{"accessToken", "refreshToken", "expiresAt", "tokenType", "oauthContext", "scope", "oauthState", "oauthCodeVerifier", "oauthRedirectURI"} {
			if value := auth[field]; value != nil {
				t.Errorf("%s restored cleared %s: %#v", stage, field, value)
			}
		}
		if auth["clientId"] != "strava-test-client" || auth["clientSecret"] != "strava-test-client-secret" {
			t.Errorf("%s removed client credentials", stage)
		}
	}
	if got := f.tokenRequests.Load(); got != 0 {
		t.Errorf("revoke/settings save made %d token requests", got)
	}
}

func TestOAuthMetadataCallbackClearsPersistedFlow(t *testing.T) {
	f := newOAuthMetadataFixture(t)
	instance, auth := f.storedAuth(t)
	auth["oauthCodeVerifier"] = "old-test-pkce-verifier"
	instance.Set("auth", auth)
	if err := f.app.Save(instance); err != nil {
		t.Fatal(err)
	}
	start := f.request(t, http.MethodPost, "/api/plugin-system/oauth/start", map[string]any{
		"pluginId": "strava", "instanceId": f.instanceID, "redirectUri": "https://wanderer.example/settings/plugins/oauth/callback",
	})
	f.request(t, http.MethodPost, "/api/plugin-system/oauth/callback", map[string]any{
		"instanceId": f.instanceID, "state": start["state"], "code": "test-authorization-code",
	})
	_, saved := f.storedAuth(t)
	for _, field := range pluginsystem.InternalOAuthTransientFields() {
		if saved[field] != nil {
			t.Errorf("callback restored cleared flow field %s", field)
		}
	}
	if saved["accessToken"] != "strava-test-access-rotated" || saved["oauthContext"] != "oauth_access_token" {
		t.Error("callback did not preserve the new grant")
	}
}

func TestOAuthMetadataSettingsCredentialEditsRemainEffective(t *testing.T) {
	for _, tc := range []struct {
		name   string
		secret any
	}{
		{name: "replacement", secret: "replacement-strava-client-secret"},
		{name: "explicit removal", secret: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newOAuthMetadataFixture(t)
			_, before := f.storedAuth(t)
			payload := f.modalPayload(t)
			submitted := payload["auth"].(map[string]any)
			submitted["clientId"] = "replacement-strava-client"
			submitted["clientSecret"] = tc.secret
			f.saveModal(t, payload)
			_, saved := f.storedAuth(t)
			if saved["clientId"] != "replacement-strava-client" || saved["clientSecret"] != tc.secret {
				t.Error("settings save restored credentials that were explicitly replaced or removed")
			}
			for _, field := range []string{"expiresAt", "tokenType", "oauthContext", "scope"} {
				if saved[field] != before[field] {
					t.Errorf("credential edit lost omitted %s", field)
				}
			}
		})
	}
}

func TestOAuthMetadataSettingsDoesNotInventMissingExpiry(t *testing.T) {
	f := newOAuthMetadataFixture(t, func(auth map[string]any) { delete(auth, "expiresAt") })
	f.saveModal(t, f.modalPayload(t))
	_, saved := f.storedAuth(t)
	if _, exists := saved["expiresAt"]; exists {
		t.Error("settings save invented expiry metadata for a previously affected grant")
	}
	if pluginsystem.OAuthNeedsRefresh(saved) {
		t.Error("missing expiry was treated as a known expiry")
	}
}
