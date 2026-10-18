package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// 1776767721 and 1778583029 each overwrite the whole index list of
// activitypub_actors. 1776767721 landed later than its timestamp suggests, so
// an instance that had already applied 1778583029 ran it afterwards and lost
// the inbox index, while a fresh install runs them in order and loses the
// unique user index instead. Without the inbox index the delete-recipient
// queries join every actor against every activity, which turned a trail delete
// into a multi-hour request. Both indexes are restored here by name, leaving
// the rest of the list as each instance has it.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		collection.AddIndex("idx_x8xyfe8q8y", false, "`inbox`", "")
		collection.AddIndex("idx_activitypub_actors_user", true, "`user`", "user IS NOT NULL AND user != ''")

		return app.Save(collection)
	}, nil)
}
