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

		if collection.Fields.GetByName("polyline") != nil {
			return nil
		}

		field := &core.TextField{}
		field.Name = "polyline"
		field.Required = false
		collection.Fields.Add(field)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("polyline")

		return app.Save(collection)
	})
}
