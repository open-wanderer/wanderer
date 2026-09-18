package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// A remote trail reaches this instance in two ways: delivered to a follower's
// inbox as Create/Update, which files it into that follower's feed, or synced
// on demand when somebody opens it, which files it nowhere. The Delete that
// follows must remove the local copy either way, and must clear every feed it
// was filed into, not just the first.

type trailDeleteFixture struct {
	app    *pbtests.TestApp
	actors *core.Collection
	trails *core.Collection
	lists  *core.Collection
	feed   *core.Collection
}

func setupTrailDeleteTestApp(t *testing.T) *trailDeleteFixture {
	t.Helper()

	t.Setenv("ORIGIN", "https://local.example")

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "is_local"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
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

	return &trailDeleteFixture{app: app, actors: actors, trails: trails, lists: lists, feed: feed}
}

func (f *trailDeleteFixture) actor(t *testing.T, name string, local bool) *core.Record {
	t.Helper()

	host := "remote.example"
	if local {
		host = "local.example"
	}
	r := core.NewRecord(f.actors)
	r.Set("iri", "https://"+host+"/api/v1/activitypub/user/"+name)
	r.Set("is_local", local)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
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
