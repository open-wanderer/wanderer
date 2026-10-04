package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(guardRecordOwnership1791110001, func(app core.App) error {
		// Keep ownership protection when rolling back unrelated schema changes.
		return nil
	})
}

// These guards constrain API updates only. Trusted model saves used by merges
// and federation can still move records, and PocketBase superusers retain
// their normal administrative access. This migration needs no webhook schema.
func guardRecordOwnership1791110001(app core.App) error {
	return app.RunInTransaction(func(tx core.App) error {
		for _, target := range []struct {
			collection string
			fields     []string
		}{
			{"trails", []string{"author"}},
			{"lists", []string{"author"}},
			{"comments", []string{"author", "trail"}},
			{"waypoints", []string{"author", "trail"}},
			{"summit_logs", []string{"author", "trail"}},
			{"plugin_instances", []string{"user", "plugin_id"}},
		} {
			collection, err := tx.FindCollectionByNameOrId(target.collection)
			if err != nil {
				return err
			}
			if collection.UpdateRule == nil {
				// A locked API stays locked; never replace nil with a public rule.
				continue
			}
			guards := make([]string, len(target.fields))
			for i, field := range target.fields {
				guards[i] = "@request.body." + field + ":changed = false"
			}
			guard := strings.Join(guards, " && ")
			previous := *collection.UpdateRule
			if previous == guard || strings.HasSuffix(previous, " && "+guard) {
				continue
			}
			if previous != "" {
				guard = "(" + previous + ") && " + guard
			}
			collection.UpdateRule = types.Pointer(guard)
			if err := tx.Save(collection); err != nil {
				return err
			}
		}
		return nil
	})
}
