package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(removePluginInstanceState1786900100, restorePluginInstanceState1786900100)
}

func removePluginInstanceState1786900100(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("plugin_instances")
	if err != nil {
		return err
	}

	collection.Fields.RemoveByName("state")
	return app.Save(collection)
}

func restorePluginInstanceState1786900100(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("plugin_instances")
	if err != nil {
		return err
	}

	if collection.Fields.GetByName("state") == nil {
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
				"hidden": false,
				"id": "json430001007",
				"maxSize": 2000000,
				"name": "state",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}`)); err != nil {
			return err
		}
	}

	return app.Save(collection)
}
