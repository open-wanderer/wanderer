package federation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"

	"pocketbase/util"
)

// All attachment downloads of one inbound activity share one budget, and the
// GPX is downloaded before the photos.

// attachmentBudgetBody serves gpxSize bytes for .gpx paths and photoSize bytes
// for everything else.
func attachmentBudgetBody(gpxSize, photoSize int) func(string) []byte {
	return func(path string) []byte {
		if strings.HasSuffix(path, ".gpx") {
			return []byte(strings.Repeat("g", gpxSize))
		}
		return []byte(strings.Repeat("p", photoSize))
	}
}

func attachmentBudgetPhotos(base, prefix string, n int) []pub.Item {
	items := make([]pub.Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, attachmentImage(fmt.Sprintf("%s/%s%03d.png", base, prefix, i)))
	}
	return items
}

// A budget smaller than the photos listed never costs the trail its GPX, even
// when the GPX is listed last.
func TestTrailCreateAttachmentCountBudget(t *testing.T) {
	const files = 10
	t.Cleanup(util.SetRemoteAttachmentBudgetLimitsForTesting(util.RemoteActivityAttachmentMaxBytes, files))
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	attachments := append(attachmentManyPhotos(server.URL, 50), attachmentGPX(server.URL+"/t.gpx"))
	if err := attachmentDeliverTrail(f, attachments...); err != nil {
		t.Fatalf("trail with 50 photos and a budget of %d files: %v", files, err)
	}

	if n := hits.total(); n != files {
		t.Errorf("server received %d requests, want exactly the budget (%d)", n, files)
	}
	stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
	if stored.GetString("gpx") == "" {
		t.Error("stored trail has no gpx file")
	}
	if n := len(stored.GetStringSlice("photos")); n != files-1 {
		t.Errorf("stored photos = %d, want %d", n, files-1)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

func TestTrailCreateAttachmentByteBudget(t *testing.T) {
	const gpxBytes, photoBytes = 4, 10
	// Room for the GPX, four photos and five bytes: the fifth photo is refused.
	t.Cleanup(util.SetRemoteAttachmentBudgetLimitsForTesting(gpxBytes+4*photoBytes+5, util.RemoteActivityAttachmentMaxFiles))
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, _ := attachmentServer(t, attachmentBudgetBody(gpxBytes, photoBytes))

	attachments := append(attachmentBudgetPhotos(server.URL, "p", 5), attachmentGPX(server.URL+"/t.gpx"))
	if err := attachmentDeliverTrail(f, attachments...); err != nil {
		t.Fatalf("trail Create: %v", err)
	}

	stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
	if stored.GetString("gpx") == "" {
		t.Error("stored trail has no gpx file")
	}
	if n := len(stored.GetStringSlice("photos")); n != 4 {
		t.Errorf("stored photos = %d, want 4", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

// The budget does not cut the 99 photos the schema allows.
func TestTrailCreateBudgetKeepsAllNinetyNinePhotos(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	attachments := append(attachmentManyPhotos(server.URL, 99), attachmentGPX(server.URL+"/t.gpx"))
	if err := attachmentDeliverTrail(f, attachments...); err != nil {
		t.Fatalf("trail with a GPX and 99 photos: %v", err)
	}

	if n := hits.total(); n != 100 {
		t.Errorf("server received %d requests, want 100", n)
	}
	stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
	if stored.GetString("gpx") == "" {
		t.Error("stored trail has no gpx file")
	}
	if n := len(stored.GetStringSlice("photos")); n != 99 {
		t.Errorf("stored photos = %d, want 99", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

// A summit log replying to an uncached trail stores that trail in the same
// activity; both draw from one budget, and the summit log keeps its GPX.
func TestSummitLogSharesBudgetWithFetchedTrail(t *testing.T) {
	const trailPhotos, logPhotos = 60, 70
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	const trailIRI = "https://remote.example.com/api/v1/trail/uncached"
	trailAct := feedDedupTrailCreate(ownerYIRI, trailIRI)
	trailObject := trailAct.Object.(*pub.Object)
	trailObject.AttributedTo = pub.IRI(ownerYIRI)
	trailObject.Attachment = pub.ItemCollection(append(
		attachmentBudgetPhotos(server.URL, "tp", trailPhotos),
		attachmentGPX(server.URL+"/trail.gpx"),
	))

	trailAuthorRefusals.reset()
	orig := fetchTrailObject
	fetchTrailObject = func(ctx context.Context, iri string) (*pub.Object, error) {
		if iri != trailIRI {
			t.Errorf("fetched %q, want %q", iri, trailIRI)
		}
		return trailObject, nil
	}
	t.Cleanup(func() {
		fetchTrailObject = orig
		trailAuthorRefusals.reset()
	})

	act := ownerSummitLogActivity(pub.CreateType, ownerYIRI, ownerRemoteLogIRI, trailIRI, "summit")
	act.Object.(*pub.Object).Attachment = pub.ItemCollection(append(
		attachmentBudgetPhotos(server.URL, "sp", logPhotos),
		attachmentGPX(server.URL+"/log.gpx"),
	))

	if err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.owner.alice, act); err != nil {
		t.Fatalf("summit log Create: %v", err)
	}

	if n := hits.total(); n != util.RemoteActivityAttachmentMaxFiles {
		t.Errorf("server received %d requests, want exactly the budget (%d)", n, util.RemoteActivityAttachmentMaxFiles)
	}
	trail := attachmentStored(t, f.app, "trails", trailIRI)
	if trail.GetString("gpx") == "" {
		t.Error("stored trail has no gpx file")
	}
	if n := len(trail.GetStringSlice("photos")); n != trailPhotos {
		t.Errorf("stored trail photos = %d, want %d", n, trailPhotos)
	}
	log := attachmentStored(t, f.app, "summit_logs", ownerRemoteLogIRI)
	if log.GetString("gpx") == "" {
		t.Error("stored summit log has no gpx file")
	}
	wantLogPhotos := util.RemoteActivityAttachmentMaxFiles - (1 + trailPhotos) - 1
	if n := len(log.GetStringSlice("photos")); n != wantLogPhotos {
		t.Errorf("stored summit log photos = %d, want %d", n, wantLogPhotos)
	}
	attachmentRequireNoTempFiles(t, tmp)
}
