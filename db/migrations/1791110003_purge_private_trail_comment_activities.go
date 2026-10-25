package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Comments on private trails used to be federated like any other. The
// Create/Update activities recorded for them carry the comment text and are
// publicly listed on the author's outbox, so they are removed. They are found
// through the trail they reply to rather than through the comment, which also
// covers comments deleted since: deleting one only records a Delete and
// leaves its Create/Update in place. Summit logs on these trails are matched
// the same way. Copies already delivered to mentioned remote actors are not
// retracted.
const purgePrivateTrailCommentActivities1791110003 = `
	DELETE FROM activitypub_activities
	WHERE type IN ('Create', 'Update')
		AND json_extract(object, '$.inReplyTo') IN (
			SELECT iri FROM trails WHERE public = FALSE AND iri != ''
		)
`

func init() {
	m.Register(purgePrivateTrailCommentActivitiesUp1791110003, nil)
}

func purgePrivateTrailCommentActivitiesUp1791110003(app core.App) error {
	_, err := app.DB().NewQuery(purgePrivateTrailCommentActivities1791110003).Execute()
	return err
}
