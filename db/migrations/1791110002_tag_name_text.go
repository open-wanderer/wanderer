package migrations

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(constrainTagNames1791110002, func(app core.App) error {
		// Plain-text safety remains in place on rollback. No tag IDs or trail
		// relations were changed, and legitimate names were never rewritten.
		return nil
	})
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
		name.Max = 5000
		name.Pattern = `^[^\x00-\x1f\x7f]*$`
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
				clean := strings.Map(func(r rune) rune {
					if r < 0x20 || r == 0x7f {
						return -1
					}
					return r
				}, original)
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
