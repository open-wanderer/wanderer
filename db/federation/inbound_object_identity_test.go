package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

func TestInboundTrailAndListObjectsCannotChangeAnotherAuthorsRecords(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, scenario := range []string{"foreign-origin", "other-author", "historical-local", "attribution"} {
			t.Run(collection+"/"+scenario, func(t *testing.T) {
				f := setupAddressingTestApp(t)
				sender := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/sender"})
				author := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/alice"})
				origin := "https://remote.example"
				attributed := sender.GetString("iri")
				if scenario == "foreign-origin" {
					origin = "https://foreign.example"
				}
				if scenario == "historical-local" {
					origin = "https://historical.example"
					author.Set("is_local", true)
					if err := f.app.Save(author); err != nil {
						t.Fatal(err)
					}
				}
				if scenario == "attribution" {
					author = sender
					attributed = "https://remote.example/users/alice"
				}
				record := f.record(t, collection, map[string]any{"author": author.Id, "iri": origin + "/api/v1/" + map[string]string{"trails": "trail", "lists": "list"}[collection] + "/private", "name": "Original", "public": false})
				object := pub.Object{ID: pub.IRI(record.GetString("iri")), Type: pub.ImageType, AttributedTo: pub.IRI(attributed), Tag: pub.ItemCollection{}}
				activity := pub.Activity{Type: pub.UpdateType, Object: &object, Actor: pub.IRI(sender.GetString("iri"))}
				var err error
				if collection == "trails" {
					_, err = util.TrailFromActivity(activity, f.app, sender)
				} else {
					_, err = util.ListFromActivity(activity, f.app, sender)
				}
				if err == nil {
					t.Fatal("inbound object accepted for another author's cached/local record")
				}
				stored, err := f.app.FindRecordById(collection, record.Id)
				if err != nil || stored.GetString("name") != "Original" || stored.GetBool("needs_full_sync") {
					t.Fatalf("protected object changed: %v %v", stored, err)
				}
				shares := core.NewBaseCollection(map[string]string{"trails": "trail_share", "lists": "list_share"}[collection])
				field := map[string]string{"trails": "trail", "lists": "list"}[collection]
				shares.Fields.Add(&core.TextField{Name: field}, &core.TextField{Name: "actor"}, &core.TextField{Name: "permission"})
				if err := f.app.Save(shares); err != nil {
					t.Fatal(err)
				}
				local := f.record(t, "activitypub_actors", map[string]any{"iri": "https://local.example/users/eve", "is_local": true})
				activity.Type = pub.AnnounceType
				activity.To = pub.ItemCollection{pub.IRI(local.GetString("iri"))}
				if collection == "trails" {
					err = processTrailAnnounceActivity(f.app, sender, activity)
				} else {
					err = processListAnnounceActivity(f.app, sender, activity)
				}
				if err == nil {
					t.Fatal("announce accepted for a different author")
				}
				count, err := f.app.CountRecords(shares.Name)
				if err != nil || count != 0 {
					t.Fatalf("different author announce granted read access: count=%d err=%v", count, err)
				}
			})
		}
	}
}

func TestInboundNewObjectIdentityMustMatchSignerAndAttribution(t *testing.T) {
	f := setupAddressingTestApp(t)
	sender := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/alice"})
	for _, iri := range []string{"https://foreign.example/api/v1/trail/new", "https://local.example/api/v1/trail/new"} {
		for _, kind := range []string{"trail", "list"} {
			object := pub.Object{ID: pub.IRI(iri), Type: pub.ImageType, Tag: pub.ItemCollection{}, Location: &pub.Place{Type: pub.PlaceType}, Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Inbound"))}
			activity := pub.Activity{Type: pub.CreateType, Object: &object}
			var err error
			if kind == "trail" {
				_, err = util.TrailFromActivity(activity, f.app, sender)
			} else {
				_, err = util.ListFromActivity(activity, f.app, sender)
			}
			if err == nil {
				t.Fatalf("new %s accepted from foreign/local origin", kind)
			}
		}
	}
	for _, attribution := range []string{"https://remote.example/users/bob", "https://local.example/users/alice"} {
		if err := validateInboundObjectAuthor(sender, &pub.Object{ID: pub.IRI("https://remote.example/api/v1/comment/new"), AttributedTo: pub.IRI(attribution)}); err == nil {
			t.Fatal("new object attribution accepted for another actor")
		}
	}
}
