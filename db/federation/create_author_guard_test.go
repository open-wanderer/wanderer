package federation

import (
	"context"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	authorGuardForeignTrailIRI = "https://third.example.com/api/v1/trail/x"
	authorGuardForeignListIRI  = "https://third.example.com/api/v1/list/x"
)

// authorGuardAddTrailsCollection is the minimal trails collection plus a
// needs_full_sync flag, so a metadata side effect is observable.
func authorGuardAddTrailsCollection(t *testing.T, app core.App) {
	t.Helper()
	trailsJSON := `[{
		"id": "pbc_trails_guard01",
		"name": "trails",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"texttrailiri001","max":0,"min":0,"name":"iri","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"texttrailauth1","max":0,"min":0,"name":"author","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"texttrailname1","max":0,"min":0,"name":"name","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"booltrailpub01","name":"public","presentable":false,"required":false,"system":false,"type":"bool"},
			{"hidden":false,"id":"booltrailsync01","name":"needs_full_sync","presentable":false,"required":false,"system":false,"type":"bool"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": []
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(trailsJSON), false); err != nil {
		t.Fatalf("create trails collection: %v", err)
	}
}

func authorGuardTestApp(t *testing.T) core.App {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	app := newInboxTestApp(t)
	addFeedCollection(t, app)
	feedDedupAddListsCollection(t, app)
	authorGuardAddTrailsCollection(t, app)
	return app
}

func authorGuardAsUpdate(act pub.Activity) pub.Activity {
	act.Type = pub.UpdateType
	return act
}

func authorGuardSeedList(t *testing.T, app core.App, iri, authorID string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatalf("find lists collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("public", true)
	r.Set("author", authorID)
	r.Set("name", "Seeded List")
	if err := app.Save(r); err != nil {
		t.Fatalf("save list: %v", err)
	}
	return r
}

func authorGuardRequireRefused(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected the activity to be refused, got nil")
	}
}

func authorGuardAssertNoRows(t *testing.T, app core.App, collection, iri string) {
	t.Helper()
	n, err := app.CountRecords(collection, dbx.HashExp{"iri": iri})
	if err != nil {
		t.Fatalf("count %s: %v", collection, err)
	}
	if n != 0 {
		t.Errorf("%s rows for %s = %d, want 0", collection, iri, n)
	}
}

func authorGuardAssertNoFeed(t *testing.T, app core.App, actorID string) {
	t.Helper()
	n, err := app.CountRecords("feed", dbx.HashExp{"actor": actorID})
	if err != nil {
		t.Fatalf("count feed: %v", err)
	}
	if n != 0 {
		t.Errorf("feed rows for %s = %d, want 0", actorID, n)
	}
}

func TestCreateTrailOnForeignHostRefused(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	act := feedDedupTrailCreate(feedDedupAuthorIRI, authorGuardForeignTrailIRI)
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act)

	authorGuardRequireRefused(t, err)
	authorGuardAssertNoRows(t, app, "trails", authorGuardForeignTrailIRI)
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestUpdateTrailOnForeignHostRefused(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	trail := seedTrailRecord(t, app, authorGuardForeignTrailIRI, true, a.author.Id)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, authorGuardForeignTrailIRI))
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act)

	authorGuardRequireRefused(t, err)
	reloaded, rerr := app.FindRecordById("trails", trail.Id)
	if rerr != nil {
		t.Fatalf("reload trail: %v", rerr)
	}
	if reloaded.GetBool("needs_full_sync") {
		t.Error("needs_full_sync was set by a refused Update")
	}
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestUpdateTrailFromNonAuthorRefused(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, a.author.Id)

	// The origin names the stored author, so the non-author stays refused.
	stubTrailOrigin(t, feedDedupTrailIRI, feedDedupAuthorIRI, nil)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupOtherIRI, feedDedupTrailIRI))
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.other, a.alice, act)

	authorGuardRequireRefused(t, err)
	reloaded, rerr := app.FindRecordById("trails", trail.Id)
	if rerr != nil {
		t.Fatalf("reload trail: %v", rerr)
	}
	if got := reloaded.GetString("author"); got != a.author.Id {
		t.Errorf("trail author = %q, want %q", got, a.author.Id)
	}
	if reloaded.GetBool("needs_full_sync") {
		t.Error("needs_full_sync was set by an Update from a non-author")
	}
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestCreateListOnForeignHostRefused(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	act := feedDedupListCreate(feedDedupAuthorIRI, authorGuardForeignListIRI)
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act)

	authorGuardRequireRefused(t, err)
	authorGuardAssertNoRows(t, app, "lists", authorGuardForeignListIRI)
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestUpdateListFromNonAuthorAddsNoFeedEntry(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	list := authorGuardSeedList(t, app, feedDedupListIRI, a.author.Id)

	act := authorGuardAsUpdate(feedDedupListCreate(feedDedupOtherIRI, feedDedupListIRI))
	err := ProcessCreateOrUpdateActivity(app, context.Background(), a.other, a.alice, act)

	authorGuardRequireRefused(t, err)
	if got := feedDedupCount(t, app, a.alice.Id, list.Id); got != 0 {
		t.Errorf("feed rows for alice = %d, want 0", got)
	}
	reloaded, rerr := app.FindRecordById("lists", list.Id)
	if rerr != nil {
		t.Fatalf("reload list: %v", rerr)
	}
	if got := reloaded.GetString("author"); got != a.author.Id {
		t.Errorf("list author = %q, want %q", got, a.author.Id)
	}
}

func TestUpdateTrailFromAuthorStillApplies(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)

	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, a.author.Id)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI))
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act); err != nil {
		t.Fatalf("Update from the stored author: %v", err)
	}

	reloaded, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatalf("reload trail: %v", err)
	}
	if !reloaded.GetBool("needs_full_sync") {
		t.Error("needs_full_sync = false after an Update from the stored author, want true")
	}
	if got := feedDedupCount(t, app, a.alice.Id, trail.Id); got != 1 {
		t.Errorf("feed rows for alice = %d, want 1", got)
	}
}
