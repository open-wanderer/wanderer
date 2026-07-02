package migrations

import (
	"encoding/json"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const trailBoundsViewQuery = `SELECT 
    a.id, a.user, 
    COALESCE(MAX(t.max_lat), 0) AS max_lat, 
    COALESCE(MAX(t.max_lon), 0) AS max_lon, 
    COALESCE(MIN(t.min_lat), 0) AS min_lat, 
    COALESCE(MIN(t.min_lon), 0) AS min_lon 
FROM activitypub_actors a 
LEFT JOIN ( 
    SELECT author AS actor_id, 
        MAX(max_lat) AS max_lat, 
        MAX(max_lon) AS max_lon, 
        MIN(min_lat) AS min_lat, 
        MIN(min_lon) AS min_lon 
    FROM trails 
    GROUP BY author 
    UNION ALL 
    SELECT ts.actor AS actor_id, 
        MAX(t.max_lat) AS max_lat, 
        MAX(t.max_lon) AS max_lon, 
        MIN(t.min_lat) AS min_lat, 
        MIN(t.min_lon) AS min_lon 
    FROM trail_share ts 
    JOIN trails t ON t.id = ts.trail 
    GROUP BY ts.actor 
    UNION ALL 
    SELECT a2.id AS actor_id, 
        p.max_lat, p.max_lon, p.min_lat, p.min_lon 
    FROM activitypub_actors a2 
    CROSS JOIN ( 
        SELECT 
            MAX(max_lat) AS max_lat, 
            MAX(max_lon) AS max_lon, 
            MIN(min_lat) AS min_lat, 
            MIN(min_lon) AS min_lon 
        FROM trails 
        WHERE public = TRUE 
    ) p 
) t ON t.actor_id = a.id 
WHERE a.user != ""
GROUP BY a.id;`

const trailStartPointBoundsViewQuery = `SELECT 
    a.id, a.user, 
    COALESCE(MAX(t.max_lat), 0) AS max_lat, 
    COALESCE(MAX(t.max_lon), 0) AS max_lon, 
    COALESCE(MIN(t.min_lat), 0) AS min_lat, 
    COALESCE(MIN(t.min_lon), 0) AS min_lon 
FROM activitypub_actors a 
LEFT JOIN ( 
    SELECT author AS actor_id, 
        MAX(lat) AS max_lat, 
        MAX(lon) AS max_lon, 
        MIN(lat) AS min_lat, 
        MIN(lon) AS min_lon 
    FROM trails 
    GROUP BY author 
    UNION ALL 
    SELECT ts.actor AS actor_id, 
        MAX(t.lat) AS max_lat, 
        MAX(t.lon) AS max_lon, 
        MIN(t.lat) AS min_lat, 
        MIN(t.lon) AS min_lon 
    FROM trail_share ts 
    JOIN trails t ON t.id = ts.trail 
    GROUP BY ts.actor 
    UNION ALL 
    SELECT a2.id AS actor_id, 
        p.max_lat, p.max_lon, p.min_lat, p.min_lon 
    FROM activitypub_actors a2 
    CROSS JOIN ( 
        SELECT 
            MAX(lat) AS max_lat, 
            MAX(lon) AS max_lon, 
            MIN(lat) AS min_lat, 
            MIN(lon) AS min_lon 
        FROM trails 
        WHERE public = TRUE 
    ) p 
) t ON t.actor_id = a.id 
WHERE a.user != ""
GROUP BY a.id;`

func init() {
	m.Register(func(app core.App) error {
		trailsCollection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		addTrailBoundsField(trailsCollection, "min_lat")
		addTrailBoundsField(trailsCollection, "max_lat")
		addTrailBoundsField(trailsCollection, "min_lon")
		addTrailBoundsField(trailsCollection, "max_lon")
		addTrailBoundsField(trailsCollection, "bounding_box_diagonal")

		if err := app.Save(trailsCollection); err != nil {
			return err
		}

		if err := backfillTrailBounds(app); err != nil {
			return err
		}

		boundingBoxCollection, err := app.FindCollectionByNameOrId("trails_bounding_box")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`{"viewQuery":`+strconvQuote(trailBoundsViewQuery)+`}`), &boundingBoxCollection); err != nil {
			return err
		}

		if err := app.Save(boundingBoxCollection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		boundingBoxCollection, err := app.FindCollectionByNameOrId("trails_bounding_box")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`{"viewQuery":`+strconvQuote(trailStartPointBoundsViewQuery)+`}`), &boundingBoxCollection); err != nil {
			return err
		}

		if err := app.Save(boundingBoxCollection); err != nil {
			return err
		}

		trailsCollection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		trailsCollection.Fields.RemoveByName("min_lat")
		trailsCollection.Fields.RemoveByName("max_lat")
		trailsCollection.Fields.RemoveByName("min_lon")
		trailsCollection.Fields.RemoveByName("max_lon")
		trailsCollection.Fields.RemoveByName("bounding_box_diagonal")

		if err := app.Save(trailsCollection); err != nil {
			return err
		}

		return nil
	})
}

func addTrailBoundsField(collection *core.Collection, name string) {
	if collection.Fields.GetByName(name) != nil {
		return
	}

	collection.Fields.Add(&core.NumberField{
		Name: name,
	})
}

func backfillTrailBounds(app core.App) error {
	const pageSize int64 = 50
	lastID := ""

	for {
		trails := []*core.Record{}
		query := app.RecordQuery("trails").
			OrderBy("id ASC").
			Limit(pageSize)

		if lastID != "" {
			query = query.AndWhere(dbx.NewExp("id > {:lastID}", dbx.Params{"lastID": lastID}))
		}

		if err := query.All(&trails); err != nil {
			return err
		}
		if len(trails) == 0 {
			return nil
		}

		for _, trail := range trails {
			if err := util.SavePolyline(app, trail); err != nil {
				if err := saveDefaultTrailBounds(app, trail); err != nil {
					return err
				}
			}
			lastID = trail.Id
		}
	}
}

func saveDefaultTrailBounds(app core.App, trail *core.Record) error {
	lat := trail.GetFloat("lat")
	lon := trail.GetFloat("lon")
	trail.Set("min_lat", lat)
	trail.Set("max_lat", lat)
	trail.Set("min_lon", lon)
	trail.Set("max_lon", lon)
	trail.Set("bounding_box_diagonal", 0)
	return app.UnsafeWithoutHooks().Save(trail)
}

func strconvQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
