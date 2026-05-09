package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("sajmiuau")
		collection.Fields.RemoveById("htr35nha")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(17, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "sajmiuau",
			"max": 0,
			"min": 0,
			"name": "external_id",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(18, []byte(`{
			"hidden": false,
			"id": "htr35nha",
			"maxSelect": 1,
			"name": "external_provider",
			"presentable": false,
			"required": false,
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
