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
			"id": "json3846545605",
			"maxSize": 0,
			"name": "settings",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		records, err := app.FindAllRecords("categories")
		if err != nil {
			return err
		}

		for _, record := range records {
			record.Set("settings", map[string]any{
				"wp_merge_enabled": true,
				"wp_merge_radius":  50,
			})
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("kjxvi8asj2igqwf")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("json3846545605")

		return app.Save(collection)
	})
}
