package routes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"pocketbase/pluginsystem"
)

func TestSyncDetailActionableRecoveryErrorsStopCalls(t *testing.T) {
	for _, phase := range []string{"initial-detail", "retry", "later-list", "later-detail"} {
		for _, inOutput := range []bool{false, true} {
			for _, tc := range []struct {
				code  string
				delay int
			}{
				{"auth_failed", 0},
				{"invalid_grant", 0},
				{"unauthorized", 90},
				{"rate_limited", 0},
				{"rate_limited", 90},
				{"provider_unavailable", 90},
				{"temporary_unavailable", 90},
			} {
				t.Run(fmt.Sprintf("%s/output=%v/%s/delay=%d", phase, inOutput, tc.code, tc.delay), func(t *testing.T) {
					f := newDetailRetryFixture(t)
					pluginErr := pluginsystem.PluginError{Code: tc.code, Message: "private actionable provider diagnostic"}
					if tc.delay > 0 {
						pluginErr.RetryAfterSeconds = &tc.delay
					}
					actionable := func(list bool) ([]byte, error) {
						if !inOutput {
							return nil, pluginsystem.PluginCallError{PluginError: pluginErr}
						}
						if list {
							return retryTestJSON(t, pluginSystemListOutput{Error: &pluginErr}), nil
						}
						return retryTestJSON(t, pluginSystemDetailOutput{Error: &pluginErr}), nil
					}
					calls := map[string]int{}
					completedLists := 0
					runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
						if export == "list_routes_v1" {
							return retryTestList(t, "before", "bad", "after"), nil
						}
						if export == "list_activities_v1" {
							completedLists++
							return actionable(true)
						}
						id, output := retryTestDetail(t, input)
						calls[id]++
						if id == "bad" {
							if phase == "initial-detail" || (calls[id] == 2 && phase == "retry") {
								return actionable(false)
							}
							return nil, retryTestFatal()
						}
						if id == "after" && phase == "later-detail" {
							return actionable(false)
						}
						if id == "after" && phase == "initial-detail" {
							// A later fatal item must never hide an earlier
							// actionable error by replacing its status.
							return nil, retryTestFatal()
						}
						return output, nil
					}}
					f.useRuntime(t, runtime)
					started := time.Now()
					result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
					finished := time.Now()
					var incomplete pluginsystem.SyncIncompleteError
					if result != nil || !errors.As(err, &incomplete) || strings.Contains(err.Error(), "private") {
						t.Fatalf("actionable partial sync returned success or private details: result=%#v, err=%v", result, err)
					}
					var capabilityErr pluginsystem.PluginCapabilityError
					var callErr pluginsystem.PluginCallError
					var retained pluginsystem.PluginError
					if errors.As(err, &capabilityErr) && capabilityErr.Err != nil {
						retained = *capabilityErr.Err
					} else if errors.As(err, &callErr) {
						retained = callErr.PluginError
					}
					if !reflect.DeepEqual(retained, pluginErr) {
						t.Fatalf("lost the actionable server-side cause: got=%#v, want=%#v", retained, pluginErr)
					}
					stored := f.stored(t, f.instanceID)
					lastError := pluginsystem.JSONMapFromRecord(stored, "last_error")
					want := pluginsystem.InstanceStatusForPluginError(pluginErr, started)
					if stored.GetString("status") != want.Status || lastError["code"] != tc.code || strings.Contains(fmt.Sprint(lastError["message"]), "private") || !stored.GetBool("enabled") || !stored.GetDateTime("last_sync_at").IsZero() {
						t.Fatalf("lost actionable status or marked full success: status=%q, last_error=%v", stored.GetString("status"), lastError)
					}
					delay := tc.delay
					if tc.code == "rate_limited" && delay == 0 {
						delay = 3600
					}
					if delay > 0 {
						got := stored.GetDateTime("retry_not_before").Time()
						if got.Before(started.Add(time.Duration(delay)*time.Second-time.Second)) || got.After(finished.Add(time.Duration(delay)*time.Second+time.Second)) || !shouldSkipPluginInstance(stored) {
							t.Fatalf("lost real provider backoff: %v, expected %d seconds", got, delay)
						}
					} else if !stored.GetDateTime("retry_not_before").IsZero() {
						t.Fatal("auth failure acquired an invented retry delay")
					}
					wantCalls := map[string]int{"before": 1, "bad": 2}
					wantLists := 0
					wantWorkers := 3
					if phase == "initial-detail" {
						wantCalls["bad"] = 1
						wantWorkers = 1
					} else if phase == "retry" {
						wantWorkers = 2
					} else {
						wantCalls["after"] = 1
					}
					if phase == "later-list" {
						wantLists = 1
						f.importedIDs(t, "before", "after")
					} else {
						f.importedIDs(t, "before")
					}
					if !reflect.DeepEqual(calls, wantCalls) || completedLists != wantLists || runtime.opened != wantWorkers || runtime.closed != runtime.opened {
						t.Fatalf("called past an actionable error or leaked a session: calls=%v, lists=%d, workers=%d/%d", calls, completedLists, runtime.opened, runtime.closed)
					}
				})
			}
		}
	}
}

func TestSyncIncompletePreservesLaterListItemLimit(t *testing.T) {
	f := newDetailRetryFixture(t)
	calls := map[string]int{}
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			return retryTestList(t, "before", "bad", "after"), nil
		}
		if export == "list_activities_v1" {
			return retryTestJSON(t, pluginSystemListOutput{Items: make([]pluginsystem.TrailSummary, defaultPluginSyncBatchLimit+1)}), nil
		}
		id, output := retryTestDetail(t, input)
		calls[id]++
		if id == "bad" {
			return nil, retryTestFatal()
		}
		return output, nil
	}}
	f.useRuntime(t, runtime)
	result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	var incomplete pluginsystem.SyncIncompleteError
	var limit pluginsystem.SyncItemLimitError
	if result != nil || !errors.As(err, &incomplete) || !errors.As(err, &limit) || limit.Returned != 51 || limit.MaxItems != 50 {
		t.Fatalf("lost partial outcome or typed list limit: result=%#v, err=%v", result, err)
	}
	f.importedIDs(t, "before", "after")
	stored := f.stored(t, f.instanceID)
	lastError := pluginsystem.JSONMapFromRecord(stored, "last_error")
	if stored.GetString("status") != "error" || lastError["code"] != "sync_item_limit_exceeded" || lastError["message"] != pluginsystem.SyncItemLimitMessage || !stored.GetDateTime("retry_not_before").IsZero() {
		t.Fatalf("item limit was masked by partial status: status=%q, last_error=%v", stored.GetString("status"), lastError)
	}
}

func TestSyncDetailRetryActionableErrorSurvivesCancellation(t *testing.T) {
	for _, code := range []string{"invalid_grant", "rate_limited"} {
		t.Run(code, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			delay := 90
			pluginErr := pluginsystem.PluginError{Code: code, RetryAfterSeconds: &delay, Message: "private provider diagnostic"}
			details := 0
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "before", "bad", "never"), nil
				}
				id, output := retryTestDetail(t, input)
				if id == "before" {
					return output, nil
				}
				if id != "bad" {
					t.Fatal("called past cancellation")
				}
				details++
				if details == 1 {
					return nil, retryTestFatal()
				}
				cancel()
				return nil, pluginsystem.PluginCallError{PluginError: pluginErr}
			}}
			f.useRuntime(t, runtime)
			result, err := syncPluginInstance(ctx, f.app, nil, f.plugin, f.stored(t, f.instanceID))
			var retained pluginsystem.PluginCallError
			if result != nil || !errors.Is(err, context.Canceled) || !errors.As(err, &retained) || retained.PluginError.Code != code {
				t.Fatalf("lost actionable cause or cancellation: result=%#v, err=%v", result, err)
			}
			f.importedIDs(t, "before")
			stored := f.stored(t, f.instanceID)
			want := pluginsystem.InstanceStatusForPluginError(pluginErr, time.Now())
			if stored.GetString("status") != want.Status || pluginsystem.JSONMapFromRecord(stored, "last_error")["code"] != code || !shouldSkipPluginInstance(stored) || details != 2 || runtime.opened != 2 || runtime.closed != 2 {
				t.Fatalf("cancellation masked real auth/backoff: status=%q, last_error=%v, details=%d, workers=%d/%d", stored.GetString("status"), stored.Get("last_error"), details, runtime.opened, runtime.closed)
			}
		})
	}
}
