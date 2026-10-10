package federation

import (
	"context"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// A summit log or comment can arrive for a trail this instance has never
// seen, which is then fetched from its origin. The copy belongs to the
// trail's author, not to whoever logged or commented: filed under the
// sender's name it would show up as theirs, and the author's later Delete
// would be refused as coming from a stranger.

// setupFetchTrailTest returns an app with the collections a fetched trail is
// stored in, and a helper creating a freshly cached actor on remote.example.
func setupFetchTrailTest(t *testing.T) (*pbtests.TestApp, func(name string) *core.Record) {
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
		&core.BoolField{Name: "is_local"},
		&core.DateField{Name: "last_fetched"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "public"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tags", "feed"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "item"})
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
	}

	actor := func(name string) *core.Record {
		r := core.NewRecord(actors)
		r.Set("iri", "https://remote.example/api/v1/activitypub/user/"+name)
		r.Set("is_local", false)
		// Freshly cached, so resolving it by IRI does not go to the network.
		r.Set("last_fetched", time.Now())
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	return app, actor
}

// remoteTrailObject is the trail at iri as its host serves it.
func remoteTrailObject(iri, authorIRI string) *pub.Object {
	remote := pub.ObjectNew(pub.NoteType)
	remote.ID = pub.IRI(iri)
	remote.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Owner's trail"))
	remote.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, ""))
	remote.AttributedTo = pub.IRI(authorIRI)
	remote.Location = &pub.Place{Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Somewhere"))}
	remote.StartTime = time.Now()
	return remote
}

// serveTrailObject answers trail fetches for trailIRI with remote.
func serveTrailObject(t *testing.T, trailIRI string, remote *pub.Object) {
	t.Helper()
	trailAuthorRefusals.reset()
	orig := fetchTrailObject
	fetchTrailObject = func(ctx context.Context, iri string) (*pub.Object, error) {
		if iri != trailIRI {
			t.Fatalf("fetched %q, want %q", iri, trailIRI)
		}
		return remote, nil
	}
	t.Cleanup(func() {
		fetchTrailObject = orig
		trailAuthorRefusals.reset()
	})
}

func TestFetchTrailAttributesToTrailAuthor(t *testing.T) {
	app, actor := setupFetchTrailTest(t)
	owner := actor("owner")
	logger := actor("logger")

	trailIRI := "https://remote.example/api/v1/trail/abc"
	remote := remoteTrailObject(trailIRI, owner.GetString("iri"))
	serveTrailObject(t, trailIRI, remote)

	// The host serving the trail may only attribute it to one of its own.
	remote.AttributedTo = pub.IRI("https://elsewhere.example/api/v1/activitypub/user/victim")
	if _, err := fetchTrail(app, context.Background(), logger, trailIRI); err == nil {
		t.Fatal("accepted a trail attributed to an actor on another host")
	}
	remote.AttributedTo = pub.IRI(owner.GetString("iri"))

	trail, err := fetchTrail(app, context.Background(), logger, trailIRI)
	if err != nil {
		t.Fatal(err)
	}

	if got := trail.GetString("author"); got != owner.Id {
		t.Fatalf("trail attributed to %q, want its author %q (the sender is %q)", got, owner.Id, logger.Id)
	}

	// Now the author's Delete is accepted rather than refused as a
	// stranger's.
	activity := pub.DeleteNew(pub.IRI("https://remote.example/api/v1/activitypub/activity/x"), pub.IRI(trailIRI))
	activity.Actor = pub.IRI(owner.GetString("iri"))
	if err := processDeleteTrailActivity(app, owner, *activity); err != nil {
		t.Fatalf("author's Delete refused: %v", err)
	}
}

// A list can contain a trail from a third host, which is imported from that
// host and filed under the author it names there.
func TestImportTrailAttributesToTrailAuthor(t *testing.T) {
	app, actor := setupFetchTrailTest(t)
	owner := actor("owner")

	trailIRI := "https://remote.example/api/v1/trail/abc"
	remote := remoteTrailObject(trailIRI, owner.GetString("iri"))
	serveTrailObject(t, trailIRI, remote)

	trail, err := ImportTrail(app, context.Background(), trailIRI)
	if err != nil {
		t.Fatal(err)
	}
	if trail.GetString("iri") != trailIRI || trail.GetString("author") != owner.Id {
		t.Fatalf("imported trail iri %q author %q, want %q by %q", trail.GetString("iri"), trail.GetString("author"), trailIRI, owner.Id)
	}
}

func TestImportTrailRefusesMisattributedTrails(t *testing.T) {
	app, actor := setupFetchTrailTest(t)
	owner := actor("owner")

	trailIRI := "https://remote.example/api/v1/trail/abc"
	remote := remoteTrailObject(trailIRI, owner.GetString("iri"))
	serveTrailObject(t, trailIRI, remote)

	remote.AttributedTo = pub.IRI("https://elsewhere.example/api/v1/activitypub/user/victim")
	if _, err := ImportTrail(app, context.Background(), trailIRI); err == nil {
		t.Error("imported a trail attributed to an actor on another host")
	}
	remote.AttributedTo = pub.IRI(owner.GetString("iri"))

	remote.ID = pub.IRI("https://remote.example/api/v1/trail/other")
	if _, err := ImportTrail(app, context.Background(), trailIRI); err == nil {
		t.Error("imported a trail the host served under another id")
	}

	if _, err := ImportTrail(app, context.Background(), "https://local.example/api/v1/trail/abc"); err == nil {
		t.Error("imported a local trail")
	}

	if n, _ := app.CountRecords("trails"); n != 0 {
		t.Errorf("%d trails stored, want none", n)
	}
}
