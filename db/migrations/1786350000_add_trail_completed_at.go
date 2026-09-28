package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	completedAtIndex1786350000          = "CREATE INDEX idx_trails_completed_at ON trails (author, completed_at) WHERE completed = 1;"
	legacyCompletedAtIndex1786350000    = "CREATE INDEX idx_trails_completed_at ON trails (completed_at) WHERE completed = true;"
	summitLogTrailAuthorIndex1786350000 = "CREATE INDEX idx_summit_logs_trail_author ON summit_logs (trail, author);"
	summitLogAuthorDateIndex1786350000  = "CREATE INDEX idx_summit_logs_author_date ON summit_logs (author, date);"

	backfillCompletedAtFromSummitLogs1786350000 = `
		UPDATE trails
		SET completed_at = oldest_logs.completed_at
		FROM (
			SELECT trail, author, MIN(date) AS completed_at
			FROM summit_logs
			WHERE date != '' AND trail != '' AND author != ''
			GROUP BY trail, author
		) AS oldest_logs
		WHERE trails.id = oldest_logs.trail
			AND trails.author = oldest_logs.author
			AND trails.completed = TRUE
			AND COALESCE(trails.completed_at, '') = ''
	`

	backfillCompletedAtFromTrail1786350000 = `
		UPDATE trails
		SET completed_at = COALESCE(NULLIF(updated, ''), created)
		WHERE completed = TRUE
			AND COALESCE(completed_at, '') = ''
	`
)

func init() {
	m.Register(upAddTrailCompletedAt1786350000, downAddTrailCompletedAt1786350000)
}

func upAddTrailCompletedAt1786350000(app core.App) error {
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		return err
	}

	if trails.Fields.GetByName("completed_at") == nil {
		if err := trails.Fields.AddMarshaledJSONAt(6, []byte(`{
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

	if !containsString1786350000(trails.Indexes, completedAtIndex1786350000) {
		trails.Indexes = append(trails.Indexes, completedAtIndex1786350000)
	}
	if err := app.Save(trails); err != nil {
		return err
	}

	summitLogs, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		return err
	}
	for _, index := range []string{
		summitLogTrailAuthorIndex1786350000,
		summitLogAuthorDateIndex1786350000,
	} {
		if !containsString1786350000(summitLogs.Indexes, index) {
			summitLogs.Indexes = append(summitLogs.Indexes, index)
		}
	}
	if err := app.Save(summitLogs); err != nil {
		return err
	}

	// Preserve the owner's oldest summit-log date as completed_at metadata for
	// trail views and possible future fallback consumers. Statistics suppress
	// the completed-trail fallback whenever the owner has a summit log; logs by
	// other users don't affect that precedence. Whatever is still blank falls
	// back to the last update, which is closer to the completion toggle than
	// the trail creation/planning date.
	if _, err := app.DB().
		NewQuery(backfillCompletedAtFromSummitLogs1786350000).
		Execute(); err != nil {
		return err
	}

	if _, err := app.DB().
		NewQuery(backfillCompletedAtFromTrail1786350000).
		Execute(); err != nil {
		return err
	}

	return nil
}

func downAddTrailCompletedAt1786350000(app core.App) error {
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		return err
	}
	trails.Indexes = removeIndexes1786350000(
		trails.Indexes,
		completedAtIndex1786350000,
		// Allows databases created from an earlier, unpublished draft of this
		// migration to roll back before applying the final version.
		legacyCompletedAtIndex1786350000,
	)
	trails.Fields.RemoveById("datecompleted1")
	if err := app.Save(trails); err != nil {
		return err
	}

	summitLogs, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		return err
	}
	summitLogs.Indexes = removeIndexes1786350000(
		summitLogs.Indexes,
		summitLogTrailAuthorIndex1786350000,
		summitLogAuthorDateIndex1786350000,
	)
	return app.Save(summitLogs)
}

func removeIndexes1786350000(indexes []string, remove ...string) []string {
	result := make([]string, 0, len(indexes))
	for _, index := range indexes {
		if !containsString1786350000(remove, index) {
			result = append(result, index)
		}
	}
	return result
}

func containsString1786350000(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
