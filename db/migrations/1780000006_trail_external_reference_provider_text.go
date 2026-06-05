package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("provider")
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "select420001002",
			"max": 128,
			"min": 1,
			"name": "provider",
			"pattern": "^[a-z0-9][a-z0-9_-]*$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("provider")
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"hidden": false,
			"id": "select420001002",
			"maxSelect": 1,
			"name": "provider",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"strava",
				"komoot",
				"hammerhead"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
