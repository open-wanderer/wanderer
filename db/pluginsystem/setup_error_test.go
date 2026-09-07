package pluginsystem

import "testing"

func TestPublicPluginSetupErrorCode(t *testing.T) {
	publicCodes := []PluginSetupErrorCode{
		SetupErrorCodeFailed,
		SetupErrorCodeManifestMissing,
		SetupErrorCodeManifestUnreadable,
		SetupErrorCodeManifestInvalid,
		SetupErrorCodeRuntimeEntrypointInvalid,
		SetupErrorCodeRuntimeEntrypointMissing,
		SetupErrorCodeRuntimeEntrypointUnreadable,
	}
	for _, code := range publicCodes {
		t.Run(string(code), func(t *testing.T) {
			if got := PublicPluginSetupErrorCode(code); got != code {
				t.Fatalf("PublicPluginSetupErrorCode(%q) = %q, want unchanged", code, got)
			}
		})
	}

	for _, code := range []PluginSetupErrorCode{"", "private:/server/path"} {
		t.Run("private_"+string(code), func(t *testing.T) {
			if got := PublicPluginSetupErrorCode(code); got != SetupErrorCodeFailed {
				t.Fatalf("PublicPluginSetupErrorCode(%q) = %q, want %q", code, got, SetupErrorCodeFailed)
			}
		})
	}
}
