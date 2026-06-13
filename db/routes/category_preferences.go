package routes

import (
	"net/http"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type categoryPreferenceReorderRequest struct {
	Categories []string `json:"categories"`
}

func CategoryPreferencesReorder(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var request categoryPreferenceReorderRequest
	if err := e.BindBody(&request); err != nil {
		return apis.NewBadRequestError("failed to read request data", err)
	}

	if err := util.ReorderUserCategoryPreferences(e.App, e.Auth.Id, request.Categories); err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	return e.JSON(http.StatusOK, map[string]any{"acknowledged": true})
}
