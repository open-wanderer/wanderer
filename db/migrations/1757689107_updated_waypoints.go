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
			"listRule": "author = @request.auth.id || trail.author.user ?= @request.auth.id || trail.public ?= true || trail.trail_share_via_trail.actor.user ?= @request.auth.id\n|| \n(trail.trail_link_share_via_trail.token != \"\" && trail.trail_link_share_via_trail.token = @request.query.share)",
			"viewRule": "author = @request.auth.id || trail.author.user ?= @request.auth.id || trail.public ?= true || trail.trail_share_via_trail.actor.user ?= @request.auth.id\n|| \n(trail.trail_link_share_via_trail.token != \"\" && trail.trail_link_share_via_trail.token = @request.query.share)"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"cascadeDelete": true,
			"collectionId": "e864strfxo14pm4",
			"hidden": false,
			"id": "relation2993194383",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "trail",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
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
			"listRule": "author = @request.auth.id || trails_via_waypoints.author.user ?= @request.auth.id || trails_via_waypoints.public ?= true || \n(@collection.trail_share.trail.id ?= trails_via_waypoints.id && @collection.trail_share.actor.user ?= @request.auth.id)\n|| \n(@collection.trail_link_share.token !=  \"\" && @collection.trail_link_share.trail.waypoints.id ?= id)",
			"viewRule": "author = @request.auth.id || trails_via_waypoints.author.user ?= @request.auth.id || trails_via_waypoints.public ?= true || \n(@collection.trail_share.trail.id ?= trails_via_waypoints.id && @collection.trail_share.actor.user ?= @request.auth.id)\n|| \n(@collection.trail_link_share.token !=  \"\" && @collection.trail_link_share.trail.waypoints.id ?= id)"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation2993194383")

		return app.Save(collection)
	})
}
