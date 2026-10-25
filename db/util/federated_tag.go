package util

import (
	"pocketbase/tagname"
	"strings"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Parallel deliveries of a new trail resolve its tags at the same time; a
// missing tag is created once and shared by all of them. Callers inside a
// transaction already hold the database writer and never wait for the lock.
var federatedTagCreation sync.Mutex

// ResolveFederatedTag normalizes an incoming plain-text name before lookup or
// creation. Empty and ASCII-space-only normalized names are intentionally
// omitted, without trimming other names. Existing tags are never renamed or
// merged; duplicate names resolve to the smallest ID.
// Failures are logged without remote text or database error details so callers
// can retain the federation import's tolerant behavior.
func ResolveFederatedTag(app core.App, name string) (*core.Record, error) {
	name = tagname.Normalize(name)
	if strings.Trim(name, " ") == "" {
		return nil, nil
	}

	collection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		app.Logger().Warn("federated tag omitted", "code", "tag_collection_unavailable")
		return nil, err
	}

	// Bind the value directly in SQL: tag text never enters the filter grammar.
	lookup := func() (*core.Record, error) {
		records := []*core.Record{}
		if err := app.RecordQuery(collection).AndWhere(dbx.HashExp{"name": name}).OrderBy("id").Limit(1).All(&records); err != nil {
			app.Logger().Warn("federated tag omitted", "code", "tag_lookup_failed")
			return nil, err
		}
		if len(records) > 0 {
			return records[0], nil
		}
		return nil, nil
	}
	if existing, err := lookup(); existing != nil || err != nil {
		return existing, err
	}
	if !app.IsTransactional() {
		federatedTagCreation.Lock()
		defer federatedTagCreation.Unlock()
		if existing, err := lookup(); existing != nil || err != nil {
			return existing, err
		}
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	if err := app.Save(record); err != nil {
		app.Logger().Warn("federated tag omitted", "code", "tag_save_failed")
		return nil, err
	}
	return record, nil
}
