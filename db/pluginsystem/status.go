package pluginsystem

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// PluginCapabilityError wraps a plugin-reported error returned inside a
// successful export response, so status mapping can treat it like runtime
// PluginCallError failures.
type PluginCapabilityError struct {
	Err *PluginError
}

func (e PluginCapabilityError) Error() string {
	if e.Err == nil {
		return "plugin error"
	}
	if e.Err.Message == "" {
		return fmt.Sprintf("plugin error %s", e.Err.Code)
	}
	return fmt.Sprintf("plugin error %s: %s", e.Err.Code, e.Err.Message)
}

// InstanceStatusUpdate contains the normalized status fields that are written
// back to plugin_instances after a failed sync.
type InstanceStatusUpdate struct {
	Status         string
	Code           string
	Message        string
	RetryNotBefore *time.Time
}

// SyncIncompleteError reports an unrecovered main-import detail failure without
// exposing worker stderr or claiming a particular trap cause. Err is retained
// for classification and server diagnostics, never as a persisted message.
type SyncIncompleteError struct{ Err error }

const SyncIncompleteMessage = "Sync incomplete: some items could not be imported. Successfully imported items were kept. The next scheduled sync will try again. If the problem persists, contact your administrator or the plugin maintainer."

func (e SyncIncompleteError) Error() string { return SyncIncompleteMessage }

func (e SyncIncompleteError) Unwrap() error { return e.Err }

// LogValue exposes bounded, allowlisted cause metadata to server logs. The
// original cause remains available through Unwrap, but its message may contain
// provider data, credentials or stderr and must not be printed automatically.
func (e SyncIncompleteError) LogValue() slog.Value {
	cause := "host_error"
	var limitErr SyncItemLimitError
	switch {
	case e.Err == nil:
		cause = "unspecified"
	case errors.Is(e.Err, context.Canceled):
		cause = "cancelled"
	case errors.Is(e.Err, context.DeadlineExceeded):
		cause = "deadline_exceeded"
	case errors.As(e.Err, &limitErr):
		cause = "sync_item_limit_exceeded"
	default:
		if pluginErr, ok := pluginErrorFromSyncError(e.Err); ok {
			switch code := strings.TrimSpace(pluginErr.Code); code {
			case "auth_failed", "invalid_grant", "unauthorized", "rate_limited",
				"provider_unavailable", "temporary_unavailable", "invalid_request",
				"internal_error", "plugin_error", "unsupported_protocol":
				cause = code
			default:
				cause = "plugin_error"
			}
		} else if IsRuntimeSessionFatalError(e.Err) {
			cause = "runtime_session_fatal"
		}
	}
	return slog.GroupValue(slog.String("code", "sync_incomplete"), slog.String("cause", cause))
}

// InstanceStatusForError converts sync/runtime errors into the persisted
// plugin_instances status fields used by the UI and cron backoff logic.
func InstanceStatusForError(err error, now time.Time) InstanceStatusUpdate {
	var incompleteErr SyncIncompleteError
	incomplete := errors.As(err, &incompleteErr)
	var limitErr SyncItemLimitError
	if errors.As(err, &limitErr) {
		return InstanceStatusUpdate{Status: "error", Code: "sync_item_limit_exceeded", Message: SyncItemLimitMessage}
	}
	if pluginErr, ok := pluginErrorFromSyncError(err); ok {
		update := InstanceStatusForPluginError(pluginErr, now)
		if !incomplete {
			return update
		}
		if actionablePluginSyncError(pluginErr) {
			// Preserve the required action/backoff, not a provider-controlled
			// diagnostic. The UI localizes known codes independently.
			switch update.Status {
			case "needs_reauth":
				update.Message = "Sync incomplete. Check the plugin connection or credentials in settings. Successfully imported items were kept."
			case "rate_limited":
				update.Message = "Sync incomplete. The provider has limited requests. Sync will resume after the retry delay. Successfully imported items were kept."
			default:
				update.Message = "Sync incomplete. The provider requested a retry delay. Sync will resume after that delay. Successfully imported items were kept."
			}
			return update
		}
	}
	if incomplete {
		return InstanceStatusUpdate{Status: "error", Code: "sync_incomplete", Message: SyncIncompleteMessage}
	}
	return InstanceStatusUpdate{
		Status:  "error",
		Code:    "provider_unavailable",
		Message: err.Error(),
	}
}

// IsActionableSyncError identifies errors that must stop main-import calls,
// including on the first detail attempt. Normal expected skips without a retry
// delay retain their existing behavior.
func IsActionableSyncError(err error) bool {
	var limitErr SyncItemLimitError
	if errors.As(err, &limitErr) {
		return true
	}
	pluginErr, ok := pluginErrorFromSyncError(err)
	return ok && actionablePluginSyncError(pluginErr)
}

func actionablePluginSyncError(pluginErr PluginError) bool {
	switch strings.TrimSpace(pluginErr.Code) {
	case "auth_failed", "invalid_grant", "unauthorized", "rate_limited":
		return true
	}
	return pluginErr.RetryAfterSeconds != nil && *pluginErr.RetryAfterSeconds > 0
}

func pluginErrorFromSyncError(err error) (PluginError, bool) {
	var capabilityErr PluginCapabilityError
	var callErr PluginCallError
	if errors.As(err, &capabilityErr) && capabilityErr.Err != nil {
		return *capabilityErr.Err, true
	}
	if errors.As(err, &callErr) {
		return callErr.PluginError, true
	}
	return PluginError{}, false
}

// InstanceStatusForPluginError maps the stable plugin error codes from the ABI
// to host instance states. retryAfterSeconds wins over default retry windows.
func InstanceStatusForPluginError(pluginErr PluginError, now time.Time) InstanceStatusUpdate {
	code := strings.TrimSpace(pluginErr.Code)
	if code == "" {
		code = "provider_unavailable"
	}
	message := strings.TrimSpace(pluginErr.Message)
	if message == "" {
		message = code
	}

	status := "error"
	switch code {
	case "auth_failed", "invalid_grant", "unauthorized":
		status = "needs_reauth"
	case "rate_limited":
		status = "rate_limited"
	case "provider_unavailable", "temporary_unavailable":
		status = "unavailable"
	}

	update := InstanceStatusUpdate{
		Status:  status,
		Code:    code,
		Message: message,
	}
	if pluginErr.RetryAfterSeconds != nil && *pluginErr.RetryAfterSeconds > 0 {
		retryNotBefore := now.Add(time.Duration(*pluginErr.RetryAfterSeconds) * time.Second)
		update.RetryNotBefore = &retryNotBefore
	} else if code == "rate_limited" {
		retryNotBefore := now.Add(time.Hour)
		update.RetryNotBefore = &retryNotBefore
	}
	return update
}
