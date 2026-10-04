package migrations

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"pocketbase/util"
)

func init() {
	m.Register(upSanitizeStoredHTML1791110000, func(core.App) error {
		// Removed unsafe markup cannot safely be restored.
		return nil
	})
}

// This backfill intentionally bypasses record-save hooks: cleaning historical
// text must not send Update activities, create mentions or change updated dates.
func upSanitizeStoredHTML1791110000(app core.App) error {
	const batchSize = 250
	// Freeze this migration's historical collection/field set. Future model
	// sanitization may add fields whose schema does not exist at this point
	// during a fresh install.
	fields := []struct{ collection, field string }{
		{"activitypub_actors", "summary"},
		{"comments", "text"},
		{"lists", "description"},
		{"settings", "bio"},
		{"summit_logs", "text"},
		{"trails", "description"},
		{"waypoints", "description"},
	}
	for _, entry := range fields {
		name := entry.collection
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName(entry.field).(*core.TextField)
		if !ok {
			return fmt.Errorf("%s.%s must be a text field", name, entry.field)
		}
		limit := field.Max
		if limit == 0 {
			limit = 5000
		}
		cursor := ""
		for {
			records, err := app.FindRecordsByFilter(collection, "id > {:cursor}", "id", batchSize, 0, dbx.Params{"cursor": cursor})
			if err != nil {
				return err
			}
			for _, record := range records {
				cursor = record.Id
				before := record.GetString(entry.field)
				after := util.SanitizeHTMLText(before, limit)
				if after == before {
					continue
				}
				patch := dbx.Params{entry.field: after}
				original := dbx.HashExp{"id": record.Id, entry.field: before}
				if _, err := app.DB().Update(collection.Name, patch, original).Execute(); err != nil {
					return err
				}
			}
			if len(records) < batchSize {
				break
			}
		}
	}
	return nil
}
