package hooks

import (
	"database/sql"
	"pocketbase/federation"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func CreateTrailLikeHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		record := e.Record

		trailId := record.GetString("trail")
		actorId := record.GetString("actor")
		actor, err := e.App.FindRecordById("activitypub_actors", actorId)
		if err != nil {
			return err
		}
		trail, err := e.App.FindRecordById("trails", trailId)
		if err != nil {
			return err
		}
		likes, err := e.App.FindAllRecords("trail_like",
			dbx.NewExp("trail = {:trailId}", dbx.Params{"trailId": trailId}),
		)
		if err != nil {
			return err
		}

		trail.Set("like_count", len(likes))
		err = e.App.UnsafeWithoutHooks().Save(trail)
		if err != nil {
			return err
		}

		actorIds := make([]string, len(likes))
		for i, r := range likes {
			actorIds[i] = r.GetString("actor")
		}
		err = util.UpdateTrailLikes(trailId, actorIds, client)
		if err != nil {
			return err
		}

		if !actor.GetBool("isLocal") {
			// this happens if someone likes a remote trail
			// we create a local copy
			// no need to create an activity for that
			return nil
		}

		err = federation.CreateLikeActivity(e.App, record)
		if err != nil {
			return err
		}

		return nil
	}
}

func DeleteTrailLikeHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {

		record := e.Record

		trailId := record.GetString("trail")
		actorId := record.GetString("actor")
		actor, err := e.App.FindRecordById("activitypub_actors", actorId)
		if err != nil {
			return err
		}
		// trail might deleted be already if this is called as part of a cascade
		trail, err := e.App.FindRecordById("trails", trailId)
		if err != nil && err == sql.ErrNoRows {
			return nil
		} else if err != nil {
			return err
		}
		likes, err := e.App.CountRecords("trail_like", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trailId}))
		if err != nil {
			return err
		}

		trail.Set("like_count", likes)
		err = e.App.UnsafeWithoutHooks().Save(trail)
		if err != nil {
			return err
		}

		err = util.UpdateTrailLikes(trailId, []string{}, client)
		if err != nil {
			return err
		}

		if !actor.GetBool("isLocal") {
			// this happens if someone likes a remote trail
			// we create a local copy
			// no need to create an activity for that
			return nil
		}

		err = federation.CreateUnlikeActivity(e.App, record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}
