package pluginsystem

import (
	"errors"
	"fmt"
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

// InstanceStatusForError converts sync/runtime errors into the persisted
// plugin_instances status fields used by the UI and cron backoff logic.
func InstanceStatusForError(err error, now time.Time) InstanceStatusUpdate {
	var capabilityErr PluginCapabilityError
	var callErr PluginCallError
	if errors.As(err, &capabilityErr) && capabilityErr.Err != nil {
		return InstanceStatusForPluginError(*capabilityErr.Err, now)
	}
	if errors.As(err, &callErr) {
		return InstanceStatusForPluginError(callErr.PluginError, now)
	}
	return InstanceStatusUpdate{
		Status:  "error",
		Code:    "provider_unavailable",
		Message: err.Error(),
	}
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
