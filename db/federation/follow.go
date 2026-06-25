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
		// Idempotency: if a follow record already exists for this pair, skip creation.
		// Duplicate Follow deliveries are normal in ActivityPub (network retries, etc.)
		// and the unique composite index on (follower, followee) would cause a constraint
		// violation on app.Save if we proceeded unconditionally.
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

		followCollection, err := app.FindCollectionByNameOrId("follows")
		if err != nil {
			return err
		}
		followRecord := core.NewRecord(followCollection)
		followRecord.Set("follower", actor.Id)
		followRecord.Set("followee", object.Id)

		// Instance-actor branch: Follow directed at the local instance actor must be
		// stored as "pending" (requires admin approval, D-05/FLCL-02).  Persist the
		// incoming Follow activity so Plan 03 can reconstruct it for Accept/Reject,
		// then return early — no Accept is sent back for instance follows.
		if object.GetString("actor_type") == "instance" && object.GetBool("is_local") {
			followRecord.Set("status", "pending")
			if err = app.Save(followRecord); err != nil {
				return err
			}

			// Persist the incoming Follow activity for later Accept/Reject.
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
			if err = app.Save(activityRecord); err != nil {
				return err
			}

			return nil
		}

		// Person-actor path: auto-accept as before.
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
	// send a notification to the followee
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

// CreateAcceptFollowActivity delivers an Accept{Follow} to the remote instance that
// sent the original Follow. It reloads the persisted incoming Follow activity from
// activitypub_activities (stored by ProcessFollowActivity in Plan 02) and wraps it
// in an Accept signed as the local instance actor.
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

	// Reload the original incoming Follow activity persisted by ProcessFollowActivity.
	// actor = remote sender IRI, object = local instance actor IRI.
	followActivityRecord, err := app.FindFirstRecordByFilter(
		"activitypub_activities",
		"actor={:actor}&&object={:object}&&type={:type}",
		dbx.Params{
			"actor":  followerActor.GetString("iri"),
			"object": followeeActor.GetString("iri"),
			"type":   string(pub.FollowType),
		},
	)
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

	// Deliver first, persist after: ensures the DB record only exists when
	// delivery has at least been enqueued. If PostActivity returns an error
	// (unlikely today since it is fire-and-forget, but possible if the goroutine
	// can't be spawned), we skip the Save so a stale accepted row is not left
	// behind while the remote instance never received the Accept.
	if err = PostActivity(app, followeeActor, acceptActivity, []string{followerActor.GetString("inbox")}); err != nil {
		return err
	}
	return app.Save(record)
}

// CreateRejectFollowActivity delivers a Reject{Follow} to the remote instance that
// sent the original Follow. Structurally identical to CreateAcceptFollowActivity but
// wraps the original Follow in a Reject. Mandatory per D-08.
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

	// Reload the original incoming Follow activity persisted by ProcessFollowActivity.
	followActivityRecord, err := app.FindFirstRecordByFilter(
		"activitypub_activities",
		"actor={:actor}&&object={:object}&&type={:type}",
		dbx.Params{
			"actor":  followerActor.GetString("iri"),
			"object": followeeActor.GetString("iri"),
			"type":   string(pub.FollowType),
		},
	)
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

	// Deliver first, persist after: same ordering as CreateAcceptFollowActivity
	// to avoid a stale rejected row when delivery has not yet been enqueued.
	if err = PostActivity(app, followeeActor, rejectActivity, []string{followerActor.GetString("inbox")}); err != nil {
		return err
	}
	return app.Save(record)
}

func ProcessAcceptActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	// CR-04: use comma-ok to guard against IRI-only Accept objects (common in
	// real-world ActivityPub implementations). Return a descriptive error instead
	// of panicking so a malformed remote Accept cannot crash the request goroutine.
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
