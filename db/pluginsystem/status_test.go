package pluginsystem

import (
	"errors"
	"testing"
	"time"
)

func TestInstanceStatusForPluginCapabilityError(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	retryAfter := 120

	update := InstanceStatusForError(PluginCapabilityError{Err: &PluginError{
		Code:              "rate_limited",
		Message:           "try later",
		RetryAfterSeconds: &retryAfter,
	}}, now)

	if update.Status != "rate_limited" {
		t.Fatalf("expected status rate_limited, got %q", update.Status)
	}
	if update.Code != "rate_limited" || update.Message != "try later" {
		t.Fatalf("unexpected error fields: %#v", update)
	}
	if update.RetryNotBefore == nil || !update.RetryNotBefore.Equal(now.Add(120*time.Second)) {
		t.Fatalf("unexpected retry time: %#v", update.RetryNotBefore)
	}
}

func TestInstanceStatusForPluginCallError(t *testing.T) {
	update := InstanceStatusForError(PluginCallError{
		PluginID: "strava",
		Export:   "list_activities_v1",
		PluginError: PluginError{
			Code: "invalid_grant",
		},
	}, time.Now())

	if update.Status != "needs_reauth" {
		t.Fatalf("expected status needs_reauth, got %q", update.Status)
	}
	if update.Code != "invalid_grant" || update.Message != "invalid_grant" {
		t.Fatalf("unexpected error fields: %#v", update)
	}
	if update.RetryNotBefore != nil {
		t.Fatalf("did not expect retry time: %#v", update.RetryNotBefore)
	}
}

func TestInstanceStatusForGenericError(t *testing.T) {
	update := InstanceStatusForError(errors.New("network unavailable"), time.Now())

	if update.Status != "error" {
		t.Fatalf("expected status error, got %q", update.Status)
	}
	if update.Code != "provider_unavailable" || update.Message != "network unavailable" {
		t.Fatalf("unexpected error fields: %#v", update)
	}
}
