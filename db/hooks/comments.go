package hooks

import (
	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func CreateCommentHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		e.Next()

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		ctx, err := util.GetSafeActorContext(e.Request, userActor)
		if err != nil {
			return err
		}

		err = federation.CreateCommentActivity(e.App, ctx, e.Record, pub.CreateType)
		if err != nil {
			return err
		}
		return nil
	}
}

func UpdateCommentHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		ctx, err := util.GetSafeActorContext(e.Request, userActor)
		if err != nil {
			return err
		}

		err = federation.CreateCommentActivity(e.App, ctx, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}
		return e.Next()

	}
}

func DeleteCommentHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		err := federation.CreateCommentDeleteActivity(e.App, client, e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	}
}
