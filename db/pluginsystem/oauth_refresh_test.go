package pluginsystem

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestOAuthRefreshFailureCode(t *testing.T) {
	endpoint := func(status int, body string) error {
		return OAuthTokenEndpointError{StatusCode: status, Status: fmt.Sprintf("%d status", status), Body: body}
	}
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"invalid grant", endpoint(400, `{"error":"invalid_grant","error_description":"expired"}`), "invalid_grant"},
		{"invalid client", endpoint(400, `{"error":"invalid_client"}`), "auth_failed"},
		{"unauthorized client", endpoint(401, `{"error":"unauthorized_client"}`), "auth_failed"},
		{"plugin sent a bad request", endpoint(400, `{"error":"invalid_request"}`), "plugin_error"},
		{"plugin uses an unsupported grant", endpoint(400, `{"error":"unsupported_grant_type"}`), "plugin_error"},
		{"strava style body without error field", endpoint(400, `{"message":"Bad Request","errors":[{"field":"refresh_token","code":"invalid"}]}`), "invalid_grant"},
		{"401 without body", endpoint(401, ""), "auth_failed"},
		{"wrapped", fmt.Errorf("oauth token refresh failed: %w", endpoint(401, "")), "auth_failed"},
		{"no refresh token stored", ErrOAuthRefreshTokenMissing, "invalid_grant"},
		{"rate limited", endpoint(429, ""), "rate_limited"},
		{"provider outage", endpoint(503, "<html>"), "provider_unavailable"},
		{"network", errors.New("dial tcp: connection refused"), "provider_unavailable"},
	}
	for _, tc := range cases {
		if got := OAuthRefreshFailureCode(tc.err); got != tc.want {
			t.Errorf("%s: OAuthRefreshFailureCode = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRefreshOAuthTokenHandlesInstanceChangesDuringExchange(t *testing.T) {
	cases := []struct {
		name         string
		change       func(*core.Record)
		wantAuth     string
		wantConfig   string
		wantStatus   string
		wantEnabled  bool
		wantConflict bool
	}{
		{
			name:        "unchanged instance accepts refresh",
			wantAuth:    "rotated",
			wantConfig:  "old",
			wantStatus:  "configured",
			wantEnabled: true,
		},
		{
			name: "reconnect wins over the old refresh chain",
			change: func(instance *core.Record) {
				instance.Set("auth", map[string]any{"clientId": "client", "refreshToken": "reconnected"})
				instance.Set("config", map[string]any{"plugin": map[string]any{"option": "new"}})
			},
			wantAuth:     "reconnected",
			wantConfig:   "new",
			wantStatus:   "configured",
			wantEnabled:  true,
			wantConflict: true,
		},
		{
			name: "config edit keeps the rotated token",
			change: func(instance *core.Record) {
				instance.Set("config", map[string]any{"plugin": map[string]any{"option": "new"}})
			},
			wantAuth:     "rotated",
			wantConfig:   "new",
			wantStatus:   "configured",
			wantEnabled:  true,
			wantConflict: true,
		},
		{
			name: "disable keeps the rotated token",
			change: func(instance *core.Record) {
				instance.Set("enabled", false)
				instance.Set("status", "disabled")
			},
			wantAuth:     "rotated",
			wantConfig:   "old",
			wantStatus:   "disabled",
			wantEnabled:  false,
			wantConflict: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := pbtests.NewTestApp(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer app.Cleanup()

			collection := core.NewBaseCollection("plugin_instances")
			collection.Fields.Add(
				&core.JSONField{Name: "auth"},
				&core.JSONField{Name: "config"},
				&core.TextField{Name: "status"},
				&core.BoolField{Name: "enabled"},
				&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
			)
			if err := app.Save(collection); err != nil {
				t.Fatal(err)
			}
			instance := core.NewRecord(collection)
			instance.Set("auth", map[string]any{"clientId": "client", "refreshToken": "old"})
			instance.Set("config", map[string]any{"plugin": map[string]any{"option": "old"}})
			instance.Set("status", "configured")
			instance.Set("enabled", true)
			if err := app.Save(instance); err != nil {
				t.Fatal(err)
			}
			snapshot, err := app.FindRecordById("plugin_instances", instance.Id)
			if err != nil {
				t.Fatal(err)
			}

			requestStarted := make(chan struct{})
			releaseResponse := make(chan struct{})
			released := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				close(requestStarted)
				<-releaseResponse
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"fresh","refresh_token":"rotated"}`))
			}))
			defer server.Close()
			defer func() {
				if !released {
					close(releaseResponse)
				}
			}()

			plugin := LocalPlugin{Manifest: Manifest{
				Auth: AuthManifest{Contexts: map[string]AuthContext{
					"oauth": {Type: AuthTypeOAuth2, TokenURL: server.URL + "/token"},
				}},
				Permissions: PermissionManifest{Network: NetworkPermissions{Connectors: []ConnectorTargetPermission{
					{Name: "api", Type: ConnectorTypePublicAPI, FixedBaseURL: server.URL, AllowedPathPrefixes: []string{"/token"}},
				}}},
			}}
			result := make(chan error, 1)
			go func() {
				auth := JSONMapFromRecord(snapshot, "auth")
				_, err := RefreshOAuthToken(context.Background(), app, plugin, snapshot, auth, "oauth")
				result <- err
			}()

			select {
			case <-requestStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("token refresh did not reach the provider")
			}
			if tc.change != nil {
				changed, err := app.FindRecordById("plugin_instances", instance.Id)
				if err != nil {
					t.Fatal(err)
				}
				tc.change(changed)
				if err := app.Save(changed); err != nil {
					t.Fatal(err)
				}
			}
			close(releaseResponse)
			released = true

			select {
			case err := <-result:
				if tc.wantConflict && !errors.Is(err, ErrPluginInstanceChanged) {
					t.Fatalf("expected refresh to abort with a concurrent-change error, got %v", err)
				}
				if !tc.wantConflict && err != nil {
					t.Fatalf("unexpected refresh error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("token refresh did not finish")
			}
			stored, err := app.FindRecordById("plugin_instances", instance.Id)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stored.GetString("auth"), tc.wantAuth) {
				t.Fatalf("unexpected stored credentials: %s", stored.GetString("auth"))
			}
			if tc.wantAuth != "rotated" && strings.Contains(stored.GetString("auth"), "rotated") {
				t.Fatalf("old refresh chain overwrote the new credentials: %s", stored.GetString("auth"))
			}
			if !strings.Contains(stored.GetString("config"), tc.wantConfig) ||
				stored.GetString("status") != tc.wantStatus || stored.GetBool("enabled") != tc.wantEnabled {
				t.Fatalf("concurrent change was not preserved: %#v", stored.FieldsData())
			}
			if !tc.wantConflict && (snapshot.GetString("auth") != stored.GetString("auth") ||
				snapshot.GetDateTime("updated").String() != stored.GetDateTime("updated").String() ||
				snapshot.Original().GetString("auth") != snapshot.GetString("auth")) {
				t.Fatal("successful refresh did not rebase the caller")
			}
		})
	}
}
