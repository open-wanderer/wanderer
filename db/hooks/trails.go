package hooks

import (
	"fmt"
	"log"
	"os"
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

		userActor, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}
		if err := util.SavePolyline(e.App, record); err != nil {
			log.Printf("failed to save polyline for trail %s: %v", record.Id, err)
		}

		// add local iri
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			return fmt.Errorf("ORIGIN not set")
		}
		if e.Record.GetString("iri") == "" {
			e.Record.Set("iri", fmt.Sprintf("%s/api/v1/trail/%s", origin, e.Record.Id))
			if err = e.App.UnsafeWithoutHooks().Save(e.Record); err != nil {
				return err
			}
		}

		if err := util.IndexTrails(e.App, []*core.Record{record}, client); err != nil {
			return err
		}

		err = e.Next()
		if err != nil {
			return err
		}

		if !userActor.GetBool("is_local") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return nil
		}

		ctx, err := util.GetSafeActorContext(nil, userActor)

		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, ctx, e.Record, activitypub.CreateType)
		if err != nil {
			return err
		}

		_, err = util.InsertIntoFeed(e.App, userActor.Id, userActor.Id, record.Id, util.TrailFeed)
		if err != nil {
			return err
		}

		return nil
	}
}

func UpdateTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		userActor, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		if record.GetString("gpx") != record.Original().GetString("gpx") {
			if err := util.SavePolyline(e.App, record); err != nil {
				log.Printf("failed to save polyline for trail %s: %v", record.Id, err)
			}
		}

		err = util.UpdateTrail(e.App, record, userActor, client)
		if err != nil {
			return err
		}
		if !userActor.GetBool("is_local") {
			// this happens if someone fetches a remote trail
			// we create a stub trail record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		ctx, err := util.GetSafeActorContext(nil, userActor)

		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, ctx, e.Record, pub.UpdateType)
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
