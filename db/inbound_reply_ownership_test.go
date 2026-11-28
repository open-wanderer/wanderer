package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestSignedInboxKeepsPrivateEntryParentAndPhotos(t *testing.T) {
	for _, typ := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
		for _, scenario := range []string{"local-entry", "sender-entry"} {
			t.Run(string(typ)+"/"+scenario, func(t *testing.T) {
				f := newTrailMentionInboxFixture(t)
				owner := f.actor(t, "owner", true)
				sender := f.actor(t, "sender", false)
				save := func(collection string, values map[string]any) *core.Record {
					return saveInboundRecord(t, f.app, collection, values)
				}
				private := save("trails", map[string]any{"name": "Private", "author": owner.Id, "iri": "https://example.com/api/v1/trail/private"})
				public := save("trails", map[string]any{"name": "Public", "author": sender.Id, "public": true, "iri": "https://remote.example/api/v1/trail/public"})
				entryAuthor := sender
				origin := "https://remote.example"
				if scenario == "local-entry" {
					entryAuthor, origin = owner, "https://example.com"
				}
				var photo bytes.Buffer
				if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
					t.Fatal(err)
				}
				file, err := filesystem.NewFileFromBytes(photo.Bytes(), "entry.png")
				if err != nil {
					t.Fatal(err)
				}
				entry := save("summit_logs", map[string]any{"author": entryAuthor.Id, "trail": private.Id, "text": "Private entry"})
				entry.Set("iri", origin+"/api/v1/summit-log/"+entry.Id)
				if err := f.app.Save(entry); err != nil {
					t.Fatal(err)
				}
				photoFields := map[string]any{
					"author": entryAuthor.Id, "type": "photo", "storage_mode": "copy",
					"remote_status": "available", "file": file,
				}
				if scenario == "sender-entry" {
					photoFields["external_provider"] = "federation"
					photoFields["external_id"] = origin + "/api/v1/assets/entryphoto"
				}
				asset := save("assets", photoFields)
				save("summit_log_assets", map[string]any{"summit_log": entry.Id, "asset": asset.Id})
				wantPhotos := []inboundSummitLogPhoto{{id: asset.Id, filename: asset.GetString("file")}}
				control := save("summit_logs", map[string]any{"author": sender.Id, "trail": public.Id, "text": "Original", "iri": "https://remote.example/api/v1/summit-log/control"})
				setupEventHandlers(&pocketbase.PocketBase{App: f.app}, newInboundSearchClient(t))
				assertPrivate := func() {
					t.Helper()
					current, err := f.app.FindRecordById("summit_logs", entry.Id)
					if err != nil {
						t.Fatal(err)
					}
					photos := inboundSummitLogPhotos(t, f.app, current.Id)
					if current.GetString("author") != entryAuthor.Id || current.GetString("trail") != private.Id || current.GetString("text") != "Private entry" || !slices.Equal(photos, wantPhotos) {
						t.Fatalf("private entry changed: author=%s trail=%s text=%q photos=%+v, want photos=%+v", current.GetString("author"), current.GetString("trail"), current.GetString("text"), photos, wantPhotos)
					}
					event := &core.RequestEvent{App: f.app}
					event.Request, event.Response = httptest.NewRequest(http.MethodGet, "/api/collections/summit_logs/records/"+entry.Id, nil), httptest.NewRecorder()
					info, err := event.RequestInfo()
					if err != nil {
						t.Fatal(err)
					}
					allowed, err := f.app.CanAccessRecord(current, info, current.Collection().ViewRule)
					if err != nil || allowed {
						t.Fatalf("private entry readable anonymously: allowed=%t err=%v", allowed, err)
					}
					parent, err := f.app.FindRecordById("trails", private.Id)
					if err != nil || parent.GetBool("public") {
						t.Fatalf("private parent changed: %v %v", parent, err)
					}
				}
				deliver := func(iri string) {
					t.Helper()
					object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, InReplyTo: pub.IRI(public.GetString("iri")), AttributedTo: pub.IRI(sender.GetString("iri")),
						Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Updated")), Tag: pub.ItemCollection{},
						Attachment: pub.ItemCollection{&pub.Object{Type: pub.DocumentType, MediaType: "text/plain"}}}
					activity := pub.Activity{ID: pub.IRI(sender.GetString("iri") + "/activity"), Type: typ, Actor: pub.IRI(sender.GetString("iri")), Object: object, To: pub.ItemCollection{pub.PublicNS}}
					body, err := json.Marshal(activity)
					if err != nil {
						t.Fatal(err)
					}
					if result := f.process(sender, owner, body); result.err != nil || result.status != http.StatusOK {
						t.Fatalf("signed inbox status=%d err=%v", result.status, result.err)
					}
				}
				assertPrivate()
				deliver(entry.GetString("iri"))
				assertPrivate()
				// A Create of a stored summit log changes nothing, so the Create
				// control is a summit log the instance does not hold yet.
				controlIRI := control.GetString("iri")
				if typ == pub.CreateType {
					controlIRI = "https://remote.example/api/v1/summit-log/new"
				}
				deliver(controlIRI)
				current, err := f.app.FindFirstRecordByData("summit_logs", "iri", controlIRI)
				if err != nil || current.GetString("text") != "Updated" {
					t.Fatalf("signed same-parent control was not applied: %v %v", current, err)
				}
			})
		}
	}
}

type inboundSummitLogPhoto struct {
	id       string
	filename string
}

func inboundSummitLogPhotos(t *testing.T, app core.App, summitLogID string) []inboundSummitLogPhoto {
	t.Helper()
	links, err := app.FindRecordsByFilter("summit_log_assets", "summit_log={:id}", "", -1, 0, dbx.Params{"id": summitLogID})
	if err != nil {
		t.Fatal(err)
	}
	storage, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	photos := make([]inboundSummitLogPhoto, 0, len(links))
	for _, link := range links {
		asset, err := app.FindRecordById("assets", link.GetString("asset"))
		if err != nil {
			t.Fatal(err)
		}
		name := asset.GetString("file")
		if asset.GetString("type") != "photo" || asset.GetString("storage_mode") != "copy" || asset.GetString("remote_status") != "available" || name == "" {
			t.Fatalf("summit log %s links to asset %s without an available copied photo", summitLogID, asset.Id)
		}
		if exists, err := storage.Exists(asset.BaseFilesPath() + "/" + name); err != nil || !exists {
			t.Fatalf("summit log %s photo %s file %q: exists=%t, err=%v", summitLogID, asset.Id, name, exists, err)
		}
		photos = append(photos, inboundSummitLogPhoto{id: asset.Id, filename: name})
	}
	return photos
}
