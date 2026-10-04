package util

import (
	"pocketbase/tagname"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ResolveFederatedTag normalizes an incoming plain-text name before lookup or
// creation. Empty normalized names are intentionally omitted. Existing tags
// are never renamed or merged; duplicate names resolve to the smallest ID.
// Failures are logged without remote text or database error details so callers
// can retain the federation import's tolerant behavior.
func ResolveFederatedTag(app core.App, name string) (*core.Record, error) {
	name = tagname.Normalize(name)
	if name == "" {
		return nil, nil
	}

	collection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		app.Logger().Warn("federated tag omitted", "code", "tag_collection_unavailable")
		return nil, err
	}

	// Bind the value directly in SQL: tag text never enters the filter grammar.
	records := []*core.Record{}
	if err := app.RecordQuery(collection).AndWhere(dbx.HashExp{"name": name}).OrderBy("id").Limit(1).All(&records); err != nil {
		app.Logger().Warn("federated tag omitted", "code", "tag_lookup_failed")
		return nil, err
	}
	if len(records) > 0 {
		return records[0], nil
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	if err := app.Save(record); err != nil {
		app.Logger().Warn("federated tag omitted", "code", "tag_save_failed")
		return nil, err
	}
	return record, nil
}
