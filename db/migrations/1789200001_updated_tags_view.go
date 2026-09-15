package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Anonymous trail views and federation pulls need to expand tag names.
// Keep listing the entire tag catalog restricted to authenticated users.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1219621782")
		if err != nil {
			return err
		}
		collection.ViewRule = types.Pointer("")
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1219621782")
		if err != nil {
			return err
		}
		collection.ViewRule = types.Pointer(`@request.auth.id != ""`)
		return app.Save(collection)
	})
}
