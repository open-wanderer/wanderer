package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "author.user = @request.auth.id || public = true || (@request.auth.id != \"\" && trail_share_via_trail.actor.user ?= @request.auth.id) || trail_link_share_via_trail.token = @request.query.share",
			"viewRule": "author.user = @request.auth.id || public = true || (@request.auth.id != \"\" && trail_share_via_trail.actor.user ?= @request.auth.id) || trail_link_share_via_trail.token = @request.query.share "
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("e864strfxo14pm4")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "author.user = @request.auth.id || public = true || (@request.auth.id != \"\" && trail_share_via_trail.actor.user ?= @request.auth.id)",
			"viewRule": "author.user = @request.auth.id || public = true || (@request.auth.id != \"\" && trail_share_via_trail.actor.user ?= @request.auth.id)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
