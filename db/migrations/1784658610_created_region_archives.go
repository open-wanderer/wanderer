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
					"id": "text2091729501",
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
					"id": "text3477051923",
					"max": 0,
					"min": 0,
					"name": "region_id",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"help": "",
					"hidden": false,
					"id": "text3477051924",
					"max": 0,
					"min": 0,
					"name": "name",
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
					"id": "number3828904901",
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
					"id": "number2279188401",
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
					"id": "number3888878401",
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
					"id": "number2217363401",
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
					"id": "select1274211101",
					"maxSelect": 1,
					"name": "status",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": [
						"building",
						"ready",
						"error"
					]
				},
				{
					"autogeneratePattern": "",
					"help": "",
					"hidden": false,
					"id": "text4192883101",
					"max": 0,
					"min": 0,
					"name": "vector_built_date",
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
					"id": "number1615430201",
					"max": null,
					"min": null,
					"name": "vector_size_bytes",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"help": "",
					"hidden": false,
					"id": "select4192883101",
					"maxSelect": 1,
					"name": "dem_status",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"building",
						"ready",
						"error"
					]
				},
				{
					"help": "",
					"hidden": false,
					"id": "number3611729501",
					"max": null,
					"min": null,
					"name": "dem_size_bytes",
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
					"id": "text737763701",
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
					"autogeneratePattern": "",
					"help": "",
					"hidden": false,
					"id": "text2064478301",
					"max": 0,
					"min": 0,
					"name": "dem_error_message",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "autodate2990389201",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085501",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_1784658610",
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_region_archives_region_id` + "`" + ` ON ` + "`" + `region_archives` + "`" + ` (` + "`" + `region_id` + "`" + `)"
			],
			"listRule": null,
			"name": "region_archives",
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
		collection, err := app.FindCollectionByNameOrId("region_archives")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
