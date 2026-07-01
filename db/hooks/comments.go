package hooks

import (
	"fmt"
	"os"
	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func CreateCommentHandler() func(e *core.RecordRequestEvent) error {
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
			e.Record.Set("iri", fmt.Sprintf("%s/api/v1/comment/%s", origin, e.Record.Id))
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
