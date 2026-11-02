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
	return processUndoActivity(app, actor, activity, nil)
}

var errUndoSuperseded = errors.New("undo superseded by a later action")

func processUndoActivity(app core.App, actor *core.Record, activity pub.Activity, superseded func() bool) error {
	object, ok := activity.Object.(*pub.Activity)
	if !ok || object == nil {
		return fmt.Errorf("undo object must be an activity")
	}
	if activity.Object.GetType() == pub.FollowType {
		return undoFollow(app, actor, activity, superseded)
	} else if activity.Object.GetType() == pub.LikeType {
		return undoLike(app, actor, activity, superseded)
	} else {
		return fmt.Errorf("unknown undo activity object type")
	}
}

// A later action noted before the check keeps the record; one noted after it
// saves only after the removal releases the writer.
func removeUndoneRecord(app core.App, record *core.Record, superseded func() bool) error {
	if superseded == nil {
		return app.Delete(record)
	}
	return app.RunInTransaction(func(tx core.App) error {
		if superseded() {
			return errUndoSuperseded
		}
		return tx.Delete(record)
	})
}

func processUnfollowActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	return undoFollow(app, actor, activity, nil)
}

// undoFollow removes the follows row an Undo{Follow} refers to, in one
// transaction.
func undoFollow(app core.App, actor *core.Record, activity pub.Activity, superseded func() bool) error {
	// this was a local follow
	if actor.GetBool("is_local") {
		return nil
	}

	followActivity, ok := activity.Object.(*pub.Activity)
	if !ok || followActivity == nil {
		return fmt.Errorf("undo object must be a follow activity")
	}

	followee, err := app.FindFirstRecordByData("activitypub_actors", "iri", followActivity.Object)
	if err != nil {
		return err
	}

	const filter = "follower={:follower} && followee={:followee}"
	params := dbx.Params{"follower": actor.Id, "followee": followee.Id}

	// Look the row up outside the writer, like every other undo; the
	// transaction re-reads it before changing anything.
	if _, err := app.FindFirstRecordByFilter("follows", filter, params); err != nil &&
		(!errors.Is(err, sql.ErrNoRows) || !isLocalInstanceRecipient(followee)) {
		return err
	}

	return app.RunInTransaction(func(txApp core.App) error {
		follow, err := txApp.FindFirstRecordByFilter("follows", filter, params)
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

		if superseded != nil && superseded() {
			return errUndoSuperseded
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
	return undoLike(app, actor, activity, nil)
}

func undoLike(app core.App, actor *core.Record, activity pub.Activity, superseded func() bool) error {
	if actor.GetBool("is_local") {
		return nil
	}

	likeActivity, ok := activity.Object.(*pub.Activity)
	if !ok || likeActivity == nil || likeActivity.Object == nil {
		return fmt.Errorf("undo object must be a like activity")
	}

	trail, err := app.FindFirstRecordByData("trails", "iri", likeActivity.Object.GetID().String())
	if err != nil {
		return err
	}

	like, err := app.FindFirstRecordByFilter("trail_like", "actor={:actor} && trail={:trail}", dbx.Params{"actor": actor.Id, "trail": trail.Id})
	if err != nil {
		return err
	}

	return removeUndoneRecord(app, like, superseded)
}
