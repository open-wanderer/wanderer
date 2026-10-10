package federation

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// create outgoing follow activity
func CreateUnfollowActivity(app core.App, follow *core.Record) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	follower := follow.GetString("follower")
	followee := follow.GetString("followee")

	followerActor, err := app.FindRecordById("activitypub_actors", follower)
	if err != nil {
		return err
	}

	followeeActor, err := app.FindRecordById("activitypub_actors", followee)
	if err != nil {
		return err
	}

	// find the original follow activity
	followActivityRecord, err := findFollowActivityRecord(app, follow, followerActor.GetString("iri"), followeeActor.GetString("iri"))
	if err != nil {
		return err
	}
	followActivity := pub.FollowNew(pub.IRI(followActivityRecord.GetString("iri")), pub.IRI(followeeActor.GetString("iri")))
	followActivity.Actor = pub.IRI(followActivityRecord.GetString("actor"))

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.UndoType))
	record.Set("object", followActivity)
	record.Set("actor", followerActor.GetString("iri"))
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}

	activity := pub.UndoNew(pub.IRI(id), followActivity)
	activity.Actor = pub.IRI(followerActor.GetString("iri"))

	return PostActivity(app, followerActor, activity, []string{followeeActor.GetString("inbox")})
}

// create outgoing unlike activity
func CreateUnlikeActivity(app core.App, like *core.Record) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	actor, err := app.FindRecordById("activitypub_actors", like.GetString("actor"))
	if err != nil {
		return err
	}

	trail, err := app.FindRecordById("trails", like.GetString("trail"))
	if err != nil {
		return err
	}

	trailAuthor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return err
	}

	object := trail.GetString("iri")

	// find the original follow activity
	likeActivityRecord, err := app.FindFirstRecordByFilter("activitypub_activities", "actor={:actor}&&object={:object}&&type={:type}", dbx.Params{"actor": actor.GetString("iri"), "object": object, "type": string(pub.LikeType)})
	if err != nil {
		return err
	}
	likeActivity := pub.LikeNew(pub.IRI(likeActivityRecord.GetString("iri")), pub.IRI(object))
	likeActivity.Actor = pub.IRI(likeActivityRecord.GetString("actor"))

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.UndoType))
	record.Set("object", likeActivity)
	record.Set("actor", actor.GetString("iri"))
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}

	activity := pub.UndoNew(pub.IRI(id), likeActivity)
	activity.Actor = pub.IRI(actor.GetString("iri"))

	recipients := []string{trailAuthor.GetString("inbox")}

	// Also deliver to peer instances; public trails only.
	if trail.GetBool("public") {
		peerInboxes, err := instanceFollowerInboxes(app)
		if err != nil {
			return err
		}
		recipients = append(recipients, peerInboxes...)
	}

	return PostActivity(app, actor, activity, recipients)
}

func ProcessUndoActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	if activity.Object == nil {
		return fmt.Errorf("undo: missing object")
	}

	if activity.Object.GetType() == pub.FollowType {
		return processUnfollowActivity(app, actor, activity)
	} else if activity.Object.GetType() == pub.LikeType {
		return processUnlikeActivity(app, actor, activity)
	} else {
		return fmt.Errorf("unknown undo activity object type")
	}
}

// processUnfollowActivity removes the follows row an Undo{Follow} refers to,
// in one transaction.
func processUnfollowActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	// this was a local follow
	if actor.GetBool("is_local") {
		return nil
	}

	followActivity, ok := activity.Object.(*pub.Activity)
	if !ok {
		return fmt.Errorf("undo: follow object is not *pub.Activity")
	}

	followee, err := app.FindFirstRecordByData("activitypub_actors", "iri", followActivity.Object)
	if err != nil {
		return err
	}

	return app.RunInTransaction(func(txApp core.App) error {
		follow, err := txApp.FindFirstRecordByFilter("follows", "follower={:follower} && followee={:followee}", dbx.Params{"follower": actor.Id, "followee": followee.Id})
		if err != nil {
			// The Undo overtook its Follow.
			if errors.Is(err, sql.ErrNoRows) && isLocalInstanceRecipient(followee) {
				return rememberUndoneFollow(txApp, actor, followee, followActivity)
			}
			return err
		}

		// An Undo for a superseded Follow does not remove the newer one.
		if current := follow.GetString("activity_iri"); isSupersededFollow(current, followActivity) {
			txApp.Logger().Info("ignoring Undo for a superseded Follow",
				"undone", followActivity.GetID().String(), "current", current)
			// It may also be a newer Follow that has not arrived yet.
			if isLocalInstanceRecipient(followee) {
				return rememberUndoneFollow(txApp, actor, followee, followActivity)
			}
			return nil
		}

		return txApp.Delete(follow)
	})
}

// rememberUndoneFollow stores an undone instance Follow that has not arrived
// yet, so its late delivery is dropped as a replay instead of opening a
// request its sender has cancelled. Only the sender's host may name it.
func rememberUndoneFollow(txApp core.App, actor, followee *core.Record, follow *pub.Activity) error {
	iri := follow.GetID().String()
	if iri == "" || !sameHost(iri, actor.GetString("iri")) {
		return nil
	}
	if _, err := txApp.FindFirstRecordByData("activitypub_activities", "iri", iri); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	txApp.Logger().Info("remembering a Follow undone before it arrived", "follow", iri, "actor", actor.GetString("iri"))
	return storeFollowActivity(txApp, actor, followee, *follow)
}

func processUnlikeActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	if actor.GetBool("is_local") {
		return nil
	}

	likeActivity, ok := activity.Object.(*pub.Activity)
	if !ok {
		return fmt.Errorf("undo: like object is not *pub.Activity")
	}
	if likeActivity == nil || likeActivity.Object == nil {
		return fmt.Errorf("undo: like is missing its object")
	}

	trail, err := app.FindFirstRecordByData("trails", "iri", likeActivity.Object.GetID().String())
	if err != nil {
		// The receiver does not hold the trail, so there is no like to remove.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	like, err := app.FindFirstRecordByFilter("trail_like", "actor={:actor} && trail={:trail}", dbx.Params{"actor": actor.Id, "trail": trail.Id})
	if err != nil {
		// Already removed by the twin delivery, or never arrived.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	return app.Delete(like)
}
