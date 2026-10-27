package routes

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/federation"
)

// Lists of origin.example name trails of trail.example. Requests are served
// in memory; the list page is read through RemoteListGet.
type listViewFixture struct {
	t        *testing.T
	app      core.App
	lock     sync.Mutex
	requests []string
	entries  []any
	trails   map[string]map[string]any
	status   int
	delay    time.Duration
	hang     bool
	// Actor documents of trail.example and how long they take.
	actorKey    string
	actorDelay  time.Duration
	listAuthors map[string]string
}

func newListViewFixture(t *testing.T) *listViewFixture {
	t.Helper()
	f := &listViewFixture{t: t, app: newRemoteRealSchemaApp(t), trails: map[string]map[string]any{}, listAuthors: map[string]string{"origin.example": "https://origin.example/api/v1/activitypub/user/anna"}}
	f.actor("anna", "origin.example")
	f.actor("ben", "trail.example")
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: f.serve}}
	}
	return f
}

func (f *listViewFixture) actor(name, host string) *core.Record {
	f.t.Helper()
	collection, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		f.t.Fatal(err)
	}
	iri := "https://" + host + "/api/v1/activitypub/user/" + name
	record := core.NewRecord(collection)
	record.Load(map[string]any{"username": name, "preferred_username": name, "domain": host, "iri": iri, "inbox": iri + "/inbox",
		"public_key": "test-key", "is_local": false, "last_fetched": time.Now()})
	if err := f.app.Save(record); err != nil {
		f.t.Fatal(err)
	}
	return record
}

func (f *listViewFixture) serve(request *http.Request) (*http.Response, error) {
	f.lock.Lock()
	f.requests = append(f.requests, request.URL.Host+request.URL.Path)
	trail := f.trails[strings.ToLower(request.URL.Host)+request.URL.Path]
	status, delay, hang, entries := f.status, f.delay, f.hang, f.entries
	listAuthor := f.listAuthors[request.URL.Host]
	f.lock.Unlock()
	respond := func(code int, payload any) (*http.Response, error) {
		body, err := json.Marshal(payload)
		return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, err
	}
	if name, ok := strings.CutPrefix(request.URL.Path, "/api/v1/activitypub/user/"); ok && f.actorKey != "" {
		select {
		case <-time.After(f.actorDelay):
		case <-request.Context().Done():
			return nil, request.Context().Err()
		}
		iri := "https://trail.example/api/v1/activitypub/user/" + name
		return respond(http.StatusOK, map[string]any{"@context": "https://www.w3.org/ns/activitystreams",
			"id": iri, "type": "Person", "preferredUsername": name, "inbox": iri + "/inbox", "outbox": iri + "/outbox",
			"publicKey": map[string]any{"id": iri + "#main-key", "owner": iri, "publicKeyPem": f.actorKey}})
	}
	if id, ok := strings.CutPrefix(request.URL.Path, "/api/v1/list/"); ok {
		return respond(http.StatusOK, map[string]any{"id": id, "iri": request.URL.Scheme + "://" + request.URL.Host + request.URL.Path, "name": "List", "public": true,
			"expand": map[string]any{"author": map[string]any{"iri": listAuthor}, "trails": entries}})
	}
	if hang {
		<-request.Context().Done()
		return nil, request.Context().Err()
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-request.Context().Done():
			return nil, request.Context().Err()
		}
	}
	if status != 0 {
		return respond(status, map[string]any{"message": "status"})
	}
	if trail == nil {
		return respond(http.StatusNotFound, map[string]any{"message": "not found"})
	}
	result := map[string]any{}
	for _, field := range strings.Split(request.URL.Query().Get("fields"), ",") {
		path := strings.Split(field, ".")
		if value, ok := trail[path[0]]; ok && len(path) == 1 {
			result[path[0]] = value
		}
	}
	expand, _ := trail["expand"].(map[string]any)
	result["expand"] = expand
	return respond(http.StatusOK, result)
}

// trail publishes a trail on trail.example and returns how the list names it.
func (f *listViewFixture) trail(id, name string, tags ...string) map[string]any {
	expandedTags := []any{}
	for _, tag := range tags {
		expandedTags = append(expandedTags, map[string]any{"name": tag})
	}
	iri := "https://trail.example/api/v1/trail/" + id
	f.lock.Lock()
	f.trails["trail.example/api/v1/trail/"+id] = map[string]any{"id": id, "iri": iri, "name": name, "public": true, "distance": 1000.0,
		"polyline": "origin-line", "like_count": 3.0, "completed": true,
		"expand": map[string]any{"author": map[string]any{"iri": "https://trail.example/api/v1/activitypub/user/ben"}, "tags": expandedTags}}
	f.lock.Unlock()
	return map[string]any{"id": "copy-" + id, "iri": iri, "name": "Listed copy", "public": true, "distance": 99000.0, "polyline": "listed-line",
		"like_count": 9.0, "expand": map[string]any{"author": map[string]any{"iri": "https://trail.example/api/v1/activitypub/user/ben"}, "tags": []any{map[string]any{"name": "Listed tag"}}}}
}

func (f *listViewFixture) list(entries ...any) {
	f.lock.Lock()
	f.entries = entries
	f.lock.Unlock()
}

// view reads the list page and waits for a sync the view started.
func (f *listViewFixture) view() []*core.Record {
	f.t.Helper()
	e := &core.RequestEvent{App: f.app}
	query := url.Values{"handle": {"@anna@origin.example"}, "expand": {"trails,trails.tags"}}
	e.Request = httptest.NewRequest(http.MethodGet, "/api/v1/list/list-id?"+query.Encode(), nil)
	e.Request.RemoteAddr = "203.0.113.7:1234"
	e.Request.SetPathValue("id", "list-id")
	e.Response = httptest.NewRecorder()
	if err := RemoteListGet(e); err != nil {
		f.t.Fatal(err)
	}
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		if _, running := listSyncing.Load("https://origin.example/api/v1/list/list-id"); !running {
			break
		}
		if time.Now().After(deadline) {
			f.t.Fatal("list sync did not finish")
		}
	}
	var body struct {
		Trails []string `json:"trails"`
	}
	if err := json.Unmarshal(e.Response.(*httptest.ResponseRecorder).Body.Bytes(), &body); err != nil {
		f.t.Fatal(err)
	}
	trails := []*core.Record{}
	for _, id := range body.Trails {
		trail, err := f.app.FindRecordById("trails", id)
		if err != nil {
			f.t.Fatal(err)
		}
		trails = append(trails, trail)
	}
	return trails
}

func (f *listViewFixture) origin() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	count := 0
	for _, request := range f.requests {
		if strings.HasPrefix(strings.ToLower(request), "trail.example/api/v1/trail/") {
			count++
		}
	}
	f.requests = nil
	return count
}

func (f *listViewFixture) age() {
	f.t.Helper()
	past := time.Now().UTC().Add(-2 * remoteSyncThreshold).Format("2006-01-02 15:04:05.000Z")
	for _, table := range []string{"lists", "trails"} {
		if _, err := f.app.DB().NewQuery("UPDATE " + table + " SET updated = {:past}").Bind(map[string]any{"past": past}).Execute(); err != nil {
			f.t.Fatal(err)
		}
	}
}

func listTrailNames(trails []*core.Record) []string {
	names := []string{}
	for _, trail := range trails {
		names = append(names, trail.GetString("name"))
	}
	return names
}

func TestRemoteListReferencesShowTheirOriginsTagsAndLine(t *testing.T) {
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin", "Wald", "Aussicht"))
	trails := f.view()
	if len(trails) != 1 {
		t.Fatalf("list trails = %v", listTrailNames(trails))
	}
	trail := trails[0]
	if errs := f.app.ExpandRecord(trail, []string{"tags"}, nil); len(errs) > 0 {
		t.Fatal(errs)
	}
	tags := []string{}
	for _, tag := range trail.ExpandedAll("tags") {
		tags = append(tags, tag.GetString("name"))
	}
	slices.Sort(tags)
	if trail.GetString("name") != "Origin" || !slices.Equal(tags, []string{"aussicht", "wald"}) && !slices.Equal(tags, []string{"Aussicht", "Wald"}) ||
		trail.GetString("polyline") == "listed-line" || trail.GetFloat("like_count") != 3 || !trail.GetBool("completed") || trail.GetFloat("distance") != 1000 {
		t.Fatalf("placeholder not taken from its origin: name=%q tags=%v polyline=%q likes=%v completed=%v distance=%v",
			trail.GetString("name"), tags, trail.GetString("polyline"), trail.GetFloat("like_count"), trail.GetBool("completed"), trail.GetFloat("distance"))
	}
	// Like a copy created by a list sync, the line follows with the next refresh.
	f.age()
	f.view()
	stored, err := f.app.FindRecordById("trails", trail.Id)
	if err != nil || stored.GetString("polyline") != "origin-line" {
		t.Fatalf("line of the origin missing after the refresh: %v", err)
	}
}

func TestRemoteListLoadsLongListsWithRegularRefreshes(t *testing.T) {
	f := newListViewFixture(t)
	entries := []any{}
	for i := range 3 {
		entries = append(entries, map[string]any{"id": fmt.Sprintf("own-%d", i), "iri": fmt.Sprintf("https://origin.example/api/v1/trail/own-%d", i), "name": "Own", "public": true,
			"expand": map[string]any{"author": map[string]any{"iri": "https://origin.example/api/v1/activitypub/user/anna"}}})
	}
	for i := range remoteListReferenceLimit + 5 {
		entries = append(entries, f.trail(fmt.Sprintf("trail-%02d", i), fmt.Sprintf("Origin %02d", i)))
	}
	f.list(entries...)
	if trails := f.view(); len(trails) != 33 || f.origin() != 30 {
		t.Fatalf("first view shows %d trails, want 33", len(trails))
	}
	if trails := f.view(); len(trails) != 33 || f.origin() != 0 {
		t.Fatalf("fresh list requested more references: trails=%d", len(trails))
	}
	if remoteListReferencesDue(f.app, "https://origin.example/api/v1/list/list-id") {
		t.Fatal("long list has a retry before its regular refresh")
	}
	f.age()
	f.view()
	if requests := f.origin(); requests > 30 || requests < 5 {
		t.Fatalf("regular refresh requested %d references", requests)
	}
	if trails := f.view(); len(trails) != 38 || f.origin() != 0 {
		t.Fatalf("list shows %d trails after the refresh, want 38", len(trails))
	}
}

func TestRemoteListAsksAgainAfterAnOriginRecovers(t *testing.T) {
	previous := remoteListReferenceRetries
	remoteListReferenceRetries = []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}
	t.Cleanup(func() { remoteListReferenceRetries = previous })
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin"))
	f.status = http.StatusServiceUnavailable
	if trails := f.view(); len(trails) != 0 {
		t.Fatalf("trail shown during the outage: %v", listTrailNames(trails))
	}
	f.lock.Lock()
	f.status = 0
	f.lock.Unlock()
	if trails := f.view(); len(trails) != 0 || f.origin() != 1 {
		t.Fatal("origin asked again before the retry delay")
	}
	time.Sleep(60 * time.Millisecond)
	f.view()
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Origin"}) {
		t.Fatalf("trail not loaded after the origin recovered: %v", listTrailNames(trails))
	}
}

func TestRemoteListFirstViewWaitsBrieflyForOtherOrigins(t *testing.T) {
	previousRetries := remoteListReferenceRetries
	remoteListReferenceRetries = []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}
	t.Cleanup(func() { remoteListReferenceRetries = previousRetries })
	previous := remoteListReferenceFirstView
	remoteListReferenceFirstView.objects, remoteListReferenceFirstView.authors = 100*time.Millisecond, 50*time.Millisecond
	t.Cleanup(func() { remoteListReferenceFirstView = previous })
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin"))
	f.delay = 400 * time.Millisecond
	started := time.Now()
	if trails := f.view(); len(trails) != 0 || time.Since(started) > 300*time.Millisecond {
		t.Fatalf("first view waited %v and shows %v", time.Since(started), listTrailNames(trails))
	}
	time.Sleep(60 * time.Millisecond)
	f.view()
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Origin"}) {
		t.Fatalf("slow origin not loaded by the next view: %v", listTrailNames(trails))
	}
}

func TestRemoteListRequiresTheExactReferenceIRI(t *testing.T) {
	for _, requested := range []string{"https://TRAIL.example/api/v1/trail/trail-one", "http://trail.example/api/v1/trail/trail-one"} {
		t.Run(requested, func(t *testing.T) {
			f := newListViewFixture(t)
			entry := f.trail("trail-one", "Origin")
			ben, _ := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://trail.example/api/v1/activitypub/user/ben")
			collection, _ := f.app.FindCollectionByNameOrId("trails")
			copy := core.NewRecord(collection)
			copy.Load(map[string]any{"name": "Stored", "iri": "https://trail.example/api/v1/trail/trail-one", "author": ben.Id, "public": true, "full_sync_completed": true})
			if err := f.app.Save(copy); err != nil {
				t.Fatal(err)
			}
			entry["iri"] = requested
			f.list(entry)
			if trails := f.view(); len(trails) != 0 || f.origin() != 1 {
				t.Fatalf("different reference accepted: %v", listTrailNames(trails))
			}
			count, err := f.app.CountRecords("trails")
			if err != nil || count != 1 {
				t.Fatalf("records=%d err=%v", count, err)
			}
			if !remoteListReferenceFailed(f.app, requested) {
				t.Fatal("different answer was not remembered for the requested IRI")
			}
		})
	}
}

func TestRemoteListRefreshesUnopenedPlaceholders(t *testing.T) {
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin"))
	f.view()
	f.lock.Lock()
	f.trails["trail.example/api/v1/trail/trail-one"]["name"] = "Renamed"
	f.lock.Unlock()
	f.age()
	f.view()
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Renamed"}) {
		t.Fatalf("placeholder not refreshed: %v", listTrailNames(trails))
	}
	f.lock.Lock()
	f.status = http.StatusForbidden
	f.lock.Unlock()
	f.age()
	f.view()
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Renamed"}) {
		t.Fatalf("refusal changed the placeholder: %v", listTrailNames(trails))
	}
}

func TestRemoteListAsksAgainWithoutOptionalFieldsWhenTooLarge(t *testing.T) {
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin"))
	f.trails["trail.example/api/v1/trail/trail-one"]["description"] = strings.Repeat("x", int(canonicalRemoteBodyLimit))
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Origin"}) || trails[0].GetString("description") != "" {
		t.Fatalf("large trail not loaded without optional fields: %v", listTrailNames(trails))
	}
	if requests := f.origin(); requests != 2 {
		t.Fatalf("origin requests = %d, want 2", requests)
	}
}

func TestRemoteListStopsAskingAnOriginThatDoesNotAnswer(t *testing.T) {
	previous, previousTimeout := remoteListReferenceFirstView, canonicalRemoteFetchTimeout
	remoteListReferenceFirstView.objects, remoteListReferenceFirstView.authors, canonicalRemoteFetchTimeout = 300*time.Millisecond, 50*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { remoteListReferenceFirstView, canonicalRemoteFetchTimeout = previous, previousTimeout })
	f := newListViewFixture(t)
	entries := []any{}
	for i := range remoteListReferenceLimit {
		entries = append(entries, f.trail(fmt.Sprintf("trail-%02d", i), "Origin"))
	}
	f.list(entries...)
	f.hang = true
	f.view()
	if requests := f.origin(); requests > canonicalRemoteFetchParallelism {
		t.Fatalf("silent origin asked %d times in one sync", requests)
	}
}

func TestRemoteListDoesNotAskAgainForAnUnstorableReference(t *testing.T) {
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin"), f.trail("trail-two", "Other"))
	f.trails["trail.example/api/v1/trail/trail-one"]["difficulty"] = "unknown-level"
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Other"}) {
		t.Fatalf("list trails = %v", listTrailNames(trails))
	}
	if !remoteListReferenceFailed(f.app, "https://trail.example/api/v1/trail/trail-one") {
		t.Fatal("unstorable reference will be requested again with the next sync")
	}
}

func TestRemoteListFirstViewLeavesSlowAuthorsToTheRetry(t *testing.T) {
	previousRetries := remoteListReferenceRetries
	remoteListReferenceRetries = []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}
	t.Cleanup(func() { remoteListReferenceRetries = previousRetries })
	previous := remoteListReferenceFirstView
	remoteListReferenceFirstView.objects, remoteListReferenceFirstView.authors = time.Second, 50*time.Millisecond
	t.Cleanup(func() { remoteListReferenceFirstView = previous })
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	f := newListViewFixture(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	f.actorKey = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	f.actorDelay = 200 * time.Millisecond
	t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: f.serve}}
	}))
	ben, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://trail.example/api/v1/activitypub/user/ben")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.app.Delete(ben); err != nil {
		t.Fatal(err)
	}
	f.list(f.trail("trail-one", "Origin"))
	if trails := f.view(); len(trails) != 0 {
		t.Fatalf("first view confirmed a slow author within its budget: %v", listTrailNames(trails))
	}
	time.Sleep(60 * time.Millisecond)
	f.view()
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Origin"}) {
		t.Fatalf("slow author not confirmed by the next view: %v", listTrailNames(trails))
	}
}

func TestRemoteListRetriesAnOverloadedOriginLater(t *testing.T) {
	f := newListViewFixture(t)
	entries := []any{}
	for i := range remoteListReferenceLimit {
		entries = append(entries, f.trail(fmt.Sprintf("trail-%02d", i), "Origin"))
	}
	f.list(entries...)
	f.status = http.StatusServiceUnavailable
	f.view()
	if requests := f.origin(); requests > canonicalRemoteFetchParallelism {
		t.Fatalf("overloaded origin asked %d times in one sync", requests)
	}
	// The skipped references wait for the retry delay, not for the next view.
	f.view()
	if requests := f.origin(); requests != 0 {
		t.Fatalf("overloaded origin asked again %d times with the next view", requests)
	}
}

func TestRemoteListReferenceRefusalsStayWithTheRequestedIRI(t *testing.T) {
	for _, mode := range []string{"status", "syntax", "type", "object", "public"} {
		t.Run(mode, func(t *testing.T) {
			f := newListViewFixture(t)
			iri := "http://trail.example/api/v1/trail/trail-one"
			requested := 0
			previous := newRemoteSyncHTTPClient
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(r *http.Request) (*http.Response, error) {
					if r.URL.Scheme == "http" && r.URL.Host == "trail.example" {
						requested++
						code, body := 200, `{"public":true}`
						switch mode {
						case "status":
							code = 404
						case "syntax":
							body = "<html>page</html>"
						case "type":
							body = `[]`
						case "object":
							body = `{"iri":"https://trail.example/api/v1/trail/other","public":true}`
						case "public":
							body = `{"iri":"http://trail.example/api/v1/trail/trail-one","public":false}`
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					}
					return f.serve(r)
				}}}
			}
			defer func() { newRemoteSyncHTTPClient = previous }()
			f.list(map[string]any{"iri": iri})
			if trails := f.view(); len(trails) != 0 || requested != 1 {
				t.Fatalf("refused trail shown: trails=%d requests=%d", len(trails), requested)
			}
			f.age()
			f.view()
			f.view()
			if requested != 1 || !remoteListReferenceFailed(f.app, iri) {
				t.Fatalf("refusal not remembered: requests=%d", requested)
			}
			f.list(f.trail("trail-one", "Origin"))
			f.age()
			f.view()
			if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Origin"}) {
				t.Fatalf("http refusal affected https: %v", listTrailNames(trails))
			}
		})
	}
}

func TestRemoteListRefusalsDoNotBlockAnotherList(t *testing.T) {
	f := newListReferenceFixture(t)
	carol := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	for i := range remoteListReferenceLimit {
		f.entries = append(f.entries, map[string]any{"iri": fmt.Sprintf("https://third.example/api/v1/trail/missing-%d", i)})
	}
	f.sync(t)
	otherAuthor := f.save(f.actors, map[string]any{"iri": "https://other.example/api/v1/actor/riley", "last_fetched": time.Now()})
	f.authors["other.example"] = otherAuthor.GetString("iri")
	other := f.save(f.lists, map[string]any{"iri": "https://other.example/api/v1/list/list-id", "author": otherAuthor.Id, "needs_full_sync": true})
	f.list = other
	iri := f.third("real", "Origin", 1000, carol.GetString("iri"))
	f.entries = []any{map[string]any{"iri": iri}}
	requests := f.sync(t)
	got := 0
	for _, r := range requests {
		if strings.HasPrefix(r, "third.example/") {
			got++
		}
	}
	if got != 1 || len(f.list.GetStringSlice("trails")) != 1 {
		t.Fatalf("another list blocked: requests=%v trails=%v", requests, f.list.GetStringSlice("trails"))
	}
}

func TestRemoteListRetryStopsWhenItsListOriginIsUnavailable(t *testing.T) {
	previousRetries := remoteListReferenceRetries
	remoteListReferenceRetries = []time.Duration{time.Minute, 4 * time.Minute}
	defer func() { remoteListReferenceRetries = previousRetries }()
	f := newListViewFixture(t)
	f.list(f.trail("trail-one", "Origin"))
	f.status = http.StatusServiceUnavailable
	f.view()
	key := f.app.DataDir() + "\x00https://origin.example/api/v1/list/list-id"
	previous := newRemoteSyncHTTPClient
	defer func() { newRemoteSyncHTTPClient = previous }()
	var lock sync.Mutex
	listRequests := 0
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(r *http.Request) (*http.Response, error) {
			if r.URL.Host == "origin.example" && r.URL.Path == "/api/v1/list/list-id" {
				lock.Lock()
				listRequests++
				lock.Unlock()
				return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
			}
			return f.serve(r)
		}}}
	}
	for step := range 3 {
		remoteListReferences.Lock()
		entry, ok := remoteListReferences.continuations[key]
		if step < 2 && !ok {
			remoteListReferences.Unlock()
			t.Fatalf("retry %d missing", step)
		}
		if ok {
			entry.due = time.Now().Add(-time.Second)
			remoteListReferences.continuations[key] = entry
		}
		remoteListReferences.Unlock()
		f.view()
		f.view()
		lock.Lock()
		count := listRequests
		lock.Unlock()
		if want := min(step+1, 2); count != want {
			t.Fatalf("step %d list requests=%d want=%d", step, count, want)
		}
	}
}

func TestRemoteListReferenceCacheIsSeparateForEachDataDirectory(t *testing.T) {
	f := newListViewFixture(t)
	iri := "https://trail.example/api/v1/trail/trail-one"
	rememberRemoteListReferenceFailure(f.app, iri)
	other := newRemoteRealSchemaApp(t)
	if remoteListReferenceFailed(other, iri) {
		t.Fatal("refusal shared between data directories")
	}
}

func TestRemoteListCancelledReferenceIsNotRemembered(t *testing.T) {
	f := newListViewFixture(t)
	iri := "https://trail.example/api/v1/trail/trail-one"
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	client := &http.Client{Transport: remoteSyncTransport{roundTrip: func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() }}}
	result := fetchRemoteListReference(f.app, ctx, client, iri)
	if result.answer != remoteListReferenceUnavailable || remoteListReferenceFailed(f.app, iri) {
		t.Fatalf("cancelled lookup = %+v", result)
	}
}

func TestRemoteListConcurrentReferencesConfirmAnUnknownAuthor(t *testing.T) {
	for _, sameList := range []bool{true, false} {
		t.Run(fmt.Sprintf("same-list-%v", sameList), func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
			f := newListViewFixture(t)
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
			f.actorKey = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
			f.delay = 100 * time.Millisecond
			f.actorDelay = 50 * time.Millisecond
			t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client { return &http.Client{Transport: remoteSyncTransport{roundTrip: f.serve}} }))
			ben, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://trail.example/api/v1/activitypub/user/ben")
			if err != nil {
				t.Fatal(err)
			}
			if err := f.app.Delete(ben); err != nil {
				t.Fatal(err)
			}
			f.list(f.trail("trail-one", "Origin", "Forest"))
			anna, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://origin.example/api/v1/activitypub/user/anna")
			if err != nil {
				t.Fatal(err)
			}
			col, err := f.app.FindCollectionByNameOrId("lists")
			if err != nil {
				t.Fatal(err)
			}
			otherAuthor := anna
			if !sameList {
				otherAuthor = f.actor("zed", "other.example")
				f.listAuthors["other.example"] = otherAuthor.GetString("iri")
			}
			ids := []string{}
			for index := range 2 {
				record := core.NewRecord(col)
				if index == 1 && sameList {
					ids = append(ids, ids[0])
					continue
				}
				origin, author := "origin.example", anna
				if index == 1 {
					origin, author = "other.example", otherAuthor
				}
				record.Load(map[string]any{"iri": fmt.Sprintf("https://%s/api/v1/list/list-%d", origin, index), "author": author.Id, "needs_full_sync": true, "name": "List"})
				if err := f.app.Save(record); err != nil {
					t.Fatal(err)
				}
				ids = append(ids, record.Id)
			}
			// Concurrent calls retain independent record snapshots.
			records := []*core.Record{}
			for _, id := range ids {
				record, err := f.app.FindRecordById("lists", id)
				if err != nil {
					t.Fatal(err)
				}
				records = append(records, record)
			}
			start := make(chan struct{})
			errs := make(chan error, 2)
			for _, record := range records {
				go func(list *core.Record) {
					<-start
					_, err := performFullListSync(f.app, t.Context(), &url.URL{}, list)
					errs <- err
				}(record)
			}
			close(start)
			for range 2 {
				if err := <-errs; err != nil {
					t.Error(err)
				}
			}
			for _, id := range ids {
				stored, err := f.app.FindRecordById("lists", id)
				if err != nil {
					t.Fatal(err)
				}
				if len(stored.GetStringSlice("trails")) != 1 {
					t.Fatalf("concurrent list lost reference: %v", stored)
				}
			}
			count, err := f.app.CountRecords("trails")
			if err != nil || count != 1 {
				t.Fatalf("trails=%d err=%v", count, err)
			}
			count, err = f.app.CountRecords("activitypub_actors", dbx.HashExp{"iri": "https://trail.example/api/v1/activitypub/user/ben"})
			if err != nil || count != 1 {
				t.Fatalf("authors=%d err=%v", count, err)
			}
			if requests := f.origin(); requests < 1 || requests > 2 {
				t.Fatalf("reference requests=%d", requests)
			}
		})
	}
}

type remoteListObservedContext struct {
	context.Context
	once   sync.Once
	joined chan struct{}
}

func (ctx *remoteListObservedContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.joined) })
	return ctx.Context.Done()
}

func TestRemoteListWaiterUsesTheCancelledStartersResult(t *testing.T) {
	f := newListViewFixture(t)
	entry := f.trail("trail-one", "Origin")
	f.list(entry)
	iri, _ := entry["iri"].(string)
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	client := &http.Client{Transport: remoteSyncTransport{roundTrip: func(r *http.Request) (*http.Response, error) {
		once.Do(func() { close(started) })
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-release:
			return f.serve(r)
		}
	}}}
	creatorCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	waiterCtx := &remoteListObservedContext{Context: t.Context(), joined: make(chan struct{})}
	creator, waiter := make(chan remoteListReferenceResult, 1), make(chan remoteListReferenceResult, 1)
	go func() { creator <- fetchRemoteListReference(f.app, creatorCtx, client, iri) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("origin lookup did not start")
	}
	go func() { waiter <- fetchRemoteListReference(f.app, waiterCtx, client, iri) }()
	select {
	case <-waiterCtx.joined:
	case <-time.After(5 * time.Second):
		t.Fatal("waiter did not join")
	}
	cancel()
	for _, done := range []chan remoteListReferenceResult{creator, waiter} {
		select {
		case result := <-done:
			if result.answer != remoteListReferenceUnavailable {
				t.Fatalf("cancelled shared result=%+v", result)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cancelled lookup did not finish")
		}
	}
	close(release)
	if remoteListReferenceFailed(f.app, iri) {
		t.Fatal("cancelled lookup was remembered as a refusal")
	}
	if trails := f.view(); !slices.Equal(listTrailNames(trails), []string{"Origin"}) {
		t.Fatalf("reference not loaded on retry: %v", listTrailNames(trails))
	}
}

func TestRemoteListReferenceRefreshesRotateWithinTheLimit(t *testing.T) {
	f := newListViewFixture(t)
	ben, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://trail.example/api/v1/activitypub/user/ben")
	if err != nil {
		t.Fatal(err)
	}
	collection, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	entries := []any{}
	for index := range 35 {
		entry := f.trail(fmt.Sprintf("trail-%02d", index), "Origin")
		record := core.NewRecord(collection)
		record.Load(map[string]any{"name": "Stored", "iri": entry["iri"], "author": ben.Id, "public": true, "needs_full_sync": true})
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	f.list(entries...)
	f.age()
	f.view()
	if requests := f.origin(); requests != 30 {
		t.Fatalf("first refresh requests=%d", requests)
	}
	f.age()
	f.view()
	if requests := f.origin(); requests != 30 {
		t.Fatalf("second refresh requests=%d", requests)
	}
	for _, trail := range f.view() {
		if trail.GetString("name") != "Origin" {
			t.Fatalf("placeholder was not reached by rotation: %s", trail.GetString("iri"))
		}
	}
}

func TestRemoteListReferenceAuthorMustRemainRemoteUntilTheWrite(t *testing.T) {
	f := newListViewFixture(t)
	ben, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://trail.example/api/v1/activitypub/user/ben")
	if err != nil {
		t.Fatal(err)
	}
	entry := f.trail("trail-one", "Origin")
	iri := entry["iri"].(string)
	listCollection, err := f.app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	list := core.NewRecord(listCollection)
	list.Set("iri", "https://origin.example/api/v1/list/list-id")
	ctx := prepareRemoteListReferences(f.app, t.Context(), list, map[string]any{"expand": map[string]any{"trails": []any{entry}}}, "https://origin.example")
	if len(remoteListReferenceStateOf(ctx).confirmed) != 1 {
		t.Fatal("origin reference was not prepared")
	}
	ben.Set("is_local", true)
	if err := f.app.Save(ben); err != nil {
		t.Fatal(err)
	}
	if err := f.app.RunInTransaction(func(tx core.App) error {
		col, err := tx.FindCollectionByNameOrId("trails")
		if err != nil {
			return err
		}
		trail, err := remoteListReferenceRecord(tx, ctx, col, iri, nil)
		if trail != nil {
			t.Error("reference stored after author changed")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	count, err := f.app.CountRecords("trails")
	if err != nil || count != 0 {
		t.Fatalf("trail records=%d err=%v", count, err)
	}
}

func TestRemoteListThirtyReferencesRetainTheirOriginFields(t *testing.T) {
	f := newListViewFixture(t)
	tags := []string{"Forest", "Ridge", "Lake", "Field", "Valley", "River", "Rock", "Hill", "Meadow", "View"}
	entries := []any{}
	for index := range 30 {
		entries = append(entries, f.trail(fmt.Sprintf("trail-%02d", index), "Origin", tags...))
	}
	f.list(entries...)
	trails := f.view()
	if len(trails) != 30 || f.origin() != 30 {
		t.Fatalf("first view trails=%d", len(trails))
	}
	f.age()
	f.view()
	for _, trail := range f.view() {
		if errs := f.app.ExpandRecord(trail, []string{"tags"}, nil); len(errs) != 0 {
			t.Fatal(errs)
		}
		if len(trail.GetStringSlice("tags")) != 10 || trail.GetString("polyline") != "origin-line" || trail.GetFloat("like_count") != 3 || !trail.GetBool("completed") || trail.GetFloat("distance") != 1000 {
			t.Fatalf("origin fields missing: iri=%s tags=%d line=%q likes=%v completed=%v distance=%v", trail.GetString("iri"), len(trail.GetStringSlice("tags")), trail.GetString("polyline"), trail.GetFloat("like_count"), trail.GetBool("completed"), trail.GetFloat("distance"))
		}
	}
}

func TestRemoteListRemembersUnstorablePlaceholderRefreshes(t *testing.T) {
	f := newListViewFixture(t)
	entry := f.trail("trail-one", "Origin", "Forest")
	iri := entry["iri"].(string)
	f.list(entry)
	trails := f.view()
	if len(trails) != 1 {
		t.Fatalf("initial trails=%d", len(trails))
	}
	previous := trails[0]
	previousDifficulty := previous.GetString("difficulty")
	if requests := f.origin(); requests != 1 {
		t.Fatalf("initial requests=%d", requests)
	}
	f.lock.Lock()
	raw := f.trails["trail.example/api/v1/trail/trail-one"]
	raw["difficulty"] = "unknown-level"
	raw["name"] = "Not stored"
	raw["distance"] = 9000.0
	f.lock.Unlock()
	f.age()
	f.view()
	if requests := f.origin(); requests != 1 {
		t.Fatalf("refresh requests=%d", requests)
	}
	if !remoteListReferenceFailed(f.app, iri) {
		t.Fatal("unstorable placeholder refresh was not remembered")
	}
	stored, err := f.app.FindRecordById("trails", previous.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("name") != "Origin" || stored.GetString("difficulty") != previousDifficulty || stored.GetFloat("distance") != 1000 || !stored.GetBool("public") || !slices.Equal(stored.GetStringSlice("tags"), previous.GetStringSlice("tags")) {
		t.Fatalf("unstorable answer changed the placeholder: %v", stored)
	}
	owner := f.actor("zed", "other.example")
	f.listAuthors["other.example"] = owner.GetString("iri")
	collection, err := f.app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	other := core.NewRecord(collection)
	other.Load(map[string]any{"name": "Other list", "iri": "https://other.example/api/v1/list/list-two", "author": owner.Id, "needs_full_sync": true})
	if err := f.app.Save(other); err != nil {
		t.Fatal(err)
	}
	result, err := performFullListSync(f.app, t.Context(), &url.URL{}, other)
	if err != nil {
		t.Fatal(err)
	}
	if requests := f.origin(); requests != 0 || len(result.GetStringSlice("trails")) != 1 || result.GetStringSlice("trails")[0] != previous.Id {
		t.Fatalf("another list repeated the unstorable refresh: requests=%d trails=%v", requests, result.GetStringSlice("trails"))
	}
}
