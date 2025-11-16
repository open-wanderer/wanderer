package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("kjxvi8asj2igqwf")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "json2689183292",
			"maxSize": 0,
			"name": "difficulty_thresholds",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("kjxvi8asj2igqwf")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("json2689183292")

		return app.Save(collection)
	})
}
