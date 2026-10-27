package federation

import (
	"fmt"
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestLocalSharesNotifyRecipientsForAuthoredRecords(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		kind := map[string]string{"trails": "trail", "lists": "list"}[collection]
		for _, shape := range []struct {
			name    string
			pattern string
		}{
			{"instance-path", "https://a.example/api/v1/%s/{id}"},
			{"alternate-host-other-id", "https://b.example/api/v1/%s/aaaaaaaaaaaaaaa"},
			{"alternate-host-own-id", "https://b.example/api/v1/%s/{id}"},
			{"encoded-path-own-id", "https://b.example/wander%%20er/api/v1/%s/{id}"},
			{"query-own-id", "https://b.example/api/v1/%s/{id}?v=1"},
			{"fragment-own-id", "https://b.example/api/v1/%s/{id}#entry"},
			{"authority-userinfo-own-id", "https://account@b.example/api/v1/%s/{id}"},
		} {
			for _, authorship := range []string{"sender", "other"} {
				t.Run(collection+"/"+shape.name+"/"+authorship, func(t *testing.T) {
					f := setupAddressingTestApp(t)
					t.Setenv("ORIGIN", "https://a.example")
					actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
					if err != nil {
						t.Fatal(err)
					}
					actors.Fields.Add(&core.TextField{Name: "user"})
					if err := f.app.Save(actors); err != nil {
						t.Fatal(err)
					}
					settings := core.NewBaseCollection("settings")
					settings.Fields.Add(&core.TextField{Name: "user"}, &core.JSONField{Name: "notifications"})
					notifications := core.NewBaseCollection("notifications")
					notifications.Fields.Add(&core.TextField{Name: "type"}, &core.TextField{Name: "author"}, &core.TextField{Name: "recipient"}, &core.JSONField{Name: "metadata"}, &core.BoolField{Name: "seen"})
					for _, target := range []*core.Collection{settings, notifications} {
						if err := f.app.Save(target); err != nil {
							t.Fatal(err)
						}
					}
					actor := func(name string) *core.Record {
						return f.record(t, "activitypub_actors", map[string]any{
							"iri":                "https://a.example/api/v1/activitypub/user/" + name,
							"preferred_username": name, "domain": "a.example", "is_local": true,
						})
					}
					sender, recipient := actor("sender"), actor("recipient")
					recipient.Set("user", recipient.Id)
					if err := f.app.Save(recipient); err != nil {
						t.Fatal(err)
					}
					f.record(t, "settings", map[string]any{
						"user": recipient.Id,
						"notifications": map[string]any{
							"trail_share": map[string]any{"web": true, "email": false},
							"list_share":  map[string]any{"web": true, "email": false},
						},
					})
					author := sender
					if authorship == "other" {
						author = actor("other")
					}
					record := f.record(t, collection, map[string]any{"author": author.Id, "name": "Shared record", "public": false})
					iri := strings.ReplaceAll(fmt.Sprintf(shape.pattern, kind), "{id}", record.Id)
					record.Set("iri", iri)
					if err := f.app.Save(record); err != nil {
						t.Fatal(err)
					}
					activity := pub.AnnounceNew(pub.IRI("https://a.example/activities/share"), pub.IRI(iri))
					activity.Actor = pub.IRI(sender.GetString("iri"))
					activity.To = pub.ItemCollection{pub.IRI(recipient.GetString("iri"))}
					if collection == "trails" {
						err = processTrailAnnounceActivity(f.app, sender, *activity)
					} else {
						err = processListAnnounceActivity(f.app, sender, *activity)
					}
					want := int64(1)
					if authorship == "other" {
						want = 0
						if err == nil {
							t.Error("share of another author's record was accepted")
						}
					} else if err != nil {
						t.Errorf("share was refused: %v", err)
					}
					count, err := f.app.CountRecords("notifications", dbx.HashExp{"type": kind + "_share", "author": sender.Id, "recipient": recipient.Id})
					if err != nil || count != want {
						t.Fatalf("share notifications=%d, want %d, err=%v", count, want, err)
					}
				})
			}
		}
	}
}
