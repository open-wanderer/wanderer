package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("lf06qip3f4d11yk")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "((@request.auth.id != \"\" && trail.author.user = @request.auth.id) || trail.public = true) || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id",
			"viewRule": "((@request.auth.id != \"\" && trail.author.user = @request.auth.id) || trail.public = true) || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("lf06qip3f4d11yk")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "((@request.auth.id != \"\" && trail.author.user = @request.auth.id) || trail.public = true) || author = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id",
			"viewRule": "((@request.auth.id != \"\" && trail.author.user = @request.auth.id) || trail.public = true) || author = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
