package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const actorHandleLookupIndex = "idx_activitypub_actors_handle_lookup"

// The ActivityPub actor, followers, following and outbox endpoints find a local
// actor with `preferred_username:lower = ? && is_local = true`. LOWER() keeps
// SQLite off the (preferred_username, domain) index, so each request scanned
// the table until the first match: on an instance with ~800k cached actors a
// new account took six seconds per request, and its follower list, which
// fetches the collection and then its first page, about fourteen. Older local
// actors may still hold a mixed-case preferred_username, so the lookup keeps
// LOWER() and the index matches it.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		collection.RemoveIndex(actorHandleLookupIndex)
		collection.Indexes = append(collection.Indexes,
			"CREATE INDEX IF NOT EXISTS `"+actorHandleLookupIndex+"` ON `activitypub_actors` (LOWER(`preferred_username`))")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		collection.RemoveIndex(actorHandleLookupIndex)

		return app.Save(collection)
	})
}
