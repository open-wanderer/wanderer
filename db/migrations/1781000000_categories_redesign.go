package migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"pocketbase/util"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		if err := resolveCategoryNameCollisions(app); err != nil {
			return fmt.Errorf("failed to resolve category name collisions: %w", err)
		}

		if err := deleteCategoryImageFiles(app); err != nil {
			return fmt.Errorf("failed to delete category image files: %w", err)
		}

		collection.Fields.RemoveById("64dsnxtb")

		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"hidden": false,
			"id": "texti4ksx4gm",
			"max": 0,
			"min": 0,
			"name": "short_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"hidden": false,
			"id": "text0r6k2h4gi",
			"max": 0,
			"min": 0,
			"name": "icon",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"hidden": false,
			"id": "jsonvkf7o88i",
			"maxSize": 0,
			"name": "translations",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if err := ensureRunningCategory(app); err != nil {
			return fmt.Errorf("failed to ensure running category: %w", err)
		}

		if err := createSubcategoriesCollection(app); err != nil {
			return err
		}

		trailsCollection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}
		if err := trailsCollection.Fields.AddMarshaledJSONAt(len(trailsCollection.Fields), []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_1781100000",
			"hidden": false,
			"id": "relphase2subct",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "subcategory",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}
		if err := trailsCollection.Fields.AddMarshaledJSONAt(len(trailsCollection.Fields), []byte(`{
			"hidden": false,
			"id": "textremotecat1",
			"max": 0,
			"min": 0,
			"name": "federated_category_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}
		if err := trailsCollection.Fields.AddMarshaledJSONAt(len(trailsCollection.Fields), []byte(`{
			"hidden": false,
			"id": "textremotesub1",
			"max": 0,
			"min": 0,
			"name": "federated_subcategory_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}
		if err := app.Save(trailsCollection); err != nil {
			return err
		}

		if err := util.PrepopulateDefaultCategoryTranslations(app); err != nil {
			return fmt.Errorf("failed to prepopulate default category translations: %w", err)
		}

		if err := util.PrepopulateDefaultCategoryIcons(app); err != nil {
			return fmt.Errorf("failed to prepopulate default category icons: %w", err)
		}

		if err := util.SeedDefaultSubcategories(app); err != nil {
			return fmt.Errorf("failed to seed default subcategories: %w", err)
		}

		if err := createUserCategoryPreferencesCollection(app); err != nil {
			return err
		}

		if err := createUserSubcategoryPreferencesCollection(app); err != nil {
			return err
		}

		if err := migrateFavouriteSportToPriority(app); err != nil {
			return fmt.Errorf("failed to migrate favourite sport to category priority: %w", err)
		}

		if err := removeSettingsCategoryField(app); err != nil {
			return fmt.Errorf("failed to remove settings.category field: %w", err)
		}

		if err := util.ValidateCategoryCollectionState(app); err != nil {
			return fmt.Errorf("categories redesign migration failed validation: %w", err)
		}

		return nil
	}, func(app core.App) error {
		if collection, err := app.FindCollectionByNameOrId("pbc_1781250000"); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		if collection, err := app.FindCollectionByNameOrId("pbc_1781200000"); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		if trailsCollection, err := app.FindCollectionByNameOrId("trails"); err == nil {
			trailsCollection.Fields.RemoveById("relphase2subct")
			trailsCollection.Fields.RemoveById("textremotecat1")
			trailsCollection.Fields.RemoveById("textremotesub1")
			if err := app.Save(trailsCollection); err != nil {
				return err
			}
		}

		if collection, err := app.FindCollectionByNameOrId("pbc_1781100000"); err == nil {
			if err := app.Delete(collection); err != nil {
				return err
			}
		}

		if settingsCollection, err := app.FindCollectionByNameOrId("settings"); err == nil {
			if err := settingsCollection.Fields.AddMarshaledJSONAt(len(settingsCollection.Fields), []byte(`{
				"cascadeDelete": false,
				"collectionId": "kjxvi8asj2igqwf",
				"hidden": false,
				"id": "owlyzl1x",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "category",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
			if err := app.Save(settingsCollection); err != nil {
				return err
			}
		}

		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("texti4ksx4gm")
		collection.Fields.RemoveById("text0r6k2h4gi")
		collection.Fields.RemoveById("jsonvkf7o88i")

		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"hidden": false,
			"id": "64dsnxtb",
			"maxSelect": 1,
			"maxSize": 5242880,
			"mimeTypes": null,
			"name": "img",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": null,
			"type": "file"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}

type categoryCollisionCandidate struct {
	id      string
	name    string
	created string
}

func resolveCategoryNameCollisions(app core.App) error {
	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	candidates := make([]categoryCollisionCandidate, 0, len(allCategories))
	for _, category := range allCategories {
		candidates = append(candidates, categoryCollisionCandidate{
			id:      category.Id,
			name:    category.GetString("name"),
			created: category.GetString("created"),
		})
	}
	resolvedNames := resolveCategoryNameCollisionCandidates(candidates)

	for _, category := range allCategories {
		resolvedName, ok := resolvedNames[category.Id]
		if !ok {
			continue
		}

		originalName := category.GetString("name")
		category.Set("name", resolvedName)
		if err := app.Save(category); err != nil {
			return fmt.Errorf("failed to resolve category name collision for %q: %w", originalName, err)
		}
	}

	return nil
}

func collisionResolvedCategoryName(name string, id string, seen map[string]struct{}) string {
	baseName := strings.TrimSpace(name)
	if baseName == "" {
		baseName = "Category"
	}

	for i := 0; ; i++ {
		suffix := id
		if i > 0 {
			suffix = fmt.Sprintf("%s-%d", id, i+1)
		}

		candidate := fmt.Sprintf("%s (%s)", baseName, suffix)
		if _, ok := seen[util.NormalizeCategoryName(candidate)]; !ok {
			return candidate
		}
	}
}

func resolveCategoryNameCollisionCandidates(candidates []categoryCollisionCandidate) map[string]string {
	sorted := append([]categoryCollisionCandidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]

		leftName := util.NormalizeCategoryName(left.name)
		rightName := util.NormalizeCategoryName(right.name)
		if leftName != rightName {
			return leftName < rightName
		}

		if left.created != right.created {
			return left.created < right.created
		}

		return left.id < right.id
	})

	seen := map[string]struct{}{}
	resolved := map[string]string{}
	for _, category := range sorted {
		normalizedName := util.NormalizeCategoryName(category.name)
		if _, ok := seen[normalizedName]; !ok {
			seen[normalizedName] = struct{}{}
			continue
		}

		resolvedName := collisionResolvedCategoryName(category.name, category.id, seen)
		resolved[category.id] = resolvedName
		seen[util.NormalizeCategoryName(resolvedName)] = struct{}{}
	}

	return resolved
}

func ensureRunningCategory(app core.App) error {
	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	if len(allCategories) == 0 {
		return util.SeedDefaultCategories(app)
	}

	for _, category := range allCategories {
		if util.NormalizeCategoryName(category.GetString("name")) == util.NormalizeCategoryName("Running") {
			return nil
		}
	}

	collection, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("name", "Running")
	record.Set("settings", map[string]any{
		"wp_merge_enabled": true,
		"wp_merge_radius":  50,
	})

	return app.Save(record)
}

func deleteCategoryImageFiles(app core.App) error {
	allCategories, err := app.FindAllRecords("categories")
	if err != nil {
		return err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fsys.Close()

	var failures []error
	for _, category := range allCategories {
		for _, filename := range categoryImageFilenames(category) {
			if filename == "" || strings.ContainsAny(filename, `/\`) {
				continue
			}

			path := category.BaseFilesPath() + "/" + filename
			if err := fsys.Delete(path); err != nil && !errors.Is(err, filesystem.ErrNotFound) {
				failures = append(failures, fmt.Errorf("failed to delete category image %q: %w", path, err))
			}

			if errs := fsys.DeletePrefix(category.BaseFilesPath() + "/thumbs_" + filename + "/"); len(errs) > 0 {
				failures = append(failures, fmt.Errorf("failed to delete category image thumbs for %q: %w", path, errors.Join(errs...)))
			}
		}
	}

	if len(failures) > 0 {
		return errors.Join(failures...)
	}

	return nil
}

func categoryImageFilenames(record *core.Record) []string {
	filenames := record.GetStringSlice("img")
	if len(filenames) > 0 {
		return filenames
	}

	filename := record.GetString("img")
	if filename == "" {
		return nil
	}

	return []string{filename}
}

func createSubcategoriesCollection(app core.App) error {
	jsonData := `{
		"createRule": null,
		"deleteRule": null,
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "kjxvi8asj2igqwf",
				"hidden": false,
				"id": "relphase2cat01",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "category",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "textphase2name",
				"max": 0,
				"min": 1,
				"name": "name",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": true,
				"system": false,
				"type": "text"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "textphase2shrt",
				"max": 0,
				"min": 0,
				"name": "short_name",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "textphase2icon",
				"max": 0,
				"min": 0,
				"name": "icon",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "textphase2badge",
				"max": 0,
				"min": 0,
				"name": "badge_icon",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "jsonphase2trns",
				"maxSize": 0,
				"name": "translations",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"id": "pbc_1781100000",
		"indexes": [],
		"listRule": "",
		"name": "subcategories",
		"system": false,
		"type": "base",
		"updateRule": null,
		"viewRule": ""
	}`

	return saveCollectionFromJSON(app, jsonData)
}

func createUserCategoryPreferencesCollection(app core.App) error {
	jsonData := `{
		"createRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"deleteRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text3208210256",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "relphase3user",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "user",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"cascadeDelete": true,
				"collectionId": "kjxvi8asj2igqwf",
				"hidden": false,
				"id": "relphase3cat",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "category",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "boolphase3visible",
				"name": "visible",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "bool"
			},
			{
				"hidden": false,
				"id": "numphase3prio",
				"max": null,
				"min": 1,
				"name": "priority",
				"onlyInt": true,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "autodate2990389176",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate3332085495",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"id": "pbc_1781200000",
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`" + `idx_user_category_preferences_user_category` + "`" + ` ON ` + "`" + `user_category_preferences` + "`" + ` (` + "`" + `user` + "`" + `, ` + "`" + `category` + "`" + `)"
		],
		"listRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"name": "user_category_preferences",
		"system": false,
		"type": "base",
		"updateRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && user = @request.auth.id"
	}`

	return saveCollectionFromJSON(app, jsonData)
}

func createUserSubcategoryPreferencesCollection(app core.App) error {
	jsonData := `{
		"createRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"deleteRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "text178125id",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"cascadeDelete": true,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "rel178125user",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "user",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"cascadeDelete": true,
				"collectionId": "pbc_1781100000",
				"hidden": false,
				"id": "rel178125subcat",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "subcategory",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "bool178125visible",
				"name": "visible",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "bool"
			},
			{
				"hidden": false,
				"id": "num178125prio",
				"max": null,
				"min": 1,
				"name": "priority",
				"onlyInt": true,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "autodate178125created",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autodate178125updated",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"id": "pbc_1781250000",
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`" + `idx_user_subcategory_preferences_user_subcategory` + "`" + ` ON ` + "`" + `user_subcategory_preferences` + "`" + ` (` + "`" + `user` + "`" + `, ` + "`" + `subcategory` + "`" + `)"
		],
		"listRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"name": "user_subcategory_preferences",
		"system": false,
		"type": "base",
		"updateRule": "@request.auth.id != \"\" && user = @request.auth.id",
		"viewRule": "@request.auth.id != \"\" && user = @request.auth.id"
	}`

	return saveCollectionFromJSON(app, jsonData)
}

// migrateFavouriteSportToPriority carries the former per-user "favourite sport"
// (settings.category) over to the new category priority model: the favourite becomes
// the user's priority-1 category. Users without a favourite get a common default
// instead of falling back to category sort order. Users who already organized
// categories by priority are left untouched.
func migrateFavouriteSportToPriority(app core.App) error {
	settingsRecords, err := app.FindAllRecords("settings")
	if err != nil {
		return err
	}

	for _, settings := range settingsRecords {
		if err := util.EnsureUserCategoryPriority(app, settings.GetString("user"), settings.GetString("category")); err != nil {
			return err
		}
	}

	return nil
}

func removeSettingsCategoryField(app core.App) error {
	settings, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		return err
	}

	settings.Fields.RemoveById("owlyzl1x")
	return app.Save(settings)
}

func saveCollectionFromJSON(app core.App, jsonData string) error {
	collection := &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
		return err
	}

	return app.Save(collection)
}
