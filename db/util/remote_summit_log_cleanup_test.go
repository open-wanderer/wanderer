package util

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteSummitLogCleanupCollectsConcurrentHooks(t *testing.T) {
	batch := NewRemoteSummitLogCleanup(nil)
	ctx := context.WithValue(context.Background(), remoteSummitLogCleanupKey{}, batch)
	var wait sync.WaitGroup
	for i := 0; i < 64; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			if !CollectRemoteSummitLogReindex(ctx, fmt.Sprint(i%4), nil) {
				t.Error("marked hook did not collect its reindex")
			}
		}(i)
	}
	wait.Wait()
	if len(batch.pending) != 4 {
		t.Fatalf("duplicate concurrent hooks were not coalesced: %d", len(batch.pending))
	}
	if CollectRemoteSummitLogReindex(context.Background(), "ordinary", nil) || CollectRemoteSummitLogReindex(nil, "ordinary", nil) {
		t.Fatal("unmarked context inherited a cleanup batch")
	}
}

type cleanupDeleteContextApp struct {
	core.App
	context       context.Context
	ordinaryCalls int
}

func (app *cleanupDeleteContextApp) Delete(core.Model) error {
	app.ordinaryCalls++
	return nil
}

func (app *cleanupDeleteContextApp) DeleteWithContext(ctx context.Context, _ core.Model) error {
	app.context = ctx
	return nil
}

func TestRemoteSummitLogCleanupMarksOnlyExplicitSummitLogDeletes(t *testing.T) {
	app := &cleanupDeleteContextApp{}
	batch := NewRemoteSummitLogCleanup(app)
	log := core.NewRecord(core.NewBaseCollection("summit_logs"))
	if err := batch.Delete(app, log); err != nil {
		t.Fatal(err)
	}
	if app.context == nil || app.context.Done() != nil || app.context.Value(remoteSummitLogCleanupKey{}) != batch {
		t.Fatal("marker must use app.Delete's background cancellation semantics")
	}
	var ordinary *RemoteSummitLogCleanup
	if err := ordinary.Delete(app, log); err != nil {
		t.Fatal(err)
	}
	if err := batch.Delete(app, core.NewRecord(core.NewBaseCollection("waypoints"))); err != nil {
		t.Fatal(err)
	}
	if app.ordinaryCalls != 2 {
		t.Fatal("unmarked or other-collection deletes changed behavior")
	}
}
