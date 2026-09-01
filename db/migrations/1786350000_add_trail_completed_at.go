package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const completedAtIndex1786350000 = "CREATE INDEX idx_trails_completed_at ON trails (completed_at) WHERE completed = true;"

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("completed_at") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
				"hidden": false,
				"id": "datecompleted1",
				"max": "",
				"min": "",
				"name": "completed_at",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			}`)); err != nil {
				return err
			}
		}

		if !containsString1786350000(collection.Indexes, completedAtIndex1786350000) {
			collection.Indexes = append(collection.Indexes, completedAtIndex1786350000)
		}
		if err := app.Save(collection); err != nil {
			return err
		}

		// An existing summit log is the best available completion date, so it is
		// applied first; whatever is still blank afterwards falls back to the
		// last update, which is closer to the completion toggle than the trail
		// creation/planning date.
		if _, err := app.DB().
			NewQuery(`
				UPDATE trails
				SET completed_at = oldest_logs.completed_at
				FROM (
					SELECT trail, MIN(date) AS completed_at
					FROM summit_logs
					WHERE date != '' AND trail != ''
					GROUP BY trail
				) AS oldest_logs
				WHERE trails.id = oldest_logs.trail
					AND trails.completed = TRUE
					AND COALESCE(trails.completed_at, '') = ''
			`).
			Execute(); err != nil {
			return err
		}

		if _, err := app.DB().
			NewQuery(`
				UPDATE trails
				SET completed_at = COALESCE(NULLIF(updated, ''), created)
				WHERE completed = TRUE
					AND COALESCE(completed_at, '') = ''
			`).
			Execute(); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}

		indexes := collection.Indexes[:0]
		for _, index := range collection.Indexes {
			if index != completedAtIndex1786350000 {
				indexes = append(indexes, index)
			}
		}
		collection.Indexes = indexes
		collection.Fields.RemoveById("datecompleted1")

		return app.Save(collection)
	})
}

func containsString1786350000(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
