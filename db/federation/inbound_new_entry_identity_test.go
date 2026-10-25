package federation

import (
	pub "github.com/go-ap/activitypub"
	"testing"
)

func TestInboundNewEntriesValidateOriginAndAttributionThroughProcessing(t *testing.T) {
	for _, kind := range []string{"comment", "summit-log"} {
		for _, scenario := range []string{"foreign-origin", "unseen-local-iri", "different-attribution", "empty-attribution-collection"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := setupAddressingTestApp(t)
				sender := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/alice"})
				trail := f.record(t, "trails", map[string]any{"iri": "https://remote.example/api/v1/trail/parent", "author": sender.Id, "public": true})
				origin := "https://remote.example"
				var attribution pub.Item = pub.IRI(sender.GetString("iri"))
				switch scenario {
				case "foreign-origin":
					origin = "https://foreign.example"
				case "unseen-local-iri":
					origin = "https://local.example"
				case "different-attribution":
					attribution = pub.IRI("https://remote.example/users/bob")
				case "empty-attribution-collection":
					attribution = pub.ItemCollection{}
				}
				object := pub.Object{ID: pub.IRI(origin + "/api/v1/" + kind + "/unseen"), Type: pub.NoteType, To: pub.ItemCollection{pub.PublicNS}, InReplyTo: pub.IRI(trail.GetString("iri")), AttributedTo: attribution, Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Replacement")), Tag: pub.ItemCollection{}}
				activity := pub.Activity{Type: pub.CreateType, Actor: pub.IRI(sender.GetString("iri")), Object: &object}
				if err := ProcessCreateOrUpdateActivity(f.app, sender, nil, activity); err == nil {
					t.Fatal("processing accepted a new object without canonical origin and author")
				}
				collection := map[string]string{"comment": "comments", "summit-log": "summit_logs"}[kind]
				count, err := f.app.CountRecords(collection)
				if err != nil || count != 0 {
					t.Fatalf("invalid new object persisted: count=%d err=%v", count, err)
				}
			})
		}
	}
}
