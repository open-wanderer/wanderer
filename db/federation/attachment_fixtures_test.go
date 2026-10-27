package federation

import (
	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"testing"
	"time"
)

type feedDedupActors struct {
	author, other, alice, bob, instance *core.Record
}

func newInboxTestApp(t *testing.T) core.App {
	t.Helper()

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       t.TempDir(),
		EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY",
	})

	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// activitypub_actors collection — same JSON as in instance_test.go
	actorsJSON := `[{
		"id": "pbc_1295301207",
		"name": "activitypub_actors",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text4166911607","max":0,"min":0,"name":"username","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text4002953752","max":0,"min":0,"name":"preferred_username","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text2812878347","max":0,"min":0,"name":"domain","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text3458754147","max":0,"min":0,"name":"summary","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"text1727648867","max":0,"min":0,"name":"public_key","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":true,"id":"text4160324774","max":0,"min":0,"name":"private_key","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"select_actor_type_001","maxSelect":1,"name":"actor_type","presentable":false,"required":false,"system":false,"type":"select","values":["person","instance"]},
			{"hidden":false,"id":"bool2193750486","name":"is_local","presentable":false,"required":false,"system":false,"type":"bool"},
			{"exceptDomains":null,"hidden":false,"id":"url126331327","name":"iri","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"},
			{"exceptDomains":null,"hidden":false,"id":"url2115105593","name":"inbox","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"},
			{"exceptDomains":null,"hidden":false,"id":"url1793578352","name":"outbox","onlyDomains":null,"presentable":false,"required":false,"system":false,"type":"url"},
			{"hidden":false,"id":"date2062531289","max":"","min":"","name":"last_fetched","presentable":false,"required":false,"system":false,"type":"date"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [
			"CREATE UNIQUE INDEX ` + "`idx_rpT7QJwWTm`" + ` ON ` + "`activitypub_actors`" + ` (` + "`iri`" + `)"
		]
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(actorsJSON), false); err != nil {
		t.Fatalf("create activitypub_actors collection: %v", err)
	}

	// follows collection — follower/followee relate to activitypub_actors (pbc_1295301207)
	followsJSON := `[{
		"id": "8obn1ukumze565i",
		"name": "follows",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"cascadeDelete":true,"collectionId":"pbc_1295301207","hidden":false,"id":"relation3117812038","maxSelect":1,"minSelect":0,"name":"follower","presentable":false,"required":true,"system":false,"type":"relation"},
			{"cascadeDelete":true,"collectionId":"pbc_1295301207","hidden":false,"id":"relation973442177","maxSelect":1,"minSelect":0,"name":"followee","presentable":false,"required":true,"system":false,"type":"relation"},
			{"hidden":false,"id":"select2063623452","maxSelect":1,"name":"status","presentable":false,"required":true,"system":false,"type":"select","values":["pending","accepted"]},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [],
		"system": false
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(followsJSON), false); err != nil {
		t.Fatalf("create follows collection: %v", err)
	}

	// activitypub_activities collection
	activitiesJSON := `[{
		"id": "pbc_3752774184",
		"name": "activitypub_activities",
		"type": "base",
		"system": false,
		"listRule": "",
		"viewRule": "",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"exceptDomains":[],"hidden":false,"id":"url2434853685","name":"iri","onlyDomains":[],"presentable":false,"required":true,"system":false,"type":"url"},
			{"autogeneratePattern":"","hidden":false,"id":"text2363381545","max":0,"min":0,"name":"type","pattern":"","presentable":false,"primaryKey":false,"required":true,"system":false,"type":"text"},
			{"hidden":false,"id":"json3616002756","maxSize":0,"name":"to","presentable":false,"required":false,"system":false,"type":"json"},
			{"hidden":false,"id":"json3685882489","maxSize":0,"name":"cc","presentable":false,"required":false,"system":false,"type":"json"},
			{"hidden":false,"id":"json2893285722","maxSize":0,"name":"object","presentable":false,"required":true,"system":false,"type":"json"},
			{"exceptDomains":[],"hidden":false,"id":"url1148540665","name":"actor","onlyDomains":[],"presentable":false,"required":true,"system":false,"type":"url"},
			{"hidden":false,"id":"date1748787223","max":"","min":"","name":"published","presentable":false,"required":true,"system":false,"type":"date"},
			{"autogeneratePattern":"","hidden":false,"id":"text1653163849","max":15,"min":15,"name":"relation","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [],
		"system": false
	}]`

	if err := app.ImportCollectionsByMarshaledJSON([]byte(activitiesJSON), false); err != nil {
		t.Fatalf("create activitypub_activities collection: %v", err)
	}

	t.Cleanup(func() {
		app.ResetBootstrapState()
	})

	return app
}

func addFeedCollection(t *testing.T, app core.App) {
	t.Helper()
	feedJSON := `[{
		"id": "pbc_feed_test00001",
		"name": "feed",
		"type": "base",
		"system": false,
		"listRule": null,
		"viewRule": null,
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textfeeditem001","max":0,"min":0,"name":"item","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textfeedactor01","max":0,"min":0,"name":"actor","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textfeedauthr1","max":0,"min":0,"name":"author","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"autogeneratePattern":"","hidden":false,"id":"textfeedtype001","max":0,"min":0,"name":"type","pattern":"","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": []
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(feedJSON), false); err != nil {
		t.Fatalf("create feed collection: %v", err)
	}
}

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

func createTestActor(t *testing.T, app core.App, iri, actorType string, isLocal bool) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatalf("find activitypub_actors: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("actor_type", actorType)
	r.Set("is_local", isLocal)
	r.Set("preferred_username", "testactor")
	r.Set("domain", "example.com")
	r.Set("inbox", iri+"/inbox")
	r.Set("outbox", iri+"/outbox")
	r.Set("last_fetched", time.Now())
	if err := app.Save(r); err != nil {
		t.Fatalf("save actor %s: %v", iri, err)
	}
	return r
}

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

const (
	feedDedupAuthorIRI   = "https://remote.example.com/api/v1/activitypub/user/rauthor"
	feedDedupOtherIRI    = "https://remote.example.com/api/v1/activitypub/user/xother"
	feedDedupAliceIRI    = "https://trails.example.com/api/v1/activitypub/user/alice"
	feedDedupBobIRI      = "https://trails.example.com/api/v1/activitypub/user/bob"
	feedDedupInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"
	feedDedupTrailIRI    = "https://remote.example.com/api/v1/trail/t1"
	feedDedupListIRI     = "https://remote.example.com/api/v1/list/l1"
)

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

func ownerText(s string) pub.NaturalLanguageValues {
	return pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, s))
}
