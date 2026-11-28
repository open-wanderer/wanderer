package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/doyensec/safeurl"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func setupAttachmentWireApp(t *testing.T) (*pbtests.TestApp, *core.Record) {
	t.Helper()
	app := setupFederatedAssetsTestApp(t)
	t.Cleanup(app.Cleanup)
	actors, _ := app.FindCollectionByNameOrId("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "preferred_username"}, &core.TextField{Name: "followers"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	author, _ := app.FindRecordById(actors, "actor0000000001")
	author.Set("iri", "https://a.example/users/author")
	if err := app.Save(author); err != nil {
		t.Fatal(err)
	}
	add := func(name string, fields ...core.Field) *core.Collection {
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	categories := add("categories", &core.TextField{Name: "name"})
	subcategories := add("subcategories", &core.TextField{Name: "name"})
	tags := add("tags", &core.TextField{Name: "name"})
	add("trails", &core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.TextField{Name: "location"}, &core.TextField{Name: "gpx"}, &core.BoolField{Name: "public"}, &core.BoolField{Name: "needs_full_sync"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1},
		&core.RelationField{Name: "subcategory", CollectionId: subcategories.Id, MaxSelect: 1},
		&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 99})
	add("lists", &core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.TextField{Name: "avatar"}, &core.BoolField{Name: "public"}, &core.BoolField{Name: "needs_full_sync"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1})
	return app, author
}

func attachmentWireObject(t *testing.T, object *pub.Object) (*pub.Object, map[string]any) {
	t.Helper()
	activity := pub.ActivityNew(pub.IRI("https://a.example/activities/one"), pub.CreateType, object)
	activity.Actor = object.AttributedTo
	wire, err := jsonld.WithContext(jsonld.IRI(pub.ActivityBaseURI), jsonld.IRI(pub.SecurityContextURI)).Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(wire, &raw); err != nil {
		t.Fatal(err)
	}
	item, err := pub.UnmarshalJSON(wire)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pub.ToActivity(item)
	if err != nil {
		t.Fatal(err)
	}
	decodedObject, err := pub.ToObject(decoded.Object)
	if err != nil {
		t.Fatal(err)
	}
	return decodedObject, raw["object"].(map[string]any)
}

func TestOutgoingTrailAndListAttachmentsStayArraysOnTheWire(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	for _, test := range []struct {
		kind   string
		photos int
		gpx    bool
	}{{"trail", 0, true}, {"trail", 1, false}, {"trail", 1, true}, {"trail", 3, false}, {"list", 0, false}} {
		t.Run(fmt.Sprintf("%s/photos%d/gpx%t", test.kind, test.photos, test.gpx), func(t *testing.T) {
			app, author := setupAttachmentWireApp(t)
			collectionName := map[string]string{"trail": "trails", "list": "lists"}[test.kind]
			collection, _ := app.FindCollectionByNameOrId(collectionName)
			record := core.NewRecord(collection)
			record.Set("author", author.Id)
			record.Set("public", true)
			record.Set("iri", "https://a.example/api/v1/"+test.kind+"/outgoing")
			if test.gpx {
				record.Set("gpx", "track.gpx")
			}
			if test.kind == "list" {
				record.Set("avatar", "avatar.jpg")
			}
			if err := app.Save(record); err != nil {
				t.Fatal(err)
			}
			for i := range test.photos {
				asset := createFederatedTestAsset(t, app, author.Id, fmt.Sprintf("https://a.example/photos/%d", i))
				asset.Set("file", fmt.Sprintf("photo%d.jpg", i))
				if err := app.Save(asset); err != nil {
					t.Fatal(err)
				}
				createFederatedTestLink(t, app, "trail_assets", "trail", record.Id, asset.Id)
			}
			var object *pub.Object
			var err error
			if test.kind == "trail" {
				object, err = ObjectFromTrail(app, record, nil)
			} else {
				object, err = ObjectFromList(app, record)
			}
			if err != nil {
				t.Fatal(err)
			}
			decoded, raw := attachmentWireObject(t, object)
			attachments, ok := raw["attachment"].([]any)
			want := test.photos
			if test.gpx || test.kind == "list" {
				want++
			}
			if !ok || len(attachments) != want {
				t.Fatalf("attachment is not an array of %d: %#v", want, raw["attachment"])
			}
			// This is the older receiver's collection conversion, after the
			// exact JSON-LD serializer used by PostActivity and wire parsing.
			oldPeerItems, err := pub.ToItemCollection(decoded.Attachment)
			if err != nil || oldPeerItems.Count() != uint(want) {
				t.Fatalf("old peer rejected attachments: %v", err)
			}
		})
	}
}

func TestIncomingBarePhotoAttachmentImportsAfterWireParsing(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, bare := range []bool{false, true} {
		t.Run(fmt.Sprintf("bare=%t", bare), func(t *testing.T) {
			app, author := setupAttachmentWireApp(t)
			asset := createFederatedTestAsset(t, app, author.Id, "https://a.example/photos/one")
			asset.Set("file", "photo.jpg")
			if err := app.Save(asset); err != nil {
				t.Fatal(err)
			}
			attachment := FederatedPhotoImage(asset, "https://a.example")
			object := &pub.Object{Type: pub.NoteType, ID: "https://a.example/api/v1/trail/incoming", AttributedTo: pub.IRI(author.GetString("iri")),
				Location: &pub.Place{Type: pub.PlaceType}, Tag: pub.ItemCollection{}}
			if bare {
				object.Attachment = attachment
			} else {
				object.Attachment = ActivityPubAttachmentArray(pub.ItemCollection{attachment})
			}
			decoded, _ := attachmentWireObject(t, object)
			trail, err := TrailFromActivity(pub.Activity{Object: decoded}, app, author)
			if err != nil {
				t.Fatal(err)
			}
			assertFederatedLinkCount(t, app, "trail_assets", "trail", trail.Id, 1)
			assertFederatedAssetCount(t, app, 1)
		})
	}
}

func TestIncomingBareGPXAndAvatarReachTheMediaPolicy(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			app, author := setupAttachmentWireApp(t)
			attachment := pub.Object{Type: pub.ImageType, URL: pub.IRI("http://127.0.0.1/attachment")}
			if kind == "trail" {
				attachment.Type = pub.DocumentType
				attachment.MediaType = "application/xml+gpx"
			}
			object := &pub.Object{Type: pub.NoteType, ID: pub.IRI("https://a.example/api/v1/" + kind + "/incoming"),
				AttributedTo: pub.IRI(author.GetString("iri")), Attachment: attachment, Location: &pub.Place{Type: pub.PlaceType}, Tag: pub.ItemCollection{}}
			decoded, _ := attachmentWireObject(t, object)
			var err error
			if kind == "trail" {
				_, err = TrailFromActivity(pub.Activity{Object: decoded}, app, author)
			} else {
				_, err = ListFromActivity(pub.Activity{Object: decoded}, app, author)
			}
			var blockedIP *safeurl.AllowedIPError
			if !errors.As(err, &blockedIP) {
				t.Fatalf("single attachment failed before the media fetch policy: %v", err)
			}
		})
	}
}
