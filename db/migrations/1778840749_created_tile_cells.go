package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"help": "",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"help": "",
					"hidden": false,
					"id": "text662905681",
					"max": 0,
					"min": 0,
					"name": "cell_key",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"help": "",
					"hidden": false,
					"id": "select1274211008",
					"maxSelect": 0,
					"name": "status",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": [
						"pending",
						"ready",
						"error"
					]
				},
				{
					"help": "",
					"hidden": false,
					"id": "number3828904802",
					"max": null,
					"min": null,
					"name": "min_lon",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"help": "",
					"hidden": false,
					"id": "number2279188374",
					"max": null,
					"min": null,
					"name": "min_lat",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"help": "",
					"hidden": false,
					"id": "number3888878381",
					"max": null,
					"min": null,
					"name": "max_lon",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"help": "",
					"hidden": false,
					"id": "number2217363417",
					"max": null,
					"min": null,
					"name": "max_lat",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"help": "",
					"hidden": false,
					"id": "number1615430137",
					"max": null,
					"min": null,
					"name": "size_bytes",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"help": "",
					"hidden": false,
					"id": "text737763667",
					"max": 0,
					"min": 0,
					"name": "error_message",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"help": "",
					"hidden": false,
					"id": "number2862860510",
					"max": null,
					"min": null,
					"name": "download_count",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_581507754",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_1tsnys1v69` + "`" + ` ON ` + "`" + `tile_cells` + "`" + ` (` + "`" + `cell_key` + "`" + `)"
			],
			"listRule": null,
			"name": "tile_cells",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_581507754")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
