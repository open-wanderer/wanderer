package migrations

import (
	"os"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/util"
)

func init() {
	client := meilisearch.New(os.Getenv("MEILI_URL"), meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))

	m.Register(func(app core.App) error {
		users, err := app.FindAllRecords("users")
		if err != nil {
			return err
		}
		for _, u := range users {
			userId := u.Id

			actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userId)
			if err != nil {
				return err
			}

			searchRules := map[string]interface{}{
				"lists": map[string]string{
					"filter": "public = true OR author = " + actor.Id + " OR shares = " + userId,
				},
				"trails": map[string]string{
					"filter": "public = true OR author = " + actor.Id + " OR shares = " + userId,
				},
			}

			token, err := util.GenerateMeilisearchToken(searchRules, client)
			if err != nil {
				return err
			}
			u.Set("token", token)
			if err := app.Save(u); err != nil {
				return err
			}

		}

		return nil
	}, func(app core.App) error {
		// add down queries...

		return nil
	})
}
