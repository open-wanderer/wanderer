package routes

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

// The same worker protocol/DB imports as production are exercised after a real
// session deadline, including the page cursor carried into a replacement worker.
func TestPluginSyncRenewsOrReopensBeforeListAndDetail(t *testing.T) {
	if os.Getenv("WANDERER_SYNC_TTL_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		for _, scenario := range []string{"1/list", "1/detail", "1/renew", "2/list", "2/detail", "2/renew"} {
			parts := strings.Split(scenario, "/")
			limit, phase := parts[0], parts[1]
			t.Run(scenario, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, executable, "-test.run=^TestPluginSyncRenewsOrReopensBeforeListAndDetail$")
				command.Env = append(os.Environ(), "WANDERER_SYNC_TTL_CHILD=1", "WANDERER_PLUGIN_WORKER_MAX="+limit, "WANDERER_SYNC_TTL_PHASE="+phase, "WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT=400ms", "GORACE=atexit_sleep_ms=0")
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
	refs.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "user"}, &core.TextField{Name: "provider"}, &core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"}, &core.AutodateField{Name: "created", OnCreate: true})
	for _, collection := range []*core.Collection{trails, refs} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "hikes"}}
	sessions := &pluginSyncRuntimeSession{runtime: pluginsystem.NewWorkerRuntime(), plugin: plugin}
	if err := sessions.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sessions.close(context.Background()) })
	result, err := syncPluginCapability(t.Context(), app, nil, sessions, plugin, pluginsystem.CapabilityManifest{Export: "list"}, pluginsystem.CapabilityManifest{Export: "detail"}, instance, actor, nil, nil, nil, false, false)
	if err != nil || result == nil || result.Imported != 2 || result.Skipped != 0 {
		t.Fatalf("session expiry lost imports: %#v %v", result, err)
	}
	saved, err := app.FindAllRecords("trails")
	if err != nil || len(saved) != 2 {
		t.Fatalf("saved trails=%d error=%v", len(saved), err)
	}
	for _, trail := range saved {
		if trail.GetString("gpx") == "" {
			t.Fatal("imported track file was lost")
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
	wantWorkers, wantLists := 2, 1
	if phase == "list" {
		wantLists = 2
	}
	if phase == "renew" {
		wantWorkers = 1
	}
	if len(workers) != wantWorkers || lists != wantLists || details != 2 {
		t.Fatalf("wrong worker/cursor reuse: workers=%d lists=%d details=%d, want %d/%d/2; calls:\n%s", len(workers), lists, details, wantWorkers, wantLists, calls)
	}
}
