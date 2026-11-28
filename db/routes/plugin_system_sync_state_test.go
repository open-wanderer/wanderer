package routes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"

	"pocketbase/pluginsystem"
)

type pluginInstanceFixture struct {
	app       *pbtests.TestApp
	instances *core.Collection
	ownerID   string
}

func newPluginInstanceFixture(t *testing.T) *pluginInstanceFixture {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	instances := core.NewBaseCollection("plugin_instances")
	instances.Fields.Add(&core.TextField{Name: "user"}, &core.TextField{Name: "plugin_id"},
		&core.BoolField{Name: "enabled"}, &core.TextField{Name: "status"},
		&core.JSONField{Name: "config"}, &core.JSONField{Name: "auth"},
		&core.JSONField{Name: "last_error"}, &core.JSONField{Name: "state"},
		&core.DateField{Name: "last_sync_at"}, &core.DateField{Name: "retry_not_before"})
	if err := app.Save(instances); err != nil {
		t.Fatal(err)
	}
	return &pluginInstanceFixture{app: app, instances: instances, ownerID: "owner"}
}

func (f *pluginInstanceFixture) instance(t *testing.T, enabled bool, retryNotBefore time.Time) string {
	t.Helper()
	record := core.NewRecord(f.instances)
	record.Set("user", f.ownerID)
	record.Set("plugin_id", "komoot")
	record.Set("enabled", enabled)
	record.Set("status", "configured")
	if !retryNotBefore.IsZero() {
		record.Set("retry_not_before", retryNotBefore)
	}
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record.Id
}

func (f *pluginInstanceFixture) stored(t *testing.T, id string) *core.Record {
	t.Helper()
	record, err := f.app.FindRecordById("plugin_instances", id)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestCronRereadsInstanceBeforeSyncing(t *testing.T) {
	f := newPluginInstanceFixture(t)
	switchedOff := f.instance(t, true, time.Time{})
	waiting := f.instance(t, true, time.Time{})
	deleted := f.instance(t, true, time.Time{})
	listed := []*core.Record{f.stored(t, switchedOff), f.stored(t, waiting), f.stored(t, deleted)}
	// The user changes each instance after the cron listed them.
	record := f.stored(t, switchedOff)
	record.Set("enabled", false)
	record.Set("status", "disabled")
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	record = f.stored(t, waiting)
	record.Set("retry_not_before", time.Now().Add(time.Hour))
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	if err := f.app.Delete(f.stored(t, deleted)); err != nil {
		t.Fatal(err)
	}
	usePluginInstanceSync(t, func(_ context.Context, _ core.App, _ meilisearch.ServiceManager, _ pluginsystem.LocalPlugin, instance *core.Record, _ func(*core.Record)) (*pluginSystemSyncResult, error) {
		t.Errorf("synced %s from a stale copy", instance.Id)
		return &pluginSystemSyncResult{}, nil
	})
	if err := syncPluginInstancesOneByOne(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, listed, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPluginSyncPanicBecomesAnError(t *testing.T) {
	f := newPluginInstanceFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	usePluginInstanceSync(t, func(context.Context, core.App, meilisearch.ServiceManager, pluginsystem.LocalPlugin, *core.Record, func(*core.Record)) (*pluginSystemSyncResult, error) {
		panic("bad plugin item")
	})

	_, skipped, err := syncFreshPluginInstance(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, instanceID, nil)
	if skipped || err == nil {
		t.Fatalf("panic not reported: skipped=%v, err=%v", skipped, err)
	}
	if strings.Contains(err.Error(), "bad plugin item") {
		t.Fatalf("error carries the panic value: %v", err)
	}
	stored := f.stored(t, instanceID)
	if stored.GetString("status") != "error" || pluginsystem.JSONMapFromRecord(stored, "last_error")["code"] != "internal_error" {
		t.Fatalf("status after panic: %q, %v", stored.GetString("status"), stored.Get("last_error"))
	}
}

func TestSyncStatusSaveKeepsChangesMadeDuringTheSync(t *testing.T) {
	f := newPluginInstanceFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	startedWith := f.stored(t, instanceID)
	// While the sync runs, the user switches the plugin off and changes its
	// settings and credentials.
	record := f.stored(t, instanceID)
	record.Set("enabled", false)
	record.Set("status", "disabled")
	record.Set("config", map[string]any{"planned": false})
	record.Set("auth", map[string]any{})
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}

	finished := time.Now()
	if err := savePluginInstanceSyncState(f.app, startedWith, map[string]any{
		"status":       "configured",
		"last_sync_at": finished,
	}); err != nil {
		t.Fatal(err)
	}
	stored := f.stored(t, instanceID)
	if stored.GetBool("enabled") || stored.GetString("status") != "disabled" {
		t.Fatalf("sync switched the plugin back on: enabled=%v, status=%q", stored.GetBool("enabled"), stored.GetString("status"))
	}
	if pluginsystem.JSONMapFromRecord(stored, "config")["planned"] != false || len(pluginsystem.JSONMapFromRecord(stored, "auth")) != 0 {
		t.Fatalf("sync restored old settings: config=%v, auth=%v", stored.Get("config"), stored.Get("auth"))
	}
	if stored.GetDateTime("last_sync_at").IsZero() {
		t.Fatal("sync did not record its own field")
	}
}

type reopenTestRuntime struct {
	opened  int
	session pluginsystem.RuntimeSession
	onOpen  func()
}

func (r *reopenTestRuntime) Call(context.Context, pluginsystem.LocalPlugin, string, []byte, pluginsystem.RequestPolicyContext, pluginsystem.RuntimeCallOptions) ([]byte, error) {
	return nil, errors.New("not used")
}

func (r *reopenTestRuntime) OpenSession(context.Context, pluginsystem.LocalPlugin, pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
	r.opened++
	if r.onOpen != nil {
		r.onOpen()
	}
	return r.session, nil
}

type recyclingSyncTestSession struct {
	pluginsystem.RuntimeSession
	renewals     int
	recycleAfter int
}

func (s *recyclingSyncTestSession) Renew(context.Context) error {
	s.renewals++
	if s.renewals > s.recycleAfter {
		return pluginsystem.ErrRuntimeSessionRecycleRequired
	}
	return nil
}

func TestPluginSyncStopsAfterInstanceChangesDuringReopen(t *testing.T) {
	for _, change := range []string{"disabled", "deleted"} {
		for _, phase := range []string{"list", "list-retry", "detail", "backfill"} {
			t.Run(change+"/"+phase, func(t *testing.T) {
				f := newPluginInstanceFixture(t)
				instanceID := f.instance(t, true, time.Time{})
				instance := f.stored(t, instanceID)
				trails := core.NewBaseCollection("trails")
				trails.Fields.Add(&core.TextField{Name: "name"})
				refs := core.NewBaseCollection("trail_external_reference")
				refs.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "user"}, &core.TextField{Name: "provider"},
					&core.TextField{Name: "external_id"}, &core.TextField{Name: "provider_category"},
					&core.DateField{Name: "provider_category_checked_at"}, &core.AutodateField{Name: "created", OnCreate: true})
				for _, collection := range []*core.Collection{trails, refs} {
					if err := f.app.Save(collection); err != nil {
						t.Fatal(err)
					}
				}
				var hostConfig map[string]any
				var ref *core.Record
				var refBefore []byte
				wantTrails := 0
				if phase == "backfill" {
					wantTrails = 1
					hostConfig = map[string]any{"categoryMapping": map[string]any{"Ride": "Biking"}}
					trail := core.NewRecord(trails)
					trail.Set("name", "Existing trail")
					if err := f.app.Save(trail); err != nil {
						t.Fatal(err)
					}
					ref = core.NewRecord(refs)
					ref.Load(map[string]any{"trail": trail.Id, "user": f.ownerID, "provider": "komoot", "external_id": "one"})
					if err := f.app.Save(ref); err != nil {
						t.Fatal(err)
					}
					var err error
					refBefore, err = json.Marshal(ref)
					if err != nil {
						t.Fatal(err)
					}
				}
				list, err := json.Marshal(pluginSystemListOutput{Items: []pluginsystem.TrailSummary{
					{Source: pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "one"}},
				}})
				if err != nil {
					t.Fatal(err)
				}
				lists, details, replacementCalls := 0, 0, 0
				call := func(_ context.Context, export string, _ []byte) ([]byte, error) {
					if export == "list" {
						lists++
						return list, nil
					}
					details++
					// A missing guard must fail on the call count, not panic later
					// because this test deliberately supplies no import actor.
					return nil, errors.New("unexpected detail after switch-off")
				}
				replacement := syncLimitTestSession{call: func(ctx context.Context, export string, input []byte) ([]byte, error) {
					replacementCalls++
					return call(ctx, export, input)
				}}
				runtime := &reopenTestRuntime{session: replacement, onOpen: func() {
					fresh := f.stored(t, instanceID)
					if change == "deleted" {
						if err := f.app.Delete(fresh); err != nil {
							t.Fatal(err)
						}
					} else {
						fresh.Set("enabled", false)
						fresh.Set("auth", map[string]any{})
						if err := f.app.Save(fresh); err != nil {
							t.Fatal(err)
						}
					}
				}}
				old := &recyclingSyncTestSession{RuntimeSession: syncLimitTestSession{call: call}, recycleAfter: 1}
				if phase == "list" {
					old.recycleAfter = 0
				}
				if phase == "list-retry" {
					old.RuntimeSession = syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) {
						lists++
						return nil, pluginsystem.RuntimeSessionFatalError{Err: errors.New("worker died")}
					}}
				}
				sessions := &pluginSyncRuntimeSession{runtime: runtime, session: old, openGeneration: 1}
				defer sessions.close(context.Background())
				_, err = syncPluginCapability(t.Context(), f.app, nil, sessions, pluginsystem.LocalPlugin{},
					pluginsystem.CapabilityManifest{Export: "list"}, pluginsystem.CapabilityManifest{Export: "detail"},
					instance, nil, map[string]any{"password": "stale-credentials"}, nil, hostConfig, false, false, nil)
				wantLists := 1
				if phase == "list" {
					wantLists = 0
				}
				if !errors.Is(err, errPluginInstanceSwitchedOff) || runtime.opened != 1 || lists != wantLists || details != 0 || replacementCalls != 0 {
					t.Fatalf("export after switch-off during reopen: opened=%d list=%d detail=%d replacement=%d err=%v", runtime.opened, lists, details, replacementCalls, err)
				}
				saved, err := f.app.FindAllRecords("trails")
				if err != nil || len(saved) != wantTrails {
					t.Fatalf("unexpected imports: trails=%d err=%v", len(saved), err)
				}
				if ref != nil {
					fresh, err := f.app.FindRecordById(refs.Id, ref.Id)
					if err != nil {
						t.Fatal(err)
					}
					after, err := json.Marshal(fresh)
					if err != nil || string(after) != string(refBefore) {
						t.Fatalf("category backfill changed after switch-off: before=%s after=%s err=%v", refBefore, after, err)
					}
				}
			})
		}
	}
}

func TestPluginSyncEnabledGuardDoesNotLookUpAnUnchangedSession(t *testing.T) {
	// This fake contains a func and cannot be compared via interface identity.
	sessions := &pluginSyncRuntimeSession{session: syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) {
		panic("no export expected")
	}}, openGeneration: 1}
	if err := sessions.ensureOpenWhileEnabled(t.Context(), nil, "no-lookup-expected"); err != nil {
		t.Fatal(err)
	}
}

func TestSyncCapabilityHandlesSessionAndSwitchOff(t *testing.T) {
	f := newPluginInstanceFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	empty, _ := json.Marshal(pluginSystemListOutput{})
	ok := syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) { return empty, nil }}

	t.Run("reopens a session the worker ended", func(t *testing.T) {
		runtime := &reopenTestRuntime{session: ok}
		killed := syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) {
			return nil, pluginsystem.RuntimeSessionFatalError{Err: errors.New("worker session timeout")}
		}}
		sessions := &pluginSyncRuntimeSession{runtime: runtime, session: killed}
		_, err := syncPluginCapability(t.Context(), f.app, nil, sessions, pluginsystem.LocalPlugin{},
			pluginsystem.CapabilityManifest{Export: "list_routes_v1"}, pluginsystem.CapabilityManifest{}, f.stored(t, instanceID), nil,
			nil, nil, nil, false, false, nil)
		if err != nil || runtime.opened != 1 {
			t.Fatalf("list after a killed session: opened %d times, err=%v", runtime.opened, err)
		}
	})
	t.Run("stops when the plugin is switched off", func(t *testing.T) {
		record := f.stored(t, instanceID)
		record.Set("enabled", false)
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
		}
		sessions := &pluginSyncRuntimeSession{session: ok}
		_, err := syncPluginCapability(t.Context(), f.app, nil, sessions, pluginsystem.LocalPlugin{},
			pluginsystem.CapabilityManifest{Export: "list_routes_v1"}, pluginsystem.CapabilityManifest{}, record, nil,
			nil, nil, nil, false, false, nil)
		if !errors.Is(err, errPluginInstanceSwitchedOff) {
			t.Fatalf("sync went on after the plugin was switched off: %v", err)
		}
	})
	refs := core.NewBaseCollection("trail_external_reference")
	refs.Fields.Add(&core.TextField{Name: "user"}, &core.TextField{Name: "provider"}, &core.TextField{Name: "external_id"})
	if err := f.app.Save(refs); err != nil {
		t.Fatal(err)
	}
	for name, remove := range map[string]func(*pluginInstanceFixture, string) error{
		"switched off": func(f *pluginInstanceFixture, id string) error {
			record, err := f.app.FindRecordById("plugin_instances", id)
			if err != nil {
				return err
			}
			record.Set("enabled", false)
			return f.app.Save(record)
		},
		"deleted": func(f *pluginInstanceFixture, id string) error {
			record, err := f.app.FindRecordById("plugin_instances", id)
			if err != nil {
				return err
			}
			return f.app.Delete(record)
		},
	} {
		t.Run("stops before the next item when "+name+" mid-batch", func(t *testing.T) {
			id := f.instance(t, true, time.Time{})
			list, _ := json.Marshal(pluginSystemListOutput{Items: []pluginsystem.TrailSummary{
				{Source: pluginsystem.TrailImportSource{ExternalID: "a"}},
				{Source: pluginsystem.TrailImportSource{ExternalID: "b"}},
				{Source: pluginsystem.TrailImportSource{ExternalID: "c"}},
			}})
			details := 0
			session := syncLimitTestSession{call: func(_ context.Context, export string, _ []byte) ([]byte, error) {
				if export == "list_routes_v1" {
					return list, nil
				}
				details++
				// The first detail call fails, so no trail is imported.
				if err := remove(f, id); err != nil {
					t.Fatal(err)
				}
				return nil, errors.New("detail failed")
			}}
			_, err := syncPluginCapability(t.Context(), f.app, nil, &pluginSyncRuntimeSession{session: session}, pluginsystem.LocalPlugin{},
				pluginsystem.CapabilityManifest{Export: "list_routes_v1"}, pluginsystem.CapabilityManifest{Export: "get_route_detail_v1"}, f.stored(t, id), nil,
				nil, nil, nil, false, false, nil)
			if !errors.Is(err, errPluginInstanceSwitchedOff) || details != 1 {
				t.Fatalf("sync went on after the plugin was %s: %d detail calls, err=%v", name, details, err)
			}
		})
	}
}

func TestSyncRestoresStatusWhenSwitchedOffDuringTheSync(t *testing.T) {
	for _, outcome := range []string{"switch-off", "error", "completion", "disconnect"} {
		t.Run(outcome, func(t *testing.T) {
			f := newPluginInstanceFixture(t)
			actors := core.NewBaseCollection("activitypub_actors")
			actors.Fields.Add(&core.TextField{Name: "user"})
			if err := f.app.Save(actors); err != nil {
				t.Fatal(err)
			}
			actor := core.NewRecord(actors)
			actor.Set("user", f.ownerID)
			if err := f.app.Save(actor); err != nil {
				t.Fatal(err)
			}
			instanceID := f.instance(t, true, time.Time{})
			empty, _ := json.Marshal(pluginSystemListOutput{})
			providerErr := errors.New("provider failed")
			// Switch off after the last list result but before the final save.
			if outcome == "completion" {
				switchedOff := false
				f.app.OnRecordUpdate().BindFunc(func(e *core.RecordEvent) error {
					if e.Record.Collection().Name == "plugin_instances" && !e.Record.GetDateTime("last_sync_at").IsZero() && !switchedOff {
						switchedOff = true
						e.Record.Set("enabled", false)
						e.Record.Set("status", "syncing")
					}
					return e.Next()
				})
			}
			session := syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) {
				record := f.stored(t, instanceID)
				record.Set("enabled", false)
				if outcome == "completion" {
					return empty, nil
				}
				if outcome == "disconnect" {
					record.Set("status", "needs_auth")
				}
				if err := f.app.Save(record); err != nil {
					t.Fatal(err)
				}
				if outcome == "error" {
					return nil, providerErr
				}
				return empty, nil
			}}
			original := pluginRuntimeFor
			pluginRuntimeFor = func(pluginsystem.LocalPlugin) (pluginsystem.Runtime, error) {
				return &reopenTestRuntime{session: session}, nil
			}
			t.Cleanup(func() { pluginRuntimeFor = original })
			plugin := pluginsystem.LocalPlugin{}
			plugin.Manifest.ID = "komoot"
			plugin.Manifest.Capabilities = []pluginsystem.CapabilityManifest{
				{Name: "list_routes", Version: "v1", Export: "list_routes_v1"},
				{Name: "get_route_detail", Version: "v1", Export: "get_route_detail_v1"},
			}

			_, err := syncPluginInstance(t.Context(), f.app, nil, plugin, f.stored(t, instanceID), nil)
			wantErr := errPluginInstanceSwitchedOff
			if outcome == "error" {
				wantErr = providerErr
			} else if outcome == "completion" {
				wantErr = nil
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("sync returned %v, want %v", err, wantErr)
			}
			stored := f.stored(t, instanceID)
			want := "configured"
			if outcome == "disconnect" {
				want = "needs_auth"
			}
			if stored.GetBool("enabled") || stored.GetString("status") != want {
				t.Fatalf("switch-off left status=%q, enabled=%v", stored.GetString("status"), stored.GetBool("enabled"))
			}
			if outcome == "switch-off" && (len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 || !stored.GetDateTime("retry_not_before").IsZero()) {
				t.Fatalf("switch-off recorded as an error: last_error=%v, retry_not_before=%v", stored.Get("last_error"), stored.GetDateTime("retry_not_before"))
			}
		})
	}
}

func TestResetInterruptedPluginSyncs(t *testing.T) {
	f := newPluginInstanceFixture(t)
	enabled := f.instance(t, true, time.Time{})
	disabled := f.instance(t, false, time.Time{})
	untouched := f.instance(t, true, time.Time{})
	for _, id := range []string{enabled, disabled} {
		record := f.stored(t, id)
		record.Set("status", "syncing")
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	record := f.stored(t, untouched)
	record.Set("status", "needs_reauth")
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}

	if err := ResetInterruptedPluginSyncs(f.app); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{enabled: "configured", disabled: "configured", untouched: "needs_reauth"} {
		if got := f.stored(t, id).GetString("status"); got != want {
			t.Errorf("status %q, want %q", got, want)
		}
	}
}

func TestCronTreatsASwitchOffDuringTheSyncAsSkipped(t *testing.T) {
	f := newPluginInstanceFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	usePluginInstanceSync(t, func(context.Context, core.App, meilisearch.ServiceManager, pluginsystem.LocalPlugin, *core.Record, func(*core.Record)) (*pluginSystemSyncResult, error) {
		return nil, errPluginInstanceSwitchedOff
	})
	if err := syncPluginInstancesOneByOne(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, []*core.Record{f.stored(t, instanceID)}, nil); err != nil {
		t.Fatalf("a switch-off was reported as a failed sync: %v", err)
	}
}

type pluginSyncFunc func(context.Context, core.App, meilisearch.ServiceManager, pluginsystem.LocalPlugin, *core.Record, func(*core.Record)) (*pluginSystemSyncResult, error)

func usePluginInstanceSync(t *testing.T, fake pluginSyncFunc) {
	t.Helper()
	original := runPluginInstanceSync
	runPluginInstanceSync = fake
	t.Cleanup(func() { runPluginInstanceSync = original })
}
