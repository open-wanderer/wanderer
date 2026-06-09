package federation

import (
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
	followActivityRecord, err := app.FindFirstRecordByFilter("activitypub_activities", "actor={:actor}&&object={:object}&&type={:type}", dbx.Params{"actor": followerActor.GetString("iri"), "object": followeeActor.GetString("iri"), "type": string(pub.FollowType)})
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

	return PostActivity(app, actor, activity, []string{trailAuthor.GetString("inbox")})
}

func ProcessUndoActivity(app core.App, actor *core.Record, activity pub.Activity) error {

	if activity.Object.GetType() == pub.FollowType {
		return processUnfollowActivity(app, actor, activity)
	} else if activity.Object.GetType() == pub.LikeType {
		return processUnlikeActivity(app, actor, activity)
	} else {
		return fmt.Errorf("unknown undo activity object type")
	}
}

func processUnfollowActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	// this was a local follow
	if actor.GetBool("is_local") {
		return nil
	}

	followActivity := activity.Object.(*pub.Activity)

	followee, err := app.FindFirstRecordByData("activitypub_actors", "iri", followActivity.Object)
	if err != nil {
		return err
	}

	follow, err := app.FindFirstRecordByFilter("follows", "follower={:follower} && followee={:followee}", dbx.Params{"follower": actor.Id, "followee": followee.Id})
	if err != nil {
		return err
	}

	err = app.Delete(follow)
	if err != nil {
		return err
	}
	return nil
}

func processUnlikeActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	if actor.GetBool("is_local") {
		return nil
	}

	likeActivity := activity.Object.(*pub.Activity)

	trail, err := app.FindFirstRecordByData("trails", "iri", likeActivity.Object.GetID().String())
	if err != nil {
		return err
	}

	like, err := app.FindFirstRecordByFilter("trail_like", "actor={:actor} && trail={:trail}", dbx.Params{"actor": actor.Id, "trail": trail.Id})
	if err != nil {
		return err
	}

	err = app.Delete(like)
	if err != nil {
		return err
	}
	return nil
}
