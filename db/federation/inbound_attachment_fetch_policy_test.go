package federation

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestInboundAttachmentsKeepPublicAddressPolicyAtEveryEntryPoint(t *testing.T) {
	for _, tc := range []struct {
		name, kind string
		image      bool
	}{
		{"trail-photo", "trail", true},
		{"trail-gpx", "trail", false},
		{"summit-photo", "summit-log", true},
		{"summit-gpx", "summit-log", false},
		{"list-avatar", "list", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupAddressingTestApp(t)
			var hits atomic.Int32
			png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j6WQAAAAASUVORK5CYII=")
			if err != nil {
				t.Fatal(err)
			}
			sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write(png)
			}))
			defer sink.Close()
			parsed, _ := url.Parse(sink.URL)
			origin := "https://a.example:" + parsed.Port()
			// The port is legitimately the canonical origin's port, so only
			// the public-address policy prevents the loopback attachment.
			sender := f.record(t, "activitypub_actors", map[string]any{"iri": origin + "/users/alice"})
			parent := f.record(t, "trails", map[string]any{"iri": origin + "/api/v1/trail/parent", "author": sender.Id, "public": true})
			for _, collection := range []string{"trails", "summit_logs", "lists"} {
				col, err := f.app.FindCollectionByNameOrId(collection)
				if err != nil {
					t.Fatal(err)
				}
				field := "gpx"
				if collection == "lists" {
					field = "avatar"
				}
				col.Fields.Add(&core.FileField{Name: field, MaxSelect: 1, MaxSize: 1024})
				if err := f.app.Save(col); err != nil {
					t.Fatal(err)
				}
			}
			assets, err := f.app.FindCollectionByNameOrId("assets")
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"type", "storage_mode", "external_provider", "external_id", "author"} {
				assets.Fields.Add(&core.TextField{Name: name})
			}
			assets.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true}, &core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1024, MimeTypes: []string{"image/png"}}, &core.JSONField{Name: "metadata"})
			if err := f.app.Save(assets); err != nil {
				t.Fatal(err)
			}
			attachment := &pub.Object{Type: pub.DocumentType, MediaType: "application/xml+gpx", URL: pub.IRI(sink.URL + "/secret.gpx")}
			if tc.image {
				attachment.Type = pub.ImageType
				attachment.MediaType = "image/png"
				attachment.ID = pub.IRI(origin + "/api/v1/assets/source")
				attachment.URL = pub.IRI(sink.URL + "/secret.png")
			}
			iri := origin + "/api/v1/" + tc.kind + "/received"
			object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, To: pub.ItemCollection{pub.PublicNS}, AttributedTo: pub.IRI(sender.GetString("iri")), InReplyTo: pub.IRI(parent.GetString("iri")), Name: pub.DefaultNaturalLanguageValue("Remote"), Content: pub.DefaultNaturalLanguageValue("Remote"), Location: &pub.Place{}, Tag: pub.ItemCollection{}, Attachment: pub.ItemCollection{attachment}}
			activity := pub.Activity{Type: pub.CreateType, Actor: pub.IRI(sender.GetString("iri")), Object: object}
			err = ProcessCreateOrUpdateActivity(f.app, sender, sender, activity)
			if hits.Load() != 0 {
				t.Fatalf("%s fetched a private-address attachment (%s requests)", tc.name, strconv.Itoa(int(hits.Load())))
			}
			if tc.kind != "list" && tc.image {
				if err != nil {
					t.Fatal(err)
				}
				if count, err := f.app.CountRecords("assets"); err != nil || count != 0 {
					t.Fatalf("private attachment materialized: count=%d err=%v", count, err)
				}
			} else if err == nil {
				t.Fatal("private-address record file was accepted")
			}
		})
	}
}
