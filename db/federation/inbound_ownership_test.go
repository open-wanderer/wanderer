package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestInboundUpdatesKeepStoredAuthorsAndTrails(t *testing.T) {
	for _, kind := range []string{"comment", "summit-log"} {
		for _, activityType := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
			for _, scenario := range []string{"local-iri", "other-author", "other-trail", "wrong-origin"} {
				t.Run(kind+"/"+string(activityType)+"/"+scenario, func(t *testing.T) {
					f := setupAddressingTestApp(t)
					owner := f.record(t, "activitypub_actors", map[string]any{"iri": "https://local.example/users/alice", "is_local": true})
					sender := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/sender"})
					third := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/carol"})
					privateTrail := f.record(t, "trails", map[string]any{"iri": "https://local.example/api/v1/trail/private", "author": owner.Id, "name": "Private", "public": false})
					publicTrail := f.record(t, "trails", map[string]any{"iri": "https://remote.example/api/v1/trail/public", "author": sender.Id, "name": "Public", "public": true})
					collection := map[string]string{"comment": "comments", "summit-log": "summit_logs"}[kind]
					iri, author, trail := "https://remote.example/api/v1/"+kind+"/original", sender.Id, privateTrail.Id
					switch scenario {
					case "local-iri":
						iri, author = "https://local.example/api/v1/"+kind+"/original", owner.Id
					case "other-author":
						author, trail = third.Id, publicTrail.Id
					case "wrong-origin":
						iri, author, trail = "https://third.example/api/v1/"+kind+"/original", third.Id, publicTrail.Id
					}
					entry := f.record(t, collection, map[string]any{"iri": iri, "author": author, "trail": trail, "text": "Private original"})
					object := pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, To: pub.ItemCollection{pub.PublicNS}, InReplyTo: pub.IRI(publicTrail.GetString("iri")), Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Replacement")), Tag: pub.ItemCollection{}}
					activity := pub.ActivityNew(pub.IRI("https://remote.example/activity/update"), activityType, &object)
					activity.Actor = pub.IRI(sender.GetString("iri"))
					if err := ProcessCreateOrUpdateActivity(f.app, sender, nil, *activity); err == nil {
						t.Fatal("inbound activity took ownership of protected content")
					}
					stored, err := f.app.FindRecordById(collection, entry.Id)
					if err != nil {
						t.Fatal(err)
					}
					if stored.GetString("author") != author || stored.GetString("trail") != trail || stored.GetString("text") != "Private original" {
						t.Fatalf("protected record changed: %v", stored.PublicExport())
					}
					if current, err := f.app.FindRecordById("trails", privateTrail.Id); err != nil || current.GetBool("public") {
						t.Fatal("private source trail changed")
					}
				})
			}
		}
	}
}

func TestInboundCanonicalAuthorCanUpdateItsOwnEntry(t *testing.T) {
	for _, kind := range []string{"comment", "summit-log"} {
		t.Run(kind, func(t *testing.T) {
			f := setupAddressingTestApp(t)
			sender := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/alice"})
			trail := f.record(t, "trails", map[string]any{"iri": "https://remote.example/api/v1/trail/public", "author": sender.Id, "name": "Trail", "public": true})
			collection := map[string]string{"comment": "comments", "summit-log": "summit_logs"}[kind]
			iri := "https://remote.example/api/v1/" + kind + "/original"
			entry := f.record(t, collection, map[string]any{"iri": iri, "author": sender.Id, "trail": trail.Id, "text": "Original"})
			object := pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, To: pub.ItemCollection{pub.PublicNS}, InReplyTo: pub.IRI(trail.GetString("iri")), AttributedTo: pub.IRI(sender.GetString("iri")), Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Updated")), Tag: pub.ItemCollection{}}
			activity := pub.ActivityNew(pub.IRI("https://remote.example/activity/update"), pub.UpdateType, &object)
			activity.Actor = pub.IRI(sender.GetString("iri"))
			if err := ProcessCreateOrUpdateActivity(f.app, sender, nil, *activity); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(collection, entry.Id)
			if err != nil || stored.GetString("text") != "Updated" || stored.GetString("author") != sender.Id || stored.GetString("trail") != trail.Id {
				t.Fatalf("canonical update failed: record=%v error=%v", stored, err)
			}
		})
	}
}
