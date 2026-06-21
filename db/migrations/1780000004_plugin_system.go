package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return createInstalledPluginsCollection(app)
	}, func(app core.App) error {
		if collection, err := app.FindCollectionByNameOrId("installed_plugins"); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}

func createInstalledPluginsCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("installed_plugins"); err == nil {
		return nil
	}

	jsonData := `{
		"createRule": null,
		"deleteRule": null,
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "textplginsid01",
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
				"hidden": false,
				"id": "textplginpid1",
				"max": 128,
				"min": 1,
				"name": "plugin_id",
				"pattern": "^[a-z0-9][a-z0-9_-]*$",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "textplginname",
				"max": 256,
				"min": 1,
				"name": "name",
				"pattern": "",
				"presentable": true,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "selectplgtype",
				"maxSelect": 1,
				"name": "type",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "select",
				"values": ["trails"]
			},
			{
				"hidden": false,
				"id": "textplginvers",
				"max": 64,
				"min": 1,
				"name": "version",
				"pattern": "",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "textplginrunt",
				"max": 32,
				"min": 1,
				"name": "runtime",
				"pattern": "",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "textplginpath",
				"max": 0,
				"min": 0,
				"name": "path",
				"pattern": "",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "jsonplginman",
				"maxSize": 2000000,
				"name": "manifest",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "jsonplgincfg",
				"maxSize": 2000000,
				"name": "config",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "selectplginst",
				"maxSelect": 1,
				"name": "status",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "select",
				"values": ["available", "disabled", "error"]
			},
			{
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
			},
			{
				"hidden": false,
				"id": "autoplgcreate",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autoplgupdate",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"id": "pbc_430002000",
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`" + `idx_installed_plugins_plugin_id` + "`" + ` ON ` + "`" + `installed_plugins` + "`" + ` (` + "`" + `plugin_id` + "`" + `)"
		],
		"listRule": null,
		"name": "installed_plugins",
		"system": false,
		"type": "base",
		"updateRule": null,
		"viewRule": null
	}`

	collection := &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
		return err
	}
	return app.Save(collection)
}
