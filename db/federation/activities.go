package federation

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The Create, Update and Announce activities recorded for a trail, list,
// comment or summit log embed a snapshot of it, and activitypub_activities
// is publicly listed (the outbox reads it). The recipient queries read the
// cc of the same rows back, so a snapshot is hidden while the object exists
// and only removed once its own Delete has gone out.

// DeleteObjectActivities removes the recorded Create, Update and Announce
// activities of an object that was deleted. Call it after the object's
// Delete was built, which reads its audience back from these rows. Replies
// remove their own rows when they are deleted, so a trail's do not go here.
func DeleteObjectActivities(app core.App, objectIRI string) error {
	if objectIRI == "" {
		return nil
	}
	_, err := app.DB().NewQuery(`
		DELETE FROM activitypub_activities
		WHERE type IN ('Create', 'Update', 'Announce')
			AND json_extract(object, '$.id') = {:iri}
	`).Bind(dbx.Params{"iri": objectIRI}).Execute()
	return err
}

// SetObjectActivitiesHidden hides or shows the recorded Create, Update and
// Announce activities of a trail or list and of the comments and summit logs
// replying to it, as it goes private or public. Hidden rows stay readable to
// the recipient queries, so a later deletion still reaches whoever holds a
// copy.
func SetObjectActivitiesHidden(app core.App, objectIRI string, hidden bool) error {
	if objectIRI == "" {
		return nil
	}
	_, err := app.DB().NewQuery(`
		UPDATE activitypub_activities
		SET hidden = {:hidden}
		WHERE type IN ('Create', 'Update', 'Announce')
			AND (json_extract(object, '$.id') = {:iri} OR json_extract(object, '$.inReplyTo') = {:iri})
	`).Bind(dbx.Params{"iri": objectIRI, "hidden": hidden}).Execute()
	return err
}
