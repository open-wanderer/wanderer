package routes

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/pluginsystem"
)

func TestRoutingPreparedProfileRejected(t *testing.T) {
	unsupportedPluginError := &pluginsystem.PluginError{Code: "unsupported_profile"}
	invalidPluginError := &pluginsystem.PluginError{Code: "invalid_request"}
	tests := []struct {
		name      string
		pluginErr *pluginsystem.PluginError
		err       error
		want      bool
	}{
		{name: "no error"},
		{name: "plugin output", pluginErr: unsupportedPluginError, want: true},
		{name: "other plugin output", pluginErr: invalidPluginError},
		{
			name:      "plugin output takes precedence over call error",
			pluginErr: invalidPluginError,
			err:       pluginsystem.PluginCallError{PluginError: *unsupportedPluginError},
		},
		{
			name: "plugin call error",
			err:  pluginsystem.PluginCallError{PluginError: *unsupportedPluginError},
			want: true,
		},
		{name: "routing error", err: routingErrorFromCode("unsupported_profile", "expired"), want: true},
		{name: "other routing error", err: routingErrorFromCode("invalid_request", "invalid")},
		{name: "other error", err: context.DeadlineExceeded},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := routingPreparedProfileRejected(test.pluginErr, test.err); got != test.want {
				t.Fatalf("routingPreparedProfileRejected() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRoutingPreparedProfileCacheDeduplicatesConcurrentPreparation(t *testing.T) {
	cache := newRoutingPreparedProfileCache(time.Minute, 8)
	var calls atomic.Int32
	prepare := func(context.Context) (string, error) {
		calls.Add(1)
		time.Sleep(10 * time.Millisecond)
		return "custom-profile", nil
	}

	var wg sync.WaitGroup
	results := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			preparedKey, err := cache.getOrPrepare(context.Background(), "same-fingerprint", prepare)
			if err != nil {
				t.Errorf("prepare profile: %v", err)
				return
			}
			results <- preparedKey
		}()
	}
	wg.Wait()
	close(results)
	for preparedKey := range results {
		if preparedKey != "custom-profile" {
			t.Fatalf("prepared key = %q, want custom-profile", preparedKey)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("prepare calls = %d, want 1", calls.Load())
	}
}

func TestRoutingPreparedProfileCacheDoesNotCacheFailures(t *testing.T) {
	cache := newRoutingPreparedProfileCache(time.Minute, 8)
	calls := 0
	prepare := func(context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", context.DeadlineExceeded
		}
		return "custom-profile", nil
	}
	if _, err := cache.getOrPrepare(context.Background(), "fingerprint", prepare); err == nil {
		t.Fatal("first profile preparation unexpectedly succeeded")
	}
	preparedKey, err := cache.getOrPrepare(context.Background(), "fingerprint", prepare)
	if err != nil {
		t.Fatalf("retry profile preparation: %v", err)
	}
	if preparedKey != "custom-profile" || calls != 2 {
		t.Fatalf("retry = %q after %d calls", preparedKey, calls)
	}
}

func TestRoutingPreparedProfileCacheUsesAbsoluteTTL(t *testing.T) {
	cache := newRoutingPreparedProfileCache(time.Minute, 8)
	var calls atomic.Int32
	prepare := func(context.Context) (string, error) {
		return "profile-" + string(rune('0'+calls.Add(1))), nil
	}
	first, err := cache.getOrPrepare(context.Background(), "fingerprint", prepare)
	if err != nil {
		t.Fatalf("first preparation: %v", err)
	}
	cache.mu.Lock()
	firstExpiry := cache.entries["fingerprint"].ExpiresAt
	cache.mu.Unlock()
	if _, err := cache.getOrPrepare(context.Background(), "fingerprint", prepare); err != nil {
		t.Fatalf("cache hit: %v", err)
	}
	cache.mu.Lock()
	if got := cache.entries["fingerprint"].ExpiresAt; !got.Equal(firstExpiry) {
		cache.mu.Unlock()
		t.Fatalf("cache hit extended absolute expiry from %v to %v", firstExpiry, got)
	}
	entry := cache.entries["fingerprint"]
	entry.ExpiresAt = time.Now().Add(-time.Second)
	cache.entries["fingerprint"] = entry
	cache.mu.Unlock()
	second, err := cache.getOrPrepare(context.Background(), "fingerprint", prepare)
	if err != nil {
		t.Fatalf("preparation after absolute expiry: %v", err)
	}
	if first == second || calls.Load() != 2 {
		t.Fatalf("absolute TTL returned %q then %q after %d calls", first, second, calls.Load())
	}
}

func TestRoutingPreparedProfileCacheOutlivesFirstCallerCancellation(t *testing.T) {
	cache := newRoutingPreparedProfileCache(time.Minute, 8)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	prepare := func(ctx context.Context) (string, error) {
		calls.Add(1)
		close(started)
		select {
		case <-release:
			return "custom-profile", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := cache.getOrPrepare(firstCtx, "fingerprint", prepare)
		firstResult <- err
	}()
	<-started
	cancelFirst()
	if err := <-firstResult; !strings.Contains(fmt.Sprint(err), context.Canceled.Error()) {
		t.Fatalf("first caller error = %v, want cancellation", err)
	}

	secondResult := make(chan error, 1)
	go func() {
		preparedKey, err := cache.getOrPrepare(context.Background(), "fingerprint", prepare)
		if err == nil && preparedKey != "custom-profile" {
			err = fmt.Errorf("prepared key = %q", preparedKey)
		}
		secondResult <- err
	}()
	close(release)
	if err := <-secondResult; err != nil {
		t.Fatalf("second caller: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("detached preparation calls = %d, want 1", calls.Load())
	}
}

func TestRoutingPreparedProfileCacheCleansUpAfterPanic(t *testing.T) {
	cache := newRoutingPreparedProfileCache(time.Minute, 8)
	if _, err := cache.getOrPrepare(context.Background(), "fingerprint", func(context.Context) (string, error) {
		panic("boom")
	}); err == nil || !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("panic preparation error = %v", err)
	}
	preparedKey, err := cache.getOrPrepare(context.Background(), "fingerprint", func(context.Context) (string, error) {
		return "recovered-profile", nil
	})
	if err != nil || preparedKey != "recovered-profile" {
		t.Fatalf("preparation after panic = %q, %v", preparedKey, err)
	}
}
