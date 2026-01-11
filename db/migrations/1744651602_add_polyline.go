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
		searchableAttributes := []string{
			"author_name",
			"name",
			"description",
			"location",
			"tags",
		}
		_, err := client.Index("trails").UpdateSearchableAttributes(&searchableAttributes)
		if err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// add down queries...

		return nil
	})
}
