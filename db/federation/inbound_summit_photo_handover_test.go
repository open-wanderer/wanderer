package federation

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"testing"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type repairedSummitPhotoClient struct{ downloads *int }

func (client repairedSummitPhotoClient) Do(*http.Request) (*http.Response, error) {
	*client.downloads++
	return nil, errors.New("migrated contributor photo should not be downloaded")
}

func TestInboundContributorEntryAdoptsMigratedPhotoAndRemovesIt(t *testing.T) {
	f := setupAddressingTestApp(t)
	trailAuthor := f.record(t, "activitypub_actors", map[string]any{"iri": "https://a.example/users/alice"})
	contributor := f.record(t, "activitypub_actors", map[string]any{"iri": "https://b.example/users/carol"})
	trail := f.record(t, "trails", map[string]any{"iri": "https://a.example/api/v1/trail/parent", "author": trailAuthor.Id, "public": true})
	entry := f.record(t, "summit_logs", map[string]any{"iri": "https://b.example/api/v1/summit-log/visit", "author": trailAuthor.Id, "trail": trail.Id, "text": "Earlier entry"})
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	const sourceName = "summit.png"
	file, err := filesystem.NewFileFromBytes(content.Bytes(), sourceName)
	if err != nil {
		t.Fatal(err)
	}
	file.Name = util.FederatedSourceFileName(sourceName)
	asset := testAddPhoto(t, f.app, entry, file, "")
	originalFile := asset.GetString("file")
	originalLink, err := f.app.FindFirstRecordByData("summit_log_assets", "summit_log", entry.Id)
	if err != nil {
		t.Fatal(err)
	}
	identity := "https://b.example/api/v1/assets/photo"
	object := pub.Object{
		ID: pub.IRI(entry.GetString("iri")), Type: pub.NoteType, InReplyTo: pub.IRI(trail.GetString("iri")),
		AttributedTo: pub.IRI(contributor.GetString("iri")), Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Updated entry")), Tag: pub.ItemCollection{},
		Attachment: pub.ItemCollection{&pub.Object{ID: pub.IRI(identity), Type: pub.ImageType, URL: pub.IRI("https://b.example/api/v1/files/assets/photo/" + sourceName)}},
	}
	downloads := 0
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(repairedSummitPhotoClient{downloads: &downloads}))
	for pass := 0; pass < 2; pass++ {
		activity := pub.Activity{ID: pub.IRI(fmt.Sprintf("https://b.example/activities/update-%d", pass)), Type: pub.UpdateType, Actor: pub.IRI(contributor.GetString("iri")), Object: &object}
		if err := processCreateOrUpdateSummitLogActivity(activity, f.app, contributor); err != nil {
			t.Fatal(err)
		}
		stored, err := f.app.FindRecordById("summit_logs", entry.Id)
		if err != nil || stored.GetString("author") != contributor.Id {
			t.Fatalf("inbound contributor author was not repaired: %v", err)
		}
		photos := testPhotoAssets(t, f.app, stored)
		if downloads != 0 || len(photos) != 1 || photos[0].Id != asset.Id || photos[0].GetString("author") != contributor.Id || photos[0].GetString("external_provider") != util.FederationAssetProvider || photos[0].GetString("external_id") != identity || photos[0].GetString("file") != originalFile {
			t.Fatalf("inbound contributor repair did not adopt the single migrated photo: pass=%d downloads=%d photos=%v", pass+1, downloads, photos)
		}
		link, err := f.app.FindFirstRecordByData("summit_log_assets", "summit_log", entry.Id)
		if err != nil || link.Id != originalLink.Id {
			t.Fatalf("inbound repair replaced the original photo link: %v", err)
		}
	}
	object.Attachment = pub.ItemCollection{}
	activity := pub.Activity{ID: "https://b.example/activities/remove-photo", Type: pub.UpdateType, Actor: pub.IRI(contributor.GetString("iri")), Object: &object}
	if err := processCreateOrUpdateSummitLogActivity(activity, f.app, contributor); err != nil {
		t.Fatal(err)
	}
	if photos := testPhotoAssets(t, f.app, entry); len(photos) != 0 || downloads != 0 {
		t.Fatalf("inbound photo removal did not remove the repaired entry link: photos=%d downloads=%d", len(photos), downloads)
	}
}
