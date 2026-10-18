package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Earlier versions of 1776767721 overwrote the whole index list of
// activitypub_actors. Instances that applied it after 1778583029 lost the inbox
// index and the uniqueness of (preferred_username, domain), see 1791200002;
// fresh installs ran them in order and lost the unique user index instead.
// Without the inbox index the delete-recipient queries join every actor
// against every activity, which turned a trail delete into a multi-hour
// request. The indexes are restored here by name, leaving the rest of the list
// as each instance has it.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}

		if err := checkNoDuplicateActorUsers(app); err != nil {
			return err
		}

		collection.AddIndex("idx_x8xyfe8q8y", false, "`inbox`", "")
		collection.AddIndex("idx_activitypub_actors_user", true, "`user`", "user IS NOT NULL AND user != ''")

		return app.Save(collection)
	}, nil)
}
