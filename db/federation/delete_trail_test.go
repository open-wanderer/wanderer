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
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}

	// The rows that name who interacted with a trail, cascading with it as
	// in the real schema.
	for _, name := range []string{"comments", "summit_logs"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
			&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1, CascadeDelete: true},
		)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"trail_like", "trail_share"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
			&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1, CascadeDelete: true},
		)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	listShare := core.NewBaseCollection("list_share")
	listShare.Fields.Add(
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, CascadeDelete: true},
	)
	if err := app.Save(listShare); err != nil {
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

// localContent is a trail or list of a local author, addressed as this
// instance would address it.
func (f *trailDeleteFixture) localContent(t *testing.T, collection *core.Collection, path string, author *core.Record, public bool) *core.Record {
	t.Helper()

	r := core.NewRecord(collection)
	r.Set("author", author.Id)
	r.Set("public", public)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	r.Set("iri", "https://local.example/api/v1/"+path+"/"+r.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *trailDeleteFixture) localTrail(t *testing.T, author *core.Record, public bool) *core.Record {
	return f.localContent(t, f.trails, "trail", author, public)
}

// interaction files a comment, summit log, like or share of the trail by
// actor; subjectField is the relation naming the actor in that collection.
func (f *trailDeleteFixture) interaction(t *testing.T, collection, subjectField string, actor, subject *core.Record) {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set(subjectField, actor.Id)
	r.Set(subject.Collection().Name[:len(subject.Collection().Name)-1], subject.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

// announced files the Announce that sharing the object with actor sent, as
// CreateAnnounceActivity records it: the actor's IRI in to.
func (f *trailDeleteFixture) announced(t *testing.T, object, actor *core.Record) {
	t.Helper()

	r := core.NewRecord(f.activities)
	r.Set("iri", "https://local.example/api/v1/activitypub/activity/"+security.RandomString(8))
	r.Set("type", "Announce")
	r.Set("object", map[string]any{"id": object.GetString("iri"), "type": "Note"})
	r.Set("to", []string{actor.GetString("iri")})
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

// recordedReply files the Create of a comment or summit log on the trail, as
// CreateCommentActivity records it: the reply's own id, the trail in
// inReplyTo, and the inboxes it was sent to in cc.
func (f *trailDeleteFixture) recordedReply(t *testing.T, trail *core.Record, sentTo ...*core.Record) {
	t.Helper()

	cc := make([]string, 0, len(sentTo))
	for _, actor := range sentTo {
		cc = append(cc, actor.GetString("inbox"))
	}

	r := core.NewRecord(f.activities)
	r.Set("iri", "https://local.example/api/v1/activitypub/activity/"+security.RandomString(8))
	r.Set("type", "Create")
	r.Set("object", map[string]any{
		"id":        "https://local.example/api/v1/comment/" + security.RandomString(8),
		"type":      "Note",
		"inReplyTo": trail.GetString("iri"),
	})
	r.Set("cc", cc)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *trailDeleteFixture) sendTrailDelete(t *testing.T, trail *core.Record) {
	t.Helper()

	audience, err := TrailDeleteRecipients(f.app, trail, trail.GetBool("public"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateTrailDeleteActivity(f.app, trail, audience); err != nil {
		t.Fatal(err)
	}
}

func (f *trailDeleteFixture) sendListDelete(t *testing.T, list *core.Record) {
	t.Helper()

	audience, err := ListDeleteRecipients(f.app, list, list.GetBool("public"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateListDeleteActivity(f.app, list, audience); err != nil {
		t.Fatal(err)
	}
}

func (f *trailDeleteFixture) deliveries(t *testing.T, name string, want int) {
	t.Helper()

	if got := f.in.wait(name, want, 5*time.Second); got != want {
		t.Errorf("%s received %d delivery(ies), want %d", name, got, want)
	}
}

// silence asserts that nothing reaches name, giving delivery time to be wrong.
func (f *trailDeleteFixture) silence(t *testing.T, name string) {
	t.Helper()

	if got := f.in.wait(name, 1, 500*time.Millisecond); got != 0 {
		t.Errorf("%s received %d delivery(ies), want none", name, got)
	}
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

// A reply from other ActivityPub software is accepted as a comment although
// its id is not a Wanderer comment IRI. Its Delete has to be understood the
// same way, or the comment outlives the post.
func TestProcessDeleteForeignCommentActivity(t *testing.T) {
	setup := func(t *testing.T) (*trailDeleteFixture, *core.Collection) {
		f := setupTrailDeleteTestApp(t)
		comments, err := f.app.FindCollectionByNameOrId("comments")
		if err != nil {
			t.Fatal(err)
		}
		comments.Fields.Add(&core.TextField{Name: "iri"})
		if err := f.app.Save(comments); err != nil {
			t.Fatal(err)
		}
		return f, comments
	}

	foreignIRI := "https://mastodon.example/users/carol/statuses/123"

	t.Run("removes the comment", func(t *testing.T) {
		f, comments := setup(t)
		carol := f.actor(t, "carol", false)
		trail := f.content(t, f.trails, "trail", f.actor(t, "author", false))

		comment := core.NewRecord(comments)
		comment.Set("iri", foreignIRI)
		comment.Set("author", carol.Id)
		comment.Set("trail", trail.Id)
		if err := f.app.Save(comment); err != nil {
			t.Fatal(err)
		}

		activity := pub.DeleteNew(pub.IRI("https://mastodon.example/users/carol/statuses/123#delete"), pub.IRI(foreignIRI))
		activity.Actor = pub.IRI(carol.GetString("iri"))
		if err := ProcessDeleteActivity(f.app, carol, *activity); err != nil {
			t.Fatalf("ProcessDeleteActivity: %v", err)
		}
		if f.exists(t, comment) {
			t.Error("comment still exists after its Delete")
		}
	})

	t.Run("ignores an object it never stored", func(t *testing.T) {
		f, _ := setup(t)
		carol := f.actor(t, "carol", false)

		activity := pub.DeleteNew(pub.IRI("https://mastodon.example/x"), pub.IRI(foreignIRI))
		activity.Actor = pub.IRI(carol.GetString("iri"))
		if err := ProcessDeleteActivity(f.app, carol, *activity); err != nil {
			t.Fatalf("ProcessDeleteActivity: %v", err)
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

		trail := f.localTrail(t, b, true)
		f.recorded(t, "Update", trail, a)

		f.sendTrailDelete(t, trail)
		f.deliveries(t, "a", 1)
	})

	// Anyone who commented on, logged a summit on, liked or was handed the
	// trail holds a copy of it, whether or not they follow its author.
	t.Run("ReachesEveryoneHoldingACopy", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		trail := f.localTrail(t, b, true)

		f.interaction(t, "comments", "author", f.actor(t, "commenter", false), trail)
		f.interaction(t, "summit_logs", "author", f.actor(t, "logger", false), trail)
		f.interaction(t, "trail_like", "actor", f.actor(t, "liker", false), trail)
		f.interaction(t, "trail_share", "actor", f.actor(t, "sharee", false), trail)
		// A share since revoked: no trail_share row, only the Announce.
		f.announced(t, trail, f.actor(t, "revoked", false))
		// Mentioned in a local comment on the trail, and so fetched it.
		f.recordedReply(t, trail, f.actor(t, "replymention", false))
		f.interaction(t, "comments", "author", f.actor(t, "bystander", false), f.localTrail(t, b, true))

		f.sendTrailDelete(t, trail)
		for _, name := range []string{"commenter", "logger", "liker", "sharee", "revoked", "replymention"} {
			f.deliveries(t, name, 1)
		}
		f.silence(t, "bystander")
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

		trail := f.localTrail(t, b, true)
		f.recorded(t, "Create", trail, both)
		f.recorded(t, "Update", trail, both)
		f.interaction(t, "comments", "author", both, trail)

		f.sendTrailDelete(t, trail)
		f.deliveries(t, "follower", 1)
		f.deliveries(t, "both", 1)
	})

	// A trail that is private by the time it is deleted was never handed to
	// followers, but whoever else holds a copy still has to hear.
	t.Run("PrivateTrailReachesHoldersNotFollowers", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		follower := f.actor(t, "follower", false)
		f.follow(t, follower, b)

		trail := f.localTrail(t, b, false)
		f.interaction(t, "trail_share", "actor", f.actor(t, "sharee", false), trail)

		f.sendTrailDelete(t, trail)
		f.deliveries(t, "sharee", 1)
		f.silence(t, "follower")
	})

	// A private trail nobody ever received is nobody's business.
	t.Run("PrivateUnsharedTrailSendsNothing", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		f.follow(t, f.actor(t, "follower", false), b)
		trail := f.localTrail(t, b, false)

		f.sendTrailDelete(t, trail)
		f.silence(t, "follower")
		if _, err := f.app.FindFirstRecordByFilter("activitypub_activities", "type = 'Delete'", dbx.Params{}); err == nil {
			t.Error("a Delete was recorded for a trail that was never handed out")
		}
	})

	// A remote author's trail is not ours to retract, however many local
	// interactions it has.
	t.Run("RemoteTrailSendsNothing", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		remote := f.actor(t, "remote", false)
		trail := f.content(t, f.trails, "trail", remote)
		f.interaction(t, "comments", "author", f.actor(t, "commenter", false), trail)

		audience, err := TrailDeleteRecipients(f.app, trail, true)
		if err != nil {
			t.Fatal(err)
		}
		if !audience.Empty() {
			t.Fatalf("audience = %+v, want none for a remote author's trail", audience)
		}
	})

	// The Delete's cc names the followers collection and the other holders
	// by inbox, as the Create did. Follower inboxes are never listed: that
	// would hand the follower list to every recipient.
	t.Run("RecordsHoldersNotFollowersInCC", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		a := f.actor(t, "a", false)
		f.follow(t, f.actor(t, "follower", false), b)

		trail := f.localTrail(t, b, true)
		f.recorded(t, "Create", trail, a)

		f.sendTrailDelete(t, trail)

		rec, err := f.app.FindFirstRecordByFilter("activitypub_activities", "type = 'Delete' && object = {:iri}", dbx.Params{"iri": trail.GetString("iri")})
		if err != nil {
			t.Fatal(err)
		}
		cc := rec.GetStringSlice("cc")
		want := []string{b.GetString("iri") + "/followers", a.GetString("inbox")}
		if len(cc) != len(want) || cc[0] != want[0] || cc[1] != want[1] {
			t.Fatalf("cc = %v, want %v", cc, want)
		}
		f.deliveries(t, "follower", 1)
	})
}

func TestCreateListDeleteActivity(t *testing.T) {
	// A list only leaves the instance by following its author or by being
	// shared, and a share once made counts even after it was revoked.
	t.Run("ReachesFollowersAndSharees", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		f.follow(t, f.actor(t, "follower", false), b)
		list := f.localContent(t, f.lists, "list", b, true)
		f.interaction(t, "list_share", "actor", f.actor(t, "sharee", false), list)
		f.announced(t, list, f.actor(t, "revoked", false))

		f.sendListDelete(t, list)
		for _, name := range []string{"follower", "sharee", "revoked"} {
			f.deliveries(t, name, 1)
		}
	})

	t.Run("PrivateListReachesShareesNotFollowers", func(t *testing.T) {
		f := setupTrailDeleteTestApp(t)

		b := f.actor(t, "b", true)
		f.follow(t, f.actor(t, "follower", false), b)
		list := f.localContent(t, f.lists, "list", b, false)
		f.interaction(t, "list_share", "actor", f.actor(t, "sharee", false), list)

		f.sendListDelete(t, list)
		f.deliveries(t, "sharee", 1)
		f.silence(t, "follower")
	})
}
