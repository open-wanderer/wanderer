package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Removes duplicate (actor, item) feed rows, keeping the earliest, and adds a
// unique index on feed(actor, item).
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_72164123")
		if err != nil {
			return err
		}

		// Idempotent: nothing to do when the index already exists
		if collection.GetIndex("idx_feed_actor_item") != "" {
			return nil
		}

		// Delete every row that has an earlier row (smaller created, or same created and
		// smaller id) for the same actor and item.
		if _, err := app.DB().NewQuery(`
			DELETE FROM feed
			WHERE EXISTS (
				SELECT 1 FROM feed AS other
				WHERE other.actor = feed.actor
				  AND other.item = feed.item
				  AND (
					other.created < feed.created
					OR (other.created = feed.created AND other.id < feed.id)
				  )
			)
		`).Execute(); err != nil {
			return err
		}

		collection.AddIndex("idx_feed_actor_item", true, "`actor`, `item`", "")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_72164123")
		if err != nil {
			return nil
		}

		collection.RemoveIndex("idx_feed_actor_item")

		return app.Save(collection)
	})
}
