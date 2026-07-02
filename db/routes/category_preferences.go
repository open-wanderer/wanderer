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

type subcategoryPreferenceReorderRequest struct {
	Category      string   `json:"category"`
	Subcategories []string `json:"subcategories"`
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

func SubcategoryPreferencesReorder(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var request subcategoryPreferenceReorderRequest
	if err := e.BindBody(&request); err != nil {
		return apis.NewBadRequestError("failed to read request data", err)
	}

	if err := util.ReorderUserSubcategoryPreferences(e.App, e.Auth.Id, request.Category, request.Subcategories); err != nil {
		return apis.NewBadRequestError(err.Error(), err)
	}

	return e.JSON(http.StatusOK, map[string]any{"acknowledged": true})
}
