package federation

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"pocketbase/util"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// create outgoing follow activity
func CreateFollowActivity(app core.App, follow *core.Record) error {
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

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

	activity := pub.FollowNew(pub.IRI(id), pub.IRI(followeeActor.GetString("iri")))
	activity.Actor = pub.IRI(followerActor.GetString("iri"))

	// Instance follows store the IRI of their Follow activity. It is written
	// directly to skip the follows hooks, and before delivery.
	if isLocalInstanceRecipient(followerActor) {
		if _, err = app.NonconcurrentDB().Update("follows", dbx.Params{"activity_iri": id}, dbx.HashExp{"id": follow.Id}).Execute(); err != nil {
			return err
		}
		follow.Set("activity_iri", id)
	}

	err = PostActivity(app, followerActor, activity, []string{followeeActor.GetString("inbox")})
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.FollowType))
	record.Set("object", followeeActor.GetString("iri"))
	record.Set("actor", followerActor.GetString("iri"))
	record.Set("published", time.Now())

	return app.Save(record)
}

// isSupersededFollow reports whether incoming names a different Follow activity
// than the one stored in current. Rows without activity_iri are never
// superseded.
func isSupersededFollow(current string, incoming pub.Item) bool {
	if current == "" || incoming == nil {
		return false
	}
	id := incoming.GetID()
	if id == "" {
		return false
	}
	return id.String() != current
}

// findFollowActivityRecord returns the stored Follow activity belonging to a
// follows row. It prefers the row's activity_iri and falls back to the newest
// Follow activity for the actor/object pair.
func findFollowActivityRecord(app core.App, follow *core.Record, actorIRI, objectIRI string) (*core.Record, error) {
	if iri := follow.GetString("activity_iri"); iri != "" {
		rec, err := app.FindFirstRecordByData("activitypub_activities", "iri", iri)
		if err == nil {
			return rec, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	records, err := app.FindRecordsByFilter(
		"activitypub_activities",
		"actor={:actor}&&object={:object}&&type={:type}",
		"-created", 1, 0,
		dbx.Params{"actor": actorIRI, "object": objectIRI, "type": string(pub.FollowType)},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

// storeFollowActivity persists an incoming Follow so Accept/Reject/Undo can be
// reconstructed later.
func storeFollowActivity(app core.App, actor, object *core.Record, activity pub.Activity) error {
	activitiesCollection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}
	activityRecord := core.NewRecord(activitiesCollection)
	activityRecord.Set("iri", activity.GetID().String())
	activityRecord.Set("type", string(pub.FollowType))
	activityRecord.Set("actor", actor.GetString("iri"))
	activityRecord.Set("object", object.GetString("iri"))
	activityRecord.Set("published", time.Now())
	return app.Save(activityRecord)
}

// processInstanceFollow handles a remote Follow of the local instance actor,
// keyed on the Follow activity IRI. The read and write run in one transaction;
// a re-Accept is sent after it commits.
func processInstanceFollow(app core.App, followCollection *core.Collection, actor, object *core.Record, activity pub.Activity) error {
	// Only remote instance actors may follow the instance actor.
	if actor.GetString("actor_type") != "instance" {
		app.Logger().Warn("refusing instance Follow from a non-instance actor", "actor", actor.GetString("iri"))
		return errFollowerNotInstance
	}

	followIRI := activity.GetID().String()
	if followIRI == "" {
		return fmt.Errorf("instance follow without activity id")
	}

	// set when an Accept must be re-sent after commit
	var resendAcceptFor *core.Record

	err := app.RunInTransaction(func(txApp core.App) error {
		// this Follow activity was already stored
		known := true
		if _, err := txApp.FindFirstRecordByData("activitypub_activities", "iri", followIRI); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			known = false
		}

		existing, existErr := txApp.FindFirstRecordByFilter(
			"follows",
			"follower={:follower} && followee={:followee}",
			dbx.Params{"follower": actor.Id, "followee": object.Id},
		)
		if existErr != nil && !errors.Is(existErr, sql.ErrNoRows) {
			return existErr
		}

		if existErr == nil && existing != nil {
			// duplicate delivery of the Follow this row stands for
			if existing.GetString("activity_iri") == followIRI || known {
				return nil
			}
			switch existing.GetString("status") {
			case "pending", "rejected":
				// a retry: the new Follow replaces the old one
				existing.Set("activity_iri", followIRI)
				existing.Set("status", "pending")
				if err := txApp.Save(existing); err != nil {
					return err
				}
				return storeFollowActivity(txApp, actor, object, activity)
			default:
				// Re-Follow of an accepted relationship: store the new Follow and re-send the
				// Accept after commit.
				existing.Set("activity_iri", followIRI)
				if err := txApp.Save(existing); err != nil {
					return err
				}
				if err := storeFollowActivity(txApp, actor, object, activity); err != nil {
					return err
				}
				resendAcceptFor = existing
				return nil
			}
		}

		// no row: a known IRI is a replay of an undone Follow
		if known {
			return nil
		}

		followRecord := core.NewRecord(followCollection)
		followRecord.Set("follower", actor.Id)
		followRecord.Set("followee", object.Id)
		followRecord.Set("status", "pending")
		followRecord.Set("activity_iri", followIRI)
		if err := txApp.Save(followRecord); err != nil {
			return err
		}
		return storeFollowActivity(txApp, actor, object, activity)
	})
	if err != nil {
		return err
	}

	if resendAcceptFor != nil {
		return CreateAcceptFollowActivity(app, resendAcceptFor)
	}
	return nil
}

// process incoming follow activity
func ProcessFollowActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	// find the followee in our db
	object, err := app.FindFirstRecordByData("activitypub_actors", "iri", activity.Object)
	if err != nil {
		return err
	}

	// a remote actor has requested the follow
	// this means we have not yet created a follow entry in our db
	if !actor.GetBool("is_local") {
		followCollection, err := app.FindCollectionByNameOrId("follows")
		if err != nil {
			return err
		}

		// Follows of the local instance actor are stored as pending and need admin
		// approval.
		if object.GetString("actor_type") == "instance" && object.GetBool("is_local") {
			return processInstanceFollow(app, followCollection, actor, object, activity)
		}

		// Skip duplicate Follow deliveries for an existing pair.
		existing, existErr := app.FindFirstRecordByFilter(
			"follows",
			"follower={:follower} && followee={:followee}",
			dbx.Params{"follower": actor.Id, "followee": object.Id},
		)
		if existErr == nil && existing != nil {
			return nil // already recorded; treat as duplicate delivery
		}
		if existErr != nil && !errors.Is(existErr, sql.ErrNoRows) {
			return existErr
		}

		followRecord := core.NewRecord(followCollection)
		followRecord.Set("follower", actor.Id)
		followRecord.Set("followee", object.Id)

		// Person actors are accepted automatically.
		followRecord.Set("status", "accepted")
		err = app.Save(followRecord)
		if err != nil {
			return err
		}
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)
	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

	// send the accept activity back to the actor's inbox
	acceptActivity := pub.AcceptNew(pub.IRI(id), activity)
	acceptActivity.Actor = activity.Object
	err = PostActivity(app, object, acceptActivity, []string{actor.GetString("inbox")})
	if err != nil {
		return err
	}

	// create record of the accept activity in our db
	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.AcceptType))
	record.Set("object", activity)
	record.Set("actor", object.GetString("iri"))
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}
	// Follow notifications are only sent for user-level follows.
	if object.GetString("actor_type") == "instance" {
		return nil
	}
	notification := util.Notification{
		Type: util.NewFollower,
		Metadata: map[string]string{
			"follower": fmt.Sprintf("@%s@%s", actor.GetString("preferred_username"), actor.GetString("domain")),
		},
		Seen:   false,
		Author: actor.Id,
	}
	return util.SendNotification(app, notification, object)

}

// CreateAcceptFollowActivity delivers an Accept wrapping the stored incoming
// Follow, signed as the local instance actor.
func CreateAcceptFollowActivity(app core.App, follow *core.Record) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	followerActor, err := app.FindRecordById("activitypub_actors", follow.GetString("follower"))
	if err != nil {
		return err
	}

	followeeActor, err := app.FindRecordById("activitypub_actors", follow.GetString("followee"))
	if err != nil {
		return err
	}

	// Reload the stored incoming Follow activity.
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

	acceptActivity := pub.AcceptNew(pub.IRI(id), followActivity)
	acceptActivity.Actor = pub.IRI(followeeActor.GetString("iri"))

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.AcceptType))
	record.Set("object", followActivity)
	record.Set("actor", followeeActor.GetString("iri"))
	record.Set("published", time.Now())

	// Deliver before saving the activity record.
	if err = PostActivity(app, followeeActor, acceptActivity, []string{followerActor.GetString("inbox")}); err != nil {
		return err
	}
	return app.Save(record)
}

// CreateRejectFollowActivity delivers a Reject wrapping the stored incoming
// Follow, signed as the local instance actor.
func CreateRejectFollowActivity(app core.App, follow *core.Record) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	followerActor, err := app.FindRecordById("activitypub_actors", follow.GetString("follower"))
	if err != nil {
		return err
	}

	followeeActor, err := app.FindRecordById("activitypub_actors", follow.GetString("followee"))
	if err != nil {
		return err
	}

	// Reload the stored incoming Follow activity.
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

	rejectActivity := pub.RejectNew(pub.IRI(id), followActivity)
	rejectActivity.Actor = pub.IRI(followeeActor.GetString("iri"))

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.RejectType))
	record.Set("object", followActivity)
	record.Set("actor", followeeActor.GetString("iri"))
	record.Set("published", time.Now())

	// Deliver before saving the activity record.
	if err = PostActivity(app, followeeActor, rejectActivity, []string{followerActor.GetString("inbox")}); err != nil {
		return err
	}
	return app.Save(record)
}

func ProcessRejectActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	followActivity, ok := activity.Object.(*pub.Activity)
	if !ok {
		return fmt.Errorf("ProcessRejectActivity: object is not *pub.Activity, got %T", activity.Object)
	}

	follower, err := app.FindFirstRecordByData("activitypub_actors", "iri", followActivity.Actor)
	if err != nil {
		return err
	}

	follow, err := app.FindFirstRecordByFilter("follows", "follower={:follower} && followee={:followee}", dbx.Params{"follower": follower.Id, "followee": actor.Id})
	if err != nil {
		return err
	}
	if current := follow.GetString("activity_iri"); isSupersededFollow(current, followActivity) {
		app.Logger().Info("ignoring Reject for a superseded Follow",
			"rejected", followActivity.GetID().String(), "current", current)
		return nil
	}
	follow.Set("status", "rejected")
	return app.Save(follow)
}

func ProcessAcceptActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	// The object may be an IRI only; return an error instead of panicking.
	followActivity, ok := activity.Object.(*pub.Activity)
	if !ok {
		return fmt.Errorf("ProcessAcceptActivity: object is not *pub.Activity, got %T", activity.Object)
	}

	follower, err := app.FindFirstRecordByData("activitypub_actors", "iri", followActivity.Actor)
	if err != nil {
		return err
	}

	follow, err := app.FindFirstRecordByFilter("follows", "follower={:follower} && followee={:followee}", dbx.Params{"follower": follower.Id, "followee": actor.Id})
	if err != nil {
		return err
	}
	if current := follow.GetString("activity_iri"); isSupersededFollow(current, followActivity) {
		app.Logger().Info("ignoring Accept for a superseded Follow",
			"accepted", followActivity.GetID().String(), "current", current)
		return nil
	}
	follow.Set("status", "accepted")
	err = app.Save(follow)
	if err != nil {
		return err
	}

	// err = util.SyncOutbox(app, actor)
	// if err != nil {
	// 	return err
	// }
	return nil
}
