package federation

import (
	"context"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	ownerLocalIRI       = "https://trails.example.com/api/v1/activitypub/user/alice"
	ownerYIRI           = "https://remote.example.com/api/v1/activitypub/user/yauthor"
	ownerXIRI           = "https://remote.example.com/api/v1/activitypub/user/xother"
	ownerZIRI           = "https://third.example.com/api/v1/activitypub/user/zthird"
	ownerMIRI           = "https://social.example/users/m"
	ownerRemoteTrailIRI = "https://remote.example.com/api/v1/trail/t1"
	ownerLocalTrailIRI  = "https://trails.example.com/api/v1/trail/lt1"

	ownerLocalCommentIRI   = "https://trails.example.com/api/v1/comment/c1"
	ownerRemoteCommentIRI  = "https://remote.example.com/api/v1/comment/c2"
	ownerForeignCommentIRI = "https://third.example.com/api/v1/comment/c3"

	ownerLocalLogIRI   = "https://trails.example.com/api/v1/summit-log/s1"
	ownerRemoteLogIRI  = "https://remote.example.com/api/v1/summit-log/s2"
	ownerForeignLogIRI = "https://third.example.com/api/v1/summit-log/s3"

	ownerLocalListIRI = "https://trails.example.com/api/v1/list/ll1"
)

type ownerFixture struct {
	app               core.App
	alice, y, x, z, m *core.Record
	remoteTrail       *core.Record
	localTrail        *core.Record
}

func ownerGuardApp(t *testing.T) *ownerFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	app := newInboxTestApp(t)
	addTrailsCollection(t, app)
	addCommentsCollection(t, app)
	addSummitLogsCollection(t, app)
	addFeedCollection(t, app)
	feedDedupAddListsCollection(t, app)

	f := &ownerFixture{app: app}
	f.alice = createTestActor(t, app, ownerLocalIRI, "person", true)
	f.y = createTestActor(t, app, ownerYIRI, "person", false)
	f.x = createTestActor(t, app, ownerXIRI, "person", false)
	f.z = createTestActor(t, app, ownerZIRI, "person", false)
	f.m = createTestActor(t, app, ownerMIRI, "person", false)
	f.remoteTrail = seedTrailRecord(t, app, ownerRemoteTrailIRI, true, f.y.Id)
	f.localTrail = seedTrailRecord(t, app, ownerLocalTrailIRI, true, f.alice.Id)
	return f
}

func ownerText(s string) pub.NaturalLanguageValues {
	return pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, s))
}

// ownerCommentActivity builds a Create or Update of a Note replying to trailIRI.
func ownerCommentActivity(typ pub.ActivityVocabularyType, actorIRI, objectIRI, trailIRI, text string) pub.Activity {
	obj := &pub.Object{
		ID:        pub.IRI(objectIRI),
		Type:      pub.NoteType,
		Content:   ownerText(text),
		InReplyTo: pub.IRI(trailIRI),
	}
	act := pub.ActivityNew(pub.IRI("https://remote.example.com/api/v1/activitypub/activity/o1"), typ, obj)
	act.Actor = pub.IRI(actorIRI)
	return *act
}

func ownerSeedComment(t *testing.T, app core.App, iri, authorID, trailID, text string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatalf("find comments collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("author", authorID)
	r.Set("trail", trailID)
	r.Set("text", text)
	if err := app.Save(r); err != nil {
		t.Fatalf("save comment: %v", err)
	}
	return r
}

// ownerAssertUnchanged reloads a stored record and checks text, author and
// trail against the values it was seeded with.
func ownerAssertUnchanged(t *testing.T, app core.App, collection string, seeded *core.Record) {
	t.Helper()
	got, err := app.FindRecordById(collection, seeded.Id)
	if err != nil {
		t.Fatalf("reload %s: %v", collection, err)
	}
	for _, field := range []string{"text", "author", "trail"} {
		if got.GetString(field) != seeded.GetString(field) {
			t.Errorf("%s %s = %q, want %q", collection, field, got.GetString(field), seeded.GetString(field))
		}
	}
}

func ownerCount(t *testing.T, app core.App, collection, iri string) int64 {
	t.Helper()
	n, err := app.CountRecords(collection, dbx.HashExp{"iri": iri})
	if err != nil {
		t.Fatalf("count %s: %v", collection, err)
	}
	return n
}

func TestUpdateLocalCommentFromRemoteRefused(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedComment(t, f.app, ownerLocalCommentIRI, f.alice.Id, f.localTrail.Id, "original")

	act := ownerCommentActivity(pub.UpdateType, ownerYIRI, ownerLocalCommentIRI, ownerLocalTrailIRI, "hijacked")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act)

	authorGuardRequireRefused(t, err)
	ownerAssertUnchanged(t, f.app, "comments", seeded)
}

func TestUpdateRemoteCommentFromOtherActorSameHostRefused(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedComment(t, f.app, ownerRemoteCommentIRI, f.y.Id, f.remoteTrail.Id, "original")

	act := ownerCommentActivity(pub.UpdateType, ownerXIRI, ownerRemoteCommentIRI, ownerRemoteTrailIRI, "hijacked")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.x, f.alice, act)

	authorGuardRequireRefused(t, err)
	ownerAssertUnchanged(t, f.app, "comments", seeded)
}

func TestUpdateRemoteCommentFromOtherHostRefused(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedComment(t, f.app, ownerRemoteCommentIRI, f.y.Id, f.remoteTrail.Id, "original")

	act := ownerCommentActivity(pub.UpdateType, ownerZIRI, ownerRemoteCommentIRI, ownerRemoteTrailIRI, "hijacked")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.z, f.alice, act)

	authorGuardRequireRefused(t, err)
	ownerAssertUnchanged(t, f.app, "comments", seeded)
}

func TestCreateCommentOnForeignHostRefused(t *testing.T) {
	f := ownerGuardApp(t)

	act := ownerCommentActivity(pub.CreateType, ownerYIRI, ownerForeignCommentIRI, ownerRemoteTrailIRI, "smuggled")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act)

	authorGuardRequireRefused(t, err)
	if n := ownerCount(t, f.app, "comments", ownerForeignCommentIRI); n != 0 {
		t.Errorf("comments rows for %s = %d, want 0", ownerForeignCommentIRI, n)
	}
}

func TestUpdateCommentFromAuthorStillApplies(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedComment(t, f.app, ownerRemoteCommentIRI, f.y.Id, f.remoteTrail.Id, "original")

	act := ownerCommentActivity(pub.UpdateType, ownerYIRI, ownerRemoteCommentIRI, ownerRemoteTrailIRI, "edited")
	if err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act); err != nil {
		t.Fatalf("Update from the stored author: %v", err)
	}

	got, err := f.app.FindRecordById("comments", seeded.Id)
	if err != nil {
		t.Fatalf("reload comment: %v", err)
	}
	if got.GetString("text") != "edited" {
		t.Errorf("text = %q, want %q", got.GetString("text"), "edited")
	}
	if got.GetString("author") != f.y.Id {
		t.Errorf("author = %q, want %q", got.GetString("author"), f.y.Id)
	}
}

func TestCreateMastodonStyleReplyStillStored(t *testing.T) {
	f := ownerGuardApp(t)
	const noteIRI = "https://social.example/users/m/statuses/1"

	act := ownerCommentActivity(pub.CreateType, ownerMIRI, noteIRI, ownerRemoteTrailIRI, "nice trail")
	if err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.m, f.alice, act); err != nil {
		t.Fatalf("Create of a reply from another ActivityPub server: %v", err)
	}

	if n := ownerCount(t, f.app, "comments", noteIRI); n != 1 {
		t.Fatalf("comments rows for %s = %d, want 1", noteIRI, n)
	}
	got, err := f.app.FindFirstRecordByData("comments", "iri", noteIRI)
	if err != nil {
		t.Fatalf("find comment: %v", err)
	}
	if got.GetString("author") != f.m.Id {
		t.Errorf("author = %q, want %q", got.GetString("author"), f.m.Id)
	}
}

// addSummitLogsCollection adds a minimal summit_logs collection without file fields.
func addSummitLogsCollection(t *testing.T, app core.App) {
	t.Helper()
	logsJSON := `[{
		"id": "pbc_summitlogs_ow1",
		"name": "summit_logs",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textsumlogiri001","max":0,"min":0,"name":"iri","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textsumlogaut001","max":0,"min":0,"name":"author","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textsumlogtrl001","max":0,"min":0,"name":"trail","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textsumlogtxt001","max":0,"min":0,"name":"text","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"datesumlogdate01","max":"","min":"","name":"date","presentable":false,"required":false,"system":false,"type":"date"},
			{"hidden":false,"id":"numsumlogdist001","max":null,"min":null,"name":"distance","onlyInt":false,"presentable":false,"required":false,"system":false,"type":"number"},
			{"hidden":false,"id":"numsumlogdur0001","max":null,"min":null,"name":"duration","onlyInt":false,"presentable":false,"required":false,"system":false,"type":"number"},
			{"hidden":false,"id":"numsumlogegain01","max":null,"min":null,"name":"elevation_gain","onlyInt":false,"presentable":false,"required":false,"system":false,"type":"number"},
			{"hidden":false,"id":"numsumlogeloss01","max":null,"min":null,"name":"elevation_loss","onlyInt":false,"presentable":false,"required":false,"system":false,"type":"number"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": []
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(logsJSON), false); err != nil {
		t.Fatalf("create summit_logs collection: %v", err)
	}
}

// ownerSummitLogActivity builds a Create or Update of a summit log replying to trailIRI.
func ownerSummitLogActivity(typ pub.ActivityVocabularyType, actorIRI, objectIRI, trailIRI, text string) pub.Activity {
	obj := &pub.Object{
		ID:        pub.IRI(objectIRI),
		Type:      pub.NoteType,
		Content:   ownerText(text),
		InReplyTo: pub.IRI(trailIRI),
		StartTime: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Tag:       pub.ItemCollection{},
	}
	act := pub.ActivityNew(pub.IRI("https://remote.example.com/api/v1/activitypub/activity/o2"), typ, obj)
	act.Actor = pub.IRI(actorIRI)
	return *act
}

func ownerSeedSummitLog(t *testing.T, app core.App, iri, authorID, trailID, text string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		t.Fatalf("find summit_logs collection: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("author", authorID)
	r.Set("trail", trailID)
	r.Set("text", text)
	if err := app.Save(r); err != nil {
		t.Fatalf("save summit log: %v", err)
	}
	return r
}

func TestUpdateLocalSummitLogFromRemoteRefused(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedSummitLog(t, f.app, ownerLocalLogIRI, f.alice.Id, f.localTrail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, ownerYIRI, ownerLocalLogIRI, ownerLocalTrailIRI, "hijacked")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act)

	authorGuardRequireRefused(t, err)
	ownerAssertUnchanged(t, f.app, "summit_logs", seeded)
}

func TestUpdateRemoteSummitLogFromOtherActorRefused(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedSummitLog(t, f.app, ownerRemoteLogIRI, f.y.Id, f.remoteTrail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, ownerXIRI, ownerRemoteLogIRI, ownerRemoteTrailIRI, "hijacked")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.x, f.alice, act)

	authorGuardRequireRefused(t, err)
	ownerAssertUnchanged(t, f.app, "summit_logs", seeded)
}

func TestUpdateRemoteSummitLogFromOtherHostRefused(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedSummitLog(t, f.app, ownerRemoteLogIRI, f.y.Id, f.remoteTrail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, ownerZIRI, ownerRemoteLogIRI, ownerRemoteTrailIRI, "hijacked")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.z, f.alice, act)

	authorGuardRequireRefused(t, err)
	ownerAssertUnchanged(t, f.app, "summit_logs", seeded)
}

func TestCreateSummitLogOnForeignHostRefused(t *testing.T) {
	f := ownerGuardApp(t)

	act := ownerSummitLogActivity(pub.CreateType, ownerYIRI, ownerForeignLogIRI, ownerRemoteTrailIRI, "smuggled")
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act)

	authorGuardRequireRefused(t, err)
	if n := ownerCount(t, f.app, "summit_logs", ownerForeignLogIRI); n != 0 {
		t.Errorf("summit_logs rows for %s = %d, want 0", ownerForeignLogIRI, n)
	}
}

func TestUpdateSummitLogFromAuthorStillApplies(t *testing.T) {
	f := ownerGuardApp(t)
	seeded := ownerSeedSummitLog(t, f.app, ownerRemoteLogIRI, f.y.Id, f.remoteTrail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, ownerYIRI, ownerRemoteLogIRI, ownerRemoteTrailIRI, "edited")
	if err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act); err != nil {
		t.Fatalf("Update from the stored author: %v", err)
	}

	got, err := f.app.FindRecordById("summit_logs", seeded.Id)
	if err != nil {
		t.Fatalf("reload summit log: %v", err)
	}
	if got.GetString("text") != "edited" {
		t.Errorf("text = %q, want %q", got.GetString("text"), "edited")
	}
	if got.GetString("author") != f.y.Id {
		t.Errorf("author = %q, want %q", got.GetString("author"), f.y.Id)
	}
}

func TestUpdateLocalTrailFromRemoteRefused(t *testing.T) {
	f := ownerGuardApp(t)

	act := authorGuardAsUpdate(feedDedupTrailCreate(ownerYIRI, ownerLocalTrailIRI))
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act)

	authorGuardRequireRefused(t, err)
	got, rerr := f.app.FindRecordById("trails", f.localTrail.Id)
	if rerr != nil {
		t.Fatalf("reload trail: %v", rerr)
	}
	if got.GetString("author") != f.alice.Id {
		t.Errorf("trail author = %q, want %q", got.GetString("author"), f.alice.Id)
	}
	if n := ownerCount(t, f.app, "trails", ownerLocalTrailIRI); n != 1 {
		t.Errorf("trails rows for %s = %d, want 1", ownerLocalTrailIRI, n)
	}
}

func TestUpdateLocalListFromRemoteRefused(t *testing.T) {
	f := ownerGuardApp(t)
	list := authorGuardSeedList(t, f.app, ownerLocalListIRI, f.alice.Id)

	act := authorGuardAsUpdate(feedDedupListCreate(ownerYIRI, ownerLocalListIRI))
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), f.y, f.alice, act)

	authorGuardRequireRefused(t, err)
	got, rerr := f.app.FindRecordById("lists", list.Id)
	if rerr != nil {
		t.Fatalf("reload list: %v", rerr)
	}
	if got.GetString("author") != f.alice.Id {
		t.Errorf("list author = %q, want %q", got.GetString("author"), f.alice.Id)
	}
	if got.GetString("description") != "" {
		t.Errorf("list description = %q, want unchanged empty", got.GetString("description"))
	}
}
