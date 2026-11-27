package util

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

type remoteSummitLogCleanupKey struct{}

// RemoteSummitLogCleanup coalesces only the search reindexes of explicitly
// marked remote-copy deletions. All per-record delete hooks still run.
type RemoteSummitLogCleanup struct {
	app     core.App
	mu      sync.Mutex
	pending map[string]meilisearch.ServiceManager
}

// NewRemoteSummitLogCleanup must receive the parent app, not a transaction app:
// Flush reads fresh committed parents after the cleanup transactions return.
func NewRemoteSummitLogCleanup(app core.App) *RemoteSummitLogCleanup {
	return &RemoteSummitLogCleanup{app: app, pending: map[string]meilisearch.ServiceManager{}}
}

// Delete opts this deletion alone into coalescing. Like app.Delete, it uses a
// background database context; the remote fetch deadline must not introduce
// different cancellation semantics merely to transport the batch marker.
// A nil batch keeps the ordinary delete behavior.
func (batch *RemoteSummitLogCleanup) Delete(app core.App, record *core.Record) error {
	if batch == nil || record.Collection().Name != "summit_logs" {
		return app.Delete(record)
	}
	return app.DeleteWithContext(context.WithValue(context.Background(), remoteSummitLogCleanupKey{}, batch), record)
}

// CollectRemoteSummitLogReindex is called only by the after-delete-success
// hook, which PocketBase defers until commit. Rolled-back deletes collect no
// work, and unrelated deletes cannot inherit a marker from an app or a flag.
func CollectRemoteSummitLogReindex(ctx context.Context, trail string, client meilisearch.ServiceManager) bool {
	if ctx == nil {
		return false
	}
	batch, _ := ctx.Value(remoteSummitLogCleanupKey{}).(*RemoteSummitLogCleanup)
	if batch == nil {
		return false
	}
	batch.mu.Lock()
	batch.pending[trail] = client
	batch.mu.Unlock()
	return true
}

// Flush is warning-only, matching the individual delete hook. Call it even on
// early-error paths: earlier child transactions may already have committed.
func (batch *RemoteSummitLogCleanup) Flush() {
	batch.mu.Lock()
	pending := batch.pending
	batch.pending = map[string]meilisearch.ServiceManager{}
	batch.mu.Unlock()
	for _, id := range slices.Sorted(maps.Keys(pending)) {
		trail, err := batch.app.FindRecordById("trails", id)
		switch {
		case err == nil:
			if err := IndexTrails(batch.app, []*core.Record{trail}, pending[id]); err != nil {
				batch.app.Logger().Warn("trail could not be reindexed after remote summit log cleanup", "trail", id, "error", err)
			}
		case errors.Is(err, sql.ErrNoRows):
			// A concurrently deleted parent has nothing left to reindex.
		default:
			batch.app.Logger().Warn("trail of deleted remote summit logs could not be looked up for reindexing", "trail", id, "error", err)
		}
	}
}
