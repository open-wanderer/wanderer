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
			"hidden": false,
			"id": "select_actor_type_001",
			"maxSelect": 1,
			"name": "actor_type",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": ["person", "instance"]
		}`)); err != nil {
			return err
		}

		// Backfill existing user actor rows to the 'person' select value
		if _, err := app.DB().NewQuery(
			"UPDATE activitypub_actors SET actor_type = 'person' WHERE actor_type = '' OR actor_type IS NULL",
		).Execute(); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("select_actor_type_001")

		return app.Save(collection)
	})
}
