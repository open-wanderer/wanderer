package hooks

import (
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func ValidateCategoryHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if err := util.ValidateCategoryRecord(e.App, e.Record); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.Next()
	}
}

func ValidateSubcategoryHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if err := util.ValidateSubcategoryRecord(e.App, e.Record); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.Next()
	}
}

func BackfillRemoteTrailCategoryHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Record.Original().Id != "" && e.Record.GetString("name") == e.Record.Original().GetString("name") {
			return e.Next()
		}

		if err := util.BackfillRemoteTrailCategory(e.App, e.Record); err != nil {
			e.App.Logger().Warn("failed to backfill remote trail categories after category save", "category", e.Record.Id, "error", err)
		}

		return e.Next()
	}
}

func BackfillRemoteTrailSubcategoryHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		original := e.Record.Original()
		if original.Id != "" &&
			e.Record.GetString("name") == original.GetString("name") &&
			e.Record.GetString("category") == original.GetString("category") {
			return e.Next()
		}

		if err := util.BackfillRemoteTrailSubcategory(e.App, e.Record); err != nil {
			e.App.Logger().Warn("failed to backfill remote trail subcategories after subcategory save", "subcategory", e.Record.Id, "error", err)
		}

		return e.Next()
	}
}

func ValidateUserCategoryPreferenceHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		requestInfo, err := e.RequestInfo()
		if err != nil {
			return err
		}

		if err := util.ValidateUserCategoryPreferenceRequest(requestBodyHasField(requestInfo.Body, "priority")); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.Next()
	}
}

func ValidateUserSubcategoryPreferenceHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		requestInfo, err := e.RequestInfo()
		if err != nil {
			return err
		}

		if err := util.ValidateUserSubcategoryPreferenceRequest(requestBodyHasField(requestInfo.Body, "priority")); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.Next()
	}
}

func ValidateTrailSubcategoryHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		requestInfo, err := e.RequestInfo()
		if err != nil {
			return err
		}

		subcategoryExplicit := requestBodyHasField(requestInfo.Body, "subcategory")
		if err := util.ValidateTrailSubcategoryRecord(e.App, e.Record, subcategoryExplicit); err != nil {
			return apis.NewBadRequestError(err.Error(), err)
		}

		return e.Next()
	}
}

func requestBodyHasField(body map[string]any, field string) bool {
	_, ok := body[field]
	if ok {
		return true
	}

	_, ok = body[field+"+"]
	if ok {
		return true
	}

	_, ok = body["+"+field]
	if ok {
		return true
	}

	_, ok = body[field+"-"]
	return ok
}
