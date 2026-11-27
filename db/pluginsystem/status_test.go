package pluginsystem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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

func TestInstanceStatusForIncompleteSync(t *testing.T) {
	for _, cause := range []error{
		nil,
		RuntimeSessionFatalError{Err: errors.New("private worker stderr and trace")},
		context.Canceled,
		PluginCapabilityError{Err: &PluginError{Code: "invalid_request", Message: "private detail"}},
		PluginCapabilityError{Err: &PluginError{Code: "provider_unavailable", Message: "private transport data"}},
		PluginCallError{PluginError: PluginError{Code: "temporary_unavailable", Message: "private provider data"}},
		PluginCapabilityError{Err: &PluginError{Code: "future_error", Message: "private unknown diagnostic"}},
	} {
		err := fmt.Errorf("sync: %w", SyncIncompleteError{Err: cause})
		update := InstanceStatusForError(err, time.Now())
		if update.Status != "error" || update.Code != "sync_incomplete" || update.Message != SyncIncompleteMessage || update.RetryNotBefore != nil {
			t.Fatalf("unexpected incomplete status for %T: %#v", cause, update)
		}
		if cause != nil && !errors.Is(err, cause) {
			t.Fatalf("lost control-flow cause: %v", err)
		}
		if err.Error() != "sync: "+SyncIncompleteMessage {
			t.Fatalf("exposed incomplete-sync diagnostics: %s", err)
		}
	}
}

func TestInstanceStatusForIncompleteSyncPreservesActionableCause(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	delay := 120
	for _, inOutput := range []bool{false, true} {
		for _, pluginErr := range []PluginError{
			{Code: "auth_failed"},
			{Code: "invalid_grant"},
			{Code: "unauthorized", RetryAfterSeconds: &delay},
			{Code: "rate_limited"},
			{Code: "rate_limited", RetryAfterSeconds: &delay},
			{Code: "provider_unavailable", RetryAfterSeconds: &delay},
			{Code: "temporary_unavailable", RetryAfterSeconds: &delay},
			{Code: "future_error", RetryAfterSeconds: &delay},
		} {
			t.Run(fmt.Sprintf("output=%v/%s/delay=%v", inOutput, pluginErr.Code, pluginErr.RetryAfterSeconds != nil), func(t *testing.T) {
				pluginErr.Message = "private provider diagnostic"
				var cause error = PluginCallError{PluginError: pluginErr}
				if inOutput {
					cause = PluginCapabilityError{Err: &pluginErr}
				}
				err := fmt.Errorf("sync: %w", SyncIncompleteError{Err: fmt.Errorf("call: %w", cause)})
				got := InstanceStatusForError(err, now)
				want := InstanceStatusForPluginError(pluginErr, now)
				if got.Status != want.Status || got.Code != want.Code || (got.RetryNotBefore == nil) != (want.RetryNotBefore == nil) {
					t.Fatalf("masked actionable cause: got=%#v, want=%#v", got, want)
				}
				if want.RetryNotBefore != nil && !got.RetryNotBefore.Equal(*want.RetryNotBefore) {
					t.Fatalf("lost provider backoff: got=%v, want=%v", got.RetryNotBefore, want.RetryNotBefore)
				}
				if strings.Contains(got.Message, "private") || got.Message == "" || !errors.Is(err, cause) || !IsActionableSyncError(err) {
					t.Fatalf("lost cause or exposed diagnostics: update=%#v, err=%v", got, err)
				}
			})
		}
	}
	limit := SyncItemLimitError{Returned: 51, MaxItems: 50}
	err := SyncIncompleteError{Err: fmt.Errorf("list: %w", limit)}
	got := InstanceStatusForError(err, now)
	if got.Status != "error" || got.Code != "sync_item_limit_exceeded" || got.Message != SyncItemLimitMessage || got.RetryNotBefore != nil || !IsActionableSyncError(err) {
		t.Fatalf("incomplete sync masked the protocol limit: %#v", got)
	}
}

func TestIsActionableSyncErrorKeepsExpectedSkipsUnchanged(t *testing.T) {
	zero := 0
	negative := -1
	for _, err := range []error{
		nil,
		errors.New("generic failure"),
		RuntimeSessionFatalError{Err: errors.New("worker ended")},
		PluginCapabilityError{Err: &PluginError{Code: "invalid_request"}},
		PluginCallError{PluginError: PluginError{Code: "provider_unavailable"}},
		PluginCapabilityError{Err: &PluginError{Code: "temporary_unavailable", RetryAfterSeconds: &zero}},
		PluginCapabilityError{Err: &PluginError{Code: "invalid_request", RetryAfterSeconds: &negative}},
	} {
		if IsActionableSyncError(err) {
			t.Fatalf("normal skip became actionable: %v", err)
		}
	}
}

func TestSyncIncompleteErrorLogsOnlySafeBoundedCauseMetadata(t *testing.T) {
	private := strings.Repeat("password=secret\n", 8192)
	for _, tc := range []struct {
		cause error
		kind  string
	}{
		{nil, "unspecified"},
		{errors.New(private), "host_error"},
		{RuntimeSessionFatalError{Err: errors.New(private)}, "runtime_session_fatal"},
		{context.Canceled, "cancelled"},
		{context.DeadlineExceeded, "deadline_exceeded"},
		{SyncItemLimitError{Returned: 51, MaxItems: 50}, "sync_item_limit_exceeded"},
		{PluginCallError{PluginError: PluginError{Code: "rate_limited", Message: private}}, "rate_limited"},
		{PluginCapabilityError{Err: &PluginError{Code: private, Message: private}}, "plugin_error"},
	} {
		err := SyncIncompleteError{Err: tc.cause}
		var out bytes.Buffer
		slog.New(slog.NewJSONHandler(&out, nil)).Warn("sync failed", "error", err)
		if out.Len() > 512 || strings.Contains(out.String(), "secret") || strings.Contains(out.String(), "password") {
			t.Fatalf("unbounded or private sync diagnostic: %d bytes", out.Len())
		}
		var log struct {
			Error struct {
				Code  string `json:"code"`
				Cause string `json:"cause"`
			} `json:"error"`
		}
		if json.Unmarshal(out.Bytes(), &log) != nil || log.Error.Code != "sync_incomplete" || log.Error.Cause != tc.kind {
			t.Fatalf("missing safe cause classification: %s", out.String())
		}
		if tc.cause != nil && !errors.Is(err, tc.cause) {
			t.Fatal("server diagnostic lost its original wrapped cause")
		}
	}
}
