package migrations

import (
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		field, ok := collection.Fields.GetByName("polyline").(*core.TextField)
		if ok {
			field.Max = util.PolylineMaxLength
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		field, ok := collection.Fields.GetByName("polyline").(*core.TextField)
		if ok {
			field.Max = 0
		}

		return app.Save(collection)
	})
}
