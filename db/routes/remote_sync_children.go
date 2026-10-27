package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// Invalid remote children do not roll back their valid siblings or the parent.
// Database and hook failures still propagate instead of being mistaken for data.
func saveRemoteSyncChild(app core.App, record *core.Record) (bool, error) {
	err := app.Save(record)
	if err == nil {
		return true, nil
	}
	var invalid validation.Errors
	if errors.As(err, &invalid) {
		app.Logger().Warn("remote sync child skipped", "collection", record.Collection().Name, "iri", record.GetString("iri"), "error", err)
		return false, nil
	}
	return false, err
}

func interruptedRemoteChildren(err error) bool {
	return isDegradableSyncError(err) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Metadata is already available when child work exhausts its budget. End the
// attempt normally and retry children through the regular refresh interval.
func finishRemoteSyncChildren(app core.App, ctx context.Context, record *core.Record, syncErr error) (*core.Record, error) {
	if syncErr != nil {
		if !interruptedRemoteChildren(syncErr) {
			return record, syncErr
		}
		app.Logger().Warn("remote child refresh interrupted", "collection", record.Collection().Name, "iri", record.GetString("iri"), "error", syncErr)
	}
	record.Set("needs_full_sync", false)
	record.Set("full_sync_completed", true)
	return record, app.SaveWithContext(context.WithoutCancel(ctx), record)
}

// Child batches leave room for other writers. Complete the refresh on the
// current trail rather than saving the earlier metadata snapshot again.
// A save after metadata may include changes this refresh did not fetch, so its
// needs_full_sync flag stays set.
func finishRemoteTrailSync(app core.App, ctx context.Context, record *core.Record, syncErr error) (*core.Record, error) {
	if syncErr != nil {
		if !interruptedRemoteChildren(syncErr) {
			return record, syncErr
		}
		app.Logger().Warn("remote child refresh interrupted", "collection", record.Collection().Name, "iri", record.GetString("iri"), "error", syncErr)
	}
	var stored *core.Record
	err := app.RunInTransaction(func(tx core.App) error {
		current, err := tx.FindRecordById(record.Collection(), record.Id)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: %w", errRemoteGone, err)
		}
		if err != nil {
			return err
		}
		if current.GetString("updated") == record.GetString("updated") {
			current.Set("needs_full_sync", false)
		}
		current.Set("full_sync_completed", true)
		stored = current
		return tx.SaveWithContext(context.WithoutCancel(ctx), current)
	})
	if err != nil {
		return record, err
	}
	return stored, nil
}
