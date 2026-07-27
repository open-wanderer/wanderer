package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("4wbv9tz5zjdrjh1")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "WITH local_actors AS (\n    SELECT id FROM activitypub_actors WHERE user != \"\"\n),\npub AS (\n    SELECT\n        MAX(distance) AS max_distance,       MIN(distance) AS min_distance,\n        MAX(elevation_gain) AS max_elevation_gain, MIN(elevation_gain) AS min_elevation_gain,\n        MAX(elevation_loss) AS max_elevation_loss, MIN(elevation_loss) AS min_elevation_loss,\n        MAX(duration) AS max_duration,       MIN(duration) AS min_duration\n    FROM trails\n    WHERE public = TRUE\n)\nSELECT\n    a.id,\n    a.user,\n    COALESCE(printf(\"%.2f\", MAX(t.max_distance)), 0) AS max_distance,\n    COALESCE(printf(\"%.2f\", MAX(t.max_elevation_gain)), 0) AS max_elevation_gain,\n    COALESCE(printf(\"%.2f\", MAX(t.max_elevation_loss)), 0) AS max_elevation_loss,\n    COALESCE(printf(\"%.2f\", MAX(t.max_duration)), 0) AS max_duration,\n    COALESCE(printf(\"%.2f\", MIN(t.min_distance)), 0) AS min_distance,\n    COALESCE(printf(\"%.2f\", MIN(t.min_elevation_gain)), 0) AS min_elevation_gain,\n    COALESCE(printf(\"%.2f\", MIN(t.min_elevation_loss)), 0) AS min_elevation_loss,\n    COALESCE(printf(\"%.2f\", MIN(t.min_duration)), 0) AS min_duration\nFROM activitypub_actors a\nLEFT JOIN (\n    SELECT author AS actor_id,\n        MAX(distance) AS max_distance, MIN(distance) AS min_distance,\n        MAX(elevation_gain) AS max_elevation_gain, MIN(elevation_gain) AS min_elevation_gain,\n        MAX(elevation_loss) AS max_elevation_loss, MIN(elevation_loss) AS min_elevation_loss,\n        MAX(duration) AS max_duration, MIN(duration) AS min_duration\n    FROM trails\n    WHERE author IN (SELECT id FROM local_actors)\n    GROUP BY author\n    UNION ALL\n    SELECT ts.actor AS actor_id,\n        MAX(t.distance), MIN(t.distance),\n        MAX(t.elevation_gain), MIN(t.elevation_gain),\n        MAX(t.elevation_loss), MIN(t.elevation_loss),\n        MAX(t.duration), MIN(t.duration)\n    FROM trail_share ts\n    JOIN trails t ON t.id = ts.trail\n    WHERE ts.actor IN (SELECT id FROM local_actors)\n    GROUP BY ts.actor\n    UNION ALL\n    SELECT a2.id AS actor_id,\n        p.max_distance, p.min_distance,\n        p.max_elevation_gain, p.min_elevation_gain,\n        p.max_elevation_loss, p.min_elevation_loss,\n        p.max_duration, p.min_duration\n    FROM local_actors a2\n    CROSS JOIN pub p\n) t ON t.actor_id = a.id\nWHERE a.user != \"\"\nGROUP BY a.id;\n"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_2ht4")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": true,
			"collectionId": "_pb_users_auth_",
			"help": "",
			"hidden": false,
			"id": "_clone_eNl9",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "user",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("4wbv9tz5zjdrjh1")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT\n    a.id,\n    a.user,\n    COALESCE(printf(\"%.2f\", MAX(t.max_distance)), 0) AS max_distance,\n    COALESCE(printf(\"%.2f\", MAX(t.max_elevation_gain)), 0) AS max_elevation_gain,\n    COALESCE(printf(\"%.2f\", MAX(t.max_elevation_loss)), 0) AS max_elevation_loss,\n    COALESCE(printf(\"%.2f\", MAX(t.max_duration)), 0) AS max_duration,\n    COALESCE(printf(\"%.2f\", MIN(t.min_distance)), 0) AS min_distance,\n    COALESCE(printf(\"%.2f\", MIN(t.min_elevation_gain)), 0) AS min_elevation_gain,\n    COALESCE(printf(\"%.2f\", MIN(t.min_elevation_loss)), 0) AS min_elevation_loss,\n    COALESCE(printf(\"%.2f\", MIN(t.min_duration)), 0) AS min_duration\nFROM activitypub_actors a\nLEFT JOIN (\n    SELECT author AS actor_id,\n        MAX(distance) AS max_distance,\n        MIN(distance) AS min_distance,\n        MAX(elevation_gain) AS max_elevation_gain,\n        MIN(elevation_gain) AS min_elevation_gain,\n        MAX(elevation_loss) AS max_elevation_loss,\n        MIN(elevation_loss) AS min_elevation_loss,\n        MAX(duration) AS max_duration,\n        MIN(duration) AS min_duration\n    FROM trails\n    GROUP BY author\n    UNION ALL\n    SELECT ts.actor AS actor_id,\n        MAX(t.distance) AS max_distance,\n        MIN(t.distance) AS min_distance,\n        MAX(t.elevation_gain) AS max_elevation_gain,\n        MIN(t.elevation_gain) AS min_elevation_gain,\n        MAX(t.elevation_loss) AS max_elevation_loss,\n        MIN(t.elevation_loss) AS min_elevation_loss,\n        MAX(t.duration) AS max_duration,\n        MIN(t.duration) AS min_duration\n    FROM trail_share ts\n    JOIN trails t ON t.id = ts.trail\n    GROUP BY ts.actor\n    UNION ALL\n    SELECT a2.id AS actor_id,\n        p.max_distance,\n        p.min_distance,\n        p.max_elevation_gain,\n        p.min_elevation_gain,\n        p.max_elevation_loss,\n        p.min_elevation_loss,\n        p.max_duration,\n        p.min_duration\n    FROM activitypub_actors a2\n    CROSS JOIN (\n        SELECT\n            MAX(distance) AS max_distance,\n            MIN(distance) AS min_distance,\n            MAX(elevation_gain) AS max_elevation_gain,\n            MIN(elevation_gain) AS min_elevation_gain,\n            MAX(elevation_loss) AS max_elevation_loss,\n            MIN(elevation_loss) AS min_elevation_loss,\n            MAX(duration) AS max_duration,\n            MIN(duration) AS min_duration\n        FROM trails\n        WHERE public = TRUE\n    ) p\n) t ON t.actor_id = a.id\nWHERE a.user != \"\"\nGROUP BY a.id;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": true,
			"collectionId": "_pb_users_auth_",
			"help": "",
			"hidden": false,
			"id": "_clone_2ht4",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "user",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_eNl9")

		return app.Save(collection)
	})
}
