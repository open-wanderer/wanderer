package federation

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The Create, Update and Announce activities recorded for a trail, list,
// comment or summit log embed a snapshot of it, and activitypub_activities
// is publicly listed (the outbox reads it). The recipient queries read the
// to and cc of the same rows back, also once the object is gone: an account's
// Delete(Actor) and a trail's Delete reach whoever a deleted reply mentioned
// only through them. So a snapshot is hidden while the object exists, and
// once it is deleted its content is dropped and the addressing kept.

// StripObjectActivities reduces the recorded Create, Update and Announce
// activities of a deleted object to its addressing. The object becomes a
// Tombstone that keeps only its id and inReplyTo, which the recipient queries
// match on, and the row is hidden for good. Call it after the object's Delete
// was built, so the Delete is addressed before anything changes.
func StripObjectActivities(app core.App, objectIRI string) error {
	if objectIRI == "" {
		return nil
	}
	_, err := app.DB().NewQuery(`
		UPDATE activitypub_activities
		SET hidden = TRUE,
			object = CASE
				WHEN json_extract(object, '$.inReplyTo') IS NULL
					THEN json_object('id', json_extract(object, '$.id'), 'type', 'Tombstone')
				ELSE json_object('id', json_extract(object, '$.id'), 'type', 'Tombstone', 'inReplyTo', json_extract(object, '$.inReplyTo'))
			END
		WHERE type IN ('Create', 'Update', 'Announce')
			AND json_extract(object, '$.id') = {:iri}
	`).Bind(dbx.Params{"iri": objectIRI}).Execute()
	return err
}

// SetObjectActivitiesHidden hides or shows the recorded Create and Update
// activities of a trail or list and of the comments and summit logs replying
// to it, as it goes private or public. Hidden rows stay readable to the
// recipient queries, so a later deletion still reaches whoever holds a copy.
// The stripped rows of deleted replies stay hidden, and so do Announces,
// which are hidden from the start (see CreateAnnounceActivity).
func SetObjectActivitiesHidden(app core.App, objectIRI string, hidden bool) error {
	if objectIRI == "" {
		return nil
	}
	_, err := app.DB().NewQuery(`
		UPDATE activitypub_activities
		SET hidden = {:hidden}
		WHERE type IN ('Create', 'Update')
			AND (json_extract(object, '$.id') = {:iri} OR json_extract(object, '$.inReplyTo') = {:iri})
			AND COALESCE(json_extract(object, '$.type'), '') != 'Tombstone'
	`).Bind(dbx.Params{"iri": objectIRI, "hidden": hidden}).Execute()
	return err
}
