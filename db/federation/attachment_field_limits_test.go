package federation

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestIncomingAttachmentsUseConfiguredFileSizes(t *testing.T) {
	for _, tc := range []struct {
		kind, field string
		defaultSize int64
	}{
		{"trail", "photos", util.RemotePhotoMaxBytes},
		{"trail", "gpx", util.RemoteGPXMaxBytes},
		{"summit-log", "photos", util.RemotePhotoMaxBytes},
		{"summit-log", "gpx", util.RemoteGPXMaxBytes},
		{"list", "avatar", util.RemoteAvatarMaxBytes},
	} {
		t.Run(tc.kind+"/"+tc.field, func(t *testing.T) {
			f := setupAttachmentWireFixture(t)
			author := f.actor(t, "author", false)
			collection := map[string]string{"trail": "trails", "summit-log": "summit_logs", "list": "lists"}[tc.kind]
			col, err := f.app.FindCollectionByNameOrId(collection)
			if err != nil {
				t.Fatal(err)
			}
			storageField := tc.field
			if tc.field == "photos" {
				col, err = f.app.FindCollectionByNameOrId("assets")
				if err != nil {
					t.Fatal(err)
				}
				storageField = "file"
			}
			data := append(attachmentWirePhoto(t), bytes.Repeat([]byte(" "), int(tc.defaultSize))...)
			filename, mediaType := "photo.png", "image/png"
			if tc.field == "gpx" {
				data = append([]byte(`<gpx version="1.1"></gpx>`), bytes.Repeat([]byte(" "), int(tc.defaultSize))...)
				filename, mediaType = "track.gpx", "application/xml+gpx"
			}
			col.Fields.GetByName(storageField).(*core.FileField).MaxSize = int64(len(data))
			if err := f.app.Save(col); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", mediaType)
				w.Write(data)
			}))
			t.Cleanup(server.Close)
			t.Cleanup(util.SetRemoteAttachmentClientForTesting(server.Client()))
			attachment := pub.Object{Type: pub.ImageType, MediaType: pub.MimeType(mediaType), URL: pub.IRI(server.URL + "/" + filename)}
			if tc.field == "gpx" {
				attachment.Type = pub.DocumentType
			}
			object := &pub.Object{ID: pub.IRI("https://remote.example/api/v1/" + tc.kind + "/entry"), Type: pub.NoteType, Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Incoming")), AttributedTo: pub.IRI(author.GetString("iri")), Location: &pub.Place{Type: pub.PlaceType}, Attachment: util.ActivityPubAttachmentArray(pub.ItemCollection{attachment})}
			if tc.kind == "summit-log" {
				trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": true, "iri": "https://remote.example/api/v1/trail/parent"})
				object.InReplyTo = pub.IRI(trail.GetString("iri"))
			}
			activity := pub.Activity{Type: pub.CreateType, Actor: pub.IRI(author.GetString("iri")), Object: object}
			var record *core.Record
			switch tc.kind {
			case "trail":
				record, err = util.TrailFromActivity(activity, f.app, author)
			case "list":
				record, err = util.ListFromActivity(activity, f.app, author)
			case "summit-log":
				err = processCreateOrUpdateSummitLogActivity(activity, f.app, author)
				if err == nil {
					record, err = f.app.FindFirstRecordByData(collection, "iri", object.ID.String())
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			fileRecord := record
			if tc.field == "photos" {
				assets := testPhotoAssets(t, f.app, record)
				if len(assets) != 1 {
					t.Fatalf("configured photo assets=%d, want one", len(assets))
				}
				fileRecord = assets[0]
			}
			files := fileRecord.GetStringSlice(storageField)
			if len(files) != 1 {
				t.Fatalf("configured attachment missing: %v", files)
			}
			fs, err := f.app.NewFilesystem()
			if err != nil {
				t.Fatal(err)
			}
			defer fs.Close()
			reader, err := fs.GetReader(fileRecord.BaseFilesPath() + "/" + files[0])
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			count, err := io.Copy(io.Discard, reader)
			if err != nil || count != int64(len(data)) {
				t.Fatalf("configured attachment incomplete: size=%d expected=%d error=%v", count, len(data), err)
			}

			col.Fields.GetByName(storageField).(*core.FileField).MaxSize = tc.defaultSize
			if err := f.app.Save(col); err != nil {
				t.Fatal(err)
			}
			object.ID += "-large"
			if tc.field == "photos" {
				// This is a new photo; an existing canonical asset can be reused.
				attachment.URL = pub.IRI(server.URL + "/above-default-" + filename)
				object.Attachment = util.ActivityPubAttachmentArray(pub.ItemCollection{attachment})
			}
			switch tc.kind {
			case "trail":
				record, err = util.TrailFromActivity(activity, f.app, author)
			case "list":
				record, err = util.ListFromActivity(activity, f.app, author)
			case "summit-log":
				err = processCreateOrUpdateSummitLogActivity(activity, f.app, author)
				if err == nil {
					record, err = f.app.FindFirstRecordByData(collection, "iri", object.ID.String())
				}
			}
			if tc.field == "photos" {
				if err != nil || len(testPhotoFiles(t, f.app, record)) != 0 {
					t.Fatalf("photo above the default field limit imported: record=%v error=%v", record, err)
				}
			} else if err == nil {
				t.Fatal("required attachment above the default field limit accepted")
			}
		})
	}
}
