package routes

import (
	"net/http"
	"pocketbase/services/assetmerge"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type assetMergeExecuteRequest struct {
	SourceAssetIDs []string `json:"sourceAssetIds"`
	TargetAssetID  string   `json:"targetAssetId"`
}

func AssetMergeSuggest(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("asset_merge_auth_required", nil)
	}
	actorID, err := authActorID(e)
	if err != nil {
		return err
	}

	response, err := assetmerge.SuggestGroups(e.App, actorID)
	if err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	return e.JSON(http.StatusOK, response)
}

func AssetMerge(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("asset_merge_auth_required", nil)
	}
	actorID, err := authActorID(e)
	if err != nil {
		return err
	}

	var request assetMergeExecuteRequest
	if err := e.BindBody(&request); err != nil {
		return apis.NewBadRequestError("asset_merge_invalid_request", err)
	}

	response, err := assetmerge.MergeWithContext(e.Request.Context(), e.App, actorID, request.TargetAssetID, request.SourceAssetIDs)
	if err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	return e.JSON(http.StatusOK, response)
}

func authActorID(e *core.RequestEvent) (string, error) {
	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	if err != nil {
		return "", apis.NewUnauthorizedError("asset_merge_actor_not_found", err)
	}
	return actor.Id, nil
}
