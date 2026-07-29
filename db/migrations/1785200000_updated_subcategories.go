package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1781100000")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"hidden": false,
			"id": "json_settings_9002",
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

		records, err := app.FindAllRecords("subcategories")
		if err != nil {
			return err
		}

		for _, record := range records {
			record.Set("settings", map[string]any{})
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1781100000")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("json_settings_9002")

		return app.Save(collection)
	})
}
