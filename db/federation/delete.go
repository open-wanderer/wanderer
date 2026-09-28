package federation

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"pocketbase/util"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// ActorDeleteRecipients returns every inbox that should be told a local actor
// is gone (see actorDeleteInboxes). Only local actors are ours to announce.
func ActorDeleteRecipients(app core.App, actor *core.Record) ([]string, error) {
	if !actor.GetBool("is_local") {
		return nil, nil
	}

	return actorDeleteInboxes(app, actor.Id, actor.GetString("iri"))
}

// DeleteActorActivities removes every activity record the actor produced:
// the Create/Update/Delete/Announce rows that embed snapshots of its content
// and are listed on its public outbox. Nothing references these rows through
// a relation, so the cascade does not reach them. Call this after
// ActorDeleteRecipients, which reads its audience back from these rows, and
// inside the deleting transaction, so a rollback restores them.
func DeleteActorActivities(app core.App, actorIRI string) error {
	if actorIRI == "" {
		return nil
	}
	_, err := app.DB().
		Delete("activitypub_activities", dbx.HashExp{"actor": actorIRI}).
		Execute()
	return err
}

func CreateActorDeleteActivity(app core.App, actor *core.Record, recipients []string) error {
	if !actor.GetBool("is_local") {
		// A remote actor being dropped locally is our own bookkeeping, not
		// something to broadcast back out to the network.
		return nil
	}

	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := "https://www.w3.org/ns/activitystreams#Public"
	cc := actor.GetString("iri") + "/followers"
	object := actor.GetString("iri")

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.DeleteType))
	record.Set("to", to)
	record.Set("cc", cc)
	record.Set("object", object)
	record.Set("actor", actor.GetString("iri"))
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(object)
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = pub.ItemCollection{pub.IRI(cc)}
	activity.Published = time.Now()

	return PostActivity(app, actor, activity, recipients)
}

// CreateTrailDeleteActivity tells the audience, as collected by
// TrailDeleteRecipients before the trail's rows were cascaded away, that the
// trail is no longer available. The trail need not be public any more: a
// trail that was handed out while public is retracted from whoever got it.
// A trail nobody ever received is not announced at all.
func CreateTrailDeleteActivity(app core.App, r *core.Record, audience DeleteAudience) error {
	if !r.GetBool("public") && audience.Empty() {
		return nil
	}
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	author, err := app.FindRecordById("activitypub_actors", r.GetString("author"))
	if err != nil {
		// The author is gone too, so this trail was removed as part of that
		// account's own cascade. There is no local actor left to attribute a
		// Delete to; the account's own Delete(Actor) is what carries the news.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if !author.GetBool("is_local") {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := "https://www.w3.org/ns/activitystreams#Public"
	object := r.GetString("iri")

	// As on the Create: followers as their collection, everyone else by
	// inbox. Listing follower inboxes would hand the follower list to every
	// recipient.
	cc := pub.ItemCollection{pub.IRI(author.GetString("iri") + "/followers")}
	for _, inbox := range audience.Holders {
		cc.Append(pub.IRI(inbox))
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.DeleteType))
	record.Set("to", to)
	record.Set("cc", cc)
	record.Set("object", object)
	record.Set("actor", author.GetString("iri"))
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()

	return PostActivity(app, author, activity, audience.Inboxes())
}

func CreateCommentDeleteActivity(app core.App, client meilisearch.ServiceManager, r *core.Record) error {

	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	author, err := app.FindRecordById("activitypub_actors", r.GetString("author"))
	if err != nil {
		// The author is gone too, so this comment was removed as part of that
		// account's own cascade. There is no local actor left to attribute a
		// Delete to, so there is nothing to send.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if !author.GetBool("is_local") {
		return nil
	}

	object := r.GetString("iri")

	// Everyone the comment text was sent to. The Create and any Update
	// activities recorded the inboxes they went out to, mentioned actors and
	// trail author alike, so the audience is read back rather than derived
	// again, and it does not depend on the trail still existing.
	recipients, err := recordedInboxes(app, object)
	if err != nil {
		return err
	}

	to := "https://www.w3.org/ns/activitystreams#Public"

	commentTrail, err := app.FindRecordById("trails", r.GetString("trail"))
	switch {
	case err == nil:
		commentTrailAuthor, err := app.FindRecordById("activitypub_actors", commentTrail.GetString("author"))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			to = commentTrailAuthor.GetString("iri")
			recipients = append(recipients, commentTrailAuthor.GetString("inbox"))
		}
	case errors.Is(err, sql.ErrNoRows):
		// The trail went first and took this comment with it. Its author can
		// no longer be looked up, but the recorded audience still stands.
	default:
		return err
	}

	recipients = remoteInboxes(recipients)
	if len(recipients) == 0 {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.Published = time.Now()

	err = PostActivity(app, author, activity, recipients)
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.DeleteType))
	record.Set("to", to)
	record.Set("object", object)
	record.Set("actor", author.GetString("iri"))
	record.Set("published", time.Now())

	return app.Save(record)
}

func CreateSummitLogDeleteActivity(app core.App, r *core.Record) error {

	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	author, err := app.FindRecordById("activitypub_actors", r.GetString("author"))
	if err != nil {
		// The author is gone too, so this summit log was removed as part of that
		// account's own cascade. There is no local actor left to attribute a
		// Delete to, so there is nothing to send.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if !author.GetBool("is_local") {
		return nil
	}

	// The log was handed to this author's followers when it was created, so
	// they are told regardless of what happened to the trail. So were the
	// actors its text mentioned, whose inboxes the Create and Update
	// activities recorded.
	recipients, err := followerInboxes(app, author.Id)
	if err != nil {
		return err
	}
	mentioned, err := recordedInboxes(app, r.GetString("iri"))
	if err != nil {
		return err
	}
	mentioned = remoteInboxes(mentioned)
	recipients = append(recipients, mentioned...)

	to := "https://www.w3.org/ns/activitystreams#Public"

	summitLogTrail, err := app.FindRecordById("trails", r.GetString("trail"))
	switch {
	case err == nil:
		summitLogTrailAuthor, err := app.FindRecordById("activitypub_actors", summitLogTrail.GetString("author"))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			to = summitLogTrailAuthor.GetString("iri")
			if author.Id != summitLogTrailAuthor.Id {
				recipients = append(recipients, summitLogTrailAuthor.GetString("inbox"))
			}
		}
	case errors.Is(err, sql.ErrNoRows):
		// The trail went first and took this log with it. Its author can no
		// longer be addressed, but the trail's own Delete only reaches the
		// trail author's followers, not this author's. They still need to hear.
	default:
		return err
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	object := r.GetString("iri")
	cc := pub.ItemCollection{pub.IRI(author.GetString("iri") + "/followers")}
	for _, inbox := range mentioned {
		cc.Append(pub.IRI(inbox))
	}

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()

	err = PostActivity(app, author, activity, recipients)
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.DeleteType))
	record.Set("to", to)
	record.Set("cc", cc)
	record.Set("object", object)
	record.Set("actor", author.GetString("iri"))
	record.Set("published", time.Now())

	return app.Save(record)
}

// CreateListDeleteActivity is CreateTrailDeleteActivity for a list, with
// the audience from ListDeleteRecipients.
func CreateListDeleteActivity(app core.App, r *core.Record, audience DeleteAudience) error {
	if !r.GetBool("public") && audience.Empty() {
		return nil
	}
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	author, err := app.FindRecordById("activitypub_actors", r.GetString("author"))
	if err != nil {
		// The author is gone too, so this list was removed as part of that
		// account's own cascade. There is no local actor left to attribute a
		// Delete to; the account's own Delete(Actor) is what carries the news.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if !author.GetBool("is_local") {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := "https://www.w3.org/ns/activitystreams#Public"
	object := r.GetString("iri")

	cc := pub.ItemCollection{pub.IRI(author.GetString("iri") + "/followers")}
	for _, inbox := range audience.Holders {
		cc.Append(pub.IRI(inbox))
	}

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()

	err = PostActivity(app, author, activity, audience.Inboxes())
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.DeleteType))
	record.Set("to", to)
	record.Set("cc", cc)
	record.Set("object", object)
	record.Set("actor", author.GetString("iri"))
	record.Set("published", time.Now())

	return app.Save(record)
}

func ProcessDeleteActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	// no need to do anything if the actor is local
	if actor.GetBool("is_local") {
		return nil
	}

	object := activity.Object.GetID().String()

	var err error
	switch util.ObjectKindFromIRI(object) {
	case util.ObjectKindActor:
		err = processDeleteActorActivity(app, actor, activity)
	case util.ObjectKindTrail:
		err = processDeleteTrailActivity(app, actor, activity)
	case util.ObjectKindComment:
		err = processDeleteCommentActivity(app, actor, activity)
	case util.ObjectKindSummitLog:
		err = processDeleteSummitLogActivity(app, actor, activity)
	case util.ObjectKindList:
		err = processDeleteListActivity(app, actor, activity)
	default:
		// Mirrors ProcessCreateOrUpdateActivity: anything else was accepted
		// as a comment, so a reply from other ActivityPub software is
		// retracted the same way. One we never stored is nothing to do.
		err = processDeleteCommentActivity(app, actor, activity)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// processDeleteActorActivity handles a remote account telling us it has been
// deleted, and removes our copy of it. Everything that account authored here is
// carried away by the schema's cascade.
//
// This is the most dangerous activity we accept, because it destroys a whole
// account's content rather than one record, so it is deliberately narrow: the
// only thing it permits is an actor deleting *itself*.
//
// The authentication happened before we got here. ActivitypubActivityProcess
// resolves the actor named in activity.Actor and verifies the request's HTTP
// signature against that actor's public key, so the record handed to us is the
// cryptographically authenticated sender and nobody else — an attacker cannot
// name a victim as the Actor without holding the victim's private key.
//
// What remains is to make sure the sender is not asking us to delete somebody
// other than itself, which is exactly the "delete a stranger's account by
// asking" case. The object must therefore be the signer, compared both against
// the IRI we authenticated and the one on the wire. Anything else is refused
// and logged rather than acted on.
func processDeleteActorActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	object := activity.Object.GetID().String()
	signer := actor.GetString("iri")
	claimed := activity.Actor.GetID().String()

	if object == "" || signer == "" || claimed == "" || object != signer || object != claimed {
		app.Logger().Warn(
			"refused federated actor deletion naming an account other than the signer",
			"object", object, "signer", signer, "claimed_actor", claimed,
		)
		return fmt.Errorf("refusing Delete of actor %q signed by %q: an actor may only delete itself", object, signer)
	}

	// A local account must never be removable by a federated message, whoever
	// signed it. ProcessDeleteActivity already returns early for a local
	// signer; this repeats the guarantee where the deletion actually happens,
	// so it cannot be lost by a later change to the dispatch above.
	if actor.GetBool("is_local") || util.IsLocalIRI(object) {
		app.Logger().Warn(
			"refused federated deletion of a local actor",
			"object", object, "signer", signer,
		)
		return fmt.Errorf("refusing federated Delete of local actor %q", object)
	}

	return app.Delete(actor)
}

func processDeleteTrailActivity(app core.App, actor *core.Record, activity pub.Activity) error {

	object := activity.Object.GetID().String()
	trail, err := app.FindFirstRecordByData("trails", "iri", object)
	if err != nil {
		return err
	}

	if trail.GetString("author") != actor.Id {
		return fmt.Errorf("actor is not trail author")
	}

	err = util.DeleteFromFeed(app, trail.Id)
	if err != nil {
		return err
	}

	return app.Delete(trail)
}

func processDeleteCommentActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	object := activity.Object.GetID().String()

	comment, err := app.FindFirstRecordByData("comments", "iri", object)
	if err != nil {
		return err
	}

	if comment.GetString("author") != actor.Id {
		return fmt.Errorf("actor is not comment author")
	}

	err = app.Delete(comment)
	if err != nil {
		return err
	}
	return nil
}

func processDeleteSummitLogActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	object := activity.Object.GetID().String()

	summitLog, err := app.FindFirstRecordByData("summit_logs", "iri", object)
	if err != nil {
		return err
	}

	if summitLog.GetString("author") != actor.Id {
		return fmt.Errorf("actor is not summit log author")
	}

	return app.Delete(summitLog)
}

func processDeleteListActivity(app core.App, actor *core.Record, activity pub.Activity) error {

	object := activity.Object.GetID().String()
	list, err := app.FindFirstRecordByData("lists", "iri", object)
	if err != nil {
		return err
	}

	if list.GetString("author") != actor.Id {
		return fmt.Errorf("actor is not list author")
	}

	err = util.DeleteFromFeed(app, list.Id)
	if err != nil {
		return err
	}

	return app.Delete(list)
}
