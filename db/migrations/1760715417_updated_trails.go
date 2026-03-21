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

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(20, []byte(`{
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
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(20, []byte(`{
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
				"komoot"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
