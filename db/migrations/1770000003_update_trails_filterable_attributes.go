package migrations

import (
	"os"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	client := meilisearch.New(os.Getenv("MEILI_URL"), meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))

	m.Register(func(app core.App) error {
		_, err := client.Index("trails").UpdateFilterableAttributes(&[]string{
			"_geo",
			"_geojson",
			"author",
			"category",
			"completed",
			"date",
			"difficulty",
			"distance",
			"elevation_gain",
			"elevation_loss",
			"public",
			"shares",
			"tags",
			"likes",
			"min_lat",
			"min_lon",
			"max_lat",
			"max_lon",
			"time_bucket_ids",
			"quad_node_ids",
			"id",
		})
		return err
	}, func(app core.App) error {
		return nil
	})
}
