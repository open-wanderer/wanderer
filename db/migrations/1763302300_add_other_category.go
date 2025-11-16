package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func init() {
	m.Register(func(app core.App) error {
		records, err := app.FindRecordsByFilter("categories", "name={:name}", "", 1, 0, dbx.Params{"name": "Other"})
		if err != nil {
			return err
		}
		if len(records) > 0 {
			return nil
		}

		collection, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		record := core.NewRecord(collection)
		record.Set("name", "Other")

		if file, err := filesystem.NewFileFromPath("migrations/initial_data/other.jpg"); err == nil {
			record.Set("img", file)
		}

		return app.Save(record)
	}, func(app core.App) error {
		records, err := app.FindRecordsByFilter("categories", "name={:name}", "", 1, 0, dbx.Params{"name": "Other"})
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}

		return app.Delete(records[0])
	})
}
