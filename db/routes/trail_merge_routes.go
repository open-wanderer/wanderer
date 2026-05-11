package routes

import (
	"net/http"
	"pocketbase/services/trailmerge"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type mergeExecuteRequest struct {
	SourceTrailID string                   `json:"sourceTrailId"`
	TargetTrailID string                   `json:"targetTrailId"`
	Settings      trailmerge.MergeSettings `json:"settings"`
}

func TrailMergeSuggest(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("trail_merge_auth_required", nil)
	}

	userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	if err != nil {
		return apis.NewBadRequestError("trail_merge_actor_not_found", err)
	}

	var request trailmerge.SuggestRequest
	if err := e.BindBody(&request); err != nil {
		return apis.NewBadRequestError("trail_merge_invalid_request", err)
	}

	if request.Mode == trailmerge.SuggestModeMaintenance {
		response, err := trailmerge.SuggestGroups(e.App, userActor.Id, request)
		if err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.JSON(http.StatusOK, response)
	}

	response, err := trailmerge.Suggest(e.App, userActor.Id, request)
	if err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	return e.JSON(http.StatusOK, response)
}

func TrailMerge(client meilisearch.ServiceManager) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apis.NewUnauthorizedError("trail_merge_auth_required", nil)
		}

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
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

		if !trailmerge.CanMerge(e.App, userActor.Id, source, target, request.Settings.Delete) {
			return apis.NewForbiddenError("trail_merge_not_allowed", nil)
		}

		ctx, err := util.GetSafeActorContext(e.Request, userActor)

		if err := trailmerge.Merge(e.App, client, ctx, userActor, request.SourceTrailID, request.TargetTrailID, request.Settings); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"acknowledged": true,
		})
	}
}
