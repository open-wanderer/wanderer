package federation

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
	"golang.org/x/net/html"
)

func CreateTrailActivity(app core.App, actor *core.Record, trail *core.Record, typ pub.ActivityVocabularyType) error {
	if !trail.GetBool("public") {
		// only broadcast the trail if it is public
		return nil
	}
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	trailAuthor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
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

	mentionedActors, err := ActorsFromMentions(app, actor, trail.GetString("description"))
	if err != nil {
		return err
	}

	mentions := []string{}
	cc := pub.ItemCollection{pub.IRI(trailAuthor.GetString("followers"))}
	tags := pub.ItemCollection{}
	for _, m := range mentionedActors {
		inbox := m.GetString("inbox")
		mention := pub.MentionNew(pub.IRI(m.GetString("iri")))
		mention.Href = pub.IRI(m.GetString("iri"))
		mention.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("@%s@%s", m.GetString("preferred_username"), m.GetString("domain"))))
		tags.Append(mention)

		mentions = append(mentions, inbox)
		cc.Append(pub.IRI(inbox))
	}

	trailObject, err := util.ObjectFromTrail(app, trail, &tags)
	if err != nil {
		return err
	}

	activity := pub.ActivityNew(pub.IRI(id), typ, trailObject)
	activity.Actor = pub.IRI(trailAuthor.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("to", []string{to})
	record.Set("cc", cc)
	record.Set("type", string(typ))
	record.Set("object", trailObject)
	record.Set("actor", trailAuthor.GetString("iri"))
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}

	follows, err := app.FindRecordsByFilter("follows", "followee={:followee}&&status='accepted'", "", -1, 0, dbx.Params{"followee": trailAuthor.Id})
	if err != nil {
		return err
	}

	recipients := mentions
	for _, f := range follows {
		follower, err := app.FindRecordById("activitypub_actors", f.GetString("follower"))
		if err != nil {
			return err
		}
		recipients = append(recipients, follower.GetString("inbox"))
	}

	return PostActivity(app, trailAuthor, activity, recipients)
}

func CreateCommentActivity(app core.App, actor *core.Record, comment *core.Record, typ pub.ActivityVocabularyType) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	// author of the comment
	commentAuthor, err := app.FindRecordById("activitypub_actors", comment.GetString("author"))
	if err != nil {
		return err
	}

	commentTrail, err := app.FindRecordById("trails", comment.GetString("trail"))
	if err != nil {
		return err
	}
	commentTrailAuthor, err := app.FindRecordById("activitypub_actors", commentTrail.GetString("author"))
	if err != nil {
		return err
	}

	activityRecordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, activityRecordId)
	to := "https://www.w3.org/ns/activitystreams#Public"

	mentionedActors, err := ActorsFromMentions(app, actor, comment.GetString("text"))
	if err != nil {
		return err
	}
	recipients := []string{}
	tags := pub.ItemCollection{}
	for _, m := range mentionedActors {
		mention := pub.MentionNew(pub.IRI(m.GetString("iri")))
		mention.Href = pub.IRI(m.GetString("iri"))
		mention.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("@%s@%s", m.GetString("preferred_username"), m.GetString("domain"))))
		tags.Append(mention)

		recipients = append(recipients, m.GetString("inbox"))
	}
	recipients = append(recipients, commentTrailAuthor.GetString("inbox"))

	cc := pub.ItemCollection{}
	for _, r := range recipients {
		cc.Append(pub.IRI(r))
	}

	author := commentAuthor.GetString("iri")

	commentObject, err := util.ObjectFromComment(app, comment, &tags)
	if err != nil {
		return err
	}

	activity := pub.ActivityNew(pub.IRI(id), typ, commentObject)
	activity.Actor = pub.IRI(author)
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = cc
	activity.Published = time.Now()
	activity.Object = commentObject

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", activityRecordId)
	record.Set("iri", id)
	record.Set("to", []string{to})
	record.Set("cc", recipients)
	record.Set("type", string(typ))
	record.Set("object", commentObject)
	record.Set("actor", author)
	record.Set("published", time.Now())

	err = app.Save(record)
	if err != nil {
		return err
	}

	return PostActivity(app, commentAuthor, activity, recipients)

}

func CreateSummitLogActivity(app core.App, actor *core.Record, summitLog *core.Record, typ pub.ActivityVocabularyType) error {

	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	summitLogAuthor, err := app.FindRecordById("activitypub_actors", summitLog.GetString("author"))
	if err != nil {
		return err
	}

	var summitLogAuthorId string
	// first check if we find the trail locally
	summitLogTrail, err := app.FindRecordById("trails", summitLog.GetString("trail"))
	if err != nil {
		return err
	}
	if !summitLogTrail.GetBool("public") {
		// only broadcast the log if the trail it belongs to is public
		return nil
	}
	summitLogAuthorId = summitLogTrail.GetString("author")

	summitLogTrailAuthor, err := app.FindRecordById("activitypub_actors", summitLogAuthorId)
	if err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	var trailIRI pub.IRI
	if summitLogTrailAuthor.GetBool("isLocal") {
		trailId := summitLog.GetString("trail")
		trailIRI = pub.IRI(fmt.Sprintf("%s/api/v1/trail/%s", origin, trailId))
	} else {
		trailIRI = pub.IRI(summitLogTrail.GetString("iri"))
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := pub.ItemCollection{pub.IRI("https://www.w3.org/ns/activitystreams#Public")}

	// someone else created the summit log on the trail -> inform the trail's author
	if summitLogAuthor.Id != summitLogTrailAuthor.Id {
		to.Append(pub.IRI(summitLogTrailAuthor.GetString("iri")))
	}

	mentionedActors, err := ActorsFromMentions(app, actor, summitLog.GetString("text"))
	if err != nil {
		return err
	}

	mentions := []string{}
	cc := pub.ItemCollection{pub.IRI(summitLogAuthor.GetString("followers"))}
	mentionTags := pub.ItemCollection{}
	for _, m := range mentionedActors {
		inbox := m.GetString("inbox")
		mention := pub.MentionNew(pub.IRI(m.GetString("iri")))
		mention.Href = pub.IRI(m.GetString("iri"))
		mention.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("@%s@%s", m.GetString("preferred_username"), m.GetString("domain"))))
		mentionTags.Append(mention)

		mentions = append(mentions, inbox)
		cc.Append(pub.IRI(inbox))
	}

	photos := summitLog.GetStringSlice("photos")

	gpx := ""
	if summitLog.GetString("gpx") != "" {
		gpx = fmt.Sprintf("%s/api/v1/files/summit_logs/%s/%s", origin, summitLog.Id, summitLog.GetString("gpx"))
	}

	attachments := make(pub.ItemCollection, max(len(photos), 2))
	for i := range len(photos) {
		iri := fmt.Sprintf("%s/api/v1/files/summit_logs/%s/%s", origin, summitLog.Id, photos[i])

		attachments[i] = pub.Document{
			Type:      pub.ImageType,
			MediaType: "image/jpeg",
			URL:       pub.IRI(iri),
		}
	}
	if gpx != "" {
		attachments.Append(pub.Document{
			Type:      pub.DocumentType,
			MediaType: "application/xml+gpx",
			URL:       pub.IRI(gpx),
		})
	}

	tags := pub.ItemCollection{
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "elevation_gain")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", summitLog.GetFloat("elevation_gain")))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "elevation_loss")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", summitLog.GetFloat("elevation_loss")))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "distance")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", summitLog.GetFloat("distance")))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "duration")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", summitLog.GetFloat("duration")))),
		},
	}

	for _, m := range mentionTags {
		tags.Append(m)
	}

	logObject := pub.ObjectNew(pub.NoteType)

	logObject.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, summitLog.GetString("text")))
	logObject.AttributedTo = pub.IRI(summitLogAuthor.GetString("iri"))
	logObject.Published = summitLog.GetDateTime("created").Time()
	logObject.ID = pub.IRI(fmt.Sprintf("%s/api/v1/summit-log/%s", origin, summitLog.Id))
	logObject.URL = pub.IRI(fmt.Sprintf("%s/trail/view/@%s/%s", origin, summitLogTrailAuthor.GetString("preferred_username"), summitLog.GetString("trail")))
	logObject.InReplyTo = trailIRI
	logObject.Tag = tags

	logObject.StartTime = summitLog.GetDateTime("date").Time()
	logObject.Attachment = attachments

	activity := pub.ActivityNew(pub.IRI(id), typ, logObject)
	activity.Actor = pub.IRI(summitLogAuthor.GetString("iri"))
	activity.To = to
	activity.CC = cc
	activity.Published = time.Now()

	follows, err := app.FindRecordsByFilter("follows", "followee={:followee}&&status='accepted'", "", -1, 0, dbx.Params{"followee": summitLogAuthor.Id})
	if err != nil {
		return err
	}

	recipients := mentions

	for _, f := range follows {
		follower, err := app.FindRecordById("activitypub_actors", f.GetString("follower"))
		if err != nil {
			return err
		}
		recipients = append(recipients, follower.GetString("inbox"))
	}

	if summitLogAuthor.Id != summitLogTrailAuthor.Id {
		recipients = append(recipients, summitLogTrailAuthor.GetString("inbox"))
	}

	err = PostActivity(app, summitLogAuthor, activity, recipients)
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("to", to)
	record.Set("cc", cc)
	record.Set("type", string(typ))
	record.Set("object", logObject)
	record.Set("actor", summitLogAuthor.GetString("iri"))
	record.Set("published", time.Now())

	return app.Save(record)
}

func CreateListActivity(app core.App, list *core.Record, typ pub.ActivityVocabularyType) error {
	if !list.GetBool("public") {
		// only broadcast the list if it is public
		return nil
	}
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	// author of the list
	listAuthor, err := app.FindRecordById("activitypub_actors", list.GetString("author"))
	if err != nil {
		return err
	}

	activityRecordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, activityRecordId)
	to := "https://www.w3.org/ns/activitystreams#Public"
	cc := listAuthor.GetString("followers")
	author := listAuthor.GetString("iri")

	listObject, err := util.ObjectFromList(app, list)
	if err != nil {
		return err
	}

	activity := pub.ActivityNew(pub.IRI(id), typ, listObject)
	activity.Actor = pub.IRI(author)
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.CC = pub.ItemCollection{pub.IRI(cc)}
	activity.Published = time.Now()
	activity.Object = listObject

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	follows, err := app.FindRecordsByFilter("follows", "followee={:followee}&&status='accepted'", "", -1, 0, dbx.Params{"followee": listAuthor.Id})
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

	err = PostActivity(app, listAuthor, activity, recipients)
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", activityRecordId)
	record.Set("iri", id)
	record.Set("to", []string{to})
	record.Set("cc", []string{cc})
	record.Set("type", string(typ))
	record.Set("object", listObject)
	record.Set("actor", author)
	record.Set("published", time.Now())

	return app.Save(record)
}

func ProcessCreateOrUpdateActivity(app core.App, actor *core.Record, recipient *core.Record, activity pub.Activity) error {

	var err error
	if strings.Contains(activity.Object.GetID().String(), "/api/v1/trail") {
		err = processCreateOrUpdateTrailActivity(activity, app, actor, recipient)
	} else if strings.Contains(activity.Object.GetID().String(), "/api/v1/summit-log") {
		err = processCreateOrUpdateSummitLogActivity(activity, app, actor)
	} else if strings.Contains(activity.Object.GetID().String(), "/api/v1/list") {
		err = processCreateOrUpdateListActivity(activity, app, actor, recipient)
	} else {
		err = processCreateOrUpdateCommentActivity(activity, app, actor)
	}

	if err != nil {
		return err
	}

	return nil

}

func processCreateOrUpdateTrailActivity(activity pub.Activity, app core.App, actor *core.Record, recipient *core.Record) error {
	trail, err := util.TrailFromActivity(activity, app, actor)
	if err != nil {
		return err
	}

	_, err = util.InsertIntoFeed(app, recipient.Id, actor.Id, trail.Id, util.TrailFeed)
	if err != nil {
		return err
	}

	trailObject, _ := pub.ToObject(activity.Object)

	for _, t := range trailObject.Tag {
		if t.GetType() == pub.MentionType {
			mention := t.(*pub.Mention)
			mentionedActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", mention.Href.GetID().String())
			if err != nil {
				continue
			}
			notification := util.Notification{
				Type: util.TrailMention,
				Metadata: map[string]string{
					"id":     trail.Id,
					"author": fmt.Sprintf("@%s@%s", actor.GetString("preferred_username"), actor.GetString("domain")),
				},
				Seen:   false,
				Author: actor.Id,
			}
			return util.SendNotification(app, notification, mentionedActor)
		}
	}

	return err
}

func processCreateOrUpdateCommentActivity(activity pub.Activity, app core.App, actor *core.Record) error {

	commentObject, err := pub.ToObject(activity.Object)
	if err != nil {
		return err
	}

	if commentObject.InReplyTo == nil {
		return fmt.Errorf("error processing comment: InReplyTo empty")
	}

	trailUrl, err := url.Parse(commentObject.InReplyTo.GetLink().String())
	if err != nil {
		return err
	}
	trailId := path.Base(trailUrl.Path)

	var trail *core.Record
	trail, err = app.FindFirstRecordByFilter("trails", "iri={:iri} || id={:id}", dbx.Params{"id": trailId, "iri": commentObject.InReplyTo.GetID().String()})

	// if the trail is not present on this instance fetch it
	if err != nil {
		if err == sql.ErrNoRows {
			trailObject, err := util.TrailObjectFromIRI(commentObject.InReplyTo.GetLink().String())
			if err != nil {
				return err
			}
			activity := pub.ActivityNew(pub.IRI("new"), pub.CreateType, trailObject)
			trail, err = util.TrailFromActivity(*activity, app, actor)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	trailAuthor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return err
	}

	// no need to do anything else if the actor is local
	if actor.GetBool("isLocal") {
		return nil
	}

	record, err := app.FindFirstRecordByData("comments", "iri", commentObject.ID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			collection, err := app.FindCollectionByNameOrId("comments")
			if err != nil {
				return err
			}

			record = core.NewRecord(collection)
		} else {
			return err
		}
	}

	record.Set("iri", commentObject.ID.String())
	record.Set("text", commentObject.Content.First().Value)
	record.Set("author", actor.Id)
	record.Set("trail", trail.Id)

	err = app.Save(record)
	if err != nil {
		return err
	}

	// send notifications to all mentioned actors
	for _, t := range commentObject.Tag {
		if t.GetType() == pub.MentionType {
			mention := t.(*pub.Mention)
			mentionedActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", mention.Href.GetID().String())
			if err != nil {
				continue
			}
			notification := util.Notification{
				Type: util.CommentMention,
				Metadata: map[string]string{
					"comment":      commentObject.Content.First().Value.String(),
					"trail_id":     trail.Id,
					"trail_name":   trail.GetString("name"),
					"trail_author": fmt.Sprintf("@%s@%s", trailAuthor.GetString("preferred_username"), trailAuthor.GetString("domain")),
				},
				Seen:   false,
				Author: actor.Id,
			}
			return util.SendNotification(app, notification, mentionedActor)
		}
	}
	if activity.Type == pub.CreateType {
		// send a notification to the trail author
		notification := util.Notification{
			Type: util.TrailComment,
			Metadata: map[string]string{
				"comment":      commentObject.Content.First().Value.String(),
				"trail_id":     trail.Id,
				"trail_name":   trail.GetString("name"),
				"trail_author": fmt.Sprintf("@%s@%s", trailAuthor.GetString("preferred_username"), trailAuthor.GetString("domain")),
			},
			Seen:   false,
			Author: actor.Id,
		}
		return util.SendNotification(app, notification, trailAuthor)
	}

	return nil
}

func processCreateOrUpdateSummitLogActivity(activity pub.Activity, app core.App, actor *core.Record) error {
	logObject, err := pub.ToObject(activity.Object)
	if err != nil {
		return err
	}

	trailIRI, err := url.Parse(logObject.InReplyTo.GetID().String())
	if err != nil {
		return err
	}
	trailId := path.Base(trailIRI.Path)

	trail, err := app.FindFirstRecordByFilter("trails", "iri={:iri} || id={:id}", dbx.Params{"id": trailId, "iri": logObject.InReplyTo.GetID().String()})
	// if the trail is not present on this instance fetch it
	if err != nil {
		if err == sql.ErrNoRows {
			trailObject, err := util.TrailObjectFromIRI(logObject.InReplyTo.GetLink().String())
			if err != nil {
				return err
			}
			activity := pub.ActivityNew(pub.IRI("new"), pub.CreateType, trailObject)
			trail, err = util.TrailFromActivity(*activity, app, actor)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	trailAuthor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return err
	}

	newSummitLog := false
	record, err := app.FindFirstRecordByData("summit_logs", "iri", logObject.ID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			collection, err := app.FindCollectionByNameOrId("summit_logs")
			if err != nil {
				return err
			}

			record = core.NewRecord(collection)
			newSummitLog = true
		} else {
			return err
		}
	}
	// no need to do anything else if the actor is local
	if actor.GetBool("isLocal") {
		return nil
	}

	var distance, duration, elevation_gain, elevation_loss float64
	tags, err := pub.ToItemCollection(logObject.Tag)
	if err != nil {
		return err
	}

	for _, tag := range tags.Collection() {
		tagObj, err := pub.ToObject(tag)
		if err != nil {
			continue
		}
		content := tagObj.Content.First().Value.String()
		switch tagObj.Name.First().Value.String() {
		case "elevation_gain":
			elevation_gain, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "elevation_loss":
			elevation_loss, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "duration":
			duration, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "distance":
			distance, err = strconv.ParseFloat(content[:len(content)-1], 64)
		}
		if err != nil {
			continue
		}
	}

	record.Set("date", logObject.StartTime)
	record.Set("text", logObject.Content.First().Value)
	record.Set("distance", distance)
	record.Set("duration", duration)
	record.Set("elevation_gain", elevation_gain)
	record.Set("elevation_loss", elevation_loss)
	record.Set("author", actor.Id)
	record.Set("trail", trail.Id)
	record.Set("iri", logObject.ID.String())

	if logObject.Attachment != nil {
		attachments, err := pub.ToItemCollection(logObject.Attachment)
		if err != nil {
			return err
		}

		photoURLs := []string{}
		gpxURL := ""
		for _, a := range attachments.Collection() {
			attachment, err := pub.ToObject(a)
			if err != nil {
				continue
			}
			if attachment.Type == pub.DocumentType && attachment.MediaType == "application/xml+gpx" {
				gpxURL = attachment.URL.GetLink().String()
			} else if attachment.Type == pub.ImageType {
				photoURLs = append(photoURLs, attachment.URL.GetLink().String())
			}
		}

		if len(photoURLs) > 0 {
			photos := make([]*filesystem.File, len(photoURLs))
			for i, purl := range photoURLs {
				photo, err := filesystem.NewFileFromURL(context.Background(), purl)
				if err != nil {
					continue
				}
				photos[i] = photo
			}

			record.Set("photos", photos)
		}

		if gpxURL != "" {
			gpx, err := filesystem.NewFileFromURL(context.Background(), gpxURL)
			if err != nil {
				return err
			}

			record.Set("gpx", gpx)
		}
	}

	err = app.Save(record)
	if err != nil {
		return err
	}

	// send notifications to all mentioned actors
	for _, t := range logObject.Tag {
		if t.GetType() == pub.MentionType {
			mention := t.(*pub.Mention)
			mentionedActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", mention.Href.GetID().String())
			if err != nil {
				continue
			}
			notification := util.Notification{
				Type: util.SummitLogMention,
				Metadata: map[string]string{
					"trail_id":     trail.Id,
					"trail_name":   trail.GetString("name"),
					"trail_author": fmt.Sprintf("@%s@%s", trailAuthor.GetString("preferred_username"), trailAuthor.GetString("domain")),
				},
				Seen:   false,
				Author: actor.Id,
			}
			return util.SendNotification(app, notification, mentionedActor)
		}
	}

	if newSummitLog {
		// send a notification to the trail author
		notification := util.Notification{
			Type: util.SummitLogCreate,
			Metadata: map[string]string{
				"trail_id":     trail.Id,
				"trail_name":   trail.GetString("name"),
				"trail_author": fmt.Sprintf("@%s@%s", trailAuthor.GetString("preferred_username"), trailAuthor.GetString("domain")),
			},
			Seen:   false,
			Author: actor.Id,
		}
		return util.SendNotification(app, notification, trailAuthor)
	}

	return nil
}

func processCreateOrUpdateListActivity(activity pub.Activity, app core.App, actor *core.Record, recipient *core.Record) error {
	list, err := util.ListFromActivity(activity, app, actor)
	if err != nil {
		return err
	}

	_, err = util.InsertIntoFeed(app, recipient.Id, actor.Id, list.Id, util.ListFeed)
	if err != nil {
		return err
	}

	return err
}

func ActorsFromMentions(app core.App, actor *core.Record, htmlStr string) ([]*core.Record, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, err
	}

	var handles []string
	var actors []*core.Record

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var isMention bool
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "mention") {
					isMention = true
					break
				}
			}
			if isMention && n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				handle := strings.TrimSpace(n.FirstChild.Data)
				if strings.HasPrefix(handle, "@") {
					handles = append(handles, handle)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	for _, h := range handles {
		actor, err := GetActorByHandle(app, actor, h, false)
		if err != nil {
			continue
		}
		actors = append(actors, actor)
	}

	return actors, nil
}
