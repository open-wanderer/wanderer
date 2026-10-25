package migrations

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Makes the user index unique so a local user can only have one actor. Only
// this index is touched: the migration may run after newer ones on upgraded
// instances, so the rest of the index list is left as each instance has it.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		if err := checkNoDuplicateActorUsers(app); err != nil {
			return err
		}

		collection.AddIndex("idx_activitypub_actors_user", true, "`user`", "user IS NOT NULL AND user != ''")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		collection.AddIndex("idx_activitypub_actors_user", false, "`user`", "user IS NOT NULL")

		return app.Save(collection)
	})
}

// checkNoDuplicateActorUsers fails with the affected users instead of
// SQLite's bare constraint error. Actors are referenced by content, so they
// cannot be merged automatically.
func checkNoDuplicateActorUsers(app core.App) error {
	var users []struct {
		User   string `db:"user"`
		Actors string `db:"actors"`
	}
	err := app.DB().NewQuery(`
		SELECT user, GROUP_CONCAT(id, ', ') AS actors
		FROM activitypub_actors
		WHERE user IS NOT NULL AND user != ''
		GROUP BY user
		HAVING COUNT(*) > 1`).All(&users)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}

	lines := make([]string, len(users))
	for i, u := range users {
		lines[i] = fmt.Sprintf("user %s: actors %s", u.User, u.Actors)
	}
	return fmt.Errorf(
		"activitypub_actors has more than one actor for these users; keep one actor per user and restart:\n%s",
		strings.Join(lines, "\n"),
	)
}
