package routes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"
)

// These fixtures use the real importer and isolated records/files, but no
// provider, HTTP server, WASM module or worker process.
type detailRetryFixture struct {
	*pluginInstanceFixture
	instanceID string
	plugin     pluginsystem.LocalPlugin
}

func newDetailRetryFixture(t *testing.T) *detailRetryFixture {
	t.Helper()
	f := &detailRetryFixture{pluginInstanceFixture: newPluginInstanceFixture(t)}
	drainTestAppStartupLogs(t, f.app)
	collection := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := f.app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	actors := collection("activitypub_actors", &core.TextField{Name: "user"})
	actor := core.NewRecord(actors)
	actor.Set("user", f.ownerID)
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	trails := collection("trails",
		&core.TextField{Name: "name"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20},
		&core.BoolField{Name: "completed"},
	)
	collection("trail_external_reference",
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.TextField{Name: "user"}, &core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	f.instanceID = f.instance(t, true, time.Time{})
	instance := f.stored(t, f.instanceID)
	instance.Set("config", map[string]any{
		"plugin": map[string]any{"after": "2026-01-01"},
		"host":   map[string]any{"createSummitLogForCompleted": false},
	})
	if err := f.app.Save(instance); err != nil {
		t.Fatal(err)
	}
	f.plugin.Manifest.ID = "komoot"
	for _, descriptor := range syncCapabilityDescriptors {
		f.plugin.Manifest.Capabilities = append(f.plugin.Manifest.Capabilities,
			pluginsystem.CapabilityManifest{Name: descriptor.CapabilityName, Version: descriptor.Version, Export: descriptor.CapabilityName + "_v1"},
			pluginsystem.CapabilityManifest{Name: descriptor.DetailName, Version: descriptor.Version, Export: descriptor.DetailName + "_v1"})
	}
	return f
}

func (f *detailRetryFixture) useRuntime(t *testing.T, runtime *detailRetryRuntime) {
	t.Helper()
	original := pluginRuntimeFor
	pluginRuntimeFor = func(pluginsystem.LocalPlugin) (pluginsystem.Runtime, error) { return runtime, nil }
	t.Cleanup(func() { pluginRuntimeFor = original })
}

func (f *detailRetryFixture) importedIDs(t *testing.T, want ...string) {
	t.Helper()
	refs, err := f.app.FindRecordsByFilter("trail_external_reference", "", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, ref := range refs {
		if got[ref.GetString("external_id")] {
			t.Fatalf("duplicate import: %s", ref.GetString("external_id"))
		}
		got[ref.GetString("external_id")] = true
		trail, err := f.app.FindRecordById("trails", ref.GetString("trail"))
		if err != nil || trail.GetString("gpx") == "" || ref.GetString("user") != f.ownerID {
			t.Fatalf("import lost its trail, GPX or owner: ref=%v, err=%v", ref, err)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("imports=%v, want %v", got, want)
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("missing import %q in %v", id, got)
		}
	}
	trails, err := f.app.FindRecordsByFilter("trails", "", "", 0, 0)
	if err != nil || len(trails) != len(want) {
		t.Fatalf("orphan or duplicate trails: count=%d, want %d, err=%v", len(trails), len(want), err)
	}
}

func (f *detailRetryFixture) incomplete(t *testing.T, err error) {
	t.Helper()
	var incomplete pluginsystem.SyncIncompleteError
	if !errors.As(err, &incomplete) || strings.Contains(err.Error(), "private") {
		t.Fatalf("missing safe incomplete error: %v", err)
	}
	stored := f.stored(t, f.instanceID)
	lastError := pluginsystem.JSONMapFromRecord(stored, "last_error")
	if stored.GetString("status") != "error" || lastError["code"] != "sync_incomplete" || lastError["message"] != pluginsystem.SyncIncompleteMessage {
		t.Fatalf("incomplete result was cleared or exposed worker diagnostics: status=%q, last_error=%v", stored.GetString("status"), lastError)
	}
	if !stored.GetBool("enabled") || !stored.GetDateTime("retry_not_before").IsZero() || shouldSkipPluginInstance(stored) {
		t.Fatal("incomplete result disabled the instance or blocked the next cron run")
	}
}

type detailRetryRuntime struct {
	t      *testing.T
	opened int
	closed int
	onOpen func(context.Context, int, pluginsystem.LocalPlugin, pluginsystem.RequestPolicyContext) error
	call   func(context.Context, int, string, []byte) ([]byte, error)
	renew  func(context.Context, int) error
}

func (r *detailRetryRuntime) Call(context.Context, pluginsystem.LocalPlugin, string, []byte, pluginsystem.RequestPolicyContext, pluginsystem.RuntimeCallOptions) ([]byte, error) {
	r.t.Fatal("sync must use sessions")
	return nil, errors.New("unexpected direct call")
}

func (r *detailRetryRuntime) OpenSession(ctx context.Context, plugin pluginsystem.LocalPlugin, policy pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
	r.opened++
	if r.onOpen != nil {
		if err := r.onOpen(ctx, r.opened, plugin, policy); err != nil {
			return nil, err
		}
	}
	return &detailRetrySession{runtime: r, worker: r.opened}, nil
}

type detailRetrySession struct {
	runtime *detailRetryRuntime
	worker  int
	dead    bool
	closed  bool
}

func (s *detailRetrySession) Call(ctx context.Context, export string, input []byte, options pluginsystem.RuntimeCallOptions) ([]byte, error) {
	s.runtime.t.Helper()
	if s.dead || s.closed {
		s.runtime.t.Fatalf("called dead/closed worker %d", s.worker)
	}
	want := pluginsystem.DetailMaxHostRequestsPerCall
	if strings.HasPrefix(export, "list_") {
		want = pluginsystem.ListMaxHostRequestsPerCall
	}
	// Assert the exact upstream budget on every initial call and retry, not
	// just an effective default that could silently permit extra requests.
	if options != (pluginsystem.RuntimeCallOptions{MaxHostRequests: want}) {
		s.runtime.t.Fatalf("%s worker %d options=%#v, want MaxHostRequests=%d", export, s.worker, options, want)
	}
	output, err := s.runtime.call(ctx, s.worker, export, input)
	if pluginsystem.IsRuntimeSessionFatalError(err) {
		s.dead = true
	}
	return output, err
}

func (s *detailRetrySession) Renew(ctx context.Context) error {
	if s.dead || s.closed {
		s.runtime.t.Fatalf("renewed dead/closed worker %d", s.worker)
	}
	if s.runtime.renew != nil {
		return s.runtime.renew(ctx, s.worker)
	}
	return nil
}

func (s *detailRetrySession) Close(context.Context) error {
	if s.closed {
		s.runtime.t.Fatalf("closed worker %d twice", s.worker)
	}
	s.closed = true
	s.runtime.closed++
	return nil
}

func retryTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func retryTestList(t *testing.T, ids ...string) []byte {
	t.Helper()
	output := pluginSystemListOutput{}
	for _, id := range ids {
		output.Items = append(output.Items, pluginsystem.TrailSummary{Source: pluginsystem.TrailImportSource{ExternalID: id}})
	}
	return retryTestJSON(t, output)
}

func retryTestDetail(t *testing.T, input []byte) (string, []byte) {
	t.Helper()
	var detail pluginSystemDetailInput
	if err := json.Unmarshal(input, &detail); err != nil {
		t.Fatal(err)
	}
	kind := detail.Summary.Kind
	if kind == "" {
		kind = "planned"
	}
	return detail.Summary.Source.ExternalID, retryTestJSON(t, pluginSystemDetailOutput{Item: pluginsystem.TrailImport{
		Source: detail.Summary.Source, Kind: kind, Name: detail.Summary.Source.ExternalID,
		Track: pluginsystem.Track{Format: "gpx", ContentBase64: base64.StdEncoding.EncodeToString([]byte(
			`<gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="46" lon="8"/><trkpt lat="46.001" lon="8.001"/></trkseg></trk></gpx>`))},
	}})
}

func retryTestFatal() error {
	return pluginsystem.RuntimeSessionFatalError{Err: errors.New("private worker stderr and trap trace")}
}

func TestSyncDetailFatalRetryImportsExactlyOnce(t *testing.T) {
	f := newDetailRetryFixture(t)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "01234567890123456789012345678901")
	auth := map[string]any{"accessToken": "fixture-token", "refreshToken": "fixture-refresh", "clientSecret": "fixture-secret"}
	instance := f.stored(t, f.instanceID)
	instance.Set("auth", auth)
	instance.Set("last_error", map[string]any{"code": "old_error"})
	if err := f.app.Save(instance); err != nil {
		t.Fatal(err)
	}
	f.plugin.Manifest.Auth.Contexts = map[string]pluginsystem.AuthContext{
		"provider": {Type: pluginsystem.AuthTypeOAuth2},
	}
	f.plugin.Manifest.Permissions.Network.Connectors = []pluginsystem.ConnectorTargetPermission{{
		Name: "provider", Type: pluginsystem.ConnectorTypePublicAPI, FixedBaseURL: "https://fixture.invalid",
		AllowedPathPrefixes: []string{"/routes/"}, Auth: []string{"provider"},
	}}
	calls := 0
	var firstInput []byte
	var firstPolicy pluginsystem.RequestPolicyContext
	runtime := &detailRetryRuntime{t: t}
	runtime.onOpen = func(_ context.Context, worker int, plugin pluginsystem.LocalPlugin, policy pluginsystem.RequestPolicyContext) error {
		if !reflect.DeepEqual(plugin, f.plugin) || !reflect.DeepEqual(policy.HostAuth, auth) {
			t.Fatal("worker did not retain its plugin/auth contract")
		}
		if worker == 1 {
			firstPolicy = policy
		} else if !reflect.DeepEqual(policy, firstPolicy) {
			t.Fatal("retry changed connector permissions or host auth")
		}
		return nil
	}
	runtime.call = func(_ context.Context, worker int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			return retryTestJSON(t, pluginSystemListOutput{Items: []pluginsystem.TrailSummary{{
				Source: pluginsystem.TrailImportSource{ExternalID: "recover", URL: "https://fixture.invalid/routes/recover?variant=planned"},
				Kind:   "planned",
			}}}), nil
		}
		if export == "list_activities_v1" {
			return retryTestList(t), nil
		}
		calls++
		f.importedIDs(t)
		var detail pluginSystemDetailInput
		if err := json.Unmarshal(input, &detail); err != nil {
			t.Fatal(err)
		}
		if detail.Instance.ID != f.instanceID || detail.Options["after"] != "2026-01-01" ||
			detail.Summary.Source.URL != "https://fixture.invalid/routes/recover?variant=planned" || detail.Summary.Kind != "planned" {
			t.Fatal("detail lost its instance, plugin options or summary/query")
		}
		if calls == 1 {
			firstInput = append([]byte(nil), input...)
			return nil, retryTestFatal()
		}
		if worker != 2 || !bytes.Equal(firstInput, input) {
			t.Fatal("detail was not retried on a new worker with identical input")
		}
		if !bytes.Contains(input, []byte("fixture-token")) || bytes.Contains(input, []byte("fixture-refresh")) || bytes.Contains(input, []byte("fixture-secret")) {
			t.Fatal("retry exposed host-only OAuth credentials to the plugin")
		}
		_, output := retryTestDetail(t, input)
		return output, nil
	}
	f.useRuntime(t, runtime)
	result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	if err != nil || result == nil || result.Imported != 1 || result.Skipped != 0 || calls != 2 || runtime.opened != 2 || runtime.closed != 2 {
		t.Fatalf("fatal retry: result=%#v, calls=%d, workers=%d/%d, err=%v", result, calls, runtime.opened, runtime.closed, err)
	}
	f.importedIDs(t, "recover")
	stored := f.stored(t, f.instanceID)
	if stored.GetString("status") != "configured" || len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 {
		t.Fatal("successful recovery was not saved as clean")
	}
}

func TestSyncDetailRepeatedFatalKeepsImportsAndContinues(t *testing.T) {
	f := newDetailRetryFixture(t)
	calls := map[string]int{}
	fail := true
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		switch export {
		case "list_routes_v1":
			var list pluginSystemListInput
			if err := json.Unmarshal(input, &list); err != nil {
				t.Fatal(err)
			}
			// A failed item must not discard the next page's cursor.
			if list.State["page"] == nil {
				var output pluginSystemListOutput
				if err := json.Unmarshal(retryTestList(t, "before", "bad"), &output); err != nil {
					t.Fatal(err)
				}
				output.HasMore, output.State = true, map[string]any{"page": 2}
				return retryTestJSON(t, output), nil
			}
			return retryTestList(t, "after"), nil
		case "list_activities_v1":
			return retryTestList(t, "activity"), nil
		}
		id, output := retryTestDetail(t, input)
		calls[id]++
		if id == "bad" && fail {
			f.importedIDs(t, "before")
			return nil, retryTestFatal()
		}
		return output, nil
	}}
	f.useRuntime(t, runtime)
	result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	f.incomplete(t, err)
	if result == nil || result.Imported != 3 || result.Skipped != 1 || calls["bad"] != 2 || runtime.opened != 3 || runtime.closed != 3 {
		t.Fatalf("isolated failure blocked later imports: result=%#v, calls=%v, workers=%d/%d", result, calls, runtime.opened, runtime.closed)
	}
	f.importedIDs(t, "before", "after", "activity")
	if f.stored(t, f.instanceID).GetDateTime("last_sync_at").IsZero() {
		t.Fatal("traversed partial sync did not record its completion time")
	}

	// A subsequent run can recover the item. Previously committed imports
	// are still skipped via their real external references, not re-imported.
	fail = false
	result, err = syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	if err != nil || result == nil || result.Imported != 1 || result.Skipped != 3 || calls["bad"] != 3 || calls["before"] != 1 || calls["after"] != 1 || calls["activity"] != 1 {
		t.Fatalf("next run lost deduplication or retry eligibility: result=%#v, calls=%v, err=%v", result, calls, err)
	}
	f.importedIDs(t, "before", "bad", "after", "activity")
	if stored := f.stored(t, f.instanceID); stored.GetString("status") != "configured" || len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 {
		t.Fatal("recovery did not clear the partial outcome")
	}
}

func TestSyncDetailSystemicFaultsAreBounded(t *testing.T) {
	f := newDetailRetryFixture(t)
	calls := map[string]int{}
	cause := retryTestFatal()
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			ids := []string{"before"}
			for i := 1; i < defaultPluginSyncBatchLimit; i++ {
				ids = append(ids, fmt.Sprintf("bad-%d", i))
			}
			return retryTestList(t, ids...), nil
		}
		if export == "list_activities_v1" {
			return retryTestList(t, "activity-after"), nil
		}
		id, output := retryTestDetail(t, input)
		calls[id]++
		if id == "before" || id == "activity-after" {
			return output, nil
		}
		return nil, cause
	}}
	f.useRuntime(t, runtime)
	result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	f.incomplete(t, err)
	f.importedIDs(t, "before", "activity-after")
	total := 0
	for i := 1; i < defaultPluginSyncBatchLimit; i++ {
		want := 1
		if i <= defaultPluginSyncFailedDetailRetryLimit {
			want++
		}
		id := fmt.Sprintf("bad-%d", i)
		if calls[id] != want {
			t.Fatalf("item %q tried %d times, want %d", id, calls[id], want)
		}
		total += calls[id]
	}
	baseline := defaultPluginSyncBatchLimit - 1
	if total != baseline+defaultPluginSyncFailedDetailRetryLimit || result == nil || result.Imported != 2 || result.Skipped != baseline || runtime.closed != runtime.opened || !errors.Is(err, cause) {
		t.Fatalf("lost traversal, cause or unsuccessful retry bound: result=%#v, calls=%v, workers=%d/%d, err=%v", result, calls, runtime.opened, runtime.closed, err)
	}
}

func TestSyncDetailRetryBudgetDoesNotStarveStableProviderOrder(t *testing.T) {
	f := newDetailRetryFixture(t)
	calls := []map[string]int{{}, {}}
	lists := []map[string]int{{}, {}}
	run := 0
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			var list pluginSystemListInput
			if err := json.Unmarshal(input, &list); err != nil {
				t.Fatal(err)
			}
			if list.State["page"] == nil {
				lists[run]["planned-first"]++
				var output pluginSystemListOutput
				if err := json.Unmarshal(retryTestList(t, "before", "bad-1", "bad-2"), &output); err != nil {
					t.Fatal(err)
				}
				output.HasMore, output.State = true, map[string]any{"page": 2}
				return retryTestJSON(t, output), nil
			}
			lists[run]["planned-later"]++
			return retryTestList(t, "bad-3", "bad-4", "planned-after"), nil
		}
		if export == "list_activities_v1" {
			lists[run]["completed"]++
			return retryTestJSON(t, pluginSystemListOutput{Items: []pluginsystem.TrailSummary{
				{Source: pluginsystem.TrailImportSource{ExternalID: "bad-activity"}, Kind: "completed"},
				{Source: pluginsystem.TrailImportSource{ExternalID: "activity-after"}, Kind: "completed"},
			}}), nil
		}
		id, output := retryTestDetail(t, input)
		calls[run][id]++
		if strings.HasPrefix(id, "bad-") {
			return nil, retryTestFatal()
		}
		return output, nil
	}}
	f.useRuntime(t, runtime)
	for ; run < 2; run++ {
		result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
		f.incomplete(t, err)
		f.importedIDs(t, "before", "planned-after", "activity-after")
		activity, findErr := f.app.FindFirstRecordByData("trails", "name", "activity-after")
		if findErr != nil || !activity.GetBool("completed") {
			t.Fatalf("run %d lost the completed activity: err=%v", run, findErr)
		}
		wantImported, wantSkipped := 3, 5
		if run == 1 {
			wantImported, wantSkipped = 0, 8
		}
		if result == nil || result.Imported != wantImported || result.Skipped != wantSkipped {
			t.Fatalf("run %d starved or duplicated healthy items: result=%#v, calls=%v", run, result, calls[run])
		}
		for _, id := range []string{"bad-1", "bad-2", "bad-3"} {
			if calls[run][id] != 2 {
				t.Fatalf("run %d did not reset/limit its unsuccessful retry budget: %s=%d", run, id, calls[run][id])
			}
		}
		for _, id := range []string{"bad-4", "bad-activity"} {
			if calls[run][id] != 1 {
				t.Fatalf("run %d retried after the budget: %s=%d", run, id, calls[run][id])
			}
		}
		if !reflect.DeepEqual(lists[run], map[string]int{"planned-first": 1, "planned-later": 1, "completed": 1}) || runtime.closed != runtime.opened {
			t.Fatalf("run %d lost later pages/capabilities or leaked a session: lists=%v, workers=%d/%d", run, lists[run], runtime.opened, runtime.closed)
		}
	}
	for _, id := range []string{"before", "planned-after", "activity-after"} {
		if calls[0][id] != 1 || calls[1][id] != 0 {
			t.Fatalf("healthy item %q was not imported exactly once: %v", id, calls)
		}
	}
}

func TestSyncDetailSuccessfulRecoveriesDoNotConsumeFailureBudget(t *testing.T) {
	f := newDetailRetryFixture(t)
	calls := map[string]int{}
	ids := []string{}
	for i := 0; i <= defaultPluginSyncFailedDetailRetryLimit; i++ {
		ids = append(ids, fmt.Sprintf("recover-%d", i))
	}
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			return retryTestList(t, append(append([]string{}, ids...), "bad")...), nil
		}
		if export == "list_activities_v1" {
			return retryTestList(t), nil
		}
		id, output := retryTestDetail(t, input)
		calls[id]++
		if id == "bad" || calls[id] == 1 {
			return nil, retryTestFatal()
		}
		return output, nil
	}}
	f.useRuntime(t, runtime)
	result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	f.incomplete(t, err)
	f.importedIDs(t, ids...)
	if result == nil || result.Imported != len(ids) || result.Skipped != 1 || calls["bad"] != 2 {
		t.Fatalf("successful recoveries exhausted the unsuccessful-retry budget: result=%#v, calls=%v", result, calls)
	}
	for _, id := range ids {
		if calls[id] != 2 {
			t.Fatalf("recoverable item %q was lost: calls=%v", id, calls)
		}
	}
}

func TestSyncDetailExhaustedRetryBudgetStillChecksCancellationAndSwitchOff(t *testing.T) {
	for _, cancelSync := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%v", cancelSync), func(t *testing.T) {
			f := newDetailRetryFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := map[string]int{}
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "before", "bad-1", "bad-2", "bad-3", "bad-4", "never"), nil
				}
				if export == "list_activities_v1" {
					t.Fatal("listed another capability after stopping")
				}
				id, output := retryTestDetail(t, input)
				calls[id]++
				if id == "before" {
					return output, nil
				}
				if id == "bad-4" {
					if cancelSync {
						cancel()
					} else {
						stored := f.stored(t, f.instanceID)
						stored.Set("enabled", false)
						stored.Set("status", "disabled")
						if err := f.app.Save(stored); err != nil {
							t.Fatal(err)
						}
					}
				}
				return nil, retryTestFatal()
			}}
			f.useRuntime(t, runtime)
			result, err := syncPluginInstance(ctx, f.app, nil, f.plugin, f.stored(t, f.instanceID))
			want := errPluginInstanceSwitchedOff
			if cancelSync {
				want = context.Canceled
				f.incomplete(t, err)
			} else if stored := f.stored(t, f.instanceID); stored.GetBool("enabled") || stored.GetString("status") != "disabled" {
				t.Fatal("exhausted-budget sync re-enabled the instance")
			}
			if result != nil || !errors.Is(err, want) || calls["bad-4"] != 1 || calls["never"] != 0 || runtime.closed != runtime.opened {
				t.Fatalf("ignored stop gates after exhausting the retry budget: result=%#v, calls=%v, workers=%d/%d, err=%v", result, calls, runtime.opened, runtime.closed, err)
			}
			f.importedIDs(t, "before")
		})
	}
}

func TestSyncDetailRetryRespectsCancellationAndSwitchOff(t *testing.T) {
	for _, phase := range []string{"before-restart", "during-startup", "failed-startup", "during-retry"} {
		for _, action := range []string{"cancel", "switch-off", "delete"} {
			t.Run(phase+"/"+action, func(t *testing.T) {
				f := newDetailRetryFixture(t)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				stop := func() {
					if action == "cancel" {
						cancel()
						return
					}
					record := f.stored(t, f.instanceID)
					if action == "delete" {
						if err := f.app.Delete(record); err != nil {
							t.Fatal(err)
						}
						return
					}
					record.Set("enabled", false)
					record.Set("status", "disabled")
					record.Set("config", map[string]any{"host": map[string]any{"planned": false}})
					if err := f.app.Save(record); err != nil {
						t.Fatal(err)
					}
				}
				details := 0
				runtime := &detailRetryRuntime{t: t}
				runtime.onOpen = func(_ context.Context, worker int, _ pluginsystem.LocalPlugin, _ pluginsystem.RequestPolicyContext) error {
					if worker == 2 && (phase == "during-startup" || phase == "failed-startup") {
						stop()
						if phase == "failed-startup" {
							return errors.New("private restart error")
						}
					}
					return nil
				}
				runtime.call = func(_ context.Context, worker int, export string, input []byte) ([]byte, error) {
					if export == "list_routes_v1" {
						return retryTestList(t, "before", "bad", "after"), nil
					}
					if export == "list_activities_v1" {
						t.Fatal("sync went on to another capability")
					}
					id, output := retryTestDetail(t, input)
					if id == "before" {
						return output, nil
					}
					if id != "bad" {
						t.Fatal("sync went on to the next item")
					}
					details++
					if worker == 1 {
						if phase == "before-restart" {
							stop()
						}
						return nil, retryTestFatal()
					}
					stop()
					// Even a valid retry must not write after a switch-off or
					// cancellation that happened while it was in flight.
					return output, nil
				}
				f.useRuntime(t, runtime)
				_, err := syncPluginInstance(ctx, f.app, nil, f.plugin, f.stored(t, f.instanceID))
				wantErr := errPluginInstanceSwitchedOff
				if action == "cancel" {
					wantErr = context.Canceled
				}
				if !errors.Is(err, wantErr) {
					t.Fatalf("lost cancellation/switch-off: %v", err)
				}
				f.importedIDs(t, "before")
				wantOpened, wantDetails := 2, 1
				if phase == "before-restart" {
					wantOpened = 1
				} else if phase == "during-retry" {
					wantDetails = 2
				}
				wantClosed := wantOpened
				if phase == "failed-startup" {
					wantClosed--
				}
				if runtime.opened != wantOpened || runtime.closed != wantClosed || details != wantDetails {
					t.Fatalf("unexpected retry after stop: workers=%d/%d, details=%d", runtime.opened, runtime.closed, details)
				}
				switch action {
				case "cancel":
					f.incomplete(t, err)
				case "switch-off":
					stored := f.stored(t, f.instanceID)
					if stored.GetBool("enabled") || stored.GetString("status") != "disabled" || len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 {
						t.Fatal("retry re-enabled the instance, overwrote its status or recorded switch-off as an error")
					}
					if host := pluginhost.HostConfig(pluginsystem.JSONMapFromRecord(stored, "config")); host["planned"] != false {
						t.Fatal("retry overwrote the user's new configuration")
					}
				case "delete":
					if _, err := f.app.FindRecordById("plugin_instances", f.instanceID); !errors.Is(err, sql.ErrNoRows) {
						t.Fatalf("retry restored a deleted instance: %v", err)
					}
				}
			})
		}
	}
}

func TestSyncDetailRestartFailureReportsIncomplete(t *testing.T) {
	f := newDetailRetryFixture(t)
	details := 0
	restartErr := errors.New("private worker restart trace")
	runtime := &detailRetryRuntime{t: t}
	runtime.onOpen = func(_ context.Context, worker int, _ pluginsystem.LocalPlugin, _ pluginsystem.RequestPolicyContext) error {
		if worker == 2 {
			return restartErr
		}
		return nil
	}
	runtime.call = func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			return retryTestList(t, "before", "bad", "after"), nil
		}
		id, output := retryTestDetail(t, input)
		if id == "before" {
			return output, nil
		}
		details++
		return nil, retryTestFatal()
	}
	f.useRuntime(t, runtime)
	_, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	f.incomplete(t, err)
	f.importedIDs(t, "before")
	if runtime.opened != 2 || runtime.closed != 1 || details != 1 || !errors.Is(err, restartErr) {
		t.Fatalf("restart failure was retried or reused a dead session: workers=%d/%d, details=%d", runtime.opened, runtime.closed, details)
	}
}

func TestSyncDetailUnusableRetryReportsIncomplete(t *testing.T) {
	for _, outcome := range []string{"structured-skip", "invalid-json", "invalid-item", "after-write-error", "later-capability-error"} {
		t.Run(outcome, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			if outcome == "after-write-error" {
				f.app.OnRecordAfterCreateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
					if e.Record.GetString("name") == "bad" {
						return errors.New("private after-write failure")
					}
					return e.Next()
				})
			}
			details := 0
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "before", "bad", "after"), nil
				}
				if export == "list_activities_v1" {
					if outcome == "later-capability-error" {
						return nil, errors.New("private later capability failure")
					}
					return retryTestList(t), nil
				}
				id, output := retryTestDetail(t, input)
				if id != "bad" {
					return output, nil
				}
				details++
				if details == 1 {
					return nil, retryTestFatal()
				}
				switch outcome {
				case "structured-skip", "later-capability-error":
					return retryTestJSON(t, pluginSystemDetailOutput{Error: &pluginsystem.PluginError{Code: "invalid_request", Message: "private detail failure"}}), nil
				case "invalid-json":
					return []byte("{"), nil
				case "invalid-item":
					return retryTestJSON(t, pluginSystemDetailOutput{}), nil
				default:
					return output, nil
				}
			}}
			f.useRuntime(t, runtime)
			_, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
			f.incomplete(t, err)
			switch outcome {
			case "invalid-item":
				f.importedIDs(t, "before")
			case "after-write-error":
				f.importedIDs(t, "before", "bad")
			default:
				f.importedIDs(t, "before", "after")
			}
			if details != 2 || runtime.opened != 2 || runtime.closed != 2 {
				t.Fatalf("retried more than once or restarted a healthy worker: details=%d, workers=%d/%d", details, runtime.opened, runtime.closed)
			}
		})
	}
}

func TestSyncDetailStructuredSkipsAreUnchanged(t *testing.T) {
	for _, tc := range []struct {
		code     string
		inOutput bool
	}{
		{"invalid_request", false},
		{"invalid_request", true},
		{"not_importable", false},
		{"not_importable", true},
	} {
		t.Run(fmt.Sprintf("%s/output=%v", tc.code, tc.inOutput), func(t *testing.T) {
			f := newDetailRetryFixture(t)
			calls := map[string]int{}
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "no-gpx", "good"), nil
				}
				if export == "list_activities_v1" {
					return retryTestList(t), nil
				}
				id, output := retryTestDetail(t, input)
				calls[id]++
				if id == "no-gpx" {
					pluginErr := pluginsystem.PluginError{Code: tc.code, Message: "This item has no GPX."}
					if tc.inOutput {
						return retryTestJSON(t, pluginSystemDetailOutput{Error: &pluginErr}), nil
					}
					return nil, pluginsystem.PluginCallError{PluginError: pluginErr}
				}
				return output, nil
			}}
			f.useRuntime(t, runtime)
			result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
			if err != nil || result == nil || result.Imported != 1 || result.Skipped != 1 || calls["no-gpx"] != 1 || calls["good"] != 1 || runtime.opened != 1 {
				t.Fatalf("normal skip changed: result=%#v, calls=%v, workers=%d, err=%v", result, calls, runtime.opened, err)
			}
			f.importedIDs(t, "good")
			if stored := f.stored(t, f.instanceID); stored.GetString("status") != "configured" || len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 {
				t.Fatal("normal structured skip was recorded as partial")
			}
		})
	}
}

func TestSyncDetailDoesNotRetryAfterImportWrites(t *testing.T) {
	f := newDetailRetryFixture(t)
	afterWriteErr := retryTestFatal()
	f.app.OnRecordAfterCreateSuccess("trails").BindFunc(func(*core.RecordEvent) error { return afterWriteErr })
	details := 0
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		if export == "list_routes_v1" {
			return retryTestList(t, "written"), nil
		}
		details++
		_, output := retryTestDetail(t, input)
		return output, nil
	}}
	f.useRuntime(t, runtime)
	_, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	if !errors.Is(err, afterWriteErr) || details != 1 || runtime.opened != 1 || runtime.closed != 1 {
		t.Fatalf("retried after import writes: details=%d, workers=%d/%d, err=%v", details, runtime.opened, runtime.closed, err)
	}
	f.importedIDs(t, "written")
}

func TestSyncDetailRetryRetainsRenewalAndAgeRecycling(t *testing.T) {
	for _, renewalErr := range []error{pluginsystem.ErrRuntimeSessionRecycleRequired, retryTestFatal()} {
		t.Run(fmt.Sprintf("%T", renewalErr), func(t *testing.T) {
			f := newDetailRetryFixture(t)
			renewals := map[int]int{}
			details := map[string][]int{}
			runtime := &detailRetryRuntime{t: t}
			runtime.renew = func(_ context.Context, worker int) error {
				renewals[worker]++
				if worker == 1 && renewals[worker] == 2 {
					return renewalErr
				}
				if worker == 3 {
					return pluginsystem.ErrRuntimeSessionRecycleRequired
				}
				return nil
			}
			runtime.call = func(_ context.Context, worker int, export string, input []byte) ([]byte, error) {
				switch export {
				case "list_routes_v1":
					return retryTestList(t, "recover", "later"), nil
				case "list_activities_v1":
					return retryTestList(t), nil
				}
				id, output := retryTestDetail(t, input)
				details[id] = append(details[id], worker)
				if id == "recover" && len(details[id]) == 1 {
					return nil, retryTestFatal()
				}
				return output, nil
			}
			f.useRuntime(t, runtime)
			result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
			if err != nil || result == nil || result.Imported != 2 || result.Skipped != 0 ||
				!reflect.DeepEqual(details, map[string][]int{"recover": {2, 3}, "later": {4}}) ||
				!reflect.DeepEqual(renewals, map[int]int{1: 2, 3: 1, 4: 1}) || runtime.opened != 4 || runtime.closed != 4 {
				t.Fatalf("retry lost healthy renewal/recycling: result=%#v details=%v renewals=%v workers=%d/%d err=%v", result, details, renewals, runtime.opened, runtime.closed, err)
			}
			f.importedIDs(t, "recover", "later")
		})
	}
}

func TestSyncDetailSuccessChecksBeforeImport(t *testing.T) {
	for _, action := range []string{"cancel", "switch-off", "delete"} {
		t.Run(action, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			details := 0
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "stopped", "never"), nil
				}
				if strings.HasPrefix(export, "list_") {
					t.Fatal("listed another capability after stopping")
				}
				details++
				stored := f.stored(t, f.instanceID)
				switch action {
				case "cancel":
					cancel()
				case "switch-off":
					stored.Set("enabled", false)
					stored.Set("status", "disabled")
					if err := f.app.Save(stored); err != nil {
						t.Fatal(err)
					}
				case "delete":
					if err := f.app.Delete(stored); err != nil {
						t.Fatal(err)
					}
				}
				_, output := retryTestDetail(t, input)
				return output, nil
			}}
			f.useRuntime(t, runtime)
			result, err := syncPluginInstance(ctx, f.app, nil, f.plugin, f.stored(t, f.instanceID))
			want := errPluginInstanceSwitchedOff
			if action == "cancel" {
				want = context.Canceled
			}
			if result != nil || !errors.Is(err, want) || details != 1 || runtime.opened != 1 || runtime.closed != 1 {
				t.Fatalf("imported after stopping or retried a healthy detail: result=%#v details=%d workers=%d/%d err=%v", result, details, runtime.opened, runtime.closed, err)
			}
			f.importedIDs(t)
		})
	}
}

type pluginSyncLookupTestApp struct {
	core.App
	lookupErr   *error
	lookups     *int
	lookupErrAt int
}

func (app pluginSyncLookupTestApp) FindRecordById(collection any, id string, filters ...func(*dbx.SelectQuery) error) (*core.Record, error) {
	if collection == "plugin_instances" {
		if app.lookups != nil {
			*app.lookups++
		}
		if *app.lookupErr != nil && (app.lookupErrAt == 0 || (app.lookups != nil && *app.lookups == app.lookupErrAt)) {
			return nil, *app.lookupErr
		}
	}
	return app.App.FindRecordById(collection, id, filters...)
}

func TestSyncDetailHealthyRenewalAvoidsDuplicateInstanceLookup(t *testing.T) {
	for lookupErrAt, phase := range []string{"healthy", "post-list-read-failure", "pre-item-read-failure", "pre-import-read-failure"} {
		t.Run(phase, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			var lookupErr error
			if lookupErrAt > 0 {
				lookupErr = errors.New("one-shot live instance lookup failed")
			}
			lookups, detailLookups := 0, 0
			app := pluginSyncLookupTestApp{App: f.app, lookupErr: &lookupErr, lookups: &lookups, lookupErrAt: lookupErrAt}
			actor, err := f.app.FindFirstRecordByData("activitypub_actors", "user", f.ownerID)
			if err != nil {
				t.Fatal(err)
			}
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "healthy"), nil
				}
				detailLookups = lookups
				_, output := retryTestDetail(t, input)
				return output, nil
			}}
			sessions := &pluginSyncRuntimeSession{runtime: runtime, plugin: f.plugin}
			if err := sessions.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer sessions.close(context.Background())
			result, err := syncPluginCapability(t.Context(), app, nil, sessions, f.plugin,
				f.plugin.Manifest.Capabilities[0], f.plugin.Manifest.Capabilities[1],
				f.stored(t, f.instanceID), actor, nil, nil, nil, false, false)
			// Ordinary checkpoints retain upstream best-effort read behavior.
			// The distinct pre-import gate must still fail closed on any error.
			if lookupErrAt == 3 {
				if result != nil || !errors.Is(err, lookupErr) {
					t.Fatalf("pre-import read failure authorized a write: result=%#v err=%v", result, err)
				}
				f.importedIDs(t)
			} else {
				if err != nil || result == nil || result.Imported != 1 {
					t.Fatalf("healthy sync lost upstream best-effort behavior: result=%#v err=%v", result, err)
				}
				f.importedIDs(t, "healthy")
			}
			if detailLookups != 2 || lookups != 3 || sessions.openGeneration != 1 {
				t.Fatalf("redundant healthy lookup or missing write gate: detail lookups=%d total=%d generation=%d", detailLookups, lookups, sessions.openGeneration)
			}
		})
	}
}

func TestSyncDetailRenewalRetainsCancellationAndStrictReadmissionChecks(t *testing.T) {
	for _, phase := range []string{"healthy-cancel", "reopen-cancel", "reopen-lookup-failure"} {
		t.Run(phase, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var lookupErr error
			app := pluginSyncLookupTestApp{App: f.app, lookupErr: &lookupErr}
			cause := errors.New("live instance lookup failed")
			renewals, details := 0, 0
			runtime := &detailRetryRuntime{t: t}
			runtime.renew = func(context.Context, int) error {
				renewals++
				if renewals == 2 {
					if phase == "healthy-cancel" {
						cancel()
						return nil
					}
					return pluginsystem.ErrRuntimeSessionRecycleRequired
				}
				return nil
			}
			runtime.onOpen = func(_ context.Context, worker int, _ pluginsystem.LocalPlugin, _ pluginsystem.RequestPolicyContext) error {
				if worker == 2 {
					if phase == "reopen-cancel" {
						cancel()
					} else {
						lookupErr = cause
					}
				}
				return nil
			}
			runtime.call = func(_ context.Context, _ int, export string, _ []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "never"), nil
				}
				details++
				return nil, errors.New("unexpected detail after stop")
			}
			sessions := &pluginSyncRuntimeSession{runtime: runtime, plugin: f.plugin}
			if err := sessions.open(ctx); err != nil {
				t.Fatal(err)
			}
			result, err := syncPluginCapability(ctx, app, nil, sessions, f.plugin,
				f.plugin.Manifest.Capabilities[0], f.plugin.Manifest.Capabilities[1],
				f.stored(t, f.instanceID), nil, nil, nil, nil, false, false)
			_ = sessions.close(context.Background())
			want, wantOpened := context.Canceled, 2
			if phase == "healthy-cancel" {
				wantOpened = 1
			} else if phase == "reopen-lookup-failure" {
				want = cause
			}
			if result != nil || !errors.Is(err, want) || details != 0 || renewals != 2 ||
				runtime.opened != wantOpened || runtime.closed != wantOpened || sessions.openGeneration != uint64(wantOpened) {
				t.Fatalf("lost renewal stop gate: result=%#v details=%d renewals=%d workers=%d/%d generation=%d err=%v", result, details, renewals, runtime.opened, runtime.closed, sessions.openGeneration, err)
			}
			f.importedIDs(t)
		})
	}
}

func TestSyncDetailRetryRequiresSuccessfulLiveLookup(t *testing.T) {
	for _, phase := range []string{"before-restart", "during-startup", "before-write"} {
		t.Run(phase, func(t *testing.T) {
			f := newDetailRetryFixture(t)
			var lookupErr error
			app := pluginSyncLookupTestApp{App: f.app, lookupErr: &lookupErr}
			cause := errors.New("live instance lookup failed")
			details := 0
			runtime := &detailRetryRuntime{t: t}
			runtime.onOpen = func(_ context.Context, worker int, _ pluginsystem.LocalPlugin, _ pluginsystem.RequestPolicyContext) error {
				if worker == 2 && phase == "during-startup" {
					lookupErr = cause
				}
				return nil
			}
			runtime.call = func(_ context.Context, worker int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "bad", "never"), nil
				}
				id, output := retryTestDetail(t, input)
				if id != "bad" {
					t.Fatal("called another item after a failed live lookup")
				}
				details++
				if worker == 1 {
					if phase == "before-restart" {
						lookupErr = cause
					}
					return nil, retryTestFatal()
				}
				lookupErr = cause
				return output, nil
			}
			sessions := &pluginSyncRuntimeSession{runtime: runtime, plugin: f.plugin}
			if err := sessions.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			result, err := syncPluginCapability(t.Context(), app, nil, sessions, f.plugin,
				f.plugin.Manifest.Capabilities[0], f.plugin.Manifest.Capabilities[1],
				f.stored(t, f.instanceID), nil, nil, nil, nil, false, false)
			_ = sessions.close(context.Background())
			wantOpened, wantDetails := 2, 1
			if phase == "before-restart" {
				wantOpened = 1
			} else if phase == "before-write" {
				wantDetails = 2
			}
			var incomplete pluginsystem.SyncIncompleteError
			if result != nil || !errors.Is(err, cause) || !errors.As(err, &incomplete) ||
				runtime.opened != wantOpened || runtime.closed != wantOpened || details != wantDetails || sessions.openGeneration != uint64(wantOpened) {
				t.Fatalf("failed lookup authorized recovery/write: result=%#v workers=%d/%d generation=%d details=%d err=%v", result, runtime.opened, runtime.closed, sessions.openGeneration, details, err)
			}
			f.importedIDs(t)
		})
	}
}

func TestSyncListRetryRemainsSeparateAndIdentical(t *testing.T) {
	f := newDetailRetryFixture(t)
	calls := map[string]int{}
	var firstInput []byte
	runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
		calls[export]++
		if export == "list_routes_v1" {
			if calls[export] == 1 {
				firstInput = append([]byte(nil), input...)
				return nil, retryTestFatal()
			}
			if !bytes.Equal(firstInput, input) {
				t.Fatal("list retry changed its cursor/auth/options/limits")
			}
			return retryTestList(t, "recover"), nil
		}
		if export == "list_activities_v1" {
			// A second fatal list error still stops, never falling through to
			// the detail retry budget or marking a recovered detail incomplete.
			return nil, retryTestFatal()
		}
		if calls[export] == 1 {
			return nil, retryTestFatal()
		}
		_, output := retryTestDetail(t, input)
		return output, nil
	}}
	f.useRuntime(t, runtime)
	result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
	var incomplete pluginsystem.SyncIncompleteError
	if result != nil || !pluginsystem.IsRuntimeSessionFatalError(err) || errors.As(err, &incomplete) ||
		!reflect.DeepEqual(calls, map[string]int{"list_routes_v1": 2, "get_route_detail_v1": 2, "list_activities_v1": 2}) ||
		runtime.opened != 4 || runtime.closed != 4 {
		t.Fatalf("changed list retry semantics: result=%#v calls=%v workers=%d/%d err=%v", result, calls, runtime.opened, runtime.closed, err)
	}
	f.importedIDs(t, "recover")
}

func TestSyncBackfillRetainsSingleAttemptAndSkipStatus(t *testing.T) {
	for _, fatal := range []bool{false, true} {
		t.Run(fmt.Sprintf("fatal=%v", fatal), func(t *testing.T) {
			f := newDetailRetryFixture(t)
			// Seed a real imported external reference with an unchecked category.
			existing := pluginsystem.TrailSummary{Source: pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "existing"}}
			runtime := &detailRetryRuntime{t: t, call: func(_ context.Context, _ int, export string, input []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "existing"), nil
				}
				if export == "list_activities_v1" {
					return retryTestList(t), nil
				}
				_, output := retryTestDetail(t, input)
				return output, nil
			}}
			f.useRuntime(t, runtime)
			if _, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID)); err != nil {
				t.Fatal(err)
			}
			ref, err := f.app.FindFirstRecordByData("trail_external_reference", "external_id", existing.Source.ExternalID)
			if err != nil {
				t.Fatal(err)
			}
			ref.Set("provider_category", "")
			ref.Set("provider_category_checked_at", "")
			if err := f.app.Save(ref); err != nil {
				t.Fatal(err)
			}
			instance := f.stored(t, f.instanceID)
			instance.Set("config", map[string]any{
				"host": map[string]any{
					"categoryMapping": map[string]any{"Ride": "Biking"},
					"completed":       false,
				},
			})
			if err := f.app.Save(instance); err != nil {
				t.Fatal(err)
			}
			details := 0
			runtime.opened, runtime.closed = 0, 0
			runtime.call = func(_ context.Context, _ int, export string, _ []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return retryTestList(t, "existing"), nil
				}
				details++
				if fatal {
					return nil, retryTestFatal()
				}
				delay := 120
				return nil, pluginsystem.PluginCallError{PluginError: pluginsystem.PluginError{Code: "rate_limited", RetryAfterSeconds: &delay}}
			}
			result, err := syncPluginInstance(t.Context(), f.app, nil, f.plugin, f.stored(t, f.instanceID))
			if err != nil || result == nil || result.Imported != 0 || result.Skipped != 1 || details != 1 || runtime.opened != 1 || runtime.closed != 1 {
				t.Fatalf("broadened backfill retry/status or opened an unused worker: result=%#v details=%d workers=%d/%d err=%v", result, details, runtime.opened, runtime.closed, err)
			}
			stored := f.stored(t, f.instanceID)
			if stored.GetString("status") != "configured" || len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 || !stored.GetDateTime("retry_not_before").IsZero() {
				t.Fatal("backfill acquired main-import incomplete/actionable handling")
			}
			fresh, err := f.app.FindRecordById("trail_external_reference", ref.Id)
			if err != nil || fresh.GetString("provider_category") != "" || !fresh.GetDateTime("provider_category_checked_at").IsZero() {
				t.Fatalf("failed backfill was marked as checked: err=%v", err)
			}
			f.importedIDs(t, "existing")
		})
	}
}
