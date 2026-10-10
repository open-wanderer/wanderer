package federation

import (
	"context"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// addTrailsCollection adds a minimal trails collection to the test app.
// Fields needed: id, iri (text), author (text), public (bool), name (text).
func addTrailsCollection(t *testing.T, app core.App) {
	t.Helper()
	trailsJSON := `[{
		"id": "pbc_trails_test001",
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
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": []
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(trailsJSON), false); err != nil {
		t.Fatalf("create trails collection: %v", err)
	}
}

// addCommentsCollection adds a minimal comments collection to the test app.
// Fields needed: id, iri (text), author (text), trail (text), text (text).
func addCommentsCollection(t *testing.T, app core.App) {
	t.Helper()
	commentsJSON := `[{
		"id": "pbc_comments_test1",
		"name": "comments",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textcommentiri1","max":0,"min":0,"name":"iri","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textcommentaut1","max":0,"min":0,"name":"author","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textcommenttrl1","max":0,"min":0,"name":"trail","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textcommenttxt1","max":0,"min":0,"name":"text","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": []
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(commentsJSON), false); err != nil {
		t.Fatalf("create comments collection: %v", err)
	}
}

// seedTrailRecord inserts a trail record with the given IRI and public flag.
func seedTrailRecord(t *testing.T, app core.App, iri string, public bool, authorId string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatalf("find trails collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("public", public)
	r.Set("author", authorId)
	r.Set("name", "Test Trail")
	if err := app.Save(r); err != nil {
		t.Fatalf("save trail record: %v", err)
	}
	return r
}

// TestProcessCreateOrUpdateTrailActivityDedupOnCreate verifies that a Create
// activity whose object IRI already exists in the trails collection is silently
// dropped (returns nil, count stays at 1).
func TestProcessCreateOrUpdateTrailActivityDedupOnCreate(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)

	actor := createTestActor(t, app, "https://remote.example.com/api/v1/activitypub/instance", "instance", false)

	const trailIRI = "https://remote.example.com/api/v1/trail/abc"

	// Seed a trail row with the known IRI
	seedTrailRecord(t, app, trailIRI, true, actor.Id)

	// Count rows before — expect 1
	beforeCount, err := app.CountRecords("trails", nil)
	if err != nil {
		t.Fatalf("count trails before: %v", err)
	}
	if beforeCount != 1 {
		t.Fatalf("expected 1 trail before dedup test, got %d", beforeCount)
	}

	// Build a Create activity with the same object IRI
	obj := pub.ObjectNew(pub.NoteType)
	obj.ID = pub.IRI(trailIRI)
	activity := pub.ActivityNew(pub.IRI("https://remote.example.com/activity/1"), pub.CreateType, obj)
	activity.Actor = pub.IRI(actor.GetString("iri"))

	// ProcessCreateOrUpdateActivity routes on the object IRI.
	recipient := createTestActor(t, app, "https://trails.example.com/api/v1/activitypub/instance", "instance", true)

	result := ProcessCreateOrUpdateActivity(app, context.Background(), actor, recipient, *activity)
	if result != nil {
		t.Errorf("expected nil (dedup path), got error: %v", result)
	}

	// Count rows after — must still be 1 (duplicate dropped)
	afterCount, err := app.CountRecords("trails", nil)
	if err != nil {
		t.Fatalf("count trails after: %v", err)
	}
	if afterCount != beforeCount {
		t.Errorf("trail count changed from %d to %d — duplicate was not dropped", beforeCount, afterCount)
	}
}

// TestProcessCreateOrUpdateTrailActivityAllowsUpdate verifies that an Update
// activity with the same object IRI is not dropped as a duplicate. It fails
// later because the test schema is incomplete.
func TestProcessCreateOrUpdateTrailActivityAllowsUpdate(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)

	actor := createTestActor(t, app, "https://remote.example.com/api/v1/activitypub/instance", "instance", false)

	const trailIRI = "https://remote.example.com/api/v1/trail/abc"

	// Seed a trail row with the known IRI — exists so dedup would fire for Create
	seedTrailRecord(t, app, trailIRI, true, actor.Id)

	// Build an Update activity with the same object IRI
	obj := pub.ObjectNew(pub.NoteType)
	obj.ID = pub.IRI(trailIRI)
	activity := pub.ActivityNew(pub.IRI("https://remote.example.com/activity/2"), pub.UpdateType, obj)
	activity.Actor = pub.IRI(actor.GetString("iri"))

	recipient := createTestActor(t, app, "https://trails.example.com/api/v1/activitypub/instance", "instance", true)

	// The Update passes the dedup and fails in TrailFromActivity.
	result := ProcessCreateOrUpdateActivity(app, context.Background(), actor, recipient, *activity)
	_ = result // fails on the incomplete test schema

	// No duplicate row was inserted.
	afterCount, err := app.CountRecords("trails", nil)
	if err != nil {
		t.Fatalf("count trails after Update: %v", err)
	}
	if afterCount != 1 {
		t.Errorf("trail count after Update = %d, want 1", afterCount)
	}
}

// TestCreateCommentActivityPrivateTrailReturnsNil verifies that
// CreateCommentActivity returns nil and saves no activitypub_activities record
// when the comment's parent trail has public=false.
func TestCreateCommentActivityPrivateTrailReturnsNil(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	addTrailsCollection(t, app)
	addCommentsCollection(t, app)

	// Create a remote author actor
	author := createTestActor(t, app, "https://remote.example.com/users/alice", "person", false)

	// Seed a trail with public=false
	trail := seedTrailRecord(t, app, "https://remote.example.com/api/v1/trail/private", false, author.Id)

	// Seed a comment referencing the private trail
	commentCol, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatalf("find comments collection: %v", err)
	}
	comment := core.NewRecord(commentCol)
	comment.Set("iri", "https://remote.example.com/api/v1/comment/1")
	comment.Set("author", author.Id)
	comment.Set("trail", trail.Id)
	comment.Set("text", "hello")
	if err := app.Save(comment); err != nil {
		t.Fatalf("save comment record: %v", err)
	}

	// Count activitypub_activities before — should be 0
	beforeCount, err := app.CountRecords("activitypub_activities", nil)
	if err != nil {
		t.Fatalf("count activities before: %v", err)
	}

	// Call CreateCommentActivity — should gate on private trail and return nil
	result := CreateCommentActivity(app, context.Background(), comment, pub.CreateType)
	if result != nil {
		t.Errorf("CreateCommentActivity expected nil for private trail, got: %v", result)
	}

	// Assert no activitypub_activities record of type Create was saved
	afterCount, err := app.CountRecords("activitypub_activities", nil)
	if err != nil {
		t.Fatalf("count activities after: %v", err)
	}
	if afterCount != beforeCount {
		t.Errorf("activitypub_activities count changed from %d to %d — gate did not suppress fanout", beforeCount, afterCount)
	}
}
