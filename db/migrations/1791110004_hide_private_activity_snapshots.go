package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Recorded Create, Update and Announce activities embed a snapshot of their
// object, and activitypub_activities is publicly listed. A hidden flag takes
// the snapshots of private trails and lists, and of the replies to them, out
// of that listing, and every Announce: a share is addressed to its recipient
// alone and tells who it went to with which permission. The rows stay, since
// the recipient queries read their to and cc back when the object is deleted
// later. Snapshots of objects deleted before
// deletions stripped them are stripped now: the content goes, the addressing
// stays, since an account's Delete(Actor) and a trail's Delete still read the
// to and cc of a deleted reply (see federation.StripObjectActivities).

const hidePrivateActivitySnapshots1791110004 = `
	UPDATE activitypub_activities
	SET hidden = TRUE
	WHERE type = 'Announce'
		OR (
			type IN ('Create', 'Update')
			AND (
				json_extract(object, '$.id') IN (
					SELECT iri FROM trails WHERE public = FALSE AND iri != ''
					UNION SELECT iri FROM lists WHERE public = FALSE AND iri != ''
				)
				OR json_extract(object, '$.inReplyTo') IN (
					SELECT iri FROM trails WHERE public = FALSE AND iri != ''
				)
			)
		)
`

// Only trails, lists, comments and summit logs are ever recorded as the
// object of these types. A summit log without an IRI was recorded with an
// empty id and is left alone.
const stripDeletedActivitySnapshots1791110004 = `
	UPDATE activitypub_activities
	SET hidden = TRUE,
		object = CASE
			WHEN json_extract(object, '$.inReplyTo') IS NULL
				THEN json_object('id', json_extract(object, '$.id'), 'type', 'Tombstone')
			ELSE json_object('id', json_extract(object, '$.id'), 'type', 'Tombstone', 'inReplyTo', json_extract(object, '$.inReplyTo'))
		END
	WHERE type IN ('Create', 'Update', 'Announce')
		AND COALESCE(json_extract(object, '$.id'), '') != ''
		AND json_extract(object, '$.id') NOT IN (
			SELECT iri FROM trails WHERE iri != ''
			UNION SELECT iri FROM lists WHERE iri != ''
			UNION SELECT iri FROM comments WHERE iri != ''
			UNION SELECT iri FROM summit_logs WHERE iri != ''
		)
`

func init() {
	m.Register(hidePrivateActivitySnapshotsUp1791110004, hidePrivateActivitySnapshotsDown1791110004)
}

func hidePrivateActivitySnapshotsUp1791110004(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}
	collection.Fields.Add(&core.BoolField{Name: "hidden"})
	collection.ListRule = types.Pointer("hidden = false")
	collection.ViewRule = types.Pointer("hidden = false")
	if err := app.Save(collection); err != nil {
		return err
	}

	for _, query := range []string{hidePrivateActivitySnapshots1791110004, stripDeletedActivitySnapshots1791110004} {
		if _, err := app.DB().NewQuery(query).Execute(); err != nil {
			return err
		}
	}
	return nil
}

func hidePrivateActivitySnapshotsDown1791110004(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}
	collection.Fields.RemoveByName("hidden")
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	return app.Save(collection)
}
