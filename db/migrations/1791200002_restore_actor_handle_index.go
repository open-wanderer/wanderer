package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Earlier versions of 1776767721 replaced the unique (preferred_username,
// domain) index with a plain one on instances that applied it after
// 1778583029. Duplicate handles can only exist on those instances, and they
// belong to remote actors that content references, so they are reported
// instead of blocking startup.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		var duplicates int
		err = app.DB().NewQuery(`
			SELECT COUNT(*) FROM (
				SELECT 1 FROM activitypub_actors
				GROUP BY preferred_username, domain
				HAVING COUNT(*) > 1
			)`).Row(&duplicates)
		if err != nil {
			return err
		}
		if duplicates > 0 {
			app.Logger().Warn(
				"activitypub_actors has duplicate (preferred_username, domain) pairs; the index on them stays non-unique",
				"pairs", duplicates,
			)
			return nil
		}

		collection.AddIndex("idx_actors_username_domain", true, "`preferred_username`, `domain`", "")

		return app.Save(collection)
	}, nil)
}
