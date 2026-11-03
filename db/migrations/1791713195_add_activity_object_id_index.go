package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Indexes the object id of recorded activities. Deleting a private trail or
// list looks up whether a Create or Update of it was ever sent, which without
// this index scans the whole table.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_activities")
		if err != nil {
			return err
		}

		if collection.GetIndex("idx_activitypub_activities_object_id") != "" {
			return nil
		}

		collection.AddIndex("idx_activitypub_activities_object_id", false, "json_extract(object, '$.id')", "")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_activities")
		if err != nil {
			return nil
		}

		collection.RemoveIndex("idx_activitypub_activities_object_id")

		return app.Save(collection)
	})
}
