package util

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// newFeedUtilTestApp builds an app with a minimal feed collection that carries
// the same unique (actor, item) index the production migration adds.
func newFeedUtilTestApp(t *testing.T) *core.BaseApp {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("feed")
	collection.Id = "pbc_feed_util_test1"
	collection.Fields.Add(
		&core.TextField{Name: "actor"},
		&core.TextField{Name: "author"},
		&core.TextField{Name: "item"},
		&core.TextField{Name: "type"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_feed_actor_item", true, "`actor`, `item`", "")
	if err := app.Save(collection); err != nil {
		t.Fatalf("create feed collection: %v", err)
	}

	return app
}

func countFeedRows(t *testing.T, app core.App, actor, item string) int64 {
	t.Helper()
	n, err := app.CountRecords("feed", dbx.HashExp{"actor": actor, "item": item})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestInsertIntoFeedLostRaceReturnsExisting simulates a parallel insert that
// wins between the initial lookup and the Save of the losing call.
func TestInsertIntoFeedLostRaceReturnsExisting(t *testing.T) {
	app := newFeedUtilTestApp(t)

	var competingId string
	var fired atomic.Bool
	app.OnRecordCreate("feed").BindFunc(func(e *core.RecordEvent) error {
		if fired.CompareAndSwap(false, true) {
			competing := core.NewRecord(e.Record.Collection())
			competing.Set("actor", "actor1")
			competing.Set("author", "author1")
			competing.Set("item", "item1")
			competing.Set("type", string(TrailFeed))
			if err := e.App.Save(competing); err != nil {
				t.Errorf("save competing row: %v", err)
			}
			competingId = competing.Id
		}
		return e.Next()
	})

	record, err := InsertIntoFeed(app, "actor1", "author1", "item1", TrailFeed)
	if err != nil {
		t.Fatalf("InsertIntoFeed lost the race and returned an error: %v", err)
	}
	if record == nil || record.Id != competingId {
		t.Fatalf("returned record = %v, want the competing row %q", record, competingId)
	}
	if n := countFeedRows(t, app, "actor1", "item1"); n != 1 {
		t.Fatalf("feed rows = %d, want 1", n)
	}
}

func TestInsertIntoFeedConcurrent(t *testing.T) {
	app := newFeedUtilTestApp(t)

	const workers = 16
	start := make(chan struct{})
	ids := make([]string, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			record, err := InsertIntoFeed(app, "actor1", "author1", "item1", TrailFeed)
			errs[i] = err
			if record != nil {
				ids[i] = record.Id
			}
		}(i)
	}
	close(start)
	wg.Wait()

	for i := 0; i < workers; i++ {
		if errs[i] != nil {
			t.Fatalf("call %d returned an error: %v", i, errs[i])
		}
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatalf("call %d returned id %q, want %q", i, ids[i], ids[0])
		}
	}
	if n := countFeedRows(t, app, "actor1", "item1"); n != 1 {
		t.Fatalf("feed rows = %d, want 1", n)
	}
}

func TestInsertIntoFeedSequentialRepeat(t *testing.T) {
	app := newFeedUtilTestApp(t)

	first, err := InsertIntoFeed(app, "actor1", "author1", "item1", TrailFeed)
	if err != nil {
		t.Fatal(err)
	}
	second, err := InsertIntoFeed(app, "actor1", "author1", "item1", TrailFeed)
	if err != nil {
		t.Fatal(err)
	}
	if first.Id != second.Id {
		t.Fatalf("second call returned %q, want %q", second.Id, first.Id)
	}
	if n := countFeedRows(t, app, "actor1", "item1"); n != 1 {
		t.Fatalf("feed rows = %d, want 1", n)
	}
}
