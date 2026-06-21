package util

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

func ValidateSubcategoryRecord(app core.App, record *core.Record) error {
	parentCategory := record.GetString("category")
	if parentCategory == "" {
		return fmt.Errorf("subcategory category is required")
	}

	name := record.GetString("name")
	normalizedName := NormalizeCategoryName(name)

	allSubcategories, err := app.FindAllRecords("subcategories")
	if err != nil {
		return err
	}

	for _, existing := range allSubcategories {
		if existing.Id == record.Id || existing.GetString("category") != parentCategory {
			continue
		}

		if NormalizeCategoryName(existing.GetString("name")) == normalizedName {
			return fmt.Errorf("subcategory name %q collides with existing subcategory %q in the same category after normalization", name, existing.GetString("name"))
		}
	}

	if _, err := ParseCategoryTranslations(record.Get("translations")); err != nil {
		return err
	}

	return nil
}

func ValidateTrailSubcategoryRecord(app core.App, record *core.Record, subcategoryExplicit bool) error {
	subcategoryID := record.GetString("subcategory")
	if subcategoryID == "" {
		return nil
	}

	categoryID := record.GetString("category")
	if categoryID == "" {
		if !subcategoryExplicit {
			record.Set("subcategory", "")
			return nil
		}

		return fmt.Errorf("trail subcategory requires a category")
	}

	subcategory, err := app.FindRecordById("subcategories", subcategoryID)
	if err != nil {
		if !subcategoryExplicit {
			record.Set("subcategory", "")
			return nil
		}

		return fmt.Errorf("trail subcategory %q does not exist: %w", subcategoryID, err)
	}

	parentCategory := subcategory.GetString("category")
	if parentCategory != categoryID {
		if !subcategoryExplicit {
			record.Set("subcategory", "")
			return nil
		}

		return fmt.Errorf("trail subcategory %q belongs to category %q, not %q", subcategoryID, parentCategory, categoryID)
	}

	return nil
}

func FindSubcategoryByNormalizedName(app core.App, categoryID string, name string) (*core.Record, error) {
	normalizedName := NormalizeCategoryName(name)
	if categoryID == "" || normalizedName == "" {
		return nil, nil
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
		return nil, err
	}

	for _, subcategory := range subcategories {
		if NormalizeCategoryName(subcategory.GetString("name")) == normalizedName {
			return subcategory, nil
		}
	}

	return nil, nil
}

func ResolveCategoryAndSubcategoryByNormalizedNames(app core.App, categoryName string, subcategoryName string) (*core.Record, *core.Record, error) {
	category, err := FindCategoryByNormalizedName(app, categoryName)
	if err != nil || category == nil {
		return category, nil, err
	}

	subcategory, err := FindSubcategoryByNormalizedName(app, category.Id, subcategoryName)
	if err != nil {
		return category, nil, err
	}

	return category, subcategory, nil
}

func BackfillRemoteTrailCategory(app core.App, category *core.Record) error {
	if category == nil || category.Id == "" {
		return nil
	}

	subcategoriesByName, err := normalizedSubcategoriesByName(app, category.Id)
	if err != nil {
		return err
	}

	trails, err := app.FindRecordsByFilter(
		"trails",
		"federated_category_name != '' && category = ''",
		"",
		0,
		0,
		nil,
	)
	if err != nil {
		return err
	}

	normalizedCategoryName := NormalizeCategoryName(category.GetString("name"))
	for _, trail := range trails {
		if NormalizeCategoryName(trail.GetString("federated_category_name")) != normalizedCategoryName {
			continue
		}

		trail.Set("category", category.Id)
		if subcategory, ok := subcategoriesByName[NormalizeCategoryName(trail.GetString("federated_subcategory_name"))]; ok {
			trail.Set("subcategory", subcategory.Id)
		}

		if err := app.Save(trail); err != nil {
			return err
		}
	}

	return nil
}

func BackfillRemoteTrailSubcategory(app core.App, subcategory *core.Record) error {
	if subcategory == nil || subcategory.Id == "" {
		return nil
	}

	category, err := app.FindRecordById("categories", subcategory.GetString("category"))
	if err != nil {
		return err
	}

	trails, err := app.FindRecordsByFilter(
		"trails",
		"federated_subcategory_name != '' && subcategory = '' && (category = {:category} || category = '')",
		"",
		0,
		0,
		map[string]any{"category": category.Id},
	)
	if err != nil {
		return err
	}

	normalizedCategoryName := NormalizeCategoryName(category.GetString("name"))
	normalizedSubcategoryName := NormalizeCategoryName(subcategory.GetString("name"))
	for _, trail := range trails {
		categoryID := trail.GetString("category")
		if categoryID == "" {
			if NormalizeCategoryName(trail.GetString("federated_category_name")) != normalizedCategoryName {
				continue
			}
			trail.Set("category", category.Id)
		}

		if NormalizeCategoryName(trail.GetString("federated_subcategory_name")) != normalizedSubcategoryName {
			continue
		}

		trail.Set("subcategory", subcategory.Id)
		if err := app.Save(trail); err != nil {
			return err
		}
	}

	return nil
}

func normalizedSubcategoriesByName(app core.App, categoryID string) (map[string]*core.Record, error) {
	subcategories, err := app.FindRecordsByFilter(
		"subcategories",
		"category = {:category}",
		"",
		0,
		0,
		map[string]any{"category": categoryID},
	)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]*core.Record, len(subcategories))
	for _, subcategory := range subcategories {
		normalizedName := NormalizeCategoryName(subcategory.GetString("name"))
		if normalizedName == "" {
			continue
		}
		byName[normalizedName] = subcategory
	}

	return byName, nil
}
