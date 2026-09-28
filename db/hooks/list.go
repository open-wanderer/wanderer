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

func CreateListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record

		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		// add local iri
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			return fmt.Errorf("ORIGIN not set")
		}
		if e.Record.GetString("iri") == "" {
			e.Record.Set("iri", fmt.Sprintf("%s/api/v1/list/%s", origin, e.Record.Id))
			if err = e.App.UnsafeWithoutHooks().Save(e.Record); err != nil {
				return err
			}
		}

		if err := util.IndexLists(e.App, []*core.Record{record}, client); err != nil {
			return err
		}

		err = e.Next()
		if err != nil {
			return err
		}

		if !author.GetBool("is_local") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return nil
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

		if !author.GetBool("is_local") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateListActivity(e.App, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}

		return nil
	}
}

const listDeleteRecipientsKey = "__delete_recipients"

// CollectListDeleteRecipientsHandler is CollectTrailDeleteRecipientsHandler
// for a list: the shares naming its recipients are cascaded away with it.
func CollectListDeleteRecipientsHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		list := e.Record
		audience, err := federation.ListDeleteRecipients(e.App, list, list.GetBool("public"))
		if err != nil {
			e.App.Logger().Error(
				"could not collect recipients to announce list deletion to",
				"list", list.Id, "error", err,
			)
			audience = federation.DeleteAudience{}
		}
		list.Set(listDeleteRecipientsKey, audience)
		return e.Next()
	}
}

func DeleteListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		_, err := client.Index("lists").DeleteDocument(record.Id, nil)
		if err != nil {
			return err
		}

		audience, _ := record.GetRaw(listDeleteRecipientsKey).(federation.DeleteAudience)
		err = federation.CreateListDeleteActivity(e.App, record, audience)
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
