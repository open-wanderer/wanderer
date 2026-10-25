package hooks

import (
	"context"
	"pocketbase/federation"
	"pocketbase/util"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func CreateActorHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		return util.IndexActors([]*core.Record{e.Record}, client)
	}
}

func UpdateActorHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		return util.UpdateActor(e.Record, client)
	}
}

const actorDeleteRecipientsKey = "__delete_recipients"

// CollectActorDeleteRecipientsHandler works out who has to be told before the
// actor and its activity records are removed, keeps the list on the record for
// AnnounceActorDeleteHandler, and purges the actor's activity records once the
// actor itself is deleted.
func CollectActorDeleteRecipientsHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		actor := e.Record

		recipients, err := federation.ActorDeleteRecipients(e.App, actor)
		if err != nil {
			e.App.Logger().Error(
				"could not collect recipients to announce actor deletion to",
				"actor", actor.Id, "error", err,
			)
			recipients = nil
		}

		actor.Set(actorDeleteRecipientsKey, recipients)

		if err := e.Next(); err != nil {
			return err
		}

		// Still inside the deleting transaction. The audience above was read
		// from these rows, so they go only now; a failure rolls the actor back
		// together with them.
		return federation.DeleteActorActivities(e.App, actor.GetString("iri"))
	}
}

func AnnounceActorDeleteHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		actor := e.Record

		recipients, _ := actor.GetRaw(actorDeleteRecipientsKey).([]string)

		if err := federation.CreateActorDeleteActivity(e.App, actor, recipients); err != nil {
			e.App.Logger().Error(
				"could not announce actor deletion",
				"actor", actor.Id, "error", err,
			)
		}

		return e.Next()
	}
}

func DeleteActorHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		ctx, cancel := context.WithTimeout(e.Context, 5*time.Second)
		defer cancel()
		task, err := client.Index("actors").DeleteDocumentWithContext(ctx, e.Record.Id, nil)
		if err != nil {
			e.App.Logger().Warn("actor deletion could not be queued in search index",
				"actor", e.Record.Id, "error", err)
		} else {
			completed, err := client.WaitForTaskWithContext(ctx, task.TaskUID, 50*time.Millisecond)
			if err != nil {
				e.App.Logger().Warn("actor deletion not confirmed by search index",
					"actor", e.Record.Id, "task", task.TaskUID, "error", err)
			} else if completed.Status != meilisearch.TaskStatusSucceeded {
				e.App.Logger().Error("search index failed to delete actor",
					"actor", e.Record.Id, "task", task.TaskUID, "status", completed.Status)
			}
		}
		return e.Next()
	}
}
