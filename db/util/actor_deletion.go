package util

import (
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// DeleteRemoteActor requires authored content to consist of remote copies.
func DeleteRemoteActor(app core.App, actor *core.Record) error {
	return app.RunInTransaction(func(txApp core.App) error {
		for _, collection := range []string{"trails", "lists", "comments", "summit_logs", "waypoints"} {
			records, err := txApp.FindRecordsByFilter(collection, "author={:actor}", "", -1, 0, dbx.Params{"actor": actor.Id})
			if err != nil {
				return err
			}
			for _, record := range records {
				if IRIHasRecordIdentity(record.GetString("iri"), collection, record.Id) {
					txApp.Logger().Warn("remote account retains records with own paths",
						"actor_iri", actor.GetString("iri"), "collection", collection, "record", record.Id)
					return errors.New("actor retains records with local paths")
				}
			}
		}
		return txApp.Delete(actor)
	})
}
