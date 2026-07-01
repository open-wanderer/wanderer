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
		_, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        "trails",
			PrimaryKey: "id",
		})
		if err != nil {
			return err
		}

		_, err = client.Index("trails").UpdateSortableAttributes(&[]string{
			"created", "date", "difficulty", "distance", "elevation_gain", "elevation_loss", "name", "duration", "author",
		})

		if err != nil {
			return err
		}

		_, err = client.Index("trails").UpdateFilterableAttributes(&[]interface{}{
			"_geo", "author", "category", "completed", "date", "difficulty", "distance", "elevation_gain", "elevation_loss", "public", "shares", "tags",
		})

		if err != nil {
			return err
		}

		// ---

		_, err = client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        "lists",
			PrimaryKey: "id",
		})
		if err != nil {
			return err
		}

		_, err = client.Index("lists").UpdateSortableAttributes(&[]string{
			"created", "name",
		})
		if err != nil {
			return err
		}

		_, err = client.Index("lists").UpdateFilterableAttributes(&[]interface{}{
			"author", "public", "shares",
		})
		if err != nil {
			return err
		}

		return nil
	}, nil)
}
