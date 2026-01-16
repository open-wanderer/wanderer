package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_rpT7QJwWTm`+"`"+` ON `+"`"+`activitypub_actors`+"`"+` (`+"`"+`iri`+"`"+`)",
				"CREATE INDEX idx_actors_username_domain\nON activitypub_actors(preferred_username, domain);",
				"CREATE INDEX idx_activitypub_actors_user ON activitypub_actors(user);"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1295301207")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_rpT7QJwWTm`+"`"+` ON `+"`"+`activitypub_actors`+"`"+` (`+"`"+`iri`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
