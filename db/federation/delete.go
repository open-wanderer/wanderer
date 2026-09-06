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
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func ActorFollowerInboxes(app core.App, actor *core.Record) ([]string, error) {
	if !actor.GetBool("is_local") {
		return nil, nil
	}

	return followerInboxes(app, actor.Id)
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

func CreateTrailDeleteActivity(app core.App, r *core.Record) error {
	if !r.GetBool("public") {
		// only broadcast the trail if it is public
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
	cc := author.GetString("iri") + "/followers"
	object := r.GetString("iri")

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
	activity.CC = pub.ItemCollection{pub.IRI(cc)}
	activity.Published = time.Now()

	recipients, err := followerInboxes(app, author.Id)
	if err != nil {
		return err
	}

	return PostActivity(app, author, activity, recipients)
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

	commentTrail, err := app.FindRecordById("trails", r.GetString("trail"))
	if err != nil {
		// The trail is gone too, so its own Delete already covers this comment.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	commentTrailAuthor, err := app.FindRecordById("activitypub_actors", commentTrail.GetString("author"))
	if err != nil {
		return err
	}

	if commentTrailAuthor.GetBool("is_local") {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := commentTrailAuthor.GetString("iri")
	object := r.GetString("iri")

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.Published = time.Now()

	err = PostActivity(app, author, activity, []string{to + "/inbox"})
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

	summitLogTrail, err := app.FindRecordById("trails", r.GetString("trail"))
	if err != nil {
		// The trail is gone too, so its own Delete already covers this log.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	summitLogTrailAuthor, err := app.FindRecordById("activitypub_actors", summitLogTrail.GetString("author"))
	if err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := summitLogTrailAuthor.GetString("iri")
	object := r.GetString("iri")
	cc := pub.ItemCollection{pub.IRI(author.GetString("iri") + "/followers")}

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()

	recipients, err := followerInboxes(app, author.Id)
	if err != nil {
		return err
	}

	if author.Id != summitLogTrailAuthor.Id {
		recipients = append(recipients, summitLogTrailAuthor.GetString("inbox"))
	}

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

func CreateListDeleteActivity(app core.App, r *core.Record) error {

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
	cc := author.GetString("iri") + "/followers"
	object := r.GetString("iri")

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = pub.ItemCollection{pub.IRI(cc)}
	activity.Published = time.Now()

	recipients, err := followerInboxes(app, author.Id)
	if err != nil {
		return err
	}

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
		err = processDeleteTrailActivity(app, activity)
	case util.ObjectKindComment:
		err = processDeleteCommentActivity(app, actor, activity)
	case util.ObjectKindSummitLog:
		err = processDeleteSummitLogActivity(app, actor, activity)
	case util.ObjectKindList:
		err = processDeleteListActivity(app, actor, activity)
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

func processDeleteTrailActivity(app core.App, activity pub.Activity) error {

	object := activity.Object.GetID().String()
	trail, err := app.FindFirstRecordByData("trails", "iri", object)
	if err != nil {
		return err
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

	err = util.DeleteFromFeed(app, list.Id)
	if err != nil {
		return err
	}

	return app.Delete(list)
}
