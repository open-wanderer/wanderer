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

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"deleteRule": "@request.auth.id != \"\" && (trail.author.user = @request.auth.id || author = @request.auth.id)",
			"updateRule": "@request.auth.id != \"\" && (trail.author.user = @request.auth.id || author = @request.auth.id)"
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
			"deleteRule": "@request.auth.id != \"\" && ((@collection.trails.waypoints.id ?= id && @collection.trails.author.user = @request.auth.id) || author = @request.auth.id)",
			"updateRule": "@request.auth.id != \"\" && ((@collection.trails.waypoints.id ?= id && @collection.trails.author.user = @request.auth.id) || author = @request.auth.id)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
