package pluginsystem

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkerSlotTimeoutAppliesToOneCaller(t *testing.T) {
	for _, limit := range []string{"1", "2"} {
		t.Run(limit, func(t *testing.T) {
			t.Setenv("WANDERER_PLUGIN_WORKER_MAX", limit)
			t.Setenv("WANDERER_PLUGIN_WORKER_SLOT_TIMEOUT", "5ms")
			workerSlotsMu.Lock()
			previous := workerSlots
			workerSlots = nil
			workerSlotsMu.Unlock()
			defer func() { workerSlotsMu.Lock(); workerSlots = previous; workerSlotsMu.Unlock() }()
			first, err := acquireWorkerSlot(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer first.Release(1)
			if limit == "2" {
				slot, err := acquireWorkerSlot(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				defer slot.Release(1)
			}
			scoped := WithWorkerSlotAcquireTimeout(t.Context(), 25*time.Millisecond)
			started := time.Now()
			if _, err := acquireWorkerSlot(scoped); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("scoped wait=%v", err)
			}
			if time.Since(started) < 20*time.Millisecond {
				t.Fatal("scoped caller used the interactive timeout")
			}
			started = time.Now()
			if _, err := acquireWorkerSlot(t.Context()); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("interactive wait=%v", err)
			}
			if time.Since(started) > 20*time.Millisecond {
				t.Fatal("scoped wait changed the interactive timeout")
			}
		})
	}
}
