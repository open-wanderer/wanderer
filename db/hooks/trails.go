package hooks

import (
	"log"
	"pocketbase/federation"
	"pocketbase/util"
	"time"

	"github.com/go-ap/activitypub"
	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func CreateTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}
		if err := util.IndexTrails(e.App, []*core.Record{record}, client); err != nil {
			return err
		}
		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote trail
			// we create a stub trail record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, author, e.Record, activitypub.CreateType)
		if err != nil {
			return err
		}

		_, err = util.InsertIntoFeed(e.App, author.Id, author.Id, record.Id, util.TrailFeed)
		if err != nil {
			return err
		}

		return nil
	}
}

func UpdateTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}
		err = util.UpdateTrail(e.App, record, author, client)
		if err != nil {
			return err
		}
		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote trail
			// we create a stub trail record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, author, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}

		return nil
	}
}

func DeleteTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		task, err := client.Index("trails").DeleteDocument(record.Id, nil)
		if err != nil {
			return err
		}

		interval := 500 * time.Millisecond
		_, err = client.WaitForTask(task.TaskUID, interval)
		if err != nil {
			log.Fatalf("Error waiting for task completion: %v", err)
		}

		err = federation.CreateTrailDeleteActivity(e.App, e.Record)
		if err != nil {
			return err
		}

		err = util.DeleteFromFeed(e.App, record.Id)
		if err != nil {
			return err
		}

		return e.Next()
	}
}
