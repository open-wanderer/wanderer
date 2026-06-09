package pluginsystem

import (
	"context"
	"net/http"
	"testing"
)

func TestValidateAuthContext(t *testing.T) {
	tests := []struct {
		name    string
		context AuthContext
		wantErr bool
	}{
		{
			name: "oauth2",
			context: AuthContext{
				Type:             AuthTypeOAuth2,
				AuthorizationURL: "https://example.com/oauth/authorize",
				TokenURL:         "https://example.com/oauth/token",
				Refresh:          &AuthRefresh{Mode: AuthRefreshModeHost},
			},
		},
		{
			name:    "missing bearer secret",
			context: AuthContext{Type: AuthTypeBearer},
			wantErr: true,
		},
		{
			name: "session",
			context: AuthContext{
				Type:         AuthTypeSession,
				SecretFields: []string{"email", "password"},
				Refresh:      &AuthRefresh{Mode: AuthRefreshModePlugin, Function: "refresh_session_v1"},
			},
		},
		{
			name:    "unsupported",
			context: AuthContext{Type: "mtls"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateAuthContext("default", test.context)
			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInjectHostRequestAuthWithBearer(t *testing.T) {
	spec := HostRequestSpec{Auth: "account"}

	err := InjectHostRequestAuth(context.Background(), AuthInjectionInput{
		Plugin: LocalPlugin{Manifest: Manifest{
			Auth: AuthManifest{Contexts: map[string]AuthContext{
				"account": {
					Type:        AuthTypeBearer,
					SecretField: "token",
				},
			}},
			Permissions: PermissionManifest{Auth: []string{"account"}},
		}},
		Auth: map[string]any{"token": "abc123"},
		Spec: &spec,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := spec.Headers[AuthHeaderAuthorization]; got != AuthSchemeBearer+" abc123" {
		t.Fatalf("unexpected authorization header: %q", got)
	}
}

func TestInjectHostRequestAuthWithAPIKeyQuery(t *testing.T) {
	spec := HostRequestSpec{
		Auth: "account",
		Target: RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/upload",
			Query:     []QueryParam{{Name: "existing", Value: "true"}},
		},
	}

	err := InjectHostRequestAuth(context.Background(), AuthInjectionInput{
		Plugin: LocalPlugin{Manifest: Manifest{
			Auth: AuthManifest{Contexts: map[string]AuthContext{
				"account": {
					Type:        AuthTypeAPIKey,
					SecretField: "apiKey",
					Placement:   AuthPlacementQuery,
					Name:        "key",
				},
			}},
			Permissions: PermissionManifest{Auth: []string{"account"}},
		}},
		Auth: map[string]any{"apiKey": "secret"},
		Spec: &spec,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Target.Query) != 2 || spec.Target.Query[1].Name != "key" || spec.Target.Query[1].Value != "secret" {
		t.Fatalf("unexpected query: %#v", spec.Target.Query)
	}
}

func TestInjectHostRequestAuthFromPolicyWithBearer(t *testing.T) {
	spec := HostRequestSpec{
		Auth:    "account",
		Headers: map[string]string{AuthHeaderAuthorization: "plugin supplied"},
	}
	manifest := Manifest{
		Auth: AuthManifest{Contexts: map[string]AuthContext{
			"account": {Type: AuthTypeBearer, SecretField: "token"},
		}},
		Permissions: PermissionManifest{Auth: []string{"account"}},
	}

	err := InjectHostRequestAuthFromPolicy(manifest, map[string]any{"token": "host-secret"}, &spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := spec.Headers[AuthHeaderAuthorization]; got != AuthSchemeBearer+" host-secret" {
		t.Fatalf("unexpected authorization header: %q", got)
	}
}

func TestInjectHostRequestAuthFromPolicyFailsWithEmptyAuth(t *testing.T) {
	spec := HostRequestSpec{
		Auth:    "account",
		Headers: map[string]string{AuthHeaderAuthorization: "plugin supplied"},
	}
	manifest := Manifest{
		Auth: AuthManifest{Contexts: map[string]AuthContext{
			"account": {Type: AuthTypeBearer, SecretField: "token"},
		}},
		Permissions: PermissionManifest{Auth: []string{"account"}},
	}

	err := InjectHostRequestAuthFromPolicy(manifest, map[string]any{}, &spec)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "bearer token is missing" {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := spec.Headers[AuthHeaderAuthorization]; got != "plugin supplied" {
		t.Fatalf("unexpected authorization header mutation: %q", got)
	}
}

func TestInjectHostRequestAuthFromPolicyRejectsSessionAuth(t *testing.T) {
	spec := HostRequestSpec{Auth: "account"}
	manifest := Manifest{
		Auth: AuthManifest{Contexts: map[string]AuthContext{
			"account": {
				Type:         AuthTypeSession,
				SecretFields: []string{"password"},
				Refresh:      &AuthRefresh{Mode: AuthRefreshModePlugin, Function: "refresh_session_v1"},
			},
		}},
		Permissions: PermissionManifest{Auth: []string{"account"}},
	}

	err := InjectHostRequestAuthFromPolicy(manifest, map[string]any{"password": "secret"}, &spec)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "session auth requires handler-managed injection" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInjectHostRequestAuthFromPolicyWithAPIKeyQuery(t *testing.T) {
	spec := HostRequestSpec{
		Auth: "account",
		Target: RequestTarget{
			Type:  "connector",
			Path:  "/assets",
			Query: []QueryParam{{Name: "api_key", Value: "plugin"}},
		},
	}
	manifest := Manifest{
		Auth: AuthManifest{Contexts: map[string]AuthContext{
			"account": {
				Type:        AuthTypeAPIKey,
				SecretField: "apiKey",
				Placement:   AuthPlacementQuery,
				Name:        "api_key",
			},
		}},
		Permissions: PermissionManifest{Auth: []string{"account"}},
	}

	err := InjectHostRequestAuthFromPolicy(manifest, map[string]any{"apiKey": "host-secret"}, &spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spec.Target.Query) != 1 || spec.Target.Query[0].Value != "host-secret" {
		t.Fatalf("unexpected query: %#v", spec.Target.Query)
	}
}

func TestInjectHostRequestAuthRequiresSpec(t *testing.T) {
	err := InjectHostRequestAuth(context.Background(), AuthInjectionInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "host request spec is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInjectHostRequestAuthValidatesPermission(t *testing.T) {
	spec := HostRequestSpec{Auth: "account"}
	err := InjectHostRequestAuth(context.Background(), AuthInjectionInput{
		Plugin: LocalPlugin{Manifest: Manifest{
			Auth: AuthManifest{Contexts: map[string]AuthContext{
				"account": {Type: AuthTypeBearer, SecretField: "token"},
			}},
		}},
		Auth: map[string]any{"token": "abc123"},
		Spec: &spec,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `auth context "account" is not permitted` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInjectRequestAuthForContextPreservesQueryOrder(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.test/media?z=last&api_key=plugin&a=first", nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		Auth: AuthManifest{Contexts: map[string]AuthContext{
			"account": {
				Type:        AuthTypeAPIKey,
				SecretField: "apiKey",
				Placement:   AuthPlacementQuery,
				Name:        "api_key",
			},
		}},
		Permissions: PermissionManifest{Auth: []string{"account"}},
	}

	err = InjectRequestAuthForContext(manifest, map[string]any{"apiKey": "host-secret"}, "account", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL.RawQuery != "z=last&a=first&api_key=host-secret" {
		t.Fatalf("unexpected raw query: %q", req.URL.RawQuery)
	}
}

func TestAuthForPluginRefresh(t *testing.T) {
	filtered := AuthForPluginRefresh(map[string]any{
		"email":       "user@example.com",
		"password":    "secret",
		"accessToken": "token",
	}, AuthContext{
		Fields:       []string{"email", "password"},
		SecretFields: []string{"password"},
	})

	if len(filtered) != 2 {
		t.Fatalf("unexpected filtered auth: %#v", filtered)
	}
	if filtered["email"] != "user@example.com" || filtered["password"] != "secret" {
		t.Fatalf("unexpected filtered auth: %#v", filtered)
	}
	if _, ok := filtered["accessToken"]; ok {
		t.Fatalf("unexpected access token in plugin refresh auth: %#v", filtered)
	}
}
