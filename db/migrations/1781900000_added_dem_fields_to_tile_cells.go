package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_581507754")
		if err != nil {
			return err
		}

		// add field: dem_status (mirrors status, but tracks the DEM archive
		// lifecycle independently so a vector-ready/DEM-failed cell can still
		// serve the basemap). required:false — existing vector-only rows must
		// re-save cleanly, and an absent value means "DEM not generated".
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"help": "",
			"hidden": false,
			"id": "select4192883017",
			"maxSelect": 1,
			"name": "dem_status",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"pending",
				"ready",
				"error"
			]
		}`)); err != nil {
			return err
		}

		// add field: dem_size_bytes (mirrors size_bytes)
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"help": "",
			"hidden": false,
			"id": "number3611729480",
			"max": null,
			"min": null,
			"name": "dem_size_bytes",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field: dem_error_message (mirrors error_message)
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"autogeneratePattern": "",
			"help": "",
			"hidden": false,
			"id": "text2064478213",
			"max": 0,
			"min": 0,
			"name": "dem_error_message",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_581507754")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("dem_status")
		collection.Fields.RemoveByName("dem_size_bytes")
		collection.Fields.RemoveByName("dem_error_message")

		return app.Save(collection)
	})
}
