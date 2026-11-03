package main

import (
	"encoding/json"
	"net/http"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// A peer's comment or summit log reaches a user inbox and the instance inbox,
// so the same Create arrives more than once. A Create of a stored reply leaves
// it as it is, also after an Update was applied; only an Update changes it.
func TestRepeatedCreateOfStoredReplyChangesNothing(t *testing.T) {
	for _, collection := range []string{"comments", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			f := newTrailMentionInboxFixture(t)
			owner := f.actor(t, "owner", true)
			sender := f.actor(t, "sender", false)
			trail := saveInboundRecord(t, f.app, "trails", map[string]any{"name": "Public", "author": sender.Id, "public": true, "iri": "https://remote.example/api/v1/trail/public"})
			iri := "https://remote.example/api/v1/comment/stored"
			if collection == "summit_logs" {
				iri = "https://remote.example/api/v1/summit-log/stored"
			}
			stored := saveInboundRecord(t, f.app, collection, map[string]any{"author": sender.Id, "trail": trail.Id, "text": "Original", "iri": iri})
			reloaded, err := f.app.FindRecordById(collection, stored.Id)
			if err != nil {
				t.Fatal(err)
			}
			updated := reloaded.GetDateTime("updated").String()

			deliveries := 0
			deliver := func(typ pub.ActivityVocabularyType, text string) {
				t.Helper()
				deliveries++
				object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, InReplyTo: pub.IRI(trail.GetString("iri")), AttributedTo: pub.IRI(sender.GetString("iri")),
					Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, text)), To: pub.ItemCollection{pub.PublicNS}}
				activity := pub.Activity{ID: pub.IRI(sender.GetString("iri") + "/activity/" + string(rune('a'+deliveries))), Type: typ, Actor: pub.IRI(sender.GetString("iri")), Object: object, To: pub.ItemCollection{pub.PublicNS}}
				body, err := json.Marshal(activity)
				if err != nil {
					t.Fatal(err)
				}
				if result := f.process(sender, owner, body); result.err != nil || result.status != http.StatusOK {
					t.Fatalf("signed inbox status=%d err=%v", result.status, result.err)
				}
			}
			current := func() *core.Record {
				t.Helper()
				record, err := f.app.FindRecordById(collection, stored.Id)
				if err != nil {
					t.Fatal(err)
				}
				return record
			}

			deliver(pub.CreateType, "Rewritten by a repeated Create")
			if record := current(); record.GetString("text") != "Original" || record.GetDateTime("updated").String() != updated {
				t.Fatalf("repeated Create changed the %s: text=%q updated=%v, want %q and %v", collection, record.GetString("text"), record.GetDateTime("updated"), "Original", updated)
			}

			deliver(pub.UpdateType, "Edited")
			if text := current().GetString("text"); text != "Edited" {
				t.Fatalf("Update was not applied: text=%q", text)
			}

			deliver(pub.CreateType, "Original")
			if text := current().GetString("text"); text != "Edited" {
				t.Fatalf("a Create processed after the Update restored older text: %q", text)
			}
		})
	}
}

// A summit log attachment without a URL has nothing to download and is
// skipped; the summit log is stored.
func TestSummitLogAttachmentWithoutURLIsSkipped(t *testing.T) {
	f := newTrailMentionInboxFixture(t)
	owner := f.actor(t, "owner", true)
	sender := f.actor(t, "sender", false)
	trail := saveInboundRecord(t, f.app, "trails", map[string]any{"name": "Public", "author": sender.Id, "public": true, "iri": "https://remote.example/api/v1/trail/public"})
	iri := "https://remote.example/api/v1/summit-log/no-url"
	object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, InReplyTo: pub.IRI(trail.GetString("iri")), AttributedTo: pub.IRI(sender.GetString("iri")),
		Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Summit")), To: pub.ItemCollection{pub.PublicNS},
		Attachment: pub.ItemCollection{
			&pub.Object{Type: pub.ImageType, MediaType: "image/png"},
			&pub.Object{Type: pub.DocumentType, MediaType: "application/xml+gpx"},
		}}
	activity := pub.Activity{ID: pub.IRI(sender.GetString("iri") + "/activity/no-url"), Type: pub.CreateType, Actor: pub.IRI(sender.GetString("iri")), Object: object, To: pub.ItemCollection{pub.PublicNS}}
	body, err := json.Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	if result := f.process(sender, owner, body); result.err != nil || result.status != http.StatusOK {
		t.Fatalf("signed inbox status=%d err=%v", result.status, result.err)
	}
	record, err := f.app.FindFirstRecordByData("summit_logs", "iri", iri)
	if err != nil {
		t.Fatalf("summit log was not stored: %v", err)
	}
	if record.GetString("gpx") != "" || len(record.GetStringSlice("photos")) != 0 {
		t.Fatalf("attachments without a URL were stored: gpx=%q photos=%v", record.GetString("gpx"), record.GetStringSlice("photos"))
	}
}
