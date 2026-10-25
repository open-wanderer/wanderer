package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// Use the legacy photo, GPX and avatar file fields used by released peers.
// Saving real files also checks that an accepted incoming attachment is stored.
func setupAttachmentWireFixture(t *testing.T) *addressingFixture {
	t.Helper()
	f := setupAddressingTestApp(t)
	for _, name := range []string{"trails", "summit_logs", "lists"} {
		collection, err := f.app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		if name == "lists" {
			collection.Fields.Add(&core.FileField{Name: "avatar", MaxSelect: 1})
		} else {
			collection.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 99}, &core.FileField{Name: "gpx", MaxSelect: 1})
		}
		if err := f.app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func attachmentWirePhoto(t *testing.T) []byte {
	t.Helper()
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return imageData.Bytes()
}

func attachmentWireFile(t *testing.T, data []byte, name string) *filesystem.File {
	t.Helper()
	file, err := filesystem.NewFileFromBytes(data, name)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func parseAttachmentWireActivity(t *testing.T, wire []byte) *pub.Activity {
	t.Helper()
	item, err := pub.UnmarshalJSON(wire)
	if err != nil {
		t.Fatal(err)
	}
	activity, err := pub.ToActivity(item)
	if err != nil {
		t.Fatal(err)
	}
	return activity
}

func TestLegacyAttachmentsStayArraysOnTheWire(t *testing.T) {
	for _, tc := range []struct {
		name   string
		kind   string
		photos int
		gpx    bool
	}{
		{name: "trail GPX only", kind: "trail", gpx: true},
		{name: "trail one photo", kind: "trail", photos: 1},
		{name: "trail several photos and GPX", kind: "trail", photos: 3, gpx: true},
		{name: "summit log GPX only", kind: "summit-log", gpx: true},
		{name: "summit log one photo", kind: "summit-log", photos: 1},
		{name: "summit log photo and GPX", kind: "summit-log", photos: 1, gpx: true},
		{name: "list avatar", kind: "list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupAttachmentWireFixture(t)
			author := f.actor(t, "author", true)
			f.follow(t, f.actor(t, "follower", false), author)
			collection := map[string]string{"trail": "trails", "summit-log": "summit_logs", "list": "lists"}[tc.kind]
			values := map[string]any{"author": author.Id, "public": true, "name": "Ridge", "text": "Summit"}
			if tc.kind == "summit-log" {
				trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": true}, "trail")
				values["trail"] = trail.Id
			}
			photos := make([]*filesystem.File, tc.photos)
			for i := range photos {
				photos[i] = attachmentWireFile(t, attachmentWirePhoto(t), "photo.png")
			}
			values["photos"] = photos
			want := tc.photos
			if tc.gpx {
				values["gpx"] = attachmentWireFile(t, []byte(`<gpx version="1.1"></gpx>`), "track.gpx")
				want++
			}
			if tc.kind == "list" {
				values["avatar"] = attachmentWireFile(t, attachmentWirePhoto(t), "avatar.png")
				want++
			}
			record := f.record(t, collection, values, tc.kind)
			var err error
			switch tc.kind {
			case "trail":
				err = CreateTrailActivity(f.app, context.Background(), record, pub.CreateType)
			case "summit-log":
				err = CreateSummitLogActivity(f.app, context.Background(), record, pub.CreateType)
			case "list":
				err = CreateListActivity(f.app, record, pub.CreateType)
			}
			if err != nil {
				t.Fatal(err)
			}
			object := f.delivered(t, "follower")["object"].(map[string]any)
			attachments, ok := object["attachment"].([]any)
			if !ok || len(attachments) != want {
				t.Fatalf("wire attachment = %#v; want an array of %d", object["attachment"], want)
			}
			decoded, err := pub.ToObject(parseAttachmentWireActivity(t, f.in.body("follower")).Object)
			if err != nil {
				t.Fatal(err)
			}
			// Use the older receiver's conversion, after the real delivery's
			// JSON-LD serialization and ActivityStreams wire parsing.
			items, err := pub.ToItemCollection(decoded.Attachment)
			if err != nil || items.Count() != uint(want) {
				t.Fatalf("older receiver rejects attachments: items=%v error=%v", items, err)
			}
		})
	}
}

func TestLegacyBareAndArrayAttachmentsImportAfterWireParsing(t *testing.T) {
	for _, kind := range []string{"trail photo", "trail GPX", "summit log photo", "summit log GPX", "list avatar"} {
		for _, bare := range []bool{false, true} {
			name := "array"
			if bare {
				name = "bare"
			}
			t.Run(kind+"/"+name, func(t *testing.T) {
				f := setupAttachmentWireFixture(t)
				author := f.actor(t, "author", false)
				field, filename, contentType := "photos", "photo.png", "image/png"
				data := attachmentWirePhoto(t)
				if kind == "trail GPX" || kind == "summit log GPX" {
					field, filename, contentType = "gpx", "track.gpx", "application/xml+gpx"
					data = []byte(`<gpx version="1.1"></gpx>`)
				} else if kind == "list avatar" {
					field = "avatar"
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", contentType)
					_, _ = w.Write(data)
				}))
				t.Cleanup(server.Close)
				t.Cleanup(util.SetRemoteAttachmentClientForTesting(server.Client()))
				attachment := pub.Object{Type: pub.ImageType, MediaType: pub.MimeType(contentType), URL: pub.IRI(server.URL + "/" + filename)}
				if field == "gpx" {
					attachment.Type = pub.DocumentType
				}
				object := &pub.Object{Type: pub.NoteType, ID: "https://remote.example/objects/incoming", AttributedTo: pub.IRI(author.GetString("iri")),
					Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Incoming")), Location: &pub.Place{Type: pub.PlaceType}}
				if bare {
					object.Attachment = attachment
				} else {
					object.Attachment = util.ActivityPubAttachmentArray(pub.ItemCollection{attachment})
				}
				if kind == "summit log photo" || kind == "summit log GPX" {
					trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": true, "iri": "https://remote.example/trails/parent"})
					object.InReplyTo = pub.IRI(trail.GetString("iri"))
				}
				activity := pub.ActivityNew("https://remote.example/activities/incoming", pub.CreateType, object)
				activity.Actor = pub.IRI(author.GetString("iri"))
				wire, err := json.Marshal(activity)
				if err != nil {
					t.Fatal(err)
				}
				decoded := parseAttachmentWireActivity(t, wire)
				var record *core.Record
				switch kind {
				case "trail photo", "trail GPX":
					record, err = util.TrailFromActivity(*decoded, f.app, author)
				case "summit log photo", "summit log GPX":
					err = processCreateOrUpdateSummitLogActivity(*decoded, f.app, author)
					if err == nil {
						record, err = f.app.FindFirstRecordByData("summit_logs", "iri", object.ID.String())
					}
				case "list avatar":
					record, err = util.ListFromActivity(*decoded, f.app, author)
				}
				if err != nil {
					t.Fatal(err)
				}
				files := record.GetStringSlice(field)
				if len(files) != 1 {
					t.Fatalf("imported %s = %v; want one file", field, files)
				}
				fs, err := f.app.NewFilesystem()
				if err != nil {
					t.Fatal(err)
				}
				defer fs.Close()
				reader, err := fs.GetReader(record.BaseFilesPath() + "/" + files[0])
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				stored, err := io.ReadAll(reader)
				if err != nil || !bytes.Equal(stored, data) {
					t.Fatalf("imported file differs: data=%q error=%v", stored, err)
				}
			})
		}
	}
}
