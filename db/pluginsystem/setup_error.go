package pluginsystem

import "errors"

// PluginSetupErrorCode identifies a safe, user-facing category for failures
// encountered while discovering a plugin bundle. The wrapped diagnostic may
// contain server paths and remains private to logs and superuser responses.
type PluginSetupErrorCode string

// Keep this public code list in sync with pluginSetupErrorKeys in
// web/src/lib/util/plugin_error_i18n.ts and the setupErrorCode OpenAPI enum in
// web/src/routes/api/v1/plugin-system/plugins/+server.ts.
const (
	SetupErrorCodeFailed                      PluginSetupErrorCode = "setup_failed"
	SetupErrorCodeManifestMissing             PluginSetupErrorCode = "manifest_missing"
	SetupErrorCodeManifestUnreadable          PluginSetupErrorCode = "manifest_unreadable"
	SetupErrorCodeManifestInvalid             PluginSetupErrorCode = "manifest_invalid"
	SetupErrorCodeRuntimeEntrypointInvalid    PluginSetupErrorCode = "runtime_entrypoint_invalid"
	SetupErrorCodeRuntimeEntrypointMissing    PluginSetupErrorCode = "runtime_entrypoint_missing"
	SetupErrorCodeRuntimeEntrypointUnreadable PluginSetupErrorCode = "runtime_entrypoint_unreadable"
)

type pluginSetupError struct {
	code PluginSetupErrorCode
	err  error
}

func (e *pluginSetupError) Error() string {
	return e.err.Error()
}

func (e *pluginSetupError) Unwrap() error {
	return e.err
}

func wrapPluginSetupError(code PluginSetupErrorCode, err error) error {
	return &pluginSetupError{code: code, err: err}
}

func pluginSetupErrorCode(err error) PluginSetupErrorCode {
	var setupError *pluginSetupError
	if errors.As(err, &setupError) && setupError.code != "" {
		return setupError.code
	}
	return SetupErrorCodeFailed
}

// PublicPluginSetupErrorCode restricts API-visible setup errors to the stable
// public allowlist. Unknown internal codes fail closed to a generic category.
func PublicPluginSetupErrorCode(code PluginSetupErrorCode) PluginSetupErrorCode {
	switch code {
	case SetupErrorCodeFailed,
		SetupErrorCodeManifestMissing,
		SetupErrorCodeManifestUnreadable,
		SetupErrorCodeManifestInvalid,
		SetupErrorCodeRuntimeEntrypointInvalid,
		SetupErrorCodeRuntimeEntrypointMissing,
		SetupErrorCodeRuntimeEntrypointUnreadable:
		return code
	default:
		return SetupErrorCodeFailed
	}
}
