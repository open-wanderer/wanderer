package migrations

import (
	"os"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	client := meilisearch.New(os.Getenv("MEILI_URL"), meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))

	m.Register(func(app core.App) error {
		_, err := client.Index("trails").UpdateFilterableAttributes(&[]any{
			"id", "_geo", "author", "category", "completed", "date", "difficulty",
			"distance", "elevation_gain", "elevation_loss", "likes", "public",
			"shares", "tags", "min_lat", "max_lat", "min_lon", "max_lon", "bounding_box_diagonal",
		})
		if err != nil {
			return err
		}

		// Re-index all trails to populate the new bounding box and diagonal fields
		const pageSize int64 = 50
		var page int64 = 0

		for {
			trails := []*core.Record{}
			err := app.RecordQuery("trails").
				Limit(pageSize).
				Offset(page * pageSize).
				All(&trails)
			if err != nil {
				return err
			}
			if len(trails) == 0 {
				break
			}

			if err := util.IndexTrails(app, trails, client); err != nil {
				return err
			}

			page++
		}

		return nil
	}, func(app core.App) error {
		_, err := client.Index("trails").UpdateFilterableAttributes(&[]any{
			"_geo", "author", "category", "completed", "date", "difficulty", "distance", "elevation_gain", "elevation_loss", "public", "shares", "tags", "likes",
		})

		return err
	})
}
