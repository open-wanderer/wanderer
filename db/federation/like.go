package federation

import (
	"fmt"
	"os"
	"pocketbase/util"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// create outgoing follow activity
func CreateLikeActivity(app core.App, like *core.Record) error {
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

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)

	activity := pub.LikeNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(actor.GetString("iri"))

	err = PostActivity(app, actor, activity, []string{trailAuthor.GetString("inbox")})
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("type", string(pub.LikeType))
	record.Set("object", object)
	record.Set("actor", actor.GetString("iri"))
	record.Set("published", time.Now())

	return app.Save(record)
}

// process incoming like activity
func ProcessLikeActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	trail, err := app.FindFirstRecordByData("trails", "iri", activity.Object.GetID().String())
	if err != nil {
		return err
	}

	trailAuthor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return err
	}

	if !actor.GetBool("is_local") {
		trailLikeCollection, err := app.FindCollectionByNameOrId("trail_like")
		if err != nil {
			return err
		}
		likeRecord := core.NewRecord(trailLikeCollection)
		likeRecord.Set("trail", trail.Id)
		likeRecord.Set("actor", actor.Id)
		err = app.Save(likeRecord)
		if err != nil {
			return err
		}
	}

	// send a notification to the trail author
	notification := util.Notification{
		Type: util.TrailLike,
		Metadata: map[string]string{
			"trail_id":     trail.Id,
			"trail_name":   trail.GetString("name"),
			"trail_author": fmt.Sprintf("@%s", trailAuthor.GetString("preferred_username")),
			"liker":        fmt.Sprintf("@%s@%s", actor.GetString("preferred_username"), actor.GetString("domain")),
		},
		Seen:   false,
		Author: actor.Id,
	}
	return util.SendNotification(app, notification, trailAuthor)

}
