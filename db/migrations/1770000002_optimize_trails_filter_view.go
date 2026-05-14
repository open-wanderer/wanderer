package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trails_filter")
		if err != nil {
			return err
		}

		viewQuery := `SELECT
    a.id,
    a.user,
    COALESCE(printf("%.2f", MAX(t.max_distance)), 0) AS max_distance,
    COALESCE(printf("%.2f", MAX(t.max_elevation_gain)), 0) AS max_elevation_gain,
    COALESCE(printf("%.2f", MAX(t.max_elevation_loss)), 0) AS max_elevation_loss,
    COALESCE(printf("%.2f", MAX(t.max_duration)), 0) AS max_duration,
    COALESCE(printf("%.2f", MIN(t.min_distance)), 0) AS min_distance,
    COALESCE(printf("%.2f", MIN(t.min_elevation_gain)), 0) AS min_elevation_gain,
    COALESCE(printf("%.2f", MIN(t.min_elevation_loss)), 0) AS min_elevation_loss,
    COALESCE(printf("%.2f", MIN(t.min_duration)), 0) AS min_duration
FROM activitypub_actors a
LEFT JOIN (
    SELECT author AS actor_id,
        MAX(distance) AS max_distance,
        MIN(distance) AS min_distance,
        MAX(elevation_gain) AS max_elevation_gain,
        MIN(elevation_gain) AS min_elevation_gain,
        MAX(elevation_loss) AS max_elevation_loss,
        MIN(elevation_loss) AS min_elevation_loss,
        MAX(duration) AS max_duration,
        MIN(duration) AS min_duration
    FROM trails
    GROUP BY author
    UNION ALL
    SELECT ts.actor AS actor_id,
        MAX(t.distance) AS max_distance,
        MIN(t.distance) AS min_distance,
        MAX(t.elevation_gain) AS max_elevation_gain,
        MIN(t.elevation_gain) AS min_elevation_gain,
        MAX(t.elevation_loss) AS max_elevation_loss,
        MIN(t.elevation_loss) AS min_elevation_loss,
        MAX(t.duration) AS max_duration,
        MIN(t.duration) AS min_duration
    FROM trail_share ts
    JOIN trails t ON t.id = ts.trail
    GROUP BY ts.actor
    UNION ALL
    SELECT a2.id AS actor_id,
        p.max_distance,
        p.min_distance,
        p.max_elevation_gain,
        p.min_elevation_gain,
        p.max_elevation_loss,
        p.min_elevation_loss,
        p.max_duration,
        p.min_duration
    FROM activitypub_actors a2
    CROSS JOIN (
        SELECT
            MAX(distance) AS max_distance,
            MIN(distance) AS min_distance,
            MAX(elevation_gain) AS max_elevation_gain,
            MIN(elevation_gain) AS min_elevation_gain,
            MAX(elevation_loss) AS max_elevation_loss,
            MIN(elevation_loss) AS min_elevation_loss,
            MAX(duration) AS max_duration,
            MIN(duration) AS min_duration
        FROM trails
        WHERE public = TRUE
    ) p
) t ON t.actor_id = a.id
GROUP BY a.id;`

		if err := json.Unmarshal([]byte(`{"viewQuery": `+jsonMarshal(viewQuery)+`}`), &collection); err != nil {
			return err
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trails_filter")
		if err != nil {
			return err
		}

		viewQuery := `SELECT activitypub_actors.id, activitypub_actors.user, COALESCE(printf("%.2f", MAX(trails.distance)), 0) AS max_distance,
  COALESCE(printf("%.2f", MAX(trails.elevation_gain)), 0) AS max_elevation_gain, 
  COALESCE(printf("%.2f", MAX(trails.elevation_loss)), 0) AS max_elevation_loss, 
  COALESCE(printf("%.2f", MAX(trails.duration)), 0) AS max_duration, 
  COALESCE(printf("%.2f", MIN(trails.distance)), 0) AS min_distance,   
  COALESCE(printf("%.2f", MIN(trails.elevation_gain)), 0) AS min_elevation_gain, 
  COALESCE(printf("%.2f", MIN(trails.elevation_loss)), 0) AS min_elevation_loss, 
  COALESCE(printf("%.2f", MIN(trails.duration)), 0) AS min_duration 
FROM activitypub_actors 
  LEFT JOIN trails ON 
  activitypub_actors.id = trails.author OR 
  trails.public = 1 OR 
  EXISTS (
    SELECT 1 
    FROM trail_share 
    WHERE trail_share.trail = trails.id 
    AND trail_share.actor = activitypub_actors.id
  ) GROUP BY activitypub_actors.id;`

		if err := json.Unmarshal([]byte(`{"viewQuery": `+jsonMarshal(viewQuery)+`}`), &collection); err != nil {
			return err
		}
		return app.Save(collection)
	})
}

func jsonMarshal(value string) string {
	b, _ := json.Marshal(value)
	return string(b)
}
