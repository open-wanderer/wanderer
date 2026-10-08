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

// findTrailLike returns the trail_like row for (trail, actor), or nil if there
// is none.
func findTrailLike(app core.App, trailId, actorId string) (*core.Record, error) {
	like, err := app.FindFirstRecordByFilter("trail_like", "trail={:trail} && actor={:actor}", dbx.Params{"trail": trailId, "actor": actorId})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return like, nil
}

// create outgoing like activity
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

	recipients := []string{trailAuthor.GetString("inbox")}

	// Also deliver to peer instances; public trails only.
	if trail.GetBool("public") {
		peerInboxes, err := instanceFollowerInboxes(app)
		if err != nil {
			return err
		}
		recipients = append(recipients, peerInboxes...)
	}

	err = PostActivity(app, actor, activity, recipients)
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

	if activity.Object == nil {
		return fmt.Errorf("like: missing object")
	}

	trail, err := app.FindFirstRecordByData("trails", "iri", activity.Object.GetID().String())
	if err != nil {
		// A like carries no content, so a receiver that does not hold the trail
		// has nothing to attach it to.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
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

		// The same like arrives at the user inbox and at the instance inbox.
		existing, err := findTrailLike(app, trail.Id, actor.Id)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil
		}

		likeRecord := core.NewRecord(trailLikeCollection)
		likeRecord.Set("trail", trail.Id)
		likeRecord.Set("actor", actor.Id)
		if err := app.Save(likeRecord); err != nil {
			// The twin delivery may have won the unique index race.
			if existing, lookupErr := findTrailLike(app, trail.Id, actor.Id); lookupErr == nil && existing != nil {
				return nil
			}
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
