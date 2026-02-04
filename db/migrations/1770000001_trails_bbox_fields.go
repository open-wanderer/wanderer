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

		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
            "hidden": false,
            "id": "minlat2c6",
            "max": null,
            "min": null,
            "name": "min_lat",
            "onlyInt": false,
            "presentable": false,
            "required": false,
            "system": false,
            "type": "number"
        }`)); err != nil {
			return err
		}
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
            "hidden": false,
            "id": "minlon2c6",
            "max": null,
            "min": null,
            "name": "min_lon",
            "onlyInt": false,
            "presentable": false,
            "required": false,
            "system": false,
            "type": "number"
        }`)); err != nil {
			return err
		}
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
            "hidden": false,
            "id": "maxlat2c6",
            "max": null,
            "min": null,
            "name": "max_lat",
            "onlyInt": false,
            "presentable": false,
            "required": false,
            "system": false,
            "type": "number"
        }`)); err != nil {
			return err
		}
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
            "hidden": false,
            "id": "maxlon2c6",
            "max": null,
            "min": null,
            "name": "max_lon",
            "onlyInt": false,
            "presentable": false,
            "required": false,
            "system": false,
            "type": "number"
        }`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("minlat2c6")
		collection.Fields.RemoveById("minlon2c6")
		collection.Fields.RemoveById("maxlat2c6")
		collection.Fields.RemoveById("maxlon2c6")

		return app.Save(collection)
	})
}
