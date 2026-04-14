package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("goeo2ubp103rzp9")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"exceptDomains": null,
			"hidden": false,
			"id": "url2434853685",
			"name": "iri",
			"onlyDomains": null,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "url"
		}`)); err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_GgX6MsdCJq`+"`"+` ON `+"`"+`waypoints`+"`"+` (`+"`"+`iri`+"`"+`) WHERE iri IS NOT NULL AND iri != \"\";"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("goeo2ubp103rzp9")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": []
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("url2434853685")

		return app.Save(collection)
	})
}
