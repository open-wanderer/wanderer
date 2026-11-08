package routes

import (
	"context"
	"os"
	"os/exec"
	"pocketbase/pluginsystem"
	"testing"
	"time"
)

func TestPluginSyncWaitsForWorkerWithinCallerDeadline(t *testing.T) {
	if os.Getenv("WANDERER_SYNC_WAIT_CHILD") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		for _, limit := range []string{"1", "2"} {
			t.Run(limit, func(t *testing.T) {
				command := exec.CommandContext(t.Context(), executable, "-test.run=^TestPluginSyncWaitsForWorkerWithinCallerDeadline$")
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
