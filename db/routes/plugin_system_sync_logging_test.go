package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

type pluginSyncLogTestApp struct {
	core.App
	logger *slog.Logger
}

func (app pluginSyncLogTestApp) Logger() *slog.Logger { return app.logger }

func TestSyncDetailRecoverySummaryIsBoundedAndCountsResponsesNotImports(t *testing.T) {
	for _, tc := range []struct {
		mode          string
		wantRecovered int
		wantImported  int
	}{
		{"success-across-capabilities", 3, 3},
		{"cancel-after-response", 1, 0},
		{"no-recovery", 0, 1},
		{"failed-detail-retry", 0, 0},
		{"list-recovery-only", 0, 1},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var logs bytes.Buffer
			app := pluginSyncLogTestApp{App: f.app, logger: slog.New(slog.NewJSONHandler(&logs, nil))}
			calls := map[string]int{}
			routeLists := 0
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					routeLists++
					if tc.mode == "list-recovery-only" && routeLists == 1 {
						return nil, retryTestFatal()
					}
					if tc.mode == "success-across-capabilities" {
						return retryTestList(t, "planned-1", "planned-2"), nil
					}
					return retryTestList(t, "item"), nil
				}
				if export == "list_activities_v1" {
					if tc.mode == "success-across-capabilities" {
						return retryTestJSON(t, pluginSystemListOutput{Items: []pluginsystem.TrailSummary{
							{Source: pluginsystem.TrailImportSource{ExternalID: "activity"}, Kind: "completed"},
						}}), nil
					}
					return retryTestList(t), nil
				}
				id, output := retryTestDetail(t, input)
				calls[id]++
				if tc.mode == "no-recovery" || tc.mode == "list-recovery-only" {
					return output, nil
				}
				if calls[id] == 1 || tc.mode == "failed-detail-retry" {
					return nil, pluginsystem.RuntimeSessionFatalError{Err: errors.New(strings.Repeat("private provider error password=secret\n", 100))}
				}
				if tc.mode == "cancel-after-response" {
					cancel()
				}
				return output, nil
			}}
			f.useRuntime(t, runtime)
			result, err := syncPluginInstance(ctx, app, nil, f.plugin, f.stored(t, f.instanceID))
			if tc.mode == "cancel-after-response" {
				if result != nil || !errors.Is(err, context.Canceled) || calls["item"] != 2 {
					t.Fatalf("unexpected cancellation: result=%#v, calls=%v, err=%v", result, calls, err)
				}
				f.importedIDs(t)
			} else if tc.mode == "failed-detail-retry" {
				f.incomplete(t, err)
				f.importedIDs(t)
			} else {
				if err != nil || result == nil || result.Imported != tc.wantImported {
					t.Fatalf("summary changed sync outcome: result=%#v, err=%v", result, err)
				}
				stored := f.stored(t, f.instanceID)
				if stored.GetString("status") != "configured" || len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 {
					t.Fatal("summary changed a clean sync status")
				}
				if tc.mode == "success-across-capabilities" {
					f.importedIDs(t, "planned-1", "planned-2", "activity")
				} else {
					f.importedIDs(t, "item")
				}
			}
			if tc.mode == "list-recovery-only" && routeLists != 2 {
				t.Fatalf("list retry was not exercised: %d calls", routeLists)
			}
			summaries := 0
			warnings := 0
			for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
				var entry map[string]any
				if err := json.Unmarshal(line, &entry); err != nil {
					t.Fatalf("invalid server log JSON: %v", err)
				}
				if entry["msg"] == "skipping plugin item after detail fetch failed" {
					warnings++
					wantCause := map[string]any{"code": "sync_incomplete", "cause": "runtime_session_fatal"}
					causeJSON, _ := json.Marshal(wantCause)
					gotJSON, _ := json.Marshal(entry["error"])
					if entry["level"] != "WARN" || entry["plugin"] != f.plugin.Manifest.ID || entry["instance"] != f.instanceID ||
						entry["capability"] != "list_routes" || entry["provider"] != "komoot" || entry["external_id"] != "item" ||
						!bytes.Equal(gotJSON, causeJSON) || len(line) > 512 || bytes.Contains(line, []byte("private")) || bytes.Contains(line, []byte("secret")) {
						t.Fatalf("changed scoped warning or exposed worker diagnostics: %s", line)
					}
				}
				if entry["msg"] != "plugin sync detail retry summary" {
					continue
				}
				summaries++
				if len(entry) != 6 || entry["level"] != "INFO" || entry["plugin"] != f.plugin.Manifest.ID || entry["instance"] != f.instanceID || entry["successful_detail_retries"] != float64(tc.wantRecovered) {
					t.Fatalf("unexpected recovery summary fields: %s", line)
				}
				if len(line) > 512 || bytes.Contains(line, []byte("private")) || bytes.Contains(line, []byte("secret")) || bytes.Contains(line, []byte("memory")) {
					t.Fatalf("recovery summary exposed diagnostics, guessed a cause or was unbounded: %d bytes", len(line))
				}
			}
			wantSummaries := 0
			if tc.wantRecovered > 0 {
				wantSummaries = 1
			}
			if summaries != wantSummaries || runtime.closed != runtime.opened {
				t.Fatalf("expected one summary only for successful detail retries: summaries=%d, want=%d, workers=%d/%d", summaries, wantSummaries, runtime.opened, runtime.closed)
			}
			wantWarnings := 0
			if tc.mode == "failed-detail-retry" {
				wantWarnings = 1
			}
			if warnings != wantWarnings {
				t.Fatalf("warnings=%d, want %d", warnings, wantWarnings)
			}
		})
	}
}
