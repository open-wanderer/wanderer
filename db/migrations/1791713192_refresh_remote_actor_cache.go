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
		// Existing remote keys are refreshed lazily using the current actor-fetch
		// contract. Until then the previous key only authenticates a self-deletion
		// the actor's origin confirms. Local key pairs and the cached
		// identity/display fields remain.
		collection.Fields.Add(&core.TextField{Name: "legacy_public_key", Hidden: true})
		if err := app.Save(collection); err != nil {
			return err
		}
		_, err = app.DB().NewQuery("UPDATE activitypub_actors SET legacy_public_key=public_key, public_key='', last_fetched='' WHERE is_local=0").Execute()
		return err
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("legacy_public_key")
		return app.Save(collection)
	})
}
