package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Removes duplicate feed rows and adds a unique index on feed(actor, type,
// item). Record ids are unique only within a collection, so the type is part
// of the key: a trail and a list with the same id are different items.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_72164123")
		if err != nil {
			return err
		}

		// Replaces the (actor, item) index of databases that ran a
		// development build of instance federation.
		collection.RemoveIndex("idx_feed_actor_item")
		if collection.GetIndex("idx_feed_actor_type_item") != "" {
			return app.Save(collection)
		}

		// Keep the earliest row of each (actor, type, item) and delete the
		// others in one window pass. Before the unique index exists a per-row
		// correlated lookup is quadratic in the table size.
		if _, err := app.DB().NewQuery(`
			DELETE FROM feed WHERE id IN (
				SELECT id FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY actor, type, item ORDER BY created, id) AS rn FROM feed
				) WHERE rn > 1
			)
		`).Execute(); err != nil {
			return err
		}

		collection.AddIndex("idx_feed_actor_type_item", true, "`actor`, `type`, `item`", "")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_72164123")
		if err != nil {
			return nil
		}

		collection.RemoveIndex("idx_feed_actor_type_item")

		return app.Save(collection)
	})
}
