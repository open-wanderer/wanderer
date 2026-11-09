package federation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"pocketbase/util"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func attachmentLimitAssets(t *testing.T, f *addressingFixture, author *core.Record, count int, remote bool) ([]*core.Record, pub.ItemCollection) {
	t.Helper()
	assets, err := f.app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"type", "storage_mode", "external_provider", "external_id", "author"} {
		assets.Fields.Add(&core.TextField{Name: name})
	}
	assets.Fields.Add(&core.JSONField{Name: "metadata"}, &core.AutodateField{Name: "created", OnCreate: true}, &core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1024})
	if err := f.app.Save(assets); err != nil {
		t.Fatal(err)
	}
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j6WQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(png, "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	result := make([]*core.Record, 0, count)
	attachments := pub.ItemCollection{}
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("https://a.example/api/v1/assets/photo-%d", i)
		record := core.NewRecord(assets)
		record.Set("author", author.Id)
		record.Set("type", "photo")
		record.Set("storage_mode", "copy")
		record.Set("file", file)
		if remote {
			record.Set("external_provider", util.FederationAssetProvider)
			record.Set("external_id", id)
		}
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
		}
		result = append(result, record)
		attachments = append(attachments, &pub.Object{ID: pub.IRI(id), Type: pub.ImageType, URL: pub.IRI("https://a.example/files/photo.png")})
	}
	return result, attachments
}

func TestInboundActivitiesLimitPhotoAttachmentsAndPreserveCappedCaches(t *testing.T) {
	for _, kind := range []string{"trail", "summit-log"} {
		for _, update := range []bool{false, true} {
			if kind == "trail" && update {
				continue
			}
			t.Run(fmt.Sprintf("%s/update=%v", kind, update), func(t *testing.T) {
				f := setupAddressingTestApp(t)
				author := f.record(t, "activitypub_actors", map[string]any{"iri": "https://a.example/users/alice"})
				parent := f.record(t, "trails", map[string]any{"iri": "https://a.example/api/v1/trail/parent", "author": author.Id, "public": true})
				assets, attachments := attachmentLimitAssets(t, f, author, 150, true)
				iri := "https://a.example/api/v1/" + kind + "/received"
				typ := pub.CreateType
				if update {
					typ = pub.UpdateType
					log := f.record(t, "summit_logs", map[string]any{"iri": iri, "author": author.Id, "trail": parent.Id})
					for _, asset := range assets {
						f.record(t, "summit_log_assets", map[string]any{"summit_log": log.Id, "asset": asset.Id})
					}
				}
				object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, To: pub.ItemCollection{pub.PublicNS}, AttributedTo: pub.IRI(author.GetString("iri")), InReplyTo: pub.IRI(parent.GetString("iri")), Content: pub.DefaultNaturalLanguageValue("Entry"), Location: &pub.Place{}, Tag: pub.ItemCollection{}, Attachment: attachments}
				activity := pub.Activity{Type: typ, Actor: pub.IRI(author.GetString("iri")), Object: object}
				processingErr := ProcessCreateOrUpdateActivity(f.app, author, author, activity)
				links := "trail_assets"
				want := util.MaxFederatedTrailPhotos
				if kind == "summit-log" {
					links = "summit_log_assets"
					want = util.MaxFederatedSummitLogPhotos
				}
				if update {
					want = 150
				}
				if count, err := f.app.CountRecords(links); err != nil || count != int64(want) {
					t.Fatalf("linked photos=%d, expected %d (err=%v)", count, want, err)
				}
				if processingErr != nil {
					t.Fatal(processingErr)
				}
			})
		}
	}
}

func TestSummitLogSenderMatchesItsPhotoAttachmentLimit(t *testing.T) {
	f := setupAddressingTestApp(t)
	author := f.actor(t, "author", true)
	follower := f.actor(t, "follower", false)
	f.follow(t, follower, author)
	trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": true, "name": "Trail"}, "trail")
	log := f.record(t, "summit_logs", map[string]any{"author": author.Id, "trail": trail.Id, "text": "Summit"}, "summit-log")
	assets, _ := attachmentLimitAssets(t, f, author, 25, false)
	for _, asset := range assets {
		f.record(t, "summit_log_assets", map[string]any{"summit_log": log.Id, "asset": asset.Id})
	}
	if err := CreateSummitLogActivity(f.app, t.Context(), log, pub.CreateType); err != nil {
		t.Fatal(err)
	}
	if f.in.wait("follower", 1, 3*time.Second) != 1 {
		t.Fatal("summit log activity was not delivered")
	}
	var activity pub.Activity
	if err := json.Unmarshal(f.in.body("follower"), &activity); err != nil {
		t.Fatal(err)
	}
	object, err := pub.ToObject(activity.Object)
	if err != nil {
		t.Fatal(err)
	}
	attachments, err := util.ActivityPubAttachmentItems(object.Attachment)
	if err != nil {
		t.Fatal(err)
	}
	if len(*attachments) != util.MaxFederatedSummitLogPhotos {
		t.Fatalf("sender delivered %d photo attachments", len(*attachments))
	}
}
