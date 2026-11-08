package util

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// FindAssetMediaRecord loads access, file and status fields without merge
// provenance or unrelated user metadata. Projected records must not be saved.
func FindAssetMediaRecord(app core.App, assetID string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return nil, err
	}
	fields := []string{}
	for _, name := range []string{"id", "author", "type", "file", "storage_mode", "external_provider", "external_id", "remote_status", "remote_error", "remote_checked_at", "remote_missing_since", "created", "updated"} {
		if collection.Fields.GetByName(name) != nil {
			fields = append(fields, "assets."+name)
		}
	}
	if collection.Fields.GetByName("metadata") != nil {
		fields = append(fields, "json('{}') AS metadata")
	}
	record, err := app.FindRecordById(collection, assetID, func(query *dbx.SelectQuery) error { query.Select(fields...); return nil })
	if err != nil {
		return nil, err
	}
	return record, nil
}
