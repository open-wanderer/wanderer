package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The previous query joined every actor against a derived table that
// aggregated all trails and shares, so each lookup of one user's box scanned
// the whole trails table. Correlating on the actor lets the author and share
// indexes do the work; the public box is computed once from a covering index.
const correlatedTrailBoundsViewQuery = `SELECT
    a.id, a.user,
    COALESCE((SELECT MAX(v) FROM (
        SELECT max_lat AS v FROM trails WHERE author = a.id
        UNION ALL
        SELECT t.max_lat FROM trail_share ts JOIN trails t ON t.id = ts.trail WHERE ts.actor = a.id
        UNION ALL
        SELECT p.max_lat
    )), 0) AS max_lat,
    COALESCE((SELECT MAX(v) FROM (
        SELECT max_lon AS v FROM trails WHERE author = a.id
        UNION ALL
        SELECT t.max_lon FROM trail_share ts JOIN trails t ON t.id = ts.trail WHERE ts.actor = a.id
        UNION ALL
        SELECT p.max_lon
    )), 0) AS max_lon,
    COALESCE((SELECT MIN(v) FROM (
        SELECT min_lat AS v FROM trails WHERE author = a.id
        UNION ALL
        SELECT t.min_lat FROM trail_share ts JOIN trails t ON t.id = ts.trail WHERE ts.actor = a.id
        UNION ALL
        SELECT p.min_lat
    )), 0) AS min_lat,
    COALESCE((SELECT MIN(v) FROM (
        SELECT min_lon AS v FROM trails WHERE author = a.id
        UNION ALL
        SELECT t.min_lon FROM trail_share ts JOIN trails t ON t.id = ts.trail WHERE ts.actor = a.id
        UNION ALL
        SELECT p.min_lon
    )), 0) AS min_lon
FROM activitypub_actors a
CROSS JOIN (
    SELECT
        MAX(max_lat) AS max_lat,
        MAX(max_lon) AS max_lon,
        MIN(min_lat) AS min_lat,
        MIN(min_lon) AS min_lon
    FROM trails
    WHERE public = TRUE
) p
WHERE a.user != ""`

func init() {
	m.Register(func(app core.App) error {
		trails, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		// SQLite does not match a partial index on "public = true" here, so
		// the index leads with public instead.
		trails.AddIndex("idx_trails_public_bounds", false, "`public`, `max_lat`, `max_lon`, `min_lat`, `min_lon`", "")
		if err := app.Save(trails); err != nil {
			return err
		}

		boundingBox, err := app.FindCollectionByNameOrId("trails_bounding_box")
		if err != nil {
			return err
		}
		boundingBox.ViewQuery = correlatedTrailBoundsViewQuery

		return app.Save(boundingBox)
	}, func(app core.App) error {
		boundingBox, err := app.FindCollectionByNameOrId("trails_bounding_box")
		if err != nil {
			return err
		}
		boundingBox.ViewQuery = trailBoundsViewQuery
		if err := app.Save(boundingBox); err != nil {
			return err
		}

		trails, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}
		trails.RemoveIndex("idx_trails_public_bounds")

		return app.Save(trails)
	})
}
