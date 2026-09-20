package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// Missing preferences retain the own-trails-only upload duplicate check.
		// Keep this separate from behavior, which map settings replace as a whole.
		collection.Fields.Add(&core.JSONField{Name: "uploadDuplicateCheck"})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		collection.Fields.RemoveByName("uploadDuplicateCheck")
		return app.Save(collection)
	})
}
