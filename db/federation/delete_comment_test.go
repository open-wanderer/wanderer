package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
)

// A comment's text goes to the actors it mentions and to the trail author, and
// the Create and Update activities record exactly which inboxes that was. The
// Delete reads that audience back, so it reaches the mentioned actors too and
// does not depend on the trail still existing.

type commentFixture struct {
	app        *pbtests.TestApp
	in         *inboxes
	actors     *core.Collection
	trails     *core.Collection
	comments   *core.Collection
	activities *core.Collection
}

func setupCommentDeleteTestApp(t *testing.T) *commentFixture {
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

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	// Non-cascading so a comment can outlive its trail, as a cascaded one
	// has by the time its post-commit hook runs.
	comments := core.NewBaseCollection("comments")
	comments.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
	)
	if err := app.Save(comments); err != nil {
		t.Fatal(err)
	}

	// JSON fields, as in the real schema: the audience query filters on
	// object.id and reads cc back as a list.
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

	return &commentFixture{
		app: app, in: newInboxes(t),
		actors: actors, trails: trails, comments: comments, activities: activities,
	}
}

func (f *commentFixture) localAuthor(t *testing.T, name string) *core.Record {
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

func (f *commentFixture) remoteActor(t *testing.T, name string) *core.Record {
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

func (f *commentFixture) trail(t *testing.T, author *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(f.trails)
	r.Set("author", author.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *commentFixture) comment(t *testing.T, author, trail *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(f.comments)
	r.Set("iri", "https://local.example/api/v1/comment/"+security.RandomString(8))
	r.Set("author", author.Id)
	r.Set("trail", trail.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// recorded stores what CreateCommentActivity would have: an activity of the
// given type whose object is the comment and whose cc is the inboxes it went
// out to.
func (f *commentFixture) recorded(t *testing.T, typ string, comment *core.Record, sentTo ...*core.Record) {
	t.Helper()

	cc := make([]string, 0, len(sentTo))
	for _, actor := range sentTo {
		cc = append(cc, actor.GetString("inbox"))
	}

	r := core.NewRecord(f.activities)
	r.Set("iri", "https://local.example/api/v1/activitypub/activity/"+security.RandomString(8))
	r.Set("type", typ)
	r.Set("object", map[string]any{"id": comment.GetString("iri"), "type": "Note"})
	r.Set("cc", cc)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *commentFixture) deleteTrail(t *testing.T, trail *core.Record) {
	t.Helper()
	if err := f.app.Delete(trail); err != nil {
		t.Fatal(err)
	}
}

func TestCreateCommentDeleteActivity(t *testing.T) {
	// The gap the reviewer's summit-log case implies for comments: the trail
	// is gone, so its author cannot be looked up, but the recorded audience
	// still says who got the text.
	t.Run("TrailGoneStillReachesRecordedAudience", func(t *testing.T) {
		f := setupCommentDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		m := f.remoteActor(t, "m")
		trail := f.trail(t, b)
		comment := f.comment(t, a, trail)
		f.recorded(t, "Create", comment, m, b)
		f.deleteTrail(t, trail)

		if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("m", 1, 5*time.Second); got != 1 {
			t.Fatalf("mentioned actor should receive the Delete, got %d", got)
		}
		if got := f.in.wait("b", 1, 5*time.Second); got != 1 {
			t.Fatalf("trail author should receive the Delete, got %d", got)
		}
	})

	// Mentioned actors were never told before, even with the trail present.
	t.Run("MentionedActorsReachedAlongsideTrailAuthor", func(t *testing.T) {
		f := setupCommentDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		m := f.remoteActor(t, "m")
		comment := f.comment(t, a, f.trail(t, b))
		f.recorded(t, "Create", comment, m, b)

		if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("m", 1, 5*time.Second); got != 1 {
			t.Fatalf("mentioned actor should receive the Delete, got %d", got)
		}
		// b is in the recorded cc and is the trail author; once, not twice.
		if got := f.in.wait("b", 1, 5*time.Second); got != 1 {
			t.Fatalf("trail author should receive the Delete exactly once, got %d", got)
		}
	})

	// An edit that added a mention went to a new audience; both are told.
	t.Run("UpdateAudienceIncluded", func(t *testing.T) {
		f := setupCommentDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		first := f.remoteActor(t, "first")
		later := f.remoteActor(t, "later")
		comment := f.comment(t, a, f.trail(t, b))
		f.recorded(t, "Create", comment, first, b)
		f.recorded(t, "Update", comment, later, b)

		if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
			t.Fatal(err)
		}

		for _, name := range []string{"first", "later", "b"} {
			if got := f.in.wait(name, 1, 5*time.Second); got != 1 {
				t.Fatalf("%s should receive the Delete once, got %d", name, got)
			}
		}
	})

	// Comments from before activities were recorded have no stored audience;
	// the trail author is still reached from the trail itself.
	t.Run("NoRecordedAudienceFallsBackToTrailAuthor", func(t *testing.T) {
		f := setupCommentDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		comment := f.comment(t, a, f.trail(t, b))

		if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("b", 1, 5*time.Second); got != 1 {
			t.Fatalf("trail author should receive the Delete, got %d", got)
		}
	})

	// A comment on a local trail with no remote mentions has nobody to tell.
	t.Run("LocalOnlyAudienceSendsNothing", func(t *testing.T) {
		f := setupCommentDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		localOwner := f.localAuthor(t, "owner")
		comment := f.comment(t, a, f.trail(t, localOwner))
		f.recorded(t, "Create", comment, localOwner)

		if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
			t.Fatal(err)
		}

		records, err := f.app.FindRecordsByFilter("activitypub_activities", "type = 'Delete'", "", 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 0 {
			t.Fatalf("expected no Delete to be recorded for a local-only audience, found %d", len(records))
		}
	})

	t.Run("AuthorGoneSendsNothing", func(t *testing.T) {
		f := setupCommentDeleteTestApp(t)

		a := f.localAuthor(t, "a")
		b := f.remoteActor(t, "b")
		comment := f.comment(t, a, f.trail(t, b))
		f.recorded(t, "Create", comment, b)

		if err := f.app.Delete(a); err != nil {
			t.Fatal(err)
		}

		if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
			t.Fatalf("expected a missing author to be tolerated, got %v", err)
		}

		if got := f.in.wait("b", 1, 300*time.Millisecond); got != 0 {
			t.Fatalf("nothing should be sent without an author to sign it, got %d", got)
		}
	})
}
