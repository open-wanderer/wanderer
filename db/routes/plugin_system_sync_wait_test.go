package routes

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"pocketbase/pluginsystem"
	"regexp"
	"testing"
	"time"
)

func TestPluginSyncWaitsForWorkerWithinCallerDeadline(t *testing.T) {
	if os.Getenv("WANDERER_SYNC_WAIT_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		testPattern := "^" + regexp.QuoteMeta(t.Name()) + "$"
		for _, limit := range []string{"1", "2"} {
			t.Run(limit, func(t *testing.T) {
				command := exec.CommandContext(t.Context(), executable, "-test.run="+testPattern)
				command.Env = append(os.Environ(), "WANDERER_SYNC_WAIT_CHILD=1", "WANDERER_PLUGIN_WORKER_MAX="+limit, "WANDERER_PLUGIN_WORKER_SLOT_TIMEOUT=5ms")
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("worker admission=%v\n%s", err, output)
				}
			})
		}
		return
	}
	syncTestWorker(t)
	runtime := pluginsystem.NewWorkerRuntime()
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "hikes"}}
	occupied, err := runtime.OpenSession(t.Context(), plugin, pluginsystem.RequestPolicyContext{})
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close(context.Background())
	if os.Getenv("WANDERER_PLUGIN_WORKER_MAX") == "2" {
		extra, err := runtime.OpenSession(t.Context(), plugin, pluginsystem.RequestPolicyContext{})
		if err != nil {
			t.Fatal(err)
		}
		defer extra.Close(context.Background())
	}
	released := make(chan struct{})
	go func() { time.Sleep(50 * time.Millisecond); _ = occupied.Close(context.Background()); close(released) }()
	defer func() { <-released }()
	sessions := pluginSyncRuntimeSession{runtime: runtime, plugin: plugin}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := sessions.open(ctx); err != nil {
		t.Fatalf("sync abandoned an available worker: %v", err)
	}
	_ = sessions.close(context.Background())
}

func TestPluginSyncRecycleYieldsWorkerSlotToWaitingCaller(t *testing.T) {
	if os.Getenv("WANDERER_SYNC_RECYCLE_WAIT_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, executable, "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
		command.Env = append(os.Environ(), "WANDERER_SYNC_RECYCLE_WAIT_CHILD=1", "WANDERER_PLUGIN_WORKER_MAX=1", "WANDERER_PLUGIN_WORKER_SLOT_TIMEOUT=5ms",
			"WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT=400ms", "WANDERER_SYNC_TTL_PHASE=slow-detail", "GORACE=atexit_sleep_ms=0")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("recycling admission=%v\n%s", err, output)
		}
		return
	}
	syncTestWorker(t)
	runtime := pluginsystem.NewWorkerRuntime()
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "hikes"}}
	sessions := &pluginSyncRuntimeSession{runtime: runtime, plugin: plugin}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := sessions.open(ctx); err != nil {
		t.Fatal(err)
	}
	defer sessions.close(context.Background())

	type admission struct {
		session pluginsystem.RuntimeSession
		err     error
	}
	waiting := make(chan admission, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		worker, err := runtime.OpenSession(pluginsystem.WithWorkerSlotAcquireTimeout(ctx, 2*time.Second), plugin, pluginsystem.RequestPolicyContext{})
		waiting <- admission{worker, err}
	}()
	<-started
	// The fixture keeps this export running beyond the soft age limit. The
	// other caller must remain queued until recycling closes the old process.
	input, err := json.Marshal(pluginSystemDetailInput{
		Auth:    map[string]any{"password": "keep-auth"},
		Options: map[string]any{"after": "keep-options"},
		Summary: pluginsystem.TrailSummary{Source: pluginsystem.TrailImportSource{Provider: "hikes", ExternalID: "first"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.session.Call(ctx, "detail", input, pluginsystem.RuntimeCallOptions{}); err != nil {
		t.Fatalf("age/idle timeout interrupted the active detail: %v", err)
	}
	select {
	case admitted := <-waiting:
		if admitted.session != nil {
			_ = admitted.session.Close(context.Background())
		}
		t.Fatalf("worker slot was released before closing the aged session: %v", admitted.err)
	default:
	}

	recycled := make(chan error, 1)
	recycleDone := make(chan struct{})
	go func() {
		defer close(recycleDone)
		recycled <- sessions.ensureOpen(ctx)
	}()
	// In particular on t.Fatal, join before the earlier session-close defer
	// accesses sessions.session. Cancellation releases pending admission.
	defer func() {
		cancel()
		<-recycleDone
	}()
	var admitted admission
	select {
	case admitted = <-waiting:
		if admitted.err != nil {
			t.Fatalf("waiting caller did not get the recycled slot: %v", admitted.err)
		}
	case <-ctx.Done():
		t.Fatal("recycling did not release its slot to the waiting caller")
	}
	defer admitted.session.Close(context.Background())
	// Reopening the sync must wait longer than the interactive 5ms test limit,
	// rather than retaining the old slot or failing with that shorter limit.
	select {
	case err := <-recycled:
		t.Fatalf("sync did not re-enter admission behind the waiting caller: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if worker, err := runtime.OpenSession(ctx, plugin, pluginsystem.RequestPolicyContext{}); !errors.Is(err, context.DeadlineExceeded) {
		if worker != nil {
			_ = worker.Close(context.Background())
		}
		t.Fatalf("sync changed the interactive admission timeout: %v", err)
	}
	if err := admitted.session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-recycled:
		if err != nil {
			t.Fatalf("sync could not reopen after yielding its slot: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("sync did not get a replacement worker")
	}
	if _, err := sessions.session.Call(ctx, "list", input, pluginsystem.RuntimeCallOptions{}); err != nil {
		t.Fatalf("replacement worker is unusable: %v", err)
	}
}
