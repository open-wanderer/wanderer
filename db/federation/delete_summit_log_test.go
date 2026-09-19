package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
)

// A summit log is delivered to its author's followers when created, so they
// have to be told when it is deleted. That includes the case where the log was
// removed by its trail's deletion: the trail's own Delete goes to the trail
// author's followers, which need not overlap with the log author's.

const summitLogTestKey = "0123456789abcdef0123456789abcdef"

// inboxes counts deliveries per path so one server can stand in for several
// remote actors.
type inboxes struct {
	server *httptest.Server
	mu     sync.Mutex
	hits   map[string]int
}

func newInboxes(t *testing.T) *inboxes {
	t.Helper()

	in := &inboxes{hits: map[string]int{}}
	in.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			in.mu.Lock()
			in.hits[r.URL.Path]++
			in.mu.Unlock()
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(in.server.Close)

	return in
}

func (in *inboxes) url(name string) string {
	return in.server.URL + "/" + name + "/inbox"
}

func (in *inboxes) count(name string) int {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.hits["/"+name+"/inbox"]
}

// wait gives asynchronous delivery time to arrive, or to be shown not to.
func (in *inboxes) wait(name string, want int, deadline time.Duration) int {
	stop := time.Now().Add(deadline)
	for time.Now().Before(stop) {
		if in.count(name) >= want {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return in.count(name)
}

type summitLogFixture struct {
	app    *pbtests.TestApp
	in     *inboxes
	actors *core.Collection
	trails *core.Collection
	logs   *core.Collection
}

func setupSummitLogDeleteTestApp(t *testing.T) *summitLogFixture {
	t.Helper()

	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "inbox"},
		&core.TextField{Name: "private_key"},
		&core.BoolField{Name: "is_local"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	follows := core.NewBaseCollection("follows")
	follows.Fields.Add(
		&core.RelationField{Name: "follower", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "followee", CollectionId: actors.Id, MaxSelect: 1},
		&core.TextField{Name: "status"},
	)
	if err := app.Save(follows); err != nil {
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	// The trail relation is not cascading here so a log can be left pointing
	// at a trail that no longer exists, which is the state a cascaded log is
	// in by the time its post-commit hook runs.
	logs := core.NewBaseCollection("summit_logs")
	logs.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
	)
	if err := app.Save(logs); err != nil {
		t.Fatal(err)
	}

	activities := core.NewBaseCollection("activitypub_activities")
	activities.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "type"},
		&core.TextField{Name: "to"},
		&core.TextField{Name: "cc"},
		&core.TextField{Name: "object"},
		&core.TextField{Name: "actor"},
		&core.TextField{Name: "published"},
	)
	if err := app.Save(activities); err != nil {
		t.Fatal(err)
	}

	return &summitLogFixture{app: app, in: newInboxes(t), actors: actors, trails: trails, logs: logs}
}

// localAuthor is an actor able to sign, which PostActivity requires.
func (f *summitLogFixture) localAuthor(t *testing.T, name string) *core.Record {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(key), summitLogTestKey)
	if err != nil {
		t.Fatal(err)
	}

	iri := "https://local.example/api/v1/activitypub/user/" + name
	r := core.NewRecord(f.actors)
	r.Set("iri", iri)
	r.Set("inbox", iri+"/inbox")
	r.Set("private_key", encrypted)
	r.Set("is_local", true)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// remoteActor's inbox is one path on the counting server.
func (f *summitLogFixture) remoteActor(t *testing.T, name string) *core.Record {
	t.Helper()

	r := core.NewRecord(f.actors)
	r.Set("iri", "https://remote.example/api/v1/activitypub/user/"+name)
	r.Set("inbox", f.in.url(name))
	r.Set("is_local", false)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *summitLogFixture) follow(t *testing.T, follower, followee *core.Record) {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set("follower", follower.Id)
	r.Set("followee", followee.Id)
	r.Set("status", "accepted")
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *summitLogFixture) trail(t *testing.T, author *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(f.trails)
	r.Set("author", author.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *summitLogFixture) summitLog(t *testing.T, author *core.Record, trailId string) *core.Record {
	t.Helper()

	r := core.NewRecord(f.logs)
	r.Set("iri", "https://local.example/api/v1/summit-log/"+security.RandomString(8))
	r.Set("author", author.Id)
	r.Set("trail", trailId)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// orphanedSummitLog returns a log whose trail has since been deleted, which is
// the state a cascaded log is in by the time its post-commit hook runs. The
// trail has to exist when the log is saved, because relation ids are checked.
func (f *summitLogFixture) orphanedSummitLog(t *testing.T, author, trailAuthor *core.Record) *core.Record {
	t.Helper()

	trail := f.trail(t, trailAuthor)
	log := f.summitLog(t, author, trail.Id)
	if err := f.app.Delete(trail); err != nil {
		t.Fatal(err)
	}
	return log
}

func TestCreateSummitLogDeleteActivity(t *testing.T) {
	// The reviewer's repro. A follows B and logs a summit on B's trail. C
	// follows A but not B, and so received the log through A. B's trail is
	// deleted, taking A's log with it. C must still be told the log is gone.
	t.Run("TrailGoneStillReachesAuthorsFollowers", func(t *testing.T) {
		f := setupSummitLogDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		c := f.remoteActor(t, "c")
		f.follow(t, c, a)

		log := f.orphanedSummitLog(t, a, b)

		if err := CreateSummitLogDeleteActivity(f.app, log); err != nil {
			t.Fatalf("expected the delete to be sent despite the missing trail, got %v", err)
		}

		if got := f.in.wait("c", 1, 5*time.Second); got != 1 {
			t.Fatalf("A's follower C should receive the log's Delete, got %d delivery(ies)", got)
		}
	})

	// The ordinary case is unchanged: followers and the trail author both hear.
	t.Run("TrailPresentReachesFollowersAndTrailAuthor", func(t *testing.T) {
		f := setupSummitLogDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		c := f.remoteActor(t, "c")
		f.follow(t, c, a)

		log := f.summitLog(t, a, f.trail(t, b).Id)

		if err := CreateSummitLogDeleteActivity(f.app, log); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("c", 1, 5*time.Second); got != 1 {
			t.Fatalf("follower C should receive the Delete, got %d", got)
		}
		if got := f.in.wait("b", 1, 5*time.Second); got != 1 {
			t.Fatalf("trail author B should receive the Delete, got %d", got)
		}
	})

	// When the trail is present its author is the To; when it is gone there
	// is nobody to address it to and it falls back to Public.
	t.Run("AddressingFollowsTrailAvailability", func(t *testing.T) {
		f := setupSummitLogDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")

		withTrail := f.summitLog(t, a, f.trail(t, b).Id)
		if err := CreateSummitLogDeleteActivity(f.app, withTrail); err != nil {
			t.Fatal(err)
		}
		rec, err := f.app.FindFirstRecordByData("activitypub_activities", "object", withTrail.GetString("iri"))
		if err != nil {
			t.Fatal(err)
		}
		if got := rec.GetString("to"); got != b.GetString("iri") {
			t.Fatalf("with the trail present, to = %q, want the trail author %q", got, b.GetString("iri"))
		}

		withoutTrail := f.orphanedSummitLog(t, a, b)
		if err := CreateSummitLogDeleteActivity(f.app, withoutTrail); err != nil {
			t.Fatal(err)
		}
		rec, err = f.app.FindFirstRecordByData("activitypub_activities", "object", withoutTrail.GetString("iri"))
		if err != nil {
			t.Fatal(err)
		}
		if got := rec.GetString("to"); got != "https://www.w3.org/ns/activitystreams#Public" {
			t.Fatalf("with the trail gone, to = %q, want Public", got)
		}
	})

	// An author that is gone too means the log fell to an account deletion,
	// and there is no local identity left to sign anything. Unchanged.
	t.Run("AuthorGoneSendsNothing", func(t *testing.T) {
		f := setupSummitLogDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		c := f.remoteActor(t, "c")
		f.follow(t, c, a)
		log := f.orphanedSummitLog(t, a, b)

		if err := f.app.Delete(a); err != nil {
			t.Fatal(err)
		}

		if err := CreateSummitLogDeleteActivity(f.app, log); err != nil {
			t.Fatalf("expected a missing author to be tolerated, got %v", err)
		}

		if got := f.in.wait("c", 1, 300*time.Millisecond); got != 0 {
			t.Fatalf("nothing should be sent without an author to sign it, got %d", got)
		}
	})
}
