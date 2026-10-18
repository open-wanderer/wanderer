package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Each user has exactly one settings record. Older instances can hold extra
// rows because the collection's create rule used to be public, so those are
// removed before the unique index is created. The oldest row is kept: it is the
// one created at sign-up and the one unsorted lookups have been returning.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("uavt73rsqcn1n13")
		if err != nil {
			return err
		}

		result, err := app.DB().NewQuery(`
			DELETE FROM settings WHERE id IN (
				SELECT id FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY user ORDER BY created, id) AS position
					FROM settings
					WHERE user IS NOT NULL AND user != ''
				)
				WHERE position > 1
			)`).Execute()
		if err != nil {
			return err
		}
		if removed, err := result.RowsAffected(); err == nil && removed > 0 {
			app.Logger().Info("removed duplicate settings records", "count", removed)
		}

		collection.AddIndex("idx_rhdd21t4cx", true, "`user`", "")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("uavt73rsqcn1n13")
		if err != nil {
			return err
		}

		collection.RemoveIndex("idx_rhdd21t4cx")

		return app.Save(collection)
	})
}
