package federation

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

// Attachment downloads must not reach loopback or private addresses, must be
// limited in size, time and number, and must not leave temp files behind.

// attachmentGPXFixtureMaxSize is larger than the production gpx maxSize so the
// oversize test hits the download limit, not the collection validation.
const attachmentGPXFixtureMaxSize = 64 << 20

type attachmentFixture struct {
	app   core.App
	owner feedDedupActors
	// y is a remote actor on remote.example.com, used for summit logs.
	y *core.Record
}

func attachmentAddCollections(t *testing.T, app core.App) {
	t.Helper()
	text := func(name string) core.Field { return &core.TextField{Name: name} }

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		text("iri"), text("author"), text("name"),
		&core.BoolField{Name: "public"},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: attachmentGPXFixtureMaxSize},
		&core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 20 << 20},
	)
	summitLogs := core.NewBaseCollection("summit_logs")
	summitLogs.Fields.Add(
		text("iri"), text("author"), text("trail"), text("text"),
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: attachmentGPXFixtureMaxSize},
		&core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 20 << 20},
	)
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		text("iri"), text("name"), text("description"), text("author"),
		&core.BoolField{Name: "public"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.FileField{Name: "avatar", MaxSelect: 1, MaxSize: 5 << 20},
	)
	for _, c := range []*core.Collection{trails, summitLogs, lists} {
		if err := app.Save(c); err != nil {
			t.Fatalf("save collection %s: %v", c.Name, err)
		}
	}
}

// attachmentApp builds an app with the file fields attachment downloads write
// to. It uses the production attachment client unless a test calls
// attachmentUseLoopbackClient.
func attachmentApp(t *testing.T) *attachmentFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	// Make sure no override leaks in from another test.
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(nil))

	app := newInboxTestApp(t)
	addFeedCollection(t, app)
	addCommentsCollection(t, app)
	attachmentAddCollections(t, app)

	f := &attachmentFixture{app: app, owner: feedDedupSeedActors(t, app)}
	f.y = createTestActor(t, app, ownerYIRI, "person", false)
	return f
}

// attachmentUseLoopbackClient reaches httptest servers through the test seam.
func attachmentUseLoopbackClient(t *testing.T) {
	t.Helper()
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(&http.Client{Timeout: 30 * time.Second}))
}

// attachmentIsolatedTempDir points os.TempDir at a fresh directory.
func attachmentIsolatedTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	return dir
}

func attachmentRequireNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	left, err := filepath.Glob(filepath.Join(dir, "wanderer-attachment-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("attachment temp files left behind: %v", left)
	}
}

type attachmentHits struct {
	mu    sync.Mutex
	count int32
	paths []string
}

func (h *attachmentHits) total() int { return int(atomic.LoadInt32(&h.count)) }

// attachmentServer serves body(path) for every request and records the hits.
func attachmentServer(t *testing.T, body func(path string) []byte) (*httptest.Server, *attachmentHits) {
	t.Helper()
	hits := &attachmentHits{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits.count, 1)
		hits.mu.Lock()
		hits.paths = append(hits.paths, r.URL.Path)
		hits.mu.Unlock()
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(body(r.URL.Path))
	}))
	t.Cleanup(server.Close)
	return server, hits
}

func attachmentSmallBody(string) []byte { return []byte("data") }

func attachmentGPX(url string) pub.Item {
	return &pub.Object{Type: pub.DocumentType, MediaType: "application/xml+gpx", URL: pub.IRI(url)}
}

func attachmentImage(url string) pub.Item {
	return &pub.Object{Type: pub.ImageType, URL: pub.IRI(url)}
}

func attachmentTrailActivity(f *attachmentFixture, attachments ...pub.Item) pub.Activity {
	act := feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI)
	act.Object.(*pub.Object).Attachment = pub.ItemCollection(attachments)
	return act
}

func attachmentDeliverTrail(f *attachmentFixture, attachments ...pub.Item) error {
	return ProcessCreateOrUpdateActivity(f.app, context.Background(), f.owner.author, f.owner.alice, attachmentTrailActivity(f, attachments...))
}

// attachmentSummitLogActivity builds a Create of a summit log by y on a trail
// of y that is seeded here.
func attachmentSummitLogActivity(t *testing.T, f *attachmentFixture, attachments ...pub.Item) pub.Activity {
	t.Helper()
	seedTrailRecord(t, f.app, ownerRemoteTrailIRI, true, f.y.Id)
	act := ownerSummitLogActivity(pub.CreateType, ownerYIRI, ownerRemoteLogIRI, ownerRemoteTrailIRI, "summit")
	act.Object.(*pub.Object).Attachment = pub.ItemCollection(attachments)
	return act
}

func attachmentCount(t *testing.T, app core.App, collection, iri string) int64 {
	t.Helper()
	n, err := app.CountRecords(collection, dbx.HashExp{"iri": iri})
	if err != nil {
		t.Fatalf("count %s: %v", collection, err)
	}
	return n
}

func attachmentStored(t *testing.T, app core.App, collection, iri string) *core.Record {
	t.Helper()
	r, err := app.FindFirstRecordByData(collection, "iri", iri)
	if err != nil {
		t.Fatalf("find %s %s: %v", collection, iri, err)
	}
	return r
}

func TestTrailCreateDoesNotFetchLoopbackAttachment(t *testing.T) {
	f := attachmentApp(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	err := attachmentDeliverTrail(f, attachmentGPX(server.URL+"/internal.gpx"))

	if err == nil {
		t.Error("trail whose GPX is on a loopback address was accepted")
	}
	if n := hits.total(); n != 0 {
		t.Errorf("loopback server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "trails", feedDedupTrailIRI); n != 0 {
		t.Errorf("trails rows = %d, want 0", n)
	}
}

func TestTrailCreateSkipsLoopbackPhoto(t *testing.T) {
	f := attachmentApp(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	if err := attachmentDeliverTrail(f, attachmentImage(server.URL+"/internal.png")); err != nil {
		t.Fatalf("trail with an unreachable photo must still be stored: %v", err)
	}

	if n := hits.total(); n != 0 {
		t.Errorf("loopback server received %d requests, want 0", n)
	}
	stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
	if photos := stored.GetStringSlice("photos"); len(photos) != 0 {
		t.Errorf("stored photos = %v, want none", photos)
	}
}

func TestSummitLogCreateDoesNotFetchLoopbackAttachment(t *testing.T) {
	f := attachmentApp(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	act := attachmentSummitLogActivity(t, f, attachmentGPX(server.URL+"/internal.gpx"))
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.owner.alice, act)

	if err == nil {
		t.Error("summit log whose GPX is on a loopback address was accepted")
	}
	if n := hits.total(); n != 0 {
		t.Errorf("loopback server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "summit_logs", ownerRemoteLogIRI); n != 0 {
		t.Errorf("summit_logs rows = %d, want 0", n)
	}
}

func TestListCreateDoesNotFetchLoopbackAvatar(t *testing.T) {
	f := attachmentApp(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	act := feedDedupListCreate(feedDedupAuthorIRI, feedDedupListIRI)
	act.Object.(*pub.Object).Attachment = pub.ItemCollection{attachmentImage(server.URL + "/internal.png")}
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.owner.author, f.owner.alice, act)

	if err == nil {
		t.Error("list whose avatar is on a loopback address was accepted")
	}
	if n := hits.total(); n != 0 {
		t.Errorf("loopback server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "lists", feedDedupListIRI); n != 0 {
		t.Errorf("lists rows = %d, want 0", n)
	}
}

func TestTrailCreateRefusesOversizedGPX(t *testing.T) {
	t.Run("one byte over the limit is refused", func(t *testing.T) {
		f := attachmentApp(t)
		attachmentUseLoopbackClient(t)
		body := make([]byte, util.RemoteGPXMaxBytes+1)
		server, _ := attachmentServer(t, func(string) []byte { return body })

		if err := attachmentDeliverTrail(f, attachmentGPX(server.URL+"/t.gpx")); err == nil {
			t.Error("GPX of RemoteGPXMaxBytes+1 bytes was accepted")
		}
		if n := attachmentCount(t, f.app, "trails", feedDedupTrailIRI); n != 0 {
			t.Errorf("trails rows = %d, want 0", n)
		}
	})

	t.Run("exactly the limit is accepted", func(t *testing.T) {
		f := attachmentApp(t)
		attachmentUseLoopbackClient(t)
		body := make([]byte, util.RemoteGPXMaxBytes)
		server, _ := attachmentServer(t, func(string) []byte { return body })

		if err := attachmentDeliverTrail(f, attachmentGPX(server.URL+"/t.gpx")); err != nil {
			t.Fatalf("GPX of exactly RemoteGPXMaxBytes bytes was refused: %v", err)
		}
		stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
		if stored.GetString("gpx") == "" {
			t.Error("stored trail has no gpx file")
		}
	})
}

func attachmentManyPhotos(base string, n int) []pub.Item {
	items := make([]pub.Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, attachmentImage(fmt.Sprintf("%s/p%03d.png", base, i)))
	}
	return items
}

func TestTrailCreateDownloadsAtMostMaxPhotos(t *testing.T) {
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	if err := attachmentDeliverTrail(f, attachmentManyPhotos(server.URL, 105)...); err != nil {
		t.Fatalf("trail with 105 photos: %v", err)
	}

	// The smaller of the photos maxSelect and the per-activity budget applies.
	want := min(99, util.RemoteActivityAttachmentMaxFiles)
	if n := hits.total(); n != want {
		t.Errorf("photo requests = %d, want %d", n, want)
	}
	stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
	if n := len(stored.GetStringSlice("photos")); n != want {
		t.Errorf("stored photos = %d, want %d", n, want)
	}
}

func TestSummitLogCreateDownloadsAtMostMaxPhotos(t *testing.T) {
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	act := attachmentSummitLogActivity(t, f, attachmentManyPhotos(server.URL, 105)...)
	if err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.owner.alice, act); err != nil {
		t.Fatalf("summit log with 105 photos: %v", err)
	}

	// The smaller of the photos maxSelect and the per-activity budget applies.
	want := min(99, util.RemoteActivityAttachmentMaxFiles)
	if n := hits.total(); n != want {
		t.Errorf("photo requests = %d, want %d", n, want)
	}
	stored := attachmentStored(t, f.app, "summit_logs", ownerRemoteLogIRI)
	if n := len(stored.GetStringSlice("photos")); n != want {
		t.Errorf("stored photos = %d, want %d", n, want)
	}
}

func TestTrailCreateLeavesNoAttachmentTempFiles(t *testing.T) {
	t.Run("after a successful Create", func(t *testing.T) {
		tmp := attachmentIsolatedTempDir(t)
		f := attachmentApp(t)
		attachmentUseLoopbackClient(t)
		server, _ := attachmentServer(t, attachmentSmallBody)

		err := attachmentDeliverTrail(f, attachmentGPX(server.URL+"/t.gpx"), attachmentImage(server.URL+"/p.png"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		stored := attachmentStored(t, f.app, "trails", feedDedupTrailIRI)
		if stored.GetString("gpx") == "" || len(stored.GetStringSlice("photos")) != 1 {
			t.Fatalf("attachments were not stored: gpx=%q photos=%v", stored.GetString("gpx"), stored.GetStringSlice("photos"))
		}
		attachmentRequireNoTempFiles(t, tmp)
	})

	t.Run("after a refused Create", func(t *testing.T) {
		tmp := attachmentIsolatedTempDir(t)
		f := attachmentApp(t)
		attachmentUseLoopbackClient(t)
		big := make([]byte, util.RemoteGPXMaxBytes+1)
		server, _ := attachmentServer(t, func(p string) []byte {
			if strings.HasSuffix(p, ".gpx") {
				return big
			}
			return []byte("data")
		})

		err := attachmentDeliverTrail(f, attachmentImage(server.URL+"/p.png"), attachmentGPX(server.URL+"/t.gpx"))
		if err == nil {
			t.Fatal("oversized GPX was accepted")
		}
		attachmentRequireNoTempFiles(t, tmp)
	})

	t.Run("after a successful summit log Create", func(t *testing.T) {
		tmp := attachmentIsolatedTempDir(t)
		f := attachmentApp(t)
		attachmentUseLoopbackClient(t)
		server, _ := attachmentServer(t, attachmentSmallBody)

		act := attachmentSummitLogActivity(t, f, attachmentGPX(server.URL+"/t.gpx"), attachmentImage(server.URL+"/p.png"))
		if err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.owner.alice, act); err != nil {
			t.Fatalf("Create: %v", err)
		}
		attachmentRequireNoTempFiles(t, tmp)
	})
}
