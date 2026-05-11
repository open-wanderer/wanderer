package federation

import (
	"fmt"
	"os"
	"pocketbase/util"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

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
		return err
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := "https://www.w3.org/ns/activitystreams#Public"
	cc := author.GetString("iri") + "/followers"
	object := fmt.Sprintf("%s/api/v1/trail/%s", origin, r.Id)

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

	follows, err := app.FindRecordsByFilter("follows", "followee={:followee}&&status='accepted'", "", -1, 0, dbx.Params{"followee": author.Id})
	if err != nil {
		return err
	}

	recipients := []string{}
	for _, f := range follows {
		follower, err := app.FindRecordById("activitypub_actors", f.GetString("follower"))
		if err != nil {
			return err
		}
		recipients = append(recipients, follower.GetString("inbox"))
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
		return err
	}

	if !author.GetBool("isLocal") {
		return nil
	}

	commentTrail, err := app.FindRecordById("trails", r.GetString("trail"))
	if err != nil {
		return err
	}

	commentTrailAuthor, err := app.FindRecordById("activitypub_actors", commentTrail.GetString("author"))
	if err != nil {
		return err
	}

	if commentTrailAuthor.GetBool("isLocal") {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := commentTrailAuthor.GetString("iri")
	object := fmt.Sprintf("%s/api/v1/comment/%s", origin, r.Id)

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
		return err
	}

	if !author.GetBool("isLocal") {
		return nil
	}

	summitLogTrail, err := app.FindRecordById("trails", r.GetString("trail"))
	if err != nil {
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
	object := fmt.Sprintf("%s/api/v1/summit-log/%s", origin, r.Id)
	cc := pub.ItemCollection{pub.IRI(author.GetString("iri") + "/followers")}

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()

	follows, err := app.FindRecordsByFilter("follows", "followee={:followee}&&status='accepted'", "", -1, 0, dbx.Params{"followee": author.Id})
	if err != nil {
		return err
	}

	recipients := []string{}

	for _, f := range follows {
		follower, err := app.FindRecordById("activitypub_actors", f.GetString("follower"))
		if err != nil {
			return err
		}
		recipients = append(recipients, follower.GetString("inbox"))
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
		return err
	}

	if !author.GetBool("isLocal") {
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
	object := fmt.Sprintf("%s/api/v1/list/%s", origin, r.Id)

	activity := pub.DeleteNew(pub.IRI(id), pub.IRI(object))
	activity.Actor = pub.IRI(author.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = pub.ItemCollection{pub.IRI(cc)}
	activity.Published = time.Now()

	follows, err := app.FindRecordsByFilter("follows", "followee={:followee}&&status='accepted'", "", -1, 0, dbx.Params{"followee": author.Id})
	if err != nil {
		return err
	}

	recipients := []string{}
	for _, f := range follows {
		follower, err := app.FindRecordById("activitypub_actors", f.GetString("follower"))
		if err != nil {
			return err
		}
		recipients = append(recipients, follower.GetString("inbox"))
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
	if actor.GetBool("isLocal") {
		return nil
	}

	object := activity.Object.GetID().String()

	var err error
	switch {
	case strings.Contains(object, "trail"):
		err = processDeleteTrailActivity(app, activity)
	case strings.Contains(object, "comment"):
		err = processDeleteCommentActivity(app, actor, activity)
	case strings.Contains(object, "summit-log"):
		err = processDeleteSummitLogActivity(app, actor, activity)
	case strings.Contains(object, "list"):
		err = processDeleteListActivity(app, actor, activity)
	}

	if err != nil {
		return err
	}

	return nil
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
