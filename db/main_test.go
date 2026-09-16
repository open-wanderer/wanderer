package main

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/tools/auth"
)

func TestConfigureOIDCScopes(t *testing.T) {
	defaultScopes := auth.Providers["oidc"]().Scopes()

	scenarios := []struct {
		name     string
		env      map[string]string
		expected map[string][]string
	}{
		{
			name: "unset leaves the provider defaults alone",
			env:  map[string]string{},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "single scope",
			env:  map[string]string{"OIDC_SCOPES": "openid"},
			expected: map[string][]string{
				"oidc":  {"openid"},
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "comma separated list",
			env:  map[string]string{"OIDC2_SCOPES": "openid,read_prefs"},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": {"openid", "read_prefs"},
				"oidc3": defaultScopes,
			},
		},
		{
			name: "surrounding whitespace is ignored",
			env:  map[string]string{"OIDC3_SCOPES": " openid , read_prefs "},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": defaultScopes,
				"oidc3": {"openid", "read_prefs"},
			},
		},
		{
			name: "empty entries are dropped",
			env:  map[string]string{"OIDC_SCOPES": "openid,,read_prefs,"},
			expected: map[string][]string{
				"oidc":  {"openid", "read_prefs"},
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "only separators leaves the provider defaults alone",
			env:  map[string]string{"OIDC_SCOPES": ",, ,"},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "each slot gets its own scopes",
			env: map[string]string{
				"OIDC_SCOPES":  "openid",
				"OIDC3_SCOPES": "openid,profile",
			},
			expected: map[string][]string{
				"oidc":  {"openid"},
				"oidc2": defaultScopes,
				"oidc3": {"openid", "profile"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			// restore the stock factories so each scenario starts from the defaults
			original := auth.Providers["oidc"]
			t.Cleanup(func() {
				for name := range oidcScopesEnv {
					auth.Providers[name] = original
				}
			})

			for _, env := range oidcScopesEnv {
				t.Setenv(env, s.env[env])
			}

			configureOIDCScopes()

			for name, expected := range s.expected {
				scopes := auth.Providers[name]().Scopes()
				if !slices.Equal(scopes, expected) {
					t.Fatalf("%s: expected scopes %v, got %v", name, expected, scopes)
				}
			}
		})
	}
}
