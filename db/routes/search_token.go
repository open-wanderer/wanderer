package routes

import (
	"net/http"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func SearchToken(client meilisearch.ServiceManager) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		searchRules := map[string]interface{}{
			"lists":  map[string]string{"filter": "public = true"},
			"trails": map[string]string{"filter": "public = true"},
		}

		if e.Auth != nil {
			userId := e.Auth.Id
			userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
			if err != nil {
				return err
			}

			searchRules = map[string]any{
				"lists": map[string]string{
					"filter": "public = true OR author = " + userActor.Id + " OR shares = " + userId,
				},
				"trails": map[string]string{
					"filter": "public = true OR author = " + userActor.Id + " OR shares = " + userId,
				},
			}
		}

		token, err := util.GenerateMeilisearchToken(searchRules, client)
		if err != nil {
			return e.InternalServerError("Failed to generate search token", err)
		}

		return e.JSON(http.StatusOK, map[string]string{
			"token": token,
		})
	}
}
