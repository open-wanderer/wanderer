package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("summit_logs")
		if err != nil {
			return err
		}
		collection.Fields.Add(&core.BoolField{Name: "federation_suppressed", Hidden: true})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("summit_logs")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("federation_suppressed")
		return app.Save(collection)
	})
}
