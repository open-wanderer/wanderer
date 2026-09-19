package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
)

// A remote trail reaches this instance in two ways: delivered to a follower's
// inbox as Create/Update, which files it into that follower's feed, or synced
// on demand when somebody opens it, which files it nowhere. The Delete that
// follows must remove the local copy either way, and must clear every feed it
// was filed into, not just the first.

type trailDeleteFixture struct {
	app        *pbtests.TestApp
	in         *inboxes
	actors     *core.Collection
	trails     *core.Collection
	lists      *core.Collection
	feed       *core.Collection
	activities *core.Collection
}

func setupTrailDeleteTestApp(t *testing.T) *trailDeleteFixture {
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
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	activities := core.NewBaseCollection("activitypub_activities")
	activities.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "type"},
		&core.JSONField{Name: "to"},
		&core.JSONField{Name: "cc"},
		&core.JSONField{Name: "object"},
		&core.TextField{Name: "actor"},
		&core.TextField{Name: "published"},
	)
	if err := app.Save(activities); err != nil {
		t.Fatal(err)
	}

	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}

	// item is plain text in the real schema too, so nothing cascades from
	// the trail to its feed entries.
	feed := core.NewBaseCollection("feed")
	feed.Fields.Add(
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.TextField{Name: "item"},
		&core.TextField{Name: "type"},
	)
	if err := app.Save(feed); err != nil {
		t.Fatal(err)
	}

	return &trailDeleteFixture{
		app: app, in: newInboxes(t),
		actors: actors, trails: trails, lists: lists, feed: feed, activities: activities,
	}
}

// actor is local, able to sign as PostActivity requires, or remote with its
// inbox on the counting server.
func (f *trailDeleteFixture) actor(t *testing.T, name string, local bool) *core.Record {
	t.Helper()

	r := core.NewRecord(f.actors)
	if local {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		encrypted, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(key), summitLogTestKey)
		if err != nil {
			t.Fatal(err)
		}
		iri := "https://local.example/api/v1/activitypub/user/" + name
		r.Set("iri", iri)
		r.Set("inbox", iri+"/inbox")
		r.Set("private_key", encrypted)
	} else {
		r.Set("iri", "https://remote.example/api/v1/activitypub/user/"+name)
		r.Set("inbox", f.in.url(name))
	}
	r.Set("is_local", local)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *trailDeleteFixture) follow(t *testing.T, follower, followee *core.Record) {
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

// localTrail is a public trail of a local author, addressed as this instance
// would address it.
func (f *trailDeleteFixture) localTrail(t *testing.T, author *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(f.trails)
	r.Set("author", author.Id)
	r.Set("public", true)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", "https://local.example/api/v1/trail/"+r.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// recorded files a Create or Update of the trail as PostActivity would have,
// with the inboxes it was sent to in cc.
func (f *trailDeleteFixture) recorded(t *testing.T, typ string, trail *core.Record, sentTo ...*core.Record) {
	t.Helper()

	cc := make([]string, 0, len(sentTo))
	for _, actor := range sentTo {
		cc = append(cc, actor.GetString("inbox"))
	}

	r := core.NewRecord(f.activities)
	r.Set("iri", "https://local.example/api/v1/activitypub/activity/"+security.RandomString(8))
	r.Set("type", typ)
	r.Set("object", map[string]any{"id": trail.GetString("iri"), "type": "Note"})
	r.Set("cc", cc)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *trailDeleteFixture) content(t *testing.T, collection *core.Collection, path string, author *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(collection)
	r.Set("author", author.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", "https://remote.example/api/v1/"+path+"/"+r.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *trailDeleteFixture) feedEntry(t *testing.T, recipient, author, item *core.Record) {
	t.Helper()

	r := core.NewRecord(f.feed)
	r.Set("actor", recipient.Id)
	r.Set("author", author.Id)
	r.Set("item", item.Id)
	r.Set("type", item.Collection().Name)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *trailDeleteFixture) feedEntries(t *testing.T, item *core.Record) int {
	t.Helper()

	records, err := f.app.FindAllRecords("feed", dbx.HashExp{"item": item.Id})
	if err != nil {
		t.Fatal(err)
	}
	return len(records)
}

func (f *trailDeleteFixture) processDelete(t *testing.T, actor, object *core.Record) error {
	t.Helper()

	activity := pub.DeleteNew(pub.IRI("https://remote.example/api/v1/activitypub/activity/x"), pub.IRI(object.GetString("iri")))
	activity.Actor = pub.IRI(actor.GetString("iri"))
	return ProcessDeleteActivity(f.app, actor, *activity)
}

func (f *trailDeleteFixture) exists(t *testing.T, r *core.Record) bool {
	t.Helper()

	_, err := f.app.FindRecordById(r.Collection().Name, r.Id)
	return err == nil
}

func TestProcessDeleteTrailActivity(t *testing.T) {
	t.Run("removes a trail that was synced on demand and never filed into a feed", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)
		author := f.actor(t, "author", false)
		trail := f.content(t, f.trails, "trail", author)

		if err := f.processDelete(t, author, trail); err != nil {
			t.Fatalf("ProcessDeleteActivity: %v", err)
		}
		if f.exists(t, trail) {
			t.Error("trail still exists after Delete")
		}
	})

	t.Run("clears the feed of every follower it was delivered to", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)
		author := f.actor(t, "author", false)
		trail := f.content(t, f.trails, "trail", author)
		other := f.content(t, f.trails, "trail", author)
		for _, name := range []string{"alice", "bob"} {
			follower := f.actor(t, name, true)
			f.feedEntry(t, follower, author, trail)
			f.feedEntry(t, follower, author, other)
		}

		if err := f.processDelete(t, author, trail); err != nil {
			t.Fatalf("ProcessDeleteActivity: %v", err)
		}
		if f.exists(t, trail) {
			t.Error("trail still exists after Delete")
		}
		if got := f.feedEntries(t, trail); got != 0 {
			t.Errorf("deleted trail still has %d feed entries", got)
		}
		if got := f.feedEntries(t, other); got != 2 {
			t.Errorf("other trail lost feed entries, %d left of 2", got)
		}
	})

	t.Run("refuses a Delete from anyone but the author", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)
		author := f.actor(t, "author", false)
		stranger := f.actor(t, "stranger", false)
		trail := f.content(t, f.trails, "trail", author)

		if err := f.processDelete(t, stranger, trail); err == nil {
			t.Fatal("expected an error for a Delete signed by a non-author")
		}
		if !f.exists(t, trail) {
			t.Error("trail was deleted on a stranger's say-so")
		}
	})
}

func TestProcessDeleteListActivity(t *testing.T) {
	t.Run("removes a list that was synced on demand and never filed into a feed", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)
		author := f.actor(t, "author", false)
		list := f.content(t, f.lists, "list", author)

		if err := f.processDelete(t, author, list); err != nil {
			t.Fatalf("ProcessDeleteActivity: %v", err)
		}
		if f.exists(t, list) {
			t.Error("list still exists after Delete")
		}
	})
}

func TestCreateTrailDeleteActivity(t *testing.T) {
	// The reviewer's repro. B follows A, nobody follows B. B's trail mentions
	// A, so A received it although A is not a follower. When B deletes the
	// trail, A must be told, or A's cached copy lives on.
	t.Run("ReachesMentionedActorsWhoDoNotFollow", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		a := f.actor(t, "a", false)
		f.follow(t, b, a)

		trail := f.localTrail(t, b)
		f.recorded(t, "Update", trail, a)

		if err := CreateTrailDeleteActivity(f.app, trail); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("a", 1, 5*time.Second); got != 1 {
			t.Fatalf("mentioned actor A should receive the Delete, got %d delivery(ies)", got)
		}
	})

	// Followers still hear, and an actor that is both a follower and was
	// mentioned hears once.
	t.Run("ReachesFollowersOnce", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		follower := f.actor(t, "follower", false)
		both := f.actor(t, "both", false)
		f.follow(t, follower, b)
		f.follow(t, both, b)

		trail := f.localTrail(t, b)
		f.recorded(t, "Create", trail, both)
		f.recorded(t, "Update", trail, both)

		if err := CreateTrailDeleteActivity(f.app, trail); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("follower", 1, 5*time.Second); got != 1 {
			t.Fatalf("follower should receive the Delete, got %d", got)
		}
		if got := f.in.wait("both", 1, 5*time.Second); got != 1 {
			t.Fatalf("follower who was also mentioned should receive the Delete once, got %d", got)
		}
	})

	// The recorded audience is written to the Delete's cc, next to the
	// followers collection, so the activity itself says who it went to.
	t.Run("RecordsMentionedInboxesInCC", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		a := f.actor(t, "a", false)

		trail := f.localTrail(t, b)
		f.recorded(t, "Create", trail, a)

		if err := CreateTrailDeleteActivity(f.app, trail); err != nil {
			t.Fatal(err)
		}

		rec, err := f.app.FindFirstRecordByFilter("activitypub_activities", "type = 'Delete' && object = {:iri}", dbx.Params{"iri": trail.GetString("iri")})
		if err != nil {
			t.Fatal(err)
		}
		cc := rec.GetStringSlice("cc")
		want := []string{b.GetString("iri") + "/followers", a.GetString("inbox")}
		if len(cc) != len(want) || cc[0] != want[0] || cc[1] != want[1] {
			t.Fatalf("cc = %v, want %v", cc, want)
		}
	})
}
