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
		env      string
		expected []string
	}{
		{
			name:     "unset leaves the provider defaults alone",
			env:      "",
			expected: defaultScopes,
		},
		{
			name:     "single scope",
			env:      "openid",
			expected: []string{"openid"},
		},
		{
			name:     "comma separated list",
			env:      "openid,read_prefs",
			expected: []string{"openid", "read_prefs"},
		},
		{
			name:     "surrounding whitespace is ignored",
			env:      " openid , read_prefs ",
			expected: []string{"openid", "read_prefs"},
		},
		{
			name:     "empty entries are dropped",
			env:      "openid,,read_prefs,",
			expected: []string{"openid", "read_prefs"},
		},
		{
			name:     "only separators leaves the provider defaults alone",
			env:      ",, ,",
			expected: defaultScopes,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			// restore the stock factories so each scenario starts from the defaults
			original := auth.Providers["oidc"]
			t.Cleanup(func() {
				for _, name := range []string{"oidc", "oidc2", "oidc3"} {
					auth.Providers[name] = original
				}
			})

			t.Setenv("OIDC_SCOPES", s.env)

			configureOIDCScopes()

			for _, name := range []string{"oidc", "oidc2", "oidc3"} {
				scopes := auth.Providers[name]().Scopes()
				if !slices.Equal(scopes, s.expected) {
					t.Fatalf("%s: expected scopes %v, got %v", name, s.expected, scopes)
				}
			}
		})
	}
}
