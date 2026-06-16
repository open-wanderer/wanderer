package federation

import (
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
	// we accept it immediately
	if !actor.GetBool("is_local") {
		followCollection, err := app.FindCollectionByNameOrId("follows")
		if err != nil {
			return err
		}
		followRecord := core.NewRecord(followCollection)
		followRecord.Set("follower", actor.Id)
		followRecord.Set("followee", object.Id)
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

func ProcessAcceptActivity(app core.App, actor *core.Record, activity pub.Activity) error {

	followActivity := activity.Object.(*pub.Activity)

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
