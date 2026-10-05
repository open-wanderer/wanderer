package migrations

import (
	"fmt"

	"pocketbase/tagname"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(constrainTagNames1791110002, restoreTagNameSchema1791110002)
}

func constrainTagNames1791110002(app core.App) error {
	return app.RunInTransaction(func(tx core.App) error {
		collection, err := tx.FindCollectionByNameOrId("tags")
		if err != nil {
			return err
		}
		name, ok := collection.Fields.GetByName("name").(*core.TextField)
		if !ok {
			return fmt.Errorf("tags.name must be a text field")
		}
		// Max=0 already means 5000 Unicode characters in PocketBase. Keep that
		// limit and the optional/non-unique name semantics of the shipped schema.
		name.Max = tagname.MaxLength
		name.Pattern = tagname.Pattern
		if err := tx.Save(collection); err != nil {
			return err
		}
		lastID := ""
		for {
			records, err := tx.FindRecordsByFilter("tags", "id > {:last}", "+id", 200, 0, dbx.Params{"last": lastID})
			if err != nil {
				return err
			}
			if len(records) == 0 {
				return nil
			}
			for _, record := range records {
				original := record.GetString("name")
				clean := tagname.Normalize(original)
				if clean == original {
					continue
				}
				// Avoid external index/federation hooks during schema migration.
				// Startup rebuilds the search documents from these stored names.
				if _, err := tx.DB().NewQuery("UPDATE tags SET name = {:name} WHERE id = {:id}").Bind(dbx.Params{
					"name": clean, "id": record.Id,
				}).Execute(); err != nil {
					return err
				}
			}
			lastID = records[len(records)-1].Id
		}
	})
}

func restoreTagNameSchema1791110002(app core.App) error {
	return app.RunInTransaction(func(tx core.App) error {
		collection, err := tx.FindCollectionByNameOrId("tags")
		if err != nil {
			return err
		}
		name, ok := collection.Fields.GetByName("name").(*core.TextField)
		if !ok {
			return fmt.Errorf("tags.name must be a text field")
		}
		// Restore the original schema only. Removed control bytes cannot be
		// reconstructed; the normalized names, tag IDs and relations stay intact.
		name.Max = 0
		name.Pattern = ""
		return tx.Save(collection)
	})
}
