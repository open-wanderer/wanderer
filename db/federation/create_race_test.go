package federation

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

// raceCreateAddTrailsCollection adds a trails collection with the production
// unique index on iri.
func raceCreateAddTrailsCollection(t *testing.T, app core.App) {
	t.Helper()
	trailsJSON := `[{
		"id": "pbc_trails_race001",
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
		"indexes": ["CREATE UNIQUE INDEX ` + "`idx_race_trails_iri`" + ` ON ` + "`trails`" + ` (` + "`iri`" + `) WHERE iri IS NOT NULL AND iri != \"\""]
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(trailsJSON), false); err != nil {
		t.Fatalf("create trails collection: %v", err)
	}
}

// raceCreateAddListsCollection adds a lists collection with the production
// unique index on iri.
func raceCreateAddListsCollection(t *testing.T, app core.App) {
	t.Helper()
	listsJSON := `[{
		"id": "pbc_lists_race0001",
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
		"indexes": ["CREATE UNIQUE INDEX ` + "`idx_race_lists_iri`" + ` ON ` + "`lists`" + ` (` + "`iri`" + `) WHERE iri IS NOT NULL AND iri != \"\""]
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(listsJSON), false); err != nil {
		t.Fatalf("create lists collection: %v", err)
	}
}

// raceCreateBarrierServer answers only once n requests have arrived (or after
// 3 seconds), holding concurrent deliveries between their lookup and insert.
func raceCreateBarrierServer(t *testing.T, n int32) *httptest.Server {
	t.Helper()
	var arrived int32
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&arrived, 1) >= n {
			once.Do(func() { close(release) })
		}
		select {
		case <-release:
		case <-time.After(3 * time.Second):
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("data"))
	}))
	t.Cleanup(server.Close)
	return server
}

func raceCreateApp(t *testing.T, addCollection func(*testing.T, core.App)) core.App {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	app := newInboxTestApp(t)
	addFeedCollection(t, app)
	addCollection(t, app)
	// The production attachment client refuses loopback.
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(&http.Client{Timeout: 10 * time.Second}))
	return app
}

// raceCreateDeliverConcurrently releases one delivery per recipient at the
// same moment and returns the errors in recipient order.
func raceCreateDeliverConcurrently(app core.App, author *core.Record, recipients []*core.Record, act pub.Activity) []error {
	errs := make([]error, len(recipients))
	start := make(chan struct{})
	var ready, done sync.WaitGroup
	for i, r := range recipients {
		ready.Add(1)
		done.Add(1)
		go func(i int, r *core.Record) {
			defer done.Done()
			ready.Done()
			<-start
			errs[i] = ProcessCreateOrUpdateActivity(app, context.Background(), author, r, act)
		}(i, r)
	}
	ready.Wait()
	close(start)
	done.Wait()
	return errs
}

func raceCreateAssertDelivered(t *testing.T, app core.App, collection, iri string, a feedDedupActors, errs []error) {
	t.Helper()
	for i, err := range errs {
		if err != nil {
			t.Errorf("delivery %d returned an error: %v", i, err)
		}
	}
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
	if got := feedDedupCount(t, app, a.instance.Id, obj.Id); got != 0 {
		t.Errorf("feed rows for instance actor = %d, want 0", got)
	}
}

func TestCreateTrailConcurrentDeliveriesFeedEveryRecipient(t *testing.T) {
	app := raceCreateApp(t, raceCreateAddTrailsCollection)
	a := feedDedupSeedActors(t, app)
	server := raceCreateBarrierServer(t, 2)

	act := feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI)
	obj := act.Object.(*pub.Object)
	obj.Attachment = pub.ItemCollection{
		&pub.Object{
			Type:      pub.DocumentType,
			MediaType: "application/xml+gpx",
			URL:       pub.IRI(fmt.Sprintf("%s/t.gpx", strings.TrimRight(server.URL, "/"))),
		},
	}

	errs := raceCreateDeliverConcurrently(app, a.author, []*core.Record{a.instance, a.alice}, act)
	raceCreateAssertDelivered(t, app, "trails", feedDedupTrailIRI, a, errs)
}

func TestCreateListConcurrentDeliveriesFeedEveryRecipient(t *testing.T) {
	app := raceCreateApp(t, raceCreateAddListsCollection)
	a := feedDedupSeedActors(t, app)
	server := raceCreateBarrierServer(t, 2)

	act := feedDedupListCreate(feedDedupAuthorIRI, feedDedupListIRI)
	obj := act.Object.(*pub.Object)
	obj.Attachment = pub.ItemCollection{
		&pub.Object{
			Type: pub.ImageType,
			URL:  pub.IRI(fmt.Sprintf("%s/a.png", strings.TrimRight(server.URL, "/"))),
		},
	}

	errs := raceCreateDeliverConcurrently(app, a.author, []*core.Record{a.instance, a.alice}, act)
	raceCreateAssertDelivered(t, app, "lists", feedDedupListIRI, a, errs)
}
