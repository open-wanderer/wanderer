package util

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

const DefaultPriorityCategoryName = "Hiking"

func ValidateUserCategoryPreferenceRequest(priorityExplicit bool) error {
	if priorityExplicit {
		return fmt.Errorf("category preference priority can only be changed through the reorder endpoint")
	}

	return nil
}

func ValidateUserSubcategoryPreferenceRequest(priorityExplicit bool) error {
	if priorityExplicit {
		return fmt.Errorf("subcategory preference priority can only be changed through the reorder endpoint")
	}

	return nil
}

func EnsureUserCategoryPriority(app core.App, userID, categoryID string) error {
	if userID == "" {
		return nil
	}

	prioritized, err := app.FindRecordsByFilter(
		"user_category_preferences",
		"user = {:user} && priority > 0",
		"",
		1,
		0,
		map[string]any{"user": userID},
	)
	if err != nil {
		return err
	}
	if len(prioritized) > 0 {
		return nil
	}

	if categoryID == "" {
		category, err := FindCategoryByNormalizedName(app, DefaultPriorityCategoryName)
		if err != nil {
			return err
		}
		if category == nil {
			return nil
		}
		categoryID = category.Id
	}

	collection, err := app.FindCollectionByNameOrId("user_category_preferences")
	if err != nil {
		return err
	}

	existing, err := app.FindRecordsByFilter(
		"user_category_preferences",
		"user = {:user} && category = {:category}",
		"",
		1,
		0,
		map[string]any{"user": userID, "category": categoryID},
	)
	if err != nil {
		return err
	}

	var record *core.Record
	if len(existing) > 0 {
		record = existing[0]
	} else {
		record = core.NewRecord(collection)
		record.Set("user", userID)
		record.Set("category", categoryID)
		record.Set("visible", true)
	}

	record.Set("priority", 1)
	return app.SaveNoValidate(record)
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
				record.Set("visible", true)
			}

			record.Set("priority", index+1)
			if err := txApp.SaveNoValidate(record); err != nil {
				return err
			}
		}

		return nil
	})
}

func ReorderUserSubcategoryPreferences(app core.App, userID, categoryID string, subcategoryIDs []string) error {
	if userID == "" {
		return fmt.Errorf("authentication required")
	}
	if categoryID == "" {
		return fmt.Errorf("category is required")
	}

	subcategories, err := app.FindRecordsByFilter(
		"subcategories",
		"category = {:category}",
		"",
		0,
		0,
		map[string]any{"category": categoryID},
	)
	if err != nil {
		return err
	}

	if len(subcategoryIDs) != len(subcategories) {
		return fmt.Errorf("reorder request must include all subcategories for the category")
	}

	validSubcategories := make(map[string]struct{}, len(subcategories))
	for _, subcategory := range subcategories {
		validSubcategories[subcategory.Id] = struct{}{}
	}

	seen := make(map[string]struct{}, len(subcategoryIDs))
	for _, subcategoryID := range subcategoryIDs {
		if _, ok := validSubcategories[subcategoryID]; !ok {
			return fmt.Errorf("unknown subcategory %q", subcategoryID)
		}
		if _, ok := seen[subcategoryID]; ok {
			return fmt.Errorf("duplicate subcategory %q", subcategoryID)
		}
		seen[subcategoryID] = struct{}{}
	}

	return app.RunInTransaction(func(txApp core.App) error {
		collection, err := txApp.FindCollectionByNameOrId("user_subcategory_preferences")
		if err != nil {
			return err
		}

		existing, err := txApp.FindRecordsByFilter(
			"user_subcategory_preferences",
			"user = {:user}",
			"",
			0,
			0,
			map[string]any{"user": userID},
		)
		if err != nil {
			return err
		}

		bySubcategory := make(map[string]*core.Record, len(existing))
		for _, record := range existing {
			bySubcategory[record.GetString("subcategory")] = record
		}

		for index, subcategoryID := range subcategoryIDs {
			record := bySubcategory[subcategoryID]
			if record == nil {
				record = core.NewRecord(collection)
				record.Set("user", userID)
				record.Set("subcategory", subcategoryID)
				record.Set("visible", true)
			}

			record.Set("priority", index+1)
			if err := txApp.SaveNoValidate(record); err != nil {
				return err
			}
		}

		return nil
	})
}
