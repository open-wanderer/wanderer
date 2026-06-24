package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text_actor_type_001",
			"max": 0,
			"min": 0,
			"name": "actor_type",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// Backfill existing user actor rows so future queries on actor_type='Person' work
		if _, err := app.DB().NewQuery(
			"UPDATE activitypub_actors SET actor_type = 'Person' WHERE actor_type = '' OR actor_type IS NULL",
		).Execute(); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("text_actor_type_001")

		return app.Save(collection)
	})
}
