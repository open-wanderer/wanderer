package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func CreateSummitLogHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
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

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		ctx, err := util.GetSafeActorContext(e.Request, userActor)
		if err != nil {
			return err
		}

		trail, err := e.App.FindRecordById("trails", e.Record.GetString("trail"))
		if err != nil {
			return err
		}

		if err := util.IndexTrails(e.App, []*core.Record{trail}, client); err != nil {
			return err
		}

		err = federation.CreateSummitLogActivity(e.App, ctx, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		return nil
	}
}

func UpdateSummitLogHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		err := e.Next()
		if err != nil {
			return err
		}

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		ctx, err := util.GetSafeActorContext(e.Request, userActor)
		if err != nil {
			return err
		}

		err = federation.CreateSummitLogActivity(e.App, ctx, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}
		return nil
	}
}

// DeleteSummitLogHandler runs on OnRecordAfterDeleteSuccess rather than on the
// delete request, so that summit logs removed by a cascade — when their trail is
// deleted, or when their author's account is — also retract their federated
// copies. Inside a transaction these hooks are deferred until after it commits,
// so the record is already gone by the time this runs and an error here cannot
// roll the deletion back.
func DeleteSummitLogHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		trail, err := e.App.FindRecordById("trails", e.Record.GetString("trail"))
		if err != nil {
			// The trail is gone too, so this summit log was removed as part of
			// that trail's own cascade. There is nothing left to reindex, and
			// the trail's delete activity already covers what federated.
			if errors.Is(err, sql.ErrNoRows) {
				return e.Next()
			}
			return err
		}

		if err := util.IndexTrails(e.App, []*core.Record{trail}, client); err != nil {
			return err
		}

		err = federation.CreateSummitLogDeleteActivity(e.App, e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	}
}
