package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("lf06qip3f4d11yk")
		if err != nil {
			return err
		}

		// update field
		//
		// comments.author was the only required relation to activitypub_actors
		// without cascadeDelete. Because a required reference that would be left
		// empty aborts the delete, any account that had ever left a comment could
		// not be deleted at all: the cascade reached comments, errored, and rolled
		// the whole transaction back, leaving even the login record in place.
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"cascadeDelete": true,
			"collectionId": "pbc_1295301207",
			"hidden": false,
			"id": "relation3182418120",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "author",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("lf06qip3f4d11yk")
		if err != nil {
			return err
		}

		// revert field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_1295301207",
			"hidden": false,
			"id": "relation3182418120",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "author",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
