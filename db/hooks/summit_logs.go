package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"pocketbase/federation"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func CreateSummitLogHandler(client meilisearch.ServiceManager, updates *FederationUpdates) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		batchID := e.Request.URL.Query().Get("federationBatch")
		if batchID != "" && !util.ValidFederationSaveBatchID(batchID) {
			return apis.NewBadRequestError("invalid federation save batch", nil)
		}

		err := e.Next()
		if err != nil {
			return err
		}

		// add local iri
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			return fmt.Errorf("ORIGIN not set")
		}
		if e.Record.GetString("iri") == "" {
			e.Record.Set("iri", fmt.Sprintf("%s/api/v1/summit-log/%s", origin, e.Record.Id))
		}
		err = e.App.UnsafeWithoutHooks().Save(e.Record)
		if err != nil {
			return err
		}

		if batchID != "" {
			updates.BeginSave(e.Record, batchID)
		}

		trail, err := e.App.FindRecordById("trails", e.Record.GetString("trail"))
		if err != nil {
			return err
		}

		if err := util.IndexTrails(e.App, []*core.Record{trail}, client); err != nil {
			return err
		}

		updates.EnqueueCreate(e.Record)

		return nil
	}
}

func UpdateSummitLogHandler(updates *FederationUpdates) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		err := e.Next()
		if err != nil {
			return err
		}

		updates.Enqueue(e.Record)
		return nil
	}
}

// DeleteSummitLogHandler runs on OnRecordAfterDeleteSuccess rather than on the
// delete request, so that summit logs removed by a cascade — when their trail is
// deleted, or when their author's account is — also retract their federated
// copies. Inside a transaction these hooks are deferred until after it commits,
// so the record is already gone by the time this runs and an error here cannot
// roll the deletion back.
//
// The Delete goes out before the trail is reindexed, and a failed reindex is
// only logged: the log is gone either way, and its followers must not miss
// the Delete because the search index is unreachable.
func DeleteSummitLogHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := errors.Join(
			federation.CreateSummitLogDeleteActivity(e.App, e.Record),
			federation.StripObjectActivities(e.App, e.Record.GetString("iri")),
		)
		if err != nil {
			return err
		}

		if util.CollectRemoteSummitLogReindex(e.Context, e.Record.GetString("trail"), client) {
			return e.Next()
		}

		trail, err := e.App.FindRecordById("trails", e.Record.GetString("trail"))
		switch {
		case err == nil:
			if err := util.IndexTrails(e.App, []*core.Record{trail}, client); err != nil {
				e.App.Logger().Warn("trail could not be reindexed after summit log deletion",
					"trail", trail.Id, "summit_log", e.Record.Id, "error", err)
			}
		case errors.Is(err, sql.ErrNoRows):
			// The trail went first and took this log with it. Nothing is left
			// to reindex.
		default:
			e.App.Logger().Warn("trail of a deleted summit log could not be looked up for reindexing",
				"summit_log", e.Record.Id, "error", err)
		}

		return e.Next()
	}
}
