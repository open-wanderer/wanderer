package util

import (
	"sync"

	"github.com/pocketbase/pocketbase/core"
)

type targetSyncCall struct {
	app core.App
	fn  func(core.App) error
}

type targetSyncBatch struct {
	depth int
	seen  map[string]struct{}
	calls []targetSyncCall
}

// Entries exist only while an explicit operation is active in an existing
// transaction. They contain callbacks, never cached target data.
var targetSyncBatches = struct {
	sync.Mutex
	active map[*core.TxAppInfo]*targetSyncBatch
}{active: make(map[*core.TxAppInfo]*targetSyncBatch)}

// WithDeferredTargetSync coalesces target syncs until the outermost scope
// returns, still inside the caller's transaction. Without a transaction it
// simply runs fn; it never opens a transaction or postpones work until commit.
//
// Normal returns flush even on error, preserving callers' partial-write
// semantics. The body error takes precedence over the first callback error.
// Panics discard the pending callbacks and continue unwinding.
func WithDeferredTargetSync(app core.App, fn func() error) (err error) {
	info := app.TxInfo()
	if info == nil {
		return fn()
	}

	targetSyncBatches.Lock()
	batch := targetSyncBatches.active[info]
	if batch == nil {
		batch = &targetSyncBatch{seen: make(map[string]struct{})}
		targetSyncBatches.active[info] = batch
	}
	batch.depth++
	targetSyncBatches.Unlock()

	returned := false
	defer func() {
		targetSyncBatches.Lock()
		// An inner panic may have discarded this scope, even if the body
		// recovered it and subsequently opened another scope on the same tx.
		if targetSyncBatches.active[info] != batch {
			targetSyncBatches.Unlock()
			return
		}
		if !returned {
			delete(targetSyncBatches.active, info)
			targetSyncBatches.Unlock()
			return
		}
		batch.depth--
		if batch.depth > 0 {
			targetSyncBatches.Unlock()
			return
		}
		delete(targetSyncBatches.active, info)
		targetSyncBatches.Unlock()

		// Remove the scope and release the mutex before callbacks: a sync may
		// itself write links or start another independently flushed scope.
		for _, call := range batch.calls {
			if syncErr := call.fn(call.app); err == nil && syncErr != nil {
				err = syncErr
			}
		}
	}()

	err = fn()
	returned = true
	return err
}

// DeferTargetSync queues the first callback for key in the active transaction
// scope, in registration order. False means the caller must sync immediately.
// The callback receives the registering app, which belongs to that same tx.
func DeferTargetSync(app core.App, key string, fn func(core.App) error) bool {
	info := app.TxInfo()
	if info == nil {
		return false
	}
	targetSyncBatches.Lock()
	defer targetSyncBatches.Unlock()
	batch := targetSyncBatches.active[info]
	if batch == nil {
		return false
	}
	if _, exists := batch.seen[key]; !exists {
		batch.seen[key] = struct{}{}
		batch.calls = append(batch.calls, targetSyncCall{app: app, fn: fn})
	}
	return true
}
