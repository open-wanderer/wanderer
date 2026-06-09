package federation

import (
	"database/sql"
	"fmt"
	"os"
	"pocketbase/util"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"

	pub "github.com/go-ap/activitypub"
)

type AnnounceType string

const (
	TrailAnnounceType AnnounceType = "trail"
	ListAnnounceType  AnnounceType = "list"
)

func CreateAnnounceActivity(app core.App, record *core.Record, typ AnnounceType) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	var subject *core.Record
	var object pub.Item
	var err error
	if typ == TrailAnnounceType {
		subject, err = app.FindRecordById("trails", record.GetString("trail"))
		if err != nil {
			return err
		}
		object, err = util.ObjectFromTrail(app, subject, nil)
		if err != nil {
			return err
		}
	} else if typ == ListAnnounceType {
		subject, err = app.FindRecordById("lists", record.GetString("list"))
		if err != nil {
			return err
		}
		object, err = util.ObjectFromList(app, subject)
		if err != nil {
			return err
		}

	} else {
		return fmt.Errorf("unknown announce type")
	}

	subjectActor, err := app.FindRecordById("activitypub_actors", subject.GetString("author"))
	if err != nil {
		return err
	}

	objectActor, err := app.FindRecordById("activitypub_actors", record.GetString("actor"))
	if err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		return err
	}

	recordId := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	id := fmt.Sprintf("%s/api/v1/activitypub/activity/%s", origin, recordId)
	to := objectActor.GetString("iri")
	actor := subjectActor.GetString("iri")

	activity := pub.AnnounceNew(pub.IRI(id), object)
	activity.To = pub.ItemCollection{pub.IRI(to)}
	activity.Actor = pub.IRI(actor)
	activity.Tag = pub.ItemCollection{
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "permission")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, record.GetString("permission"))),
		},
	}

	err = PostActivity(app, subjectActor, activity, []string{objectActor.GetString("inbox")})
	if err != nil {
		return err
	}

	activityRecord := core.NewRecord(collection)
	activityRecord.Set("id", recordId)
	activityRecord.Set("iri", id)
	activityRecord.Set("to", []string{to})
	activityRecord.Set("type", string(pub.AnnounceType))
	activityRecord.Set("object", object)
	activityRecord.Set("actor", actor)
	activityRecord.Set("published", time.Now())

	return app.Save(activityRecord)
}

// process incoming announce activity
func ProcessAnnounceActivity(app core.App, actor *core.Record, activity pub.Activity) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	object := activity.Object.GetID().String()

	if strings.Contains(object, "/api/v1/trail") {
		processTrailAnnounceActivity(app, actor, activity)

	} else if strings.Contains(object, "/api/v1/list") {
		processListAnnounceActivity(app, actor, activity)
	} else {
		return fmt.Errorf("unknown announce type")
	}
	return nil

}

func processTrailAnnounceActivity(app core.App, actor *core.Record, activity pub.Activity) error {

	objectActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", activity.To[0].GetID().String())
	if err != nil {
		return err
	}

	var trail *core.Record
	if !actor.GetBool("is_local") {
		trail, err = util.TrailFromActivity(activity, app, actor)
		if err != nil {
			return err
		}

		permission := "view"
		// tags, err := pub.ToItemCollection(activity.Tag)
		// if err != nil {
		// 	return err
		// }

		// for _, tag := range tags.Collection() {
		// 	tagObj, err := pub.ToObject(tag)
		// 	if err != nil {
		// 		continue
		// 	}
		// 	name := tagObj.Name.First().Value.String()
		// 	content := tagObj.Content.First().Value.String()
		// 	if name == "permission" {
		// 		permission = content
		// 	}
		// }

		record, err := app.FindFirstRecordByFilter("trail_share", "trail={:trailId}&&actor={:actorId}", dbx.Params{"trailId": trail.Id, "actorId": objectActor.Id})
		if err != nil {
			if err == sql.ErrNoRows {
				collection, err := app.FindCollectionByNameOrId("trail_share")
				if err != nil {
					return err
				}
				record = core.NewRecord(collection)
				record.Set("trail", trail.Id)
				record.Set("actor", objectActor.Id)
			} else {
				return err
			}
		}

		record.Set("permission", permission)

		err = app.Save(record)
		if err != nil {
			return err
		}
	} else {
		trail, err = app.FindFirstRecordByData("trails", "iri", activity.Object.GetID().String())
		if err != nil {
			return err
		}
	}

	notification := util.Notification{
		Type: util.TrailShare,
		Metadata: map[string]string{
			"id":     trail.Id,
			"trail":  trail.GetString("name"),
			"author": fmt.Sprintf("@%s@%s", actor.GetString("preferred_username"), actor.GetString("domain")),
		},
		Seen:   false,
		Author: actor.Id,
	}
	err = util.SendNotification(app, notification, objectActor)
	if err != nil {
		return err
	}

	return nil
}

func processListAnnounceActivity(app core.App, actor *core.Record, activity pub.Activity) error {

	objectActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", activity.To[0].GetID().String())
	if err != nil {
		return err
	}

	var list *core.Record
	if !actor.GetBool("is_local") {
		list, err = util.ListFromActivity(activity, app, actor)
		if err != nil {
			return err
		}

		record, err := app.FindFirstRecordByFilter("list_share", "list={:listId}&&actor={:actorId}", dbx.Params{"listId": list.Id, "actorId": objectActor.Id})
		if err != nil {
			if err == sql.ErrNoRows {
				collection, err := app.FindCollectionByNameOrId("list_share")
				if err != nil {
					return err
				}
				record = core.NewRecord(collection)
				record.Set("list", list.Id)
				record.Set("actor", objectActor.Id)
				record.Set("permission", "view")

			} else {
				return err
			}
		}

		err = app.Save(record)
		if err != nil {
			return err
		}
	} else {
		list, err = app.FindFirstRecordByData("lists", "iri", activity.Object.GetID().String())
		if err != nil {
			return err
		}
	}

	notification := util.Notification{
		Type: util.ListShare,
		Metadata: map[string]string{
			"id":     list.Id,
			"list":   list.GetString("name"),
			"author": fmt.Sprintf("@%s@%s", actor.GetString("preferred_username"), actor.GetString("domain")),
		},
		Seen:   false,
		Author: actor.Id,
	}
	err = util.SendNotification(app, notification, objectActor)
	if err != nil {
		return err
	}

	return nil

}
