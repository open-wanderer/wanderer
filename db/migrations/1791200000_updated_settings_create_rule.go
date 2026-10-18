package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Settings records are created by the user create hook. The public create rule
// let anyone add a record for any user, which then showed up as that user's
// bio and privacy setting.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer(`@request.auth.id != "" && @request.body.user = @request.auth.id`)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("")

		return app.Save(collection)
	})
}
