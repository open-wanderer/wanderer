package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
		if err != nil {
			return err
		}

		// // remove field
		collection.Fields.RemoveById("in1traur")

		// // remove field
		collection.Fields.RemoveById("wxwomfd5")

		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
