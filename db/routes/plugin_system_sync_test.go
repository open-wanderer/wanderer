package routes

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pocketbase/plugins/importer"
)

func TestStartBoundedPluginAssetAutoAttachLimitsConcurrency(t *testing.T) {
	const limit = 2
	slots := make(chan struct{}, limit)
	started := make(chan struct{}, limit+1)
	release := make(chan struct{})
	var tasks sync.WaitGroup
	tasks.Add(limit + 1)

	scheduled := make(chan struct{})
	scheduleErr := make(chan error, 1)
	go func() {
		for range limit + 1 {
			if err := startBoundedPluginAssetAutoAttach(context.Background(), slots, func() {
				started <- struct{}{}
				<-release
				tasks.Done()
			}); err != nil {
				scheduleErr <- err
				return
			}
		}
		close(scheduled)
	}()

	for range limit {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for bounded tasks to start")
		}
	}
	select {
	case <-started:
		t.Fatal("task started above concurrency limit")
	case <-scheduled:
		t.Fatal("all tasks were scheduled while every slot was occupied")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	done := make(chan struct{})
	go func() {
		tasks.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bounded tasks to finish")
	}
	select {
	case <-scheduled:
	case err := <-scheduleErr:
		t.Fatalf("unexpected scheduling error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for all tasks to be scheduled")
	}
}

func TestStartBoundedPluginAssetAutoAttachStopsWaitingOnCancellation(t *testing.T) {
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	taskStarted := false

	err := startBoundedPluginAssetAutoAttach(ctx, slots, func() { taskStarted = true })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
	if taskStarted {
		t.Fatal("task started after context cancellation")
	}
	if len(slots) != 1 {
		t.Fatalf("slot count = %d, want occupied slot to remain unchanged", len(slots))
	}
}

func TestCategoryMappingPreservesExplicitEmptyMap(t *testing.T) {
	mapping := categoryMapping(map[string]any{
		"categoryMapping": map[string]any{},
	})
	if mapping == nil {
		t.Fatal("expected explicit empty category mapping to be preserved")
	}
	if len(mapping) != 0 {
		t.Fatalf("expected empty category mapping, got %#v", mapping)
	}
}

func TestCategoryMappingNilWhenMissing(t *testing.T) {
	if mapping := categoryMapping(map[string]any{}); mapping != nil {
		t.Fatalf("expected missing category mapping to be nil, got %#v", mapping)
	}
}

func TestCategoryMappingPreservesBlankProviderMapping(t *testing.T) {
	mapping := categoryMapping(map[string]any{
		"categoryMapping": map[string]any{
			"Ride": "",
		},
	})
	if mapping == nil {
		t.Fatal("expected category mapping")
	}
	if value, ok := mapping["Ride"]; !ok || value != (importer.CategoryMappingValue{}) {
		t.Fatalf("expected blank provider mapping to be preserved, got %#v", mapping)
	}
}

func TestCategoryMappingParsesStructuredTarget(t *testing.T) {
	mapping := categoryMapping(map[string]any{
		"categoryMapping": map[string]any{
			"TrailRun": map[string]any{
				"category":    "Running",
				"subcategory": "Trail",
			},
		},
	})
	want := importer.CategoryMappingValue{Category: "Running", Subcategory: "Trail"}
	if value, ok := mapping["TrailRun"]; !ok || value != want {
		t.Fatalf("structured provider mapping = %#v, want %#v", mapping, want)
	}
}
