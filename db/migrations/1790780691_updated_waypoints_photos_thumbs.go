package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("goeo2ubp103rzp9")
		if err != nil {
			return err
		}

		field, _ := collection.Fields.GetByName("photos").(*core.FileField)
		if field == nil {
			return fmt.Errorf("waypoints.photos file field not found")
		}

		field.Thumbs = []string{"600x0", "100x0"}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("goeo2ubp103rzp9")
		if err != nil {
			return err
		}

		field, _ := collection.Fields.GetByName("photos").(*core.FileField)
		if field == nil {
			return fmt.Errorf("waypoints.photos file field not found")
		}

		field.Thumbs = nil

		return app.Save(collection)
	})
}
