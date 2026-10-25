package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	backfillFullSyncCompleted1789200000 = `
		UPDATE %s
		SET full_sync_completed = TRUE
		WHERE iri != ''
			AND needs_full_sync = FALSE
	`
)

func init() {
	m.Register(upAddFullSyncCompleted1789200000, downAddFullSyncCompleted1789200000)
}

func upAddFullSyncCompleted1789200000(app core.App) error {
	for _, name := range []string{"trails", "lists"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("full_sync_completed") == nil {
			if err := collection.Fields.AddMarshaledJSON([]byte(`{
				"hidden": false,
				"id": "boolfullsyncdone",
				"name": "full_sync_completed",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "bool"
			}`)); err != nil {
				return err
			}
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if _, err := app.DB().
			NewQuery(fmt.Sprintf(backfillFullSyncCompleted1789200000, name)).
			Execute(); err != nil {
			return err
		}
	}

	return nil
}

func downAddFullSyncCompleted1789200000(app core.App) error {
	for _, name := range []string{"trails", "lists"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("boolfullsyncdone")

		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}
