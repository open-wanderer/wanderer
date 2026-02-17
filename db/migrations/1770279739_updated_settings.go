package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("uavt73rsqcn1n13")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"hidden": false,
			"id": "0sepzvkh",
			"maxSelect": 1,
			"name": "language",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"en",
				"de",
				"fr",
				"hu",
				"it",
				"nl",
				"pl",
				"pt",
				"zh",
				"es",
				"eu",
				"ru",
				"no",
				"cs"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("uavt73rsqcn1n13")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"hidden": false,
			"id": "0sepzvkh",
			"maxSelect": 1,
			"name": "language",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"en",
				"de",
				"fr",
				"hu",
				"it",
				"nl",
				"pl",
				"pt",
				"zh",
				"es",
				"eu",
				"ru",
				"no"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
