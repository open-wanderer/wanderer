package federation

import (
	"database/sql"
	"errors"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// followerInboxes returns inbox URLs for all accepted followers of actorId
// in a single JOIN query instead of one query per follower.
func followerInboxes(app core.App, actorId string) ([]string, error) {
	rows, err := app.DB().
		Select("aa.inbox").
		From("follows f").
		InnerJoin("activitypub_actors aa", dbx.NewExp("f.follower = aa.id")).
		Where(dbx.NewExp("f.followee = {:followee} AND f.status = 'accepted' AND aa.inbox != ''",
			dbx.Params{"followee": actorId})).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inboxes []string
	for rows.Next() {
		var inbox string
		if err := rows.Scan(&inbox); err != nil {
			return nil, err
		}
		inboxes = append(inboxes, inbox)
	}
	return inboxes, rows.Err()
}

// actorDeleteInboxes returns the inboxes to notify when actorId is deleted:
//   - accepted followers
//   - actors it follows, any status
//   - authors of remote trails it commented on, logged a summit on, or liked
//   - remote actors who commented on, logged a summit on, or liked its trails
//   - the other party of every trail or list share it is involved in
func actorDeleteInboxes(app core.App, actorId string, actorIRI string) ([]string, error) {
	rows, err := app.DB().NewQuery(`
		SELECT aa.inbox
		FROM follows f
		INNER JOIN activitypub_actors aa ON f.follower = aa.id
		WHERE f.followee = {:actor} AND f.status = 'accepted' AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM follows f
		INNER JOIN activitypub_actors aa ON f.followee = aa.id
		WHERE f.follower = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM comments c
		INNER JOIN trails t ON c.trail = t.id
		INNER JOIN activitypub_actors aa ON t.author = aa.id
		WHERE c.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM summit_logs s
		INNER JOIN trails t ON s.trail = t.id
		INNER JOIN activitypub_actors aa ON t.author = aa.id
		WHERE s.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM trail_like l
		INNER JOIN trails t ON l.trail = t.id
		INNER JOIN activitypub_actors aa ON t.author = aa.id
		WHERE l.actor = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM comments c
		INNER JOIN trails t ON c.trail = t.id
		INNER JOIN activitypub_actors aa ON c.author = aa.id
		WHERE t.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM summit_logs s
		INNER JOIN trails t ON s.trail = t.id
		INNER JOIN activitypub_actors aa ON s.author = aa.id
		WHERE t.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM trail_like l
		INNER JOIN trails t ON l.trail = t.id
		INNER JOIN activitypub_actors aa ON l.actor = aa.id
		WHERE t.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM trail_share ts
		INNER JOIN trails t ON ts.trail = t.id
		INNER JOIN activitypub_actors aa ON aa.id = ts.actor
		WHERE t.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM trail_share ts
		INNER JOIN trails t ON ts.trail = t.id
		INNER JOIN activitypub_actors aa ON aa.id = t.author
		WHERE ts.actor = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM list_share ls
		INNER JOIN lists li ON ls.list = li.id
		INNER JOIN activitypub_actors aa ON aa.id = ls.actor
		WHERE li.author = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM list_share ls
		INNER JOIN lists li ON ls.list = li.id
		INNER JOIN activitypub_actors aa ON aa.id = li.author
		WHERE ls.actor = {:actor} AND aa.is_local = 0 AND aa.inbox != ''
		UNION
		SELECT aa.inbox
		FROM activitypub_activities ac, json_each(ac.cc) recipient
		INNER JOIN activitypub_actors aa ON aa.inbox = recipient.value
		WHERE ac.actor = {:iri} AND ac.type IN ('Create', 'Update') AND aa.is_local = 0 AND aa.inbox != ''
	`).Bind(dbx.Params{"actor": actorId, "iri": actorIRI}).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inboxes []string
	for rows.Next() {
		var inbox string
		if err := rows.Scan(&inbox); err != nil {
			return nil, err
		}
		inboxes = append(inboxes, inbox)
	}
	return inboxes, rows.Err()
}

// DeleteAudience is who a retraction of a trail or list goes to. Followers
// are addressed as the author's followers collection and never listed by
// inbox on the wire; Holders received the object for another reason and are
// addressed individually.
type DeleteAudience struct {
	Followers []string
	Holders   []string
}

// Inboxes returns every inbox to deliver to, remote and without duplicates.
func (a DeleteAudience) Inboxes() []string {
	return remoteInboxes(append(append([]string{}, a.Followers...), a.Holders...))
}

func (a DeleteAudience) Empty() bool {
	return len(a.Followers) == 0 && len(a.Holders) == 0
}

// TrailDeleteRecipients returns whom to tell that a trail is no longer
// available: its followers if it was public, plus everyone who provably
// holds a copy of it for another reason (see holderInboxes). public says
// whether the trail was public when it was handed out; a trail that just
// went private still has to reach the followers it was public to.
//
// Only a local author's trail is ours to retract. Call this before the trail
// is deleted: the comment, summit log, like and share rows it reads are
// cascaded away with the trail.
func TrailDeleteRecipients(app core.App, trail *core.Record, public bool) (DeleteAudience, error) {
	return deleteRecipients(app, trail, public, trailHolderQuery)
}

// ListDeleteRecipients is TrailDeleteRecipients for a list.
func ListDeleteRecipients(app core.App, list *core.Record, public bool) (DeleteAudience, error) {
	return deleteRecipients(app, list, public, listHolderQuery)
}

func deleteRecipients(app core.App, record *core.Record, public bool, holders string) (DeleteAudience, error) {
	var audience DeleteAudience

	author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return audience, nil
		}
		return audience, err
	}
	if !author.GetBool("is_local") {
		return audience, nil
	}

	if public {
		audience.Followers, err = followerInboxes(app, author.Id)
		if err != nil {
			return audience, err
		}
	}

	others, err := holderInboxes(app, holders, record.Id, record.GetString("iri"))
	if err != nil {
		return audience, err
	}
	audience.Holders = remoteInboxes(others)

	return audience, nil
}

// trailHolderQuery selects the remote inboxes holding a copy of a trail for a
// reason other than following its author:
//   - remote actors who commented on, logged a summit on, or liked it
//   - remote actors it is shared with
//   - actors it mentioned, recorded in the cc of its Create and Update
//   - actors a local comment or summit log on it mentioned, recorded in the
//     cc of that reply's Create and Update; a mentioned actor's instance
//     fetches the trail the reply is to
//   - actors it was announced to, recorded in the to of its Announce, which
//     still counts a share that has since been revoked
const trailHolderQuery = `
	SELECT aa.inbox
	FROM comments c
	INNER JOIN activitypub_actors aa ON c.author = aa.id
	WHERE c.trail = {:id} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM summit_logs s
	INNER JOIN activitypub_actors aa ON s.author = aa.id
	WHERE s.trail = {:id} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM trail_like l
	INNER JOIN activitypub_actors aa ON l.actor = aa.id
	WHERE l.trail = {:id} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM trail_share ts
	INNER JOIN activitypub_actors aa ON ts.actor = aa.id
	WHERE ts.trail = {:id} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM activitypub_activities ac, json_each(ac.cc) recipient
	INNER JOIN activitypub_actors aa ON aa.inbox = recipient.value
	WHERE ac.type IN ('Create', 'Update') AND json_extract(ac.object, '$.id') = {:iri} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM activitypub_activities ac, json_each(ac.cc) recipient
	INNER JOIN activitypub_actors aa ON aa.inbox = recipient.value
	WHERE ac.type IN ('Create', 'Update') AND json_extract(ac.object, '$.inReplyTo') = {:iri} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM activitypub_activities ac, json_each(ac.[to]) recipient
	INNER JOIN activitypub_actors aa ON aa.iri = recipient.value
	WHERE ac.type = 'Announce' AND json_extract(ac.object, '$.id') = {:iri} AND aa.is_local = 0 AND aa.inbox != ''
`

// listHolderQuery is trailHolderQuery for a list, which is only ever handed
// out by sharing.
const listHolderQuery = `
	SELECT aa.inbox
	FROM list_share ls
	INNER JOIN activitypub_actors aa ON ls.actor = aa.id
	WHERE ls.list = {:id} AND aa.is_local = 0 AND aa.inbox != ''
	UNION
	SELECT aa.inbox
	FROM activitypub_activities ac, json_each(ac.[to]) recipient
	INNER JOIN activitypub_actors aa ON aa.iri = recipient.value
	WHERE ac.type = 'Announce' AND json_extract(ac.object, '$.id') = {:iri} AND aa.is_local = 0 AND aa.inbox != ''
`

func holderInboxes(app core.App, query, id, iri string) ([]string, error) {
	rows, err := app.DB().NewQuery(query).Bind(dbx.Params{"id": id, "iri": iri}).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inboxes []string
	for rows.Next() {
		var inbox string
		if err := rows.Scan(&inbox); err != nil {
			return nil, err
		}
		inboxes = append(inboxes, inbox)
	}
	return inboxes, rows.Err()
}

// recordedInboxes returns every inbox an object was sent to, read back from
// the cc of its recorded Create and Update activities. Mentioned actors are
// addressed there without any follow relationship, and an object edited to
// mention someone new went to both audiences, so all of them count.
func recordedInboxes(app core.App, objectIRI string) ([]string, error) {
	records, err := app.FindRecordsByFilter(
		"activitypub_activities",
		"(type = 'Create' || type = 'Update') && object.id = {:iri}",
		"", 0, 0,
		dbx.Params{"iri": objectIRI},
	)
	if err != nil {
		return nil, err
	}

	var inboxes []string
	for _, record := range records {
		inboxes = append(inboxes, record.GetStringSlice("cc")...)
	}
	return inboxes, nil
}

// remoteInboxes drops duplicates and anything on this instance. A Delete to
// our own inbox would only describe a record that is already gone.
func remoteInboxes(inboxes []string) []string {
	seen := make(map[string]struct{}, len(inboxes))
	out := make([]string, 0, len(inboxes))
	for _, inbox := range inboxes {
		if inbox == "" || util.IsLocalIRI(inbox) {
			continue
		}
		if _, dup := seen[inbox]; dup {
			continue
		}
		seen[inbox] = struct{}{}
		out = append(out, inbox)
	}
	return out
}
