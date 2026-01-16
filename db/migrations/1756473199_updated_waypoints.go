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
			"listRule": "author = @request.auth.id || trails_via_waypoints.author.user ?= @request.auth.id || trails_via_waypoints.public ?= true || \n(@collection.trail_share.trail.id ?= trails_via_waypoints.id && @collection.trail_share.actor.user ?= @request.auth.id)\n|| \n(@collection.trail_link_share.token !=  \"\" && @collection.trail_link_share.trail.waypoints.id ?= id)",
			"viewRule": "author = @request.auth.id || trails_via_waypoints.author.user ?= @request.auth.id || trails_via_waypoints.public ?= true || \n(@collection.trail_share.trail.id ?= trails_via_waypoints.id && @collection.trail_share.actor.user ?= @request.auth.id)\n|| \n(@collection.trail_link_share.token !=  \"\" && @collection.trail_link_share.trail.waypoints.id ?= id)"
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
			"listRule": "author = @request.auth.id || trails_via_waypoints.author.user ?= @request.auth.id || trails_via_waypoints.public ?= true || \n(@collection.trail_share.trail.id ?= trails_via_waypoints.id && @collection.trail_share.actor.user ?= @request.auth.id)\n|| \n(@collection.trail_link_share.trail.id ?= trails_via_waypoints.id && @collection.trail_link_share.token = @request.query.share)",
			"viewRule": "author = @request.auth.id || trails_via_waypoints.author.user ?= @request.auth.id || trails_via_waypoints.public ?= true || \n(@collection.trail_share.trail.id ?= trails_via_waypoints.id && @collection.trail_share.actor.user ?= @request.auth.id)\n|| \n(@collection.trail_link_share.trail.id ?= trails_via_waypoints.id && @collection.trail_link_share.token = @request.query.share)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
