package pluginsystem

import (
	"context"
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
		URL:  "https://example.com/upload?existing=true",
		Auth: "account",
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
		}},
		Auth: map[string]any{"apiKey": "secret"},
		Spec: &spec,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.URL != "https://example.com/upload?existing=true&key=secret" {
		t.Fatalf("unexpected url: %q", spec.URL)
	}
}

func TestAuthForPluginRefresh(t *testing.T) {
	filtered := AuthForPluginRefresh(map[string]any{
		"email":       "user@example.com",
		"password":    "secret",
		"accessToken": "token",
	}, AuthContext{
		SecretFields: []string{"email", "password"},
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
