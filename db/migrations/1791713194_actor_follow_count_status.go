package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}
		// Existing caches retain their freshness. Later actor fetches mark
		// collection reads not yet attempted in the cache window as pending.
		collection.Fields.Add(
			&core.BoolField{Name: "follower_count_pending", Hidden: true},
			&core.BoolField{Name: "following_count_pending", Hidden: true},
		)
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("follower_count_pending")
		collection.Fields.RemoveByName("following_count_pending")
		return app.Save(collection)
	})
}
