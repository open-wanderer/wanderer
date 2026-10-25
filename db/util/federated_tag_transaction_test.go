package util

import (
	"context"
	"fmt"
	"testing"
	"time"

	"pocketbase/tagname"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type tagSaveContextApp struct {
	core.App
	ctx context.Context
}

func (app tagSaveContextApp) Save(model core.Model) error {
	return app.App.SaveWithContext(app.ctx, model)
}

func TestResolveFederatedTagDoesNotBlockAnOpenTransaction(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := core.NewBaseCollection("tags")
	collection.Fields.Add(&core.TextField{Name: "name", Max: tagname.MaxLength, Pattern: tagname.Pattern})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transactionStarted, normalSaving := make(chan struct{}), make(chan struct{})
	app.OnRecordCreate("tags").BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("name") == "outside" {
			close(normalSaving)
		}
		return event.Next()
	})
	transactionDone, normalDone := make(chan error, 1), make(chan error, 1)
	go func() {
		transactionDone <- app.RunInTransaction(func(tx core.App) error {
			initial := core.NewRecord(collection)
			initial.Set("name", "initial")
			if err := tx.Save(initial); err != nil {
				return err
			}
			close(transactionStarted)
			select {
			case <-normalSaving:
			case <-ctx.Done():
				return ctx.Err()
			}
			record, err := ResolveFederatedTag(tx, "inside")
			if err == nil && record == nil {
				return fmt.Errorf("transaction did not resolve its tag")
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
		record, err := ResolveFederatedTag(tagSaveContextApp{App: app, ctx: ctx}, "outside")
		if err == nil && record == nil {
			err = fmt.Errorf("normal caller did not resolve its tag")
		}
		normalDone <- err
	}()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for remaining := 2; remaining > 0; remaining-- {
		select {
		case err := <-transactionDone:
			if err != nil {
				t.Errorf("transaction tag: %v", err)
			}
		case err := <-normalDone:
			if err != nil {
				t.Errorf("normal tag: %v", err)
			}
		case <-deadline.C:
			t.Errorf("tag resolutions exceeded one second")
			cancel()
			// Cancelling the normal writer releases its process lock, so a
			// broken transaction-lock variant can finish and be cleaned up.
			for ; remaining > 0; remaining-- {
				select {
				case <-transactionDone:
				case <-normalDone:
				case <-time.After(3 * time.Second):
					t.Fatal("tag callers did not finish after cancellation")
				}
			}
			return
		}
	}
	if time.Since(started) >= time.Second {
		t.Fatal("tag resolutions exceeded one second")
	}
	if count, err := app.CountRecords("tags"); err != nil || count != 3 {
		t.Fatalf("stored tags: %d, want 3; error: %v", count, err)
	}
}
