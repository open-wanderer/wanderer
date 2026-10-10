package federation

import (
	"context"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	feedDedupAuthorIRI   = "https://remote.example.com/api/v1/activitypub/user/rauthor"
	feedDedupOtherIRI    = "https://remote.example.com/api/v1/activitypub/user/xother"
	feedDedupAliceIRI    = "https://trails.example.com/api/v1/activitypub/user/alice"
	feedDedupBobIRI      = "https://trails.example.com/api/v1/activitypub/user/bob"
	feedDedupInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"
	feedDedupTrailIRI    = "https://remote.example.com/api/v1/trail/t1"
	feedDedupListIRI     = "https://remote.example.com/api/v1/list/l1"
)

// feedDedupAddListsCollection adds a minimal lists collection.
func feedDedupAddListsCollection(t *testing.T, app core.App) {
	t.Helper()
	listsJSON := `[{
		"id": "pbc_lists_feed_t01",
		"name": "lists",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textlistiri0001","max":0,"min":0,"name":"iri","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textlistname001","max":0,"min":0,"name":"name","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textlistdesc001","max":0,"min":0,"name":"description","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textlistauth001","max":0,"min":0,"name":"author","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"boollistpublic1","name":"public","presentable":false,"required":false,"system":false,"type":"bool"},
			{"hidden":false,"id":"boollistsync001","name":"needs_full_sync","presentable":false,"required":false,"system":false,"type":"bool"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": []
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(listsJSON), false); err != nil {
		t.Fatalf("create lists collection: %v", err)
	}
}

func newFeedDedupTestApp(t *testing.T) core.App {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	app := newInboxTestApp(t)
	addTrailsCollection(t, app)
	addFeedCollection(t, app)
	feedDedupAddListsCollection(t, app)
	return app
}

func feedDedupTrailCreate(actorIRI, trailIRI string) pub.Activity {
	obj := &pub.Object{
		ID:      pub.IRI(trailIRI),
		Type:    pub.NoteType,
		Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Trail")),
		Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "")),
		Location: &pub.Place{
			Type:      pub.PlaceType,
			Latitude:  47.5,
			Longitude: 11.5,
			Name:      pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test Place")),
		},
	}
	act := pub.ActivityNew(pub.IRI("https://remote.example.com/api/v1/activitypub/activity/c1"), pub.CreateType, obj)
	act.Actor = pub.IRI(actorIRI)
	return *act
}

func feedDedupListCreate(actorIRI, listIRI string) pub.Activity {
	obj := &pub.Object{
		ID:      pub.IRI(listIRI),
		Type:    pub.NoteType,
		Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Test List")),
		Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "desc")),
	}
	act := pub.ActivityNew(pub.IRI("https://remote.example.com/api/v1/activitypub/activity/c2"), pub.CreateType, obj)
	act.Actor = pub.IRI(actorIRI)
	return *act
}

func feedDedupCount(t *testing.T, app core.App, actorID, itemID string) int {
	t.Helper()
	n, err := app.CountRecords("feed", dbx.HashExp{"actor": actorID, "item": itemID})
	if err != nil {
		t.Fatalf("count feed: %v", err)
	}
	return int(n)
}

type feedDedupActors struct {
	author, other, alice, bob, instance *core.Record
}

func feedDedupSeedActors(t *testing.T, app core.App) feedDedupActors {
	t.Helper()
	return feedDedupActors{
		author:   createTestActor(t, app, feedDedupAuthorIRI, "person", false),
		other:    createTestActor(t, app, feedDedupOtherIRI, "person", false),
		alice:    createTestActor(t, app, feedDedupAliceIRI, "person", true),
		bob:      createTestActor(t, app, feedDedupBobIRI, "person", true),
		instance: createTestActor(t, app, feedDedupInstanceIRI, "instance", true),
	}
}

func feedDedupDeliver(t *testing.T, app core.App, actor, recipient *core.Record, act pub.Activity) {
	t.Helper()
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), actor, recipient, act); err != nil {
		t.Fatalf("ProcessCreateOrUpdateActivity to %s: %v", recipient.GetString("iri"), err)
	}
}

func feedDedupAssertCounts(t *testing.T, app core.App, collection, iri string, a feedDedupActors) {
	t.Helper()
	n, err := app.CountRecords(collection, dbx.HashExp{"iri": iri})
	if err != nil {
		t.Fatalf("count %s: %v", collection, err)
	}
	if n != 1 {
		t.Fatalf("%s rows for %s = %d, want 1", collection, iri, n)
	}
	obj, err := app.FindFirstRecordByData(collection, "iri", iri)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	if got := feedDedupCount(t, app, a.alice.Id, obj.Id); got != 1 {
		t.Errorf("feed rows for alice = %d, want 1", got)
	}
	if got := feedDedupCount(t, app, a.bob.Id, obj.Id); got != 1 {
		t.Errorf("feed rows for bob = %d, want 1", got)
	}
	if got := feedDedupCount(t, app, a.instance.Id, obj.Id); got != 0 {
		t.Errorf("feed rows for instance actor = %d, want 0", got)
	}
}

func TestCreateTrailFeedEntryPerRecipient(t *testing.T) {
	app := newFeedDedupTestApp(t)
	a := feedDedupSeedActors(t, app)
	act := feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI)

	for _, r := range []*core.Record{a.instance, a.alice, a.bob, a.alice} {
		feedDedupDeliver(t, app, a.author, r, act)
	}
	feedDedupAssertCounts(t, app, "trails", feedDedupTrailIRI, a)
}

func TestCreateTrailFeedEntryUserInboxFirst(t *testing.T) {
	app := newFeedDedupTestApp(t)
	a := feedDedupSeedActors(t, app)
	act := feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI)

	for _, r := range []*core.Record{a.alice, a.instance, a.bob} {
		feedDedupDeliver(t, app, a.author, r, act)
	}
	feedDedupAssertCounts(t, app, "trails", feedDedupTrailIRI, a)
}

func TestCreateListFeedEntryPerRecipient(t *testing.T) {
	app := newFeedDedupTestApp(t)
	a := feedDedupSeedActors(t, app)
	act := feedDedupListCreate(feedDedupAuthorIRI, feedDedupListIRI)

	for _, r := range []*core.Record{a.instance, a.alice, a.bob, a.alice} {
		feedDedupDeliver(t, app, a.author, r, act)
	}
	feedDedupAssertCounts(t, app, "lists", feedDedupListIRI, a)
}

func TestCreateTrailDuplicateFromNonAuthorAddsNoFeedEntry(t *testing.T) {
	app := newFeedDedupTestApp(t)
	a := feedDedupSeedActors(t, app)

	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, a.author.Id)

	// The origin names the stored author, so the non-author stays refused.
	stubTrailOrigin(t, feedDedupTrailIRI, feedDedupAuthorIRI, nil)

	act := feedDedupTrailCreate(feedDedupOtherIRI, feedDedupTrailIRI)
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), a.other, a.alice, act); err == nil {
		t.Error("duplicate Create from a non-author was accepted, want an error")
	}

	if got := feedDedupCount(t, app, a.alice.Id, trail.Id); got != 0 {
		t.Errorf("feed rows for alice = %d, want 0", got)
	}
	reloaded, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatalf("reload trail: %v", err)
	}
	if got := reloaded.GetString("author"); got != a.author.Id {
		t.Errorf("trail author = %q, want %q", got, a.author.Id)
	}
}
