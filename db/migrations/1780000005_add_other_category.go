package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func init() {
	m.Register(func(app core.App) error {
		categories, err := app.FindAllRecords("categories")
		if err != nil {
			return err
		}
		if len(categories) == 0 {
			return nil
		}

		existing, _ := app.FindFirstRecordByData("categories", "name", "Other")
		if existing != nil {
			return nil
		}

		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		record := core.NewRecord(collection)
		record.Set("name", "Other")
		record.Set("settings", map[string]any{
			"wp_merge_enabled": true,
			"wp_merge_radius":  50,
		})
		if file, err := filesystem.NewFileFromPath("migrations/initial_data/other.jpg"); err == nil {
			record.Set("img", file)
		}
		return app.Save(record)
	}, func(app core.App) error {
		record, _ := app.FindFirstRecordByData("categories", "name", "Other")
		if record == nil {
			return nil
		}
		return app.Delete(record)
	})
}
