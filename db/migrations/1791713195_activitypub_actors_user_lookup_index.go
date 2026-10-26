package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const actorUserLookupIndex = "idx_activitypub_actors_user_lookup"

// The unique user index is partial on a non-empty user, which SQLite cannot
// infer from `user = ?`, so every lookup of a local actor by user scanned the
// whole table. The scan stops at the first match, so it only showed for accounts
// whose actor sits behind the remote actor cache: a new account on an instance
// with ~800k actors took six seconds per request. IF NOT EXISTS lets the
// migration run on instances where the index was created by hand.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		collection.RemoveIndex(actorUserLookupIndex)
		collection.Indexes = append(collection.Indexes,
			"CREATE INDEX IF NOT EXISTS `"+actorUserLookupIndex+"` ON `activitypub_actors` (`user`)")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		collection.RemoveIndex(actorUserLookupIndex)

		return app.Save(collection)
	})
}
