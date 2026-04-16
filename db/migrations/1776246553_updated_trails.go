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

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "bool989355118",
			"name": "completed",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		err = app.Save(collection)
		if err != nil {
			return err
		}

		logs, err := app.FindAllRecords("summit_logs")
		if err != nil {
			return err
		}

		for _, l := range logs {
			trail, err := app.FindRecordById("trails", l.GetString("trail"))
			if err != nil {
				continue
			}
			trail.Set("completed", true)
			err = app.Save(trail)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("bool989355118")

		return app.Save(collection)
	})
}
