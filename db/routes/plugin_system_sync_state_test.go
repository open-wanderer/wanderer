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
	usePluginInstanceSync(t, func(_ context.Context, _ core.App, _ meilisearch.ServiceManager, _ pluginsystem.LocalPlugin, instance *core.Record) (*pluginSystemSyncResult, error) {
		t.Errorf("synced %s from a stale copy", instance.Id)
		return &pluginSystemSyncResult{}, nil
	})
	if err := syncPluginInstancesOneByOne(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, listed); err != nil {
		t.Fatal(err)
	}
}

func TestPluginSyncPanicBecomesAnError(t *testing.T) {
	f := newPluginInstanceFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	usePluginInstanceSync(t, func(context.Context, core.App, meilisearch.ServiceManager, pluginsystem.LocalPlugin, *core.Record) (*pluginSystemSyncResult, error) {
		panic("bad plugin item")
	})

	_, skipped, err := syncFreshPluginInstance(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, instanceID)
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
}

func (r *reopenTestRuntime) Call(context.Context, pluginsystem.LocalPlugin, string, []byte, pluginsystem.RequestPolicyContext) ([]byte, error) {
	return nil, errors.New("not used")
}

func (r *reopenTestRuntime) OpenSession(context.Context, pluginsystem.LocalPlugin, pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
	r.opened++
	return r.session, nil
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
			nil, nil, nil, false, false)
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
			nil, nil, nil, false, false)
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
				nil, nil, nil, false, false)
			if !errors.Is(err, errPluginInstanceSwitchedOff) || details != 1 {
				t.Fatalf("sync went on after the plugin was %s: %d detail calls, err=%v", name, details, err)
			}
		})
	}
}

func TestSyncKeepsErrorFieldsUntouchedWhenSwitchedOffDuringTheSync(t *testing.T) {
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
	// The user switches the plugin off while the plugin is listing.
	session := syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) {
		record := f.stored(t, instanceID)
		record.Set("enabled", false)
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
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

	_, err := syncPluginInstance(t.Context(), f.app, nil, plugin, f.stored(t, instanceID))
	if !errors.Is(err, errPluginInstanceSwitchedOff) {
		t.Fatalf("sync of a switched-off plugin: %v", err)
	}
	stored := f.stored(t, instanceID)
	if len(pluginsystem.JSONMapFromRecord(stored, "last_error")) != 0 || !stored.GetDateTime("retry_not_before").IsZero() {
		t.Fatalf("switch-off recorded as an error: last_error=%v, retry_not_before=%v", stored.Get("last_error"), stored.GetDateTime("retry_not_before"))
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
	for id, want := range map[string]string{enabled: "configured", disabled: "disabled", untouched: "needs_reauth"} {
		if got := f.stored(t, id).GetString("status"); got != want {
			t.Errorf("status %q, want %q", got, want)
		}
	}
}

func TestCronTreatsASwitchOffDuringTheSyncAsSkipped(t *testing.T) {
	f := newPluginInstanceFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	usePluginInstanceSync(t, func(context.Context, core.App, meilisearch.ServiceManager, pluginsystem.LocalPlugin, *core.Record) (*pluginSystemSyncResult, error) {
		return nil, errPluginInstanceSwitchedOff
	})
	if err := syncPluginInstancesOneByOne(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, []*core.Record{f.stored(t, instanceID)}); err != nil {
		t.Fatalf("a switch-off was reported as a failed sync: %v", err)
	}
}

type pluginSyncFunc func(context.Context, core.App, meilisearch.ServiceManager, pluginsystem.LocalPlugin, *core.Record) (*pluginSystemSyncResult, error)

func usePluginInstanceSync(t *testing.T, fake pluginSyncFunc) {
	t.Helper()
	original := runPluginInstanceSync
	runPluginInstanceSync = fake
	t.Cleanup(func() { runPluginInstanceSync = original })
}
