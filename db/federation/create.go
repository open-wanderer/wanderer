package federation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
	"golang.org/x/net/html"
)

func CreateTrailActivity(app core.App, ctx context.Context, trail *core.Record, typ pub.ActivityVocabularyType) error {
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

	mentionedActors, err := ActorsFromMentions(app, ctx, trail.GetString("description"))
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
	// GoToSocial reads visibility from the object's to/cc.
	trailObject.To = activity.To
	trailObject.CC = activity.CC
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

	inboxes, err := followerInboxes(app, trailAuthor.Id)
	if err != nil {
		return err
	}
	recipients := append(mentions, inboxes...)

	// Also deliver to peer instances.
	instanceInboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		return err
	}
	recipients = append(recipients, instanceInboxes...)

	return PostActivity(app, trailAuthor, activity, recipients)
}

func CreateCommentActivity(app core.App, ctx context.Context, comment *core.Record, typ pub.ActivityVocabularyType) error {
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
	// No fanout for comments on private trails.
	if !commentTrail.GetBool("public") {
		return nil
	}
	commentTrailAuthor, err := app.FindRecordById("activitypub_actors", commentTrail.GetString("author"))
	if err != nil {
		return err
	}

	activityRecordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, activityRecordId)
	to := "https://www.w3.org/ns/activitystreams#Public"

	mentionedActors, err := ActorsFromMentions(app, ctx, comment.GetString("text"))
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
	// Only a remote trail author needs HTTP delivery.
	if !commentTrailAuthor.GetBool("is_local") {
		recipients = append(recipients, commentTrailAuthor.GetString("inbox"))
	}

	// Deliver to the comment author's followers.
	followerInboxList, err := followerInboxes(app, commentAuthor.Id)
	if err != nil {
		return err
	}
	recipients = append(recipients, followerInboxList...)

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
	// GoToSocial reads visibility from the object's to/cc.
	commentObject.To = activity.To
	commentObject.CC = activity.CC
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

	// Also deliver to peer instances.
	instanceInboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		return err
	}
	recipients = append(recipients, instanceInboxes...)

	return PostActivity(app, commentAuthor, activity, recipients)

}

func CreateSummitLogActivity(app core.App, ctx context.Context, summitLog *core.Record, typ pub.ActivityVocabularyType) error {

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
	trailIRI = pub.IRI(summitLogTrail.GetString("iri"))

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := pub.ItemCollection{pub.IRI("https://www.w3.org/ns/activitystreams#Public")}

	// someone else created the summit log on the trail -> inform the trail's author
	if summitLogAuthor.Id != summitLogTrailAuthor.Id {
		to.Append(pub.IRI(summitLogTrailAuthor.GetString("iri")))
	}

	mentionedActors, err := ActorsFromMentions(app, ctx, summitLog.GetString("text"))
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

	attachments := make(pub.ItemCollection, 0, len(photos)+1)
	for i := range len(photos) {
		iri := fmt.Sprintf("%s/api/v1/files/summit_logs/%s/%s", origin, summitLog.Id, photos[i])
		attachments.Append(pub.Document{
			Type:      pub.ImageType,
			MediaType: "image/jpeg",
			URL:       pub.IRI(iri),
		})
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
	logObject.ID = pub.IRI(summitLog.GetString("iri"))
	logObject.URL = pub.IRI(fmt.Sprintf("%s/trail/view/@%s/%s", origin, summitLogTrailAuthor.GetString("preferred_username"), summitLog.GetString("trail")))
	logObject.InReplyTo = trailIRI
	logObject.Tag = tags

	logObject.StartTime = summitLog.GetDateTime("date").Time()
	logObject.Attachment = attachments

	activity := pub.ActivityNew(pub.IRI(id), typ, logObject)
	activity.Actor = pub.IRI(summitLogAuthor.GetString("iri"))
	activity.To = to
	activity.CC = cc
	// GoToSocial reads visibility from the object's to/cc.
	logObject.To = activity.To
	logObject.CC = activity.CC
	activity.Published = time.Now()

	// Save the activity record before delivery.
	record := core.NewRecord(collection)
	record.Set("id", recordId)
	record.Set("iri", id)
	record.Set("to", to)
	record.Set("cc", cc)
	record.Set("type", string(typ))
	record.Set("object", logObject)
	record.Set("actor", summitLogAuthor.GetString("iri"))
	record.Set("published", time.Now())

	if err := app.Save(record); err != nil {
		return err
	}

	inboxes, err := followerInboxes(app, summitLogAuthor.Id)
	if err != nil {
		return err
	}
	recipients := append(mentions, inboxes...)

	if summitLogAuthor.Id != summitLogTrailAuthor.Id {
		recipients = append(recipients, summitLogTrailAuthor.GetString("inbox"))
	}

	// Also deliver to peer instances.
	instanceInboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		return err
	}
	recipients = append(recipients, instanceInboxes...)

	return PostActivity(app, summitLogAuthor, activity, recipients)
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
	// GoToSocial reads visibility from the object's to/cc.
	listObject.To = activity.To
	listObject.CC = activity.CC
	activity.Published = time.Now()
	activity.Object = listObject

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	// Save the activity record before delivery.
	record := core.NewRecord(collection)
	record.Set("id", activityRecordId)
	record.Set("iri", id)
	record.Set("to", []string{to})
	record.Set("cc", []string{cc})
	record.Set("type", string(typ))
	record.Set("object", listObject)
	record.Set("actor", author)
	record.Set("published", time.Now())

	if err := app.Save(record); err != nil {
		return err
	}

	recipients, err := followerInboxes(app, listAuthor.Id)
	if err != nil {
		return err
	}

	// Also deliver to peer instances.
	instanceInboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		return err
	}
	recipients = append(recipients, instanceInboxes...)

	return PostActivity(app, listAuthor, activity, recipients)
}

func ProcessCreateOrUpdateActivity(app core.App, ctx context.Context, actor *core.Record, recipient *core.Record, activity pub.Activity) error {
	ctx, cancel := util.WithRemoteAttachmentBudget(ctx)
	defer cancel()

	var err error
	switch util.ObjectKindFromIRI(activity.Object.GetID().String()) {
	case util.ObjectKindTrail:
		err = processCreateOrUpdateTrailActivity(ctx, activity, app, actor, recipient)
	case util.ObjectKindSummitLog:
		err = processCreateOrUpdateSummitLogActivity(ctx, activity, app, actor)
	case util.ObjectKindList:
		err = processCreateOrUpdateListActivity(ctx, activity, app, actor, recipient)
	default:
		// Unchanged fallback: anything else is treated as a comment, which is
		// what lets replies from other ActivityPub software be accepted.
		err = processCreateOrUpdateCommentActivity(ctx, activity, app, actor)
	}

	if err != nil {
		return err
	}

	return nil

}

// isLocalInstanceRecipient reports whether r is this instance's own instance
// actor. It receives federated content but has no personal feed.
func isLocalInstanceRecipient(r *core.Record) bool {
	return r != nil && r.GetString("actor_type") == "instance" && r.GetBool("is_local")
}

// checkRemoteObjectOwnership refuses a remote Create or Update of a comment or
// summit log whose IRI is empty, local, on a host other than the signer's, or
// already stored under another author. It returns the stored row, or nil when the
// object is new.
func checkRemoteObjectOwnership(app core.App, collection, objectIRI string, signer *core.Record) (*core.Record, error) {
	if objectIRI == "" {
		return nil, fmt.Errorf("activity object has no id")
	}
	if util.IsLocalIRI(objectIRI) {
		return nil, fmt.Errorf("refusing remote activity referencing local object %s", objectIRI)
	}
	if !sameHost(objectIRI, signer.GetString("iri")) {
		return nil, fmt.Errorf("object %s is not on the signer's host", objectIRI)
	}

	existing, err := app.FindFirstRecordByData(collection, "iri", objectIRI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if existing.GetString("author") != signer.Id {
		return nil, fmt.Errorf("object %s is not authored by the signer", objectIRI)
	}
	return existing, nil
}

func processCreateOrUpdateTrailActivity(ctx context.Context, activity pub.Activity, app core.App, actor *core.Record, recipient *core.Record) error {
	objectIRI := activity.Object.GetID().String()
	if !sameHost(objectIRI, actor.GetString("iri")) {
		return fmt.Errorf("object %s is not on the signer's host", objectIRI)
	}

	// A duplicate Create keeps the stored trail but still adds the recipient's feed
	// entry. Only the stored author may create or update it.
	existing, err := app.FindFirstRecordByData("trails", "iri", objectIRI)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		existing = nil
	}
	if existing != nil && existing.GetString("author") != actor.Id {
		// The host check above guarantees this fetch goes to the signer's own host.
		if err := confirmTrailAuthorAtOrigin(ctx, objectIRI, actor); err != nil {
			return fmt.Errorf("object %s is not authored by the signer: %w", objectIRI, err)
		}
		previous := existing.GetString("author")
		existing.Set("author", actor.Id)
		if err := app.Save(existing); err != nil {
			return err
		}
		app.Logger().Info("repaired legacy trail author", "iri", objectIRI, "previous_author", previous, "new_author", actor.Id)
	}

	trail := existing
	duplicate := activity.Type == pub.CreateType && existing != nil
	if !duplicate {
		trail, err = util.TrailFromActivity(ctx, activity, app, actor)
		if err != nil {
			// A failed save of a row that already existed is a real failure.
			if existing != nil {
				return err
			}
			// A concurrent delivery of the same object stored it first; continue with that
			// row so this recipient still gets a feed entry.
			stored, rerr := app.FindFirstRecordByData("trails", "iri", objectIRI)
			if rerr != nil || stored == nil {
				return err
			}
			if stored.GetString("author") != actor.Id {
				return fmt.Errorf("object %s is not authored by the signer", objectIRI)
			}
			app.Logger().Info("lost insert race against a parallel delivery", "iri", objectIRI)
			duplicate = true
			trail = stored
		}
	}

	// Per-recipient feed entry (idempotent).
	if recipient != nil && !isLocalInstanceRecipient(recipient) {
		if _, err := util.InsertIntoFeed(app, recipient.Id, actor.Id, trail.Id, util.TrailFeed); err != nil {
			return err
		}
	}

	if duplicate {
		// Duplicate deliveries do not re-send mention notifications.
		return nil
	}

	trailObject, err := pub.ToObject(activity.Object)
	if err != nil {
		return err
	}

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
			util.SendNotification(app, notification, mentionedActor)
		}
	}

	return nil
}

func processCreateOrUpdateCommentActivity(ctx context.Context, activity pub.Activity, app core.App, actor *core.Record) error {
	// Check ownership before any lookup, write or remote fetch.
	if !actor.GetBool("is_local") {
		if _, err := checkRemoteObjectOwnership(app, "comments", activity.Object.GetID().String(), actor); err != nil {
			return err
		}
	}

	// Broadcast-loop dedup: drop a duplicate Create whose content IRI is already stored.
	if activity.Type == pub.CreateType {
		objectIRI := activity.Object.GetID().String()
		existing, derr := app.FindFirstRecordByData("comments", "iri", objectIRI)
		if derr == nil && existing != nil {
			return nil // already have this comment
		}
		if derr != nil && !errors.Is(derr, sql.ErrNoRows) {
			return derr
		}
	}

	commentObject, err := pub.ToObject(activity.Object)
	if err != nil {
		return err
	}

	if commentObject.InReplyTo == nil {
		return fmt.Errorf("error processing comment: InReplyTo empty")
	}

	var trail *core.Record
	trail, err = app.FindFirstRecordByData("trails", "iri", commentObject.InReplyTo.GetLink().String())

	// if the trail is not present on this instance fetch it
	if err != nil {
		if err == sql.ErrNoRows {
			trail, err = fetchTrail(app, ctx, actor, commentObject.InReplyTo.GetLink().String())
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
	if actor.GetBool("is_local") {
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
			util.SendNotification(app, notification, mentionedActor)
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

func processCreateOrUpdateSummitLogActivity(ctx context.Context, activity pub.Activity, app core.App, actor *core.Record) error {
	ctx, cancel := util.WithRemoteAttachmentBudget(ctx)
	defer cancel()

	// Check ownership before any lookup, write or remote fetch.
	if !actor.GetBool("is_local") {
		if _, err := checkRemoteObjectOwnership(app, "summit_logs", activity.Object.GetID().String(), actor); err != nil {
			return err
		}
	}

	// Broadcast-loop dedup: drop a duplicate Create whose content IRI is already stored.
	if activity.Type == pub.CreateType {
		objectIRI := activity.Object.GetID().String()
		existing, derr := app.FindFirstRecordByData("summit_logs", "iri", objectIRI)
		if derr == nil && existing != nil {
			return nil // already have this summit log
		}
		if derr != nil && !errors.Is(derr, sql.ErrNoRows) {
			return derr
		}
	}

	logObject, err := pub.ToObject(activity.Object)
	if err != nil {
		return err
	}

	trail, err := app.FindFirstRecordByData("trails", "iri", logObject.InReplyTo.GetID().String())
	// if the trail is not present on this instance fetch it
	if err != nil {
		if err == sql.ErrNoRows {
			trail, err = fetchTrail(app, ctx, actor, logObject.InReplyTo.GetLink().String())
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
	if actor.GetBool("is_local") {
		return nil
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
		if len(content) == 0 {
			continue // guard against empty content — prevents index-out-of-range panic
		}
		numeric := content[:len(content)-1] // strip unit suffix
		switch tagObj.Name.First().Value.String() {
		case "elevation_gain":
			elevation_gain, err = strconv.ParseFloat(numeric, 64)
		case "elevation_loss":
			elevation_loss, err = strconv.ParseFloat(numeric, 64)
		case "duration":
			duration, err = strconv.ParseFloat(numeric, 64)
		case "distance":
			distance, err = strconv.ParseFloat(numeric, 64)
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
		photoLimit := util.RemotePhotoLimit(app, "summit_logs")
		ignoredPhotos := 0
		gpxURL := ""
		for _, a := range attachments.Collection() {
			attachment, err := pub.ToObject(a)
			if err != nil {
				continue
			}
			if attachment.Type == pub.DocumentType && attachment.MediaType == "application/xml+gpx" {
				gpxURL = attachment.URL.GetLink().String()
			} else if attachment.Type == pub.ImageType {
				if len(photoURLs) >= photoLimit {
					ignoredPhotos++
					continue
				}
				photoURLs = append(photoURLs, attachment.URL.GetLink().String())
			}
		}
		if ignoredPhotos > 0 {
			app.Logger().Info("ignoring remote summit log photos past the collection limit",
				"iri", logObject.ID.String(), "limit", photoLimit, "ignored", ignoredPhotos)
		}

		// Download the GPX before the photos.
		if gpxURL != "" {
			gpx, cleanup, err := util.DownloadRemoteFile(ctx, gpxURL, util.RemoteGPXMaxBytes, actor.GetString("iri"))
			defer cleanup()
			if err != nil {
				return err
			}

			record.Set("gpx", gpx)
		}

		if len(photoURLs) == 0 {
			record.Set("photos", []*filesystem.File{})
		} else {
			photos := []*filesystem.File{}
			overBudget := 0
			for _, purl := range photoURLs {
				photo, cleanup, err := util.DownloadRemoteFile(ctx, purl, util.RemotePhotoMaxBytes, actor.GetString("iri"))
				defer cleanup()
				if err != nil {
					if errors.Is(err, util.ErrRemoteAttachmentBudgetExhausted) {
						overBudget++
					}
					continue
				}
				photos = append(photos, photo)
			}
			if overBudget > 0 {
				app.Logger().Info("skipping remote photos past the activity attachment budget",
					"iri", logObject.ID.String(), "skipped", overBudget)
			}

			if len(photos) > 0 {
				record.Set("photos", photos)
			}
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
			util.SendNotification(app, notification, mentionedActor)
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

func processCreateOrUpdateListActivity(ctx context.Context, activity pub.Activity, app core.App, actor *core.Record, recipient *core.Record) error {
	objectIRI := activity.Object.GetID().String()
	if !sameHost(objectIRI, actor.GetString("iri")) {
		return fmt.Errorf("object %s is not on the signer's host", objectIRI)
	}

	// A duplicate Create keeps the stored list but still adds the recipient's feed
	// entry. Only the stored author may create or update it.
	existing, err := app.FindFirstRecordByData("lists", "iri", objectIRI)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		existing = nil
	}
	if existing != nil && existing.GetString("author") != actor.Id {
		return fmt.Errorf("object %s is not authored by the signer", objectIRI)
	}

	list := existing
	if activity.Type != pub.CreateType || existing == nil {
		list, err = util.ListFromActivity(ctx, activity, app, actor)
		if err != nil {
			// A failed save of a row that already existed is a real failure.
			if existing != nil {
				return err
			}
			// A concurrent delivery of the same object stored it first; continue with that
			// row.
			stored, rerr := app.FindFirstRecordByData("lists", "iri", objectIRI)
			if rerr != nil || stored == nil {
				return err
			}
			if stored.GetString("author") != actor.Id {
				return fmt.Errorf("object %s is not authored by the signer", objectIRI)
			}
			app.Logger().Info("lost insert race against a parallel delivery", "iri", objectIRI)
			list = stored
		}
	}

	// Per-recipient feed entry (idempotent).
	if recipient != nil && !isLocalInstanceRecipient(recipient) {
		if _, err := util.InsertIntoFeed(app, recipient.Id, actor.Id, list.Id, util.ListFeed); err != nil {
			return err
		}
	}

	return nil
}

// fetchTrailObject is util.TrailObjectFromIRI, replaceable in tests.
var fetchTrailObject = util.TrailObjectFromIRI

const (
	// trailAuthorConfirmTimeout bounds one origin confirmation fetch.
	trailAuthorConfirmTimeout = 15 * time.Second
	// trailAuthorConfirmRateLimitID is the rate-limit identifier for origin
	// confirmation fetches.
	trailAuthorConfirmRateLimitID = "trail-author-confirm"
	// trailAuthorRefusalTTL is how long a refusal by the origin is remembered.
	trailAuthorRefusalTTL = 5 * time.Minute
	// trailAuthorTransientRefusalTTL is how long a failed confirmation (fetch
	// error, timeout, rate limit, non-2xx) is remembered.
	trailAuthorTransientRefusalTTL = 30 * time.Second
	// trailAuthorRefusalMaxEntries bounds the refusal memory.
	trailAuthorRefusalMaxEntries = 4096
)

// trailAuthorRefusalMemory remembers refused (object, signer) confirmations.
type trailAuthorRefusalMemory struct {
	mu      sync.Mutex
	entries map[string]time.Time
	now     func() time.Time
}

var trailAuthorRefusals = &trailAuthorRefusalMemory{now: time.Now}

func (m *trailAuthorRefusalMemory) recent(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	expiry, ok := m.entries[key]
	return ok && expiry.After(m.now())
}

// remember stores a refusal for ttl. When full, expired entries are dropped
// first, then the whole memory is cleared.
func (m *trailAuthorRefusalMemory) remember(key string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if m.entries == nil {
		m.entries = map[string]time.Time{}
	}
	if _, ok := m.entries[key]; !ok && len(m.entries) >= trailAuthorRefusalMaxEntries {
		for k, expiry := range m.entries {
			if !expiry.After(now) {
				delete(m.entries, k)
			}
		}
		if len(m.entries) >= trailAuthorRefusalMaxEntries {
			clear(m.entries)
		}
	}
	m.entries[key] = now.Add(ttl)
}

func (m *trailAuthorRefusalMemory) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = nil
	m.now = time.Now
}

// confirmTrailAuthorAtOrigin returns nil only if the trail's origin attributes
// the trail at objectIRI to signer: the fetched object's id must equal objectIRI
// and its attributedTo must equal the signer's IRI. It is used to repair trails
// stored under the wrong author. The fetch has its own rate-limit identifier,
// a trailAuthorConfirmTimeout deadline and a body limit; refusals are remembered
// per object and signer.
func confirmTrailAuthorAtOrigin(ctx context.Context, objectIRI string, signer *core.Record) error {
	key := objectIRI + "\n" + signer.GetString("iri")
	if trailAuthorRefusals.recent(key) {
		return fmt.Errorf("origin recently refused to confirm this author change")
	}

	fetchCtx, cancel := context.WithTimeout(ctx, trailAuthorConfirmTimeout)
	defer cancel()
	fetchCtx = util.WithRateLimitIdentifier(fetchCtx, trailAuthorConfirmRateLimitID)

	trailObject, err := fetchTrailObject(fetchCtx, objectIRI)
	if err != nil {
		trailAuthorRefusals.remember(key, trailAuthorTransientRefusalTTL)
		return err
	}
	if trailObject.ID.String() != objectIRI {
		trailAuthorRefusals.remember(key, trailAuthorRefusalTTL)
		return fmt.Errorf("origin returned %q instead of %q", trailObject.ID.String(), objectIRI)
	}
	if trailObject.AttributedTo == nil || trailObject.AttributedTo.GetLink().String() != signer.GetString("iri") {
		trailAuthorRefusals.remember(key, trailAuthorRefusalTTL)
		return fmt.Errorf("origin does not attribute the trail to the signer")
	}
	return nil
}

// fetchTrail stores a copy of the remote trail at iri that a comment or
// summit log by sender replies to. The trail belongs to whoever it is
// attributed to, not to the sender: a copy filed under the sender's name
// would show up as theirs and would refuse the real author's Delete.
//
// The object is fetched from the trail's host without authentication, so
// its attributedTo is only trusted for an actor on that same host: a
// Wanderer trail is always its author's, and a host naming an actor
// elsewhere would otherwise file content under a stranger's name.
func fetchTrail(app core.App, ctx context.Context, sender *core.Record, iri string) (*core.Record, error) {
	trailObject, err := fetchTrailObject(ctx, iri)
	if err != nil {
		return nil, err
	}

	var authorIRI string
	if trailObject.AttributedTo != nil {
		authorIRI = trailObject.AttributedTo.GetLink().String()
	}
	if authorIRI == "" {
		return nil, fmt.Errorf("trail %s is attributed to nobody", iri)
	}
	if !sameHost(authorIRI, iri) {
		return nil, fmt.Errorf("trail %s is attributed to %s on another host", iri, authorIRI)
	}

	author := sender
	if authorIRI != sender.GetString("iri") {
		ctx, err := util.GetSafeActorContext(nil, sender)
		if err != nil {
			return nil, err
		}
		// A cached actor comes back even when refreshing it failed, and is
		// good enough to attribute the copy to.
		author, err = GetActorByIRI(app, ctx, authorIRI, false)
		if author == nil {
			return nil, err
		}
	}

	activity := pub.ActivityNew(pub.IRI("new"), pub.CreateType, trailObject)
	return util.TrailFromActivity(ctx, *activity, app, author)
}

func sameHost(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	return errA == nil && errB == nil && ua.Host != "" && strings.EqualFold(ua.Host, ub.Host)
}

func ActorsFromMentions(app core.App, ctx context.Context, htmlStr string) ([]*core.Record, error) {
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
		actor, err := GetActorByHandle(app, ctx, h, false)
		if err != nil {
			continue
		}
		actors = append(actors, actor)
	}

	return actors, nil
}
