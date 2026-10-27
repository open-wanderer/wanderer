package util

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type feedSaveContextApp struct {
	core.App
	ctx context.Context
}

func (app feedSaveContextApp) Save(model core.Model) error {
	return app.App.SaveWithContext(app.ctx, model)
}

func TestInsertIntoFeedDoesNotBlockAnOpenTransaction(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := core.NewBaseCollection("feed")
	for _, name := range []string{"actor", "author", "item", "type"} {
		collection.Fields.Add(&core.TextField{Name: name})
	}
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transactionStarted, normalSaving := make(chan struct{}), make(chan struct{})
	app.OnRecordCreate("feed").BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("item") == "outside" {
			close(normalSaving)
		}
		return event.Next()
	})
	transactionDone, normalDone := make(chan error, 1), make(chan error, 1)
	go func() {
		transactionDone <- app.RunInTransaction(func(tx core.App) error {
			initial := core.NewRecord(collection)
			initial.Set("item", "initial")
			if err := tx.Save(initial); err != nil {
				return err
			}
			close(transactionStarted)
			select {
			case <-normalSaving:
			case <-ctx.Done():
				return ctx.Err()
			}
			record, err := InsertIntoFeed(tx, "reader", "author", "inside", TrailFeed)
			if err == nil && record == nil {
				return fmt.Errorf("transaction did not create its entry")
			}
			return err
		})
	}()
	select {
	case <-transactionStarted:
	case err := <-transactionDone:
		t.Fatalf("transaction did not begin: %v", err)
	case <-time.After(time.Second):
		t.Fatal("transaction did not begin")
	}
	started := time.Now()
	go func() {
		record, err := InsertIntoFeed(feedSaveContextApp{App: app, ctx: ctx}, "reader", "author", "outside", TrailFeed)
		if err == nil && record == nil {
			err = fmt.Errorf("normal caller did not create its entry")
		}
		normalDone <- err
	}()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for remaining := 2; remaining > 0; remaining-- {
		select {
		case err := <-transactionDone:
			if err != nil {
				t.Errorf("transaction feed: %v", err)
			}
		case err := <-normalDone:
			if err != nil {
				t.Errorf("normal feed: %v", err)
			}
		case <-deadline.C:
			t.Errorf("feed insertions exceeded one second")
			cancel()
			for ; remaining > 0; remaining-- {
				select {
				case <-transactionDone:
				case <-normalDone:
				case <-time.After(3 * time.Second):
					t.Fatal("feed callers did not finish after cancellation")
				}
			}
			return
		}
	}
	if time.Since(started) >= time.Second {
		t.Fatal("feed insertions exceeded one second")
	}
	if count, err := app.CountRecords("feed"); err != nil || count != 3 {
		t.Fatalf("stored feed entries: %d, want 3; error: %v", count, err)
	}
}
