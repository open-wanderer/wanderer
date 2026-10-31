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

		// Keep the earliest row of each (actor, item) pair (smallest created, then
		// smallest id) and delete the others in one window pass. Before the unique
		// index exists a per-row correlated lookup is quadratic in the table size.
		if _, err := app.DB().NewQuery(`
			DELETE FROM feed WHERE id IN (
				SELECT id FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY actor, item ORDER BY created, id) AS rn FROM feed
				) WHERE rn > 1
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
