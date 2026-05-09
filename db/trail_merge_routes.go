package main

import (
	"net/http"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/trailmerge"
)

type mergeExecuteRequest struct {
	SourceTrailID string                   `json:"sourceTrailId"`
	TargetTrailID string                   `json:"targetTrailId"`
	Settings      trailmerge.MergeSettings `json:"settings"`
}

func registerTrailMergeRoutes(se *core.ServeEvent, client meilisearch.ServiceManager) {
	se.Router.POST("/trail-merge/suggest", func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apis.NewUnauthorizedError("trail_merge_auth_required", nil)
		}

		actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return apis.NewBadRequestError("trail_merge_actor_not_found", err)
		}

		var request trailmerge.SuggestRequest
		if err := e.BindBody(&request); err != nil {
			return apis.NewBadRequestError("trail_merge_invalid_request", err)
		}

		if request.Mode == trailmerge.SuggestModeMaintenance {
			response, err := trailmerge.SuggestGroups(e.App, actor.Id, request)
			if err != nil {
				return apis.NewBadRequestError(err.Error(), err)
			}

			return e.JSON(http.StatusOK, response)
		}

		response, err := trailmerge.Suggest(e.App, actor.Id, request)
		if err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.JSON(http.StatusOK, response)
	})

	se.Router.POST("/trail-merge", func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apis.NewUnauthorizedError("trail_merge_auth_required", nil)
		}

		actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return apis.NewBadRequestError("trail_merge_actor_not_found", err)
		}

		var request mergeExecuteRequest
		if err := e.BindBody(&request); err != nil {
			return apis.NewBadRequestError("trail_merge_invalid_request", err)
		}

		source, err := e.App.FindRecordById("trails", request.SourceTrailID)
		if err != nil {
			return apis.NewBadRequestError("trail_merge_source_not_found", err)
		}
		target, err := e.App.FindRecordById("trails", request.TargetTrailID)
		if err != nil {
			return apis.NewBadRequestError("trail_merge_target_not_found", err)
		}

		if !trailmerge.CanMerge(e.App, actor.Id, source, target, request.Settings.Delete) {
			return apis.NewForbiddenError("trail_merge_not_allowed", nil)
		}

		if err := trailmerge.Merge(e.App, client, actor, request.SourceTrailID, request.TargetTrailID, request.Settings); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"acknowledged": true,
		})
	})
}
