package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("urytyc428mwlbqq")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n    a.id, a.user, \n    COALESCE(MAX(t.max_lat), 0) AS max_lat, \n    COALESCE(MAX(t.max_lon), 0) AS max_lon, \n    COALESCE(MIN(t.min_lat), 0) AS min_lat, \n    COALESCE(MIN(t.min_lon), 0) AS min_lon \nFROM activitypub_actors a \nLEFT JOIN ( \n    SELECT author AS actor_id, \n        MAX(lat) AS max_lat, \n        MAX(lon) AS max_lon, \n        MIN(lat) AS min_lat, \n        MIN(lon) AS min_lon \n    FROM trails \n    GROUP BY author \n    UNION ALL \n    SELECT ts.actor AS actor_id, \n        MAX(t.lat) AS max_lat, \n        MAX(t.lon) AS max_lon, \n        MIN(t.lat) AS min_lat, \n        MIN(t.lon) AS min_lon \n    FROM trail_share ts \n    JOIN trails t ON t.id = ts.trail \n    GROUP BY ts.actor \n    UNION ALL \n    SELECT a2.id AS actor_id, \n        p.max_lat, p.max_lon, p.min_lat, p.min_lon \n    FROM activitypub_actors a2 \n    CROSS JOIN ( \n        SELECT \n            MAX(lat) AS max_lat, \n            MAX(lon) AS max_lon, \n            MIN(lat) AS min_lat, \n            MIN(lon) AS min_lon \n        FROM trails \n        WHERE public = TRUE \n    ) p \n) t ON t.actor_id = a.id \nGROUP BY a.id;"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("urytyc428mwlbqq")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n    activitypub_actors.id, activitypub_actors.user, \n    COALESCE(MAX(trails.lat), 0) AS max_lat, \n    COALESCE(MAX(trails.lon), 0) AS max_lon, \n    COALESCE(MIN(trails.lat), 0) AS min_lat, \n    COALESCE(MIN(trails.lon), 0) AS min_lon \nFROM activitypub_actors \nLEFT JOIN trails \n    ON activitypub_actors.id = trails.author \n    OR trails.public = TRUE \n    OR EXISTS (\n        SELECT 1 \n        FROM trail_share \n        WHERE trail_share.trail = trails.id \n        AND trail_share.actor = activitypub_actors.id\n    ) \nGROUP BY activitypub_actors.id;"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
