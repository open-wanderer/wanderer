package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The UNION left PocketBase unable to infer the type of `actor`, so it became a
// JSON field and the list rule's actor.user.settings_via_user.privacy check
// never matched: anyone could list the feed of a private account. Selecting
// the actor id through a join makes `actor` a relation again.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("profile_feed")
		if err != nil {
			return err
		}

		collection.ViewQuery = `SELECT
    (ROW_NUMBER() OVER ()) as id,
    a.id as actor,
    feed.item,
    feed.type,
    feed.created
FROM
    (
        SELECT
            author as actor,
            id as item,
            'list' as type,
            created
        FROM
            lists
        WHERE lists.public = TRUE
        UNION
        SELECT
            author as actor,
            id as item,
            'trail' as type,
            created
        FROM trails
        WHERE trails.public = TRUE
    ) feed
    JOIN activitypub_actors a ON a.id = feed.actor
ORDER BY feed.created desc;`

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("profile_feed")
		if err != nil {
			return err
		}

		collection.ViewQuery = `SELECT
    (ROW_NUMBER() OVER ()) as id,
    actor,
    item,
    type,
    created
FROM
    (
        SELECT
            author as actor,
            id as item,
            'list' as type,
            created
        FROM
            lists
        WHERE lists.public = TRUE
        UNION
        SELECT
            author as actor,
            id as item,
            'trail' as type,
            created
        FROM trails
        WHERE trails.public = TRUE
    )
ORDER BY created desc;`

		return app.Save(collection)
	})
}
