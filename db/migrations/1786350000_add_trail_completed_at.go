package migrations

import (
	"github.com/pocketbase/dbx"
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

		trails, err := app.FindRecordsByFilter("trails", "completed=true", "", -1, 0)
		if err != nil {
			return err
		}

		for _, trail := range trails {
			// An existing summit log is the best available completion date. For
			// explicitly completed trails without one, the last update is closer
			// to the completion toggle than the trail creation/planning date.
			completedAt := trail.GetDateTime("updated")
			if completedAt.IsZero() {
				completedAt = trail.GetDateTime("created")
			}

			logs, err := app.FindRecordsByFilter(
				"summit_logs",
				"trail={:trail}",
				"+date",
				1,
				0,
				dbx.Params{"trail": trail.Id},
			)
			if err != nil {
				return err
			}
			if len(logs) > 0 {
				completedAt = logs[0].GetDateTime("date")
			}

			trail.Set("completed_at", completedAt)
			if err := app.UnsafeWithoutHooks().Save(trail); err != nil {
				return err
			}
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
