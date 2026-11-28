package routes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/pluginsystem"
)

type syncExpiryLogHandler struct {
	slog.Handler
	message string
	once    sync.Once
}

func (h *syncExpiryLogHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Message == h.message {
		h.once.Do(func() { time.Sleep(600 * time.Millisecond) })
	}
	return h.Handler.Handle(ctx, record)
}

// Exercise idle expiry and cooperative age-based recycling through the real
// worker protocol and DB import, including page cursors, auth and backfills.
func TestPluginSyncRenewsOrReopensBeforeListAndDetail(t *testing.T) {
	if os.Getenv("WANDERER_SYNC_TTL_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		testPattern := "^" + regexp.QuoteMeta(t.Name()) + "$"
		for _, scenario := range []string{
			"1/list", "1/detail", "1/renew", "1/recycle-list", "1/recycle-detail", "1/recycle-backfill", "1/slow-detail",
			"2/list", "2/detail", "2/renew", "2/recycle-list", "2/recycle-detail", "2/recycle-backfill", "2/slow-detail",
		} {
			parts := strings.Split(scenario, "/")
			limit, phase := parts[0], parts[1]
			t.Run(scenario, func(t *testing.T) {
				sessionTimeout := "400ms"
				if phase == "renew" {
					// Reuse well below the age limit, with ample startup/DB margin.
					// Timer renewal itself is covered deterministically by worker tests.
					sessionTimeout = "10s"
				}
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, executable, "-test.run="+testPattern)
				command.Env = append(os.Environ(), "WANDERER_SYNC_TTL_CHILD=1", "WANDERER_PLUGIN_WORKER_MAX="+limit, "WANDERER_SYNC_TTL_PHASE="+phase, "WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT="+sessionTimeout, "GORACE=atexit_sleep_ms=0")
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("actual session-expiry sync: %v\n%s", err, output)
				}
			})
		}
		return
	}
	syncTestWorker(t)
	callsPath := filepath.Join(t.TempDir(), "calls")
	t.Setenv("WANDERER_SYNC_RPC_LOG", callsPath)
	base, instance, actor := newSyncRuntimeTestApp(t)
	phase := os.Getenv("WANDERER_SYNC_TTL_PHASE")
	wantItems := 2
	if phase == "recycle-detail" || phase == "recycle-backfill" {
		wantItems = 3
	}
	pauseMessage := ""
	if phase == "list" {
		pauseMessage = "imported plugin trail"
	}
	if phase == "detail" {
		pauseMessage = "plugin capability batch returned items"
	}
	logger := slog.New(&syncExpiryLogHandler{Handler: slog.NewTextHandler(os.Stderr, nil), message: pauseMessage})
	app := assetImportLoggingApp{App: base, logger: logger}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "author"}, &core.BoolField{Name: "completed"}, &core.DateField{Name: "date"}, &core.DateField{Name: "completed_at"}, &core.FileField{Name: "gpx", MaxSelect: 1})
	refs := core.NewBaseCollection("trail_external_reference")
	refs.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "user"}, &core.TextField{Name: "provider"}, &core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"}, &core.AutodateField{Name: "created", OnCreate: true},
		&core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"})
	for _, collection := range []*core.Collection{trails, refs} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	var hostConfig map[string]any
	if phase == "recycle-backfill" {
		hostConfig = map[string]any{"categoryMapping": map[string]any{"Ride": "Biking"}}
		for _, id := range []string{"first", "second", "third"} {
			trail := core.NewRecord(trails)
			trail.Set("name", "Existing "+id)
			if err := app.Save(trail); err != nil {
				t.Fatal(err)
			}
			ref := core.NewRecord(refs)
			ref.Load(map[string]any{"trail": trail.Id, "user": instance.GetString("user"), "provider": "hikes", "external_id": id, "plugin_id": "hikes"})
			if err := app.Save(ref); err != nil {
				t.Fatal(err)
			}
		}
	}
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "hikes", Auth: pluginsystem.AuthManifest{Contexts: map[string]pluginsystem.AuthContext{
		"session": {Type: pluginsystem.AuthTypeSession, SecretFields: []string{"password"}},
	}}}}
	sessions := &pluginSyncRuntimeSession{runtime: pluginsystem.NewWorkerRuntime(), plugin: plugin}
	if err := sessions.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sessions.close(context.Background()) })
	result, err := syncPluginCapability(t.Context(), app, nil, sessions, plugin, pluginsystem.CapabilityManifest{Export: "list"}, pluginsystem.CapabilityManifest{Export: "detail"}, instance, actor,
		map[string]any{"password": "keep-auth"}, map[string]any{"after": "keep-options"}, hostConfig, false, false, nil)
	wantImported, wantSkipped := wantItems, 0
	if phase == "recycle-backfill" {
		wantImported, wantSkipped = 0, wantItems
	}
	if err != nil || result == nil || result.Imported != wantImported || result.Skipped != wantSkipped {
		t.Fatalf("session expiry lost imports: %#v %v", result, err)
	}
	saved, err := app.FindAllRecords("trails")
	if err != nil || len(saved) != wantItems {
		t.Fatalf("saved trails=%d error=%v", len(saved), err)
	}
	for _, trail := range saved {
		if phase != "recycle-backfill" && trail.GetString("gpx") == "" {
			t.Fatal("imported track file was lost")
		}
	}
	if phase == "recycle-backfill" {
		savedRefs, err := app.FindAllRecords("trail_external_reference")
		if err != nil || len(savedRefs) != wantItems {
			t.Fatalf("saved backfill refs=%d error=%v", len(savedRefs), err)
		}
		for _, ref := range savedRefs {
			if ref.GetString("provider_category") != "Ride" || ref.GetDateTime("provider_category_checked_at").IsZero() {
				t.Fatalf("backfill lost after recycling: %s", ref.GetString("external_id"))
			}
		}
	}
	calls, err := os.ReadFile(callsPath)
	if err != nil {
		t.Fatal(err)
	}
	workers := map[string]bool{}
	lists, details := 0, 0
	for _, line := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			t.Fatalf("invalid RPC log: %q", line)
		}
		workers[fields[0]] = true
		switch fields[1] {
		case "list":
			lists++
		case "detail":
			details++
		}
	}
	wantLists := 1
	if phase == "list" || phase == "recycle-list" {
		wantLists = 2
	}
	// Slow hosts may legitimately recycle more often. Keep all call/import and
	// backfill counts exact, but do not require precisely two worker lifetimes.
	// The worker's deterministic age test (with an unexpired idle timer) proves
	// that removing the age check fails independently of idle expiry/reopening.
	validWorkers := len(workers) >= 2
	if phase == "renew" {
		validWorkers = len(workers) == 1
	}
	if !validWorkers || lists != wantLists || details != wantItems {
		t.Fatalf("wrong worker/cursor reuse: workers=%d lists=%d details=%d, phase=%s, want lists=%d details=%d; calls:\n%s", len(workers), lists, details, phase, wantLists, wantItems, calls)
	}
}

type syncRenewalTestSession struct {
	pluginsystem.RuntimeSession
	renewErr error
	onClose  func()
}

func (s *syncRenewalTestSession) Renew(context.Context) error { return s.renewErr }
func (s *syncRenewalTestSession) Close(context.Context) error {
	s.onClose()
	return nil
}

func TestPluginSyncEnsureOpenOnlyRecyclesExplicitRequestsOrFatalErrors(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		err     error
		recycle bool
	}{
		{"healthy", nil, false},
		{"aged", pluginsystem.ErrRuntimeSessionRecycleRequired, true},
		{"wrapped aged", fmt.Errorf("renew: %w", pluginsystem.ErrRuntimeSessionRecycleRequired), true},
		{"fatal", pluginsystem.RuntimeSessionFatalError{Err: errors.New("worker died")}, true},
		{"canceled", context.Canceled, false},
		{"other error", errors.New("renewal failed"), false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			runtime := &reopenTestRuntime{}
			closed := false
			current := &syncRenewalTestSession{renewErr: scenario.err, onClose: func() {
				if runtime.opened != 0 {
					t.Fatal("replacement was opened before closing the old session")
				}
				closed = true
			}}
			sessions := &pluginSyncRuntimeSession{runtime: runtime, session: current}
			err := sessions.ensureOpen(t.Context())
			if scenario.recycle {
				if err != nil || !closed || runtime.opened != 1 {
					t.Fatalf("session was not recycled: closed=%v opened=%d err=%v", closed, runtime.opened, err)
				}
			} else if !errors.Is(err, scenario.err) || closed || runtime.opened != 0 {
				t.Fatalf("unrequested recycle or lost error: closed=%v opened=%d err=%v", closed, runtime.opened, err)
			}
		})
	}
}
