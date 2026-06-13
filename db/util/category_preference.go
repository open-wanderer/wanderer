package util

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func ValidateUserCategoryPreferenceRequest(priorityExplicit bool) error {
	if priorityExplicit {
		return fmt.Errorf("category preference priority can only be changed through the reorder endpoint")
	}

	return nil
}

func ReorderUserCategoryPreferences(app core.App, userID string, categoryIDs []string) error {
	if userID == "" {
		return fmt.Errorf("authentication required")
	}

	categories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	if len(categoryIDs) != len(categories) {
		return fmt.Errorf("reorder request must include all categories")
	}

	validCategories := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		validCategories[category.Id] = struct{}{}
	}

	seen := make(map[string]struct{}, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		if _, ok := validCategories[categoryID]; !ok {
			return fmt.Errorf("unknown category %q", categoryID)
		}
		if _, ok := seen[categoryID]; ok {
			return fmt.Errorf("duplicate category %q", categoryID)
		}
		seen[categoryID] = struct{}{}
	}

	return app.RunInTransaction(func(txApp core.App) error {
		collection, err := txApp.FindCollectionByNameOrId("user_category_preferences")
		if err != nil {
			return err
		}

		existing, err := txApp.FindRecordsByFilter(
			"user_category_preferences",
			"user = {:user}",
			"",
			0,
			0,
			map[string]any{"user": userID},
		)
		if err != nil {
			return err
		}

		byCategory := make(map[string]*core.Record, len(existing))
		for _, record := range existing {
			byCategory[record.GetString("category")] = record
		}

		for index, categoryID := range categoryIDs {
			record := byCategory[categoryID]
			if record == nil {
				record = core.NewRecord(collection)
				record.Set("user", userID)
				record.Set("category", categoryID)
			}

			record.Set("priority", index+1)
			if err := txApp.SaveNoValidate(record); err != nil {
				return err
			}
		}

		return nil
	})
}
