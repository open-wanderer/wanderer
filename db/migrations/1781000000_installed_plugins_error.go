package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("installed_plugins")
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("error") != nil {
			return nil
		}
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"hidden": false,
			"id": "textplginerr",
			"max": 0,
			"min": 0,
			"name": "error",
			"pattern": "",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("installed_plugins")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("error")
		return app.Save(collection)
	})
}
