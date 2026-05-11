package hooks

import (
	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func CreateListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record

		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		if err := util.IndexLists(e.App, []*core.Record{record}, client); err != nil {
			return err
		}

		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateListActivity(e.App, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		_, err = util.InsertIntoFeed(e.App, author.Id, author.Id, record.Id, util.ListFeed)
		if err != nil {
			return err
		}

		return nil
	}
}

func UpdateListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		err = util.UpdateList(e.App, record, author, client)
		if err != nil {
			return err
		}

		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateListActivity(e.App, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		return nil
	}
}

func DeleteListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		_, err := client.Index("lists").DeleteDocument(record.Id, nil)
		if err != nil {
			return err
		}

		err = federation.CreateListDeleteActivity(e.App, record)
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
