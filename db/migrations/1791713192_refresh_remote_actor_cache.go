package migrations

import (
	"fmt"

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
		collection.Fields.Add(&core.TextField{
			Name:   "legacy_public_key",
			Hidden: true,
		})
		if err := app.Save(collection); err != nil {
			return err
		}

		const batchSize = 5000
		var total int64
		var batches int

		for {
			result, err := app.DB().NewQuery(fmt.Sprintf(`
				UPDATE activitypub_actors
				SET
					legacy_public_key = public_key,
					public_key = '',
					last_fetched = ''
				WHERE rowid IN (
					SELECT rowid
					FROM activitypub_actors
					WHERE is_local = 0
					  AND (public_key != '' OR last_fetched != '')
					ORDER BY rowid
					LIMIT %d
				)
			`, batchSize)).Execute()
			if err != nil {
				return err
			}

			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				break
			}

			total += affected
			batches++

			if batches%20 == 0 {
				fmt.Printf(
					"[1791713192] Elaborati %d attori remoti in %d blocchi\n",
					total,
					batches,
				)
			}
		}

		fmt.Printf(
			"[1791713192] Completato: %d attori aggiornati in %d blocchi\n",
			total,
			batches,
		)

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("legacy_public_key")
		return app.Save(collection)
	})
}
