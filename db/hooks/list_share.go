package hooks

import (
	"pocketbase/federation"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func CreateListShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		record := e.Record
		listId := record.GetString("list")
		shares, err := e.App.FindAllRecords("list_share",
			dbx.NewExp("list = {:listId}", dbx.Params{"listId": listId}),
		)
		if err != nil {
			return err
		}
		actorIds := make([]string, len(shares))
		for i, r := range shares {
			actorIds[i] = r.GetString("actor")
		}
		err = util.UpdateListShares(listId, actorIds, client)

		if err != nil {
			return err
		}

		err = federation.CreateAnnounceActivity(e.App, record, federation.ListAnnounceType)
		if err != nil {
			return err
		}

		return nil
	}
}

func DeleteListShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		record := e.Record
		listId := record.GetString("list")
		err := util.UpdateListShares(listId, []string{}, client)
		if err != nil {
			return err
		}
		return e.Next()
	}
}
