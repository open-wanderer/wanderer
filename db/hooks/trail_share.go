package hooks

import (
	"pocketbase/federation"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func CreateTrailShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		record := e.Record

		trailId := record.GetString("trail")
		shares, err := e.App.FindAllRecords("trail_share",
			dbx.NewExp("trail = {:trailId}", dbx.Params{"trailId": trailId}),
		)
		if err != nil {
			return err
		}
		actorIds := make([]string, len(shares))
		for i, r := range shares {
			actorIds[i] = r.GetString("actor")
		}
		err = util.UpdateTrailShares(trailId, actorIds, client)
		if err != nil {
			return err
		}

		err = federation.CreateAnnounceActivity(e.App, record, federation.TrailAnnounceType)
		if err != nil {
			return err
		}

		return nil
	}
}

func DeleteTrailShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		record := e.Record

		trailId := record.GetString("trail")
		err := util.UpdateTrailShares(trailId, []string{}, client)
		if err != nil {
			return err
		}
		return e.Next()
	}
}
