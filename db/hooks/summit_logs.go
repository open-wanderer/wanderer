package hooks

import (
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

func DeleteSummitLogHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
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

		err = federation.CreateSummitLogDeleteActivity(e.App, e.Record)
		if err != nil {
			return err
		}
		return nil
	}
}
