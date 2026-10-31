package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds an optional activity_iri text field to follows. Instance follows store
// the IRI of their Follow activity in it; user-level follows leave it empty.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
		if err != nil {
			return err
		}

		// Idempotent: nothing to do when the field already exists
		if collection.Fields.GetByName("activity_iri") != nil {
			return nil
		}
		collection.Fields.Add(&core.TextField{Name: "activity_iri"})

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
		if err != nil {
			return nil
		}

		collection.Fields.RemoveByName("activity_iri")

		return app.Save(collection)
	})
}
