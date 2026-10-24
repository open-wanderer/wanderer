package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestAnnounceRejectsEmptyRecipientAndLocalSignerWithoutOwnership(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, scenario := range []string{"empty-recipient", "local-signer"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := setupAddressingTestApp(t)
				signer := f.actor(t, "signer", true)
				owner := f.actor(t, "owner", false)
				// A remote recipient needs no notification preferences or email.
				recipient := f.actor(t, "recipient", false)
				collection := map[string]string{"trail": "trails", "list": "lists"}[kind]
				record := f.record(t, collection, map[string]any{"author": owner.Id, "name": "Original", "iri": "https://local.example/api/v1/" + kind + "/unowned"})
				activity := pub.Activity{Type: pub.AnnounceType, Actor: pub.IRI(signer.GetString("iri")), Object: pub.IRI(record.GetString("iri"))}
				if scenario != "empty-recipient" {
					activity.To = pub.ItemCollection{pub.IRI(recipient.GetString("iri"))}
				}
				if err := ProcessAnnounceActivity(f.app, signer, activity); err == nil {
					t.Fatal("announce accepted without recipient or local ownership")
				}
				stored, err := f.app.FindRecordById(collection, record.Id)
				if err != nil || stored.GetString("author") != owner.Id || stored.GetString("name") != "Original" {
					t.Fatalf("rejected announce changed the object: %v %v", stored, err)
				}
			})
		}
	}
}

func TestInboundSummitLogDoesNotRewriteALocalOwnerOfAnImportedIdentity(t *testing.T) {
	f := setupAddressingTestApp(t)
	owner := f.actor(t, "owner", true)
	sender := f.actor(t, "sender", false)
	trail := f.record(t, "trails", map[string]any{"author": owner.Id, "name": "Private local"}, "trail")
	log := f.record(t, "summit_logs", map[string]any{"trail": trail.Id, "author": owner.Id, "text": "Original", "iri": "https://remote.example/api/v1/summit-log/imported-id"})
	object := pub.Object{ID: pub.IRI(log.GetString("iri")), Type: pub.NoteType, AttributedTo: pub.IRI(sender.GetString("iri")),
		InReplyTo: pub.IRI(trail.GetString("iri")), Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Replacement")), Tag: pub.ItemCollection{}}
	activity := pub.Activity{Type: pub.UpdateType, Actor: pub.IRI(sender.GetString("iri")), Object: &object}
	if err := ProcessCreateOrUpdateActivity(f.app, sender, nil, activity); err == nil {
		t.Fatal("imported identity granted ownership of a local user's log")
	}
	stored, err := f.app.FindRecordById("summit_logs", log.Id)
	if err != nil || stored.GetString("author") != owner.Id || stored.GetString("text") != "Original" || stored.GetString("trail") != trail.Id {
		t.Fatalf("local log was rewritten before rejection: %v %v", stored, err)
	}
}

func TestInboundOwnedTrailAndListDeletesRemoveTheirFeedEntries(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f := setupAddressingTestApp(t)
			sender := f.actor(t, "sender", false)
			collection := map[string]string{"trail": "trails", "list": "lists"}[kind]
			record := f.record(t, collection, map[string]any{"author": sender.Id, "iri": "https://remote.example/api/v1/" + kind + "/remote-id"})
			feed := f.record(t, "feed", map[string]any{"actor": sender.Id, "author": sender.Id, "item": record.Id, "type": kind})
			activity := pub.Activity{Type: pub.DeleteType, Actor: pub.IRI(sender.GetString("iri")), Object: pub.IRI(record.GetString("iri"))}
			if err := ProcessDeleteActivity(f.app, sender, activity); err != nil {
				t.Fatal(err)
			}
			if _, err := f.app.FindRecordById(collection, record.Id); err == nil {
				t.Fatal("owned remote object was not deleted")
			}
			if _, err := f.app.FindRecordById("feed", feed.Id); err == nil {
				t.Fatal("object deletion left its feed entry")
			}
		})
	}
}
