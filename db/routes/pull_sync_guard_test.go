package routes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Pull-sync must only store a trail origin's own content for that trail.

const (
	pullOrigin = "https://remote.example"
	pullThird  = "https://third.example"
)

// pullOriginTransport plays the origin: it answers by URL path with canned
// JSON and 404 for everything else.
type pullOriginTransport struct {
	mu    sync.Mutex
	paths map[string]string
	seen  []string
}

func (t *pullOriginTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.seen = append(t.seen, r.URL.String())
	body, ok := t.paths[r.URL.Path]
	t.mu.Unlock()

	status := http.StatusOK
	if !ok {
		status = http.StatusNotFound
		body = `{"status":404}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

type pullFixture struct {
	app       *pbtests.TestApp
	origin    *pullOriginTransport
	lou       *core.Record
	alice     *core.Record
	mallory   *core.Record
	carol     *core.Record
	localT    *core.Record
	remoteR   *core.Record
	remoteR2  *core.Record
	comments  *core.Collection
	summits   *core.Collection
	waypoints *core.Collection
}

func setupPullFixture(t *testing.T) *pullFixture {
	t.Helper()

	t.Setenv("ORIGIN", "https://local.example")

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	save := func(model core.Model) {
		t.Helper()
		if err := app.Save(model); err != nil {
			t.Fatal(err)
		}
	}

	f := &pullFixture{app: app, origin: &pullOriginTransport{paths: map[string]string{}}}

	original := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: f.origin}
	}
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "is_local"},
		&core.DateField{Name: "last_fetched"},
	)
	actors.ViewRule = types.Pointer("")
	actors.ListRule = actors.ViewRule
	save(actors)

	newActor := func(name, domain, iri string, local bool) *core.Record {
		r := core.NewRecord(actors)
		r.Set("preferred_username", name)
		r.Set("domain", domain)
		r.Set("iri", iri)
		r.Set("is_local", local)
		r.Set("last_fetched", types.NowDateTime())
		save(r)
		return r
	}
	f.lou = newActor("lou", "local.example", "https://local.example/api/v1/activitypub/user/lou", true)
	f.alice = newActor("alice", "remote.example", pullOrigin+"/actor/alice", false)
	f.mallory = newActor("mallory", "remote.example", pullOrigin+"/actor/mallory", false)
	f.carol = newActor("carol", "third.example", pullThird+"/actor/carol", false)

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.BoolField{Name: "public"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
		&core.DateField{Name: "updated"},
	)
	trails.ViewRule = types.Pointer("")
	trails.ListRule = trails.ViewRule
	save(trails)

	child := func(name string, extra ...core.Field) *core.Collection {
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
			&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
			&core.TextField{Name: "iri"},
		)
		c.Fields.Add(extra...)
		c.ViewRule = types.Pointer("")
		c.ListRule = c.ViewRule
		save(c)
		return c
	}
	f.comments = child("comments", &core.TextField{Name: "text"})
	f.summits = child("summit_logs", &core.TextField{Name: "text"})
	f.waypoints = child("waypoints", &core.TextField{Name: "name"})

	newTrail := func(author *core.Record, host string) *core.Record {
		r := core.NewRecord(trails)
		r.Set("author", author.Id)
		r.Set("public", true)
		r.Set("name", "trail")
		r.Set("full_sync_completed", true)
		save(r)
		r.Set("iri", host+"/api/v1/trail/"+r.Id)
		save(r)
		return r
	}
	f.localT = newTrail(f.lou, "https://local.example")
	f.remoteR = newTrail(f.alice, pullOrigin)
	f.remoteR2 = newTrail(f.alice, pullOrigin)
	return f
}

// row stores a child row of one of the three collections under the given IRI
// host and returns it.
func (f *pullFixture) row(t *testing.T, c *core.Collection, author, trail *core.Record, iri, text string) *core.Record {
	t.Helper()
	r := core.NewRecord(c)
	r.Set("author", author.Id)
	r.Set("trail", trail.Id)
	r.Set("iri", iri)
	r.Set(textField(c), text)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func textField(c *core.Collection) string {
	if c.Name == "waypoints" {
		return "name"
	}
	return "text"
}

// assertUntouched fails when the stored row no longer has the given text,
// author and trail.
func (f *pullFixture) assertUntouched(t *testing.T, label string, r *core.Record, wantText string) {
	t.Helper()
	got, err := f.app.FindRecordById(r.Collection().Name, r.Id)
	if err != nil {
		t.Errorf("%s: row vanished: %v", label, err)
		return
	}
	field := textField(r.Collection())
	if got.GetString(field) != wantText {
		t.Errorf("%s: %s = %q, want %q", label, field, got.GetString(field), wantText)
	}
	if got.GetString("author") != r.GetString("author") {
		t.Errorf("%s: author changed from %s to %s", label, r.GetString("author"), got.GetString("author"))
	}
	if got.GetString("trail") != r.GetString("trail") {
		t.Errorf("%s: trail changed from %s to %s", label, r.GetString("trail"), got.GetString("trail"))
	}
	if got.GetString("iri") != r.GetString("iri") {
		t.Errorf("%s: iri changed from %s to %s", label, r.GetString("iri"), got.GetString("iri"))
	}
}

func (f *pullFixture) mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func authorExpand(actor *core.Record) map[string]any {
	return map[string]any{"author": map[string]any{"iri": actor.GetString("iri")}}
}

// pullComments drives RemoteTrailCommentsList for the trail with the origin
// answering the comment list with items.
func (f *pullFixture) pullComments(t *testing.T, trail *core.Record, items []map[string]any) {
	t.Helper()
	f.origin.paths["/api/v1/comment"] = f.mustJSON(map[string]any{"items": items})

	request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments?sort=id", nil)
	request.SetPathValue("id", trail.Id)
	response := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.App, e.Request, e.Response = f.app, request, response
	if err := RemoteTrailCommentsList(e); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", response.Code, response.Body.String())
	}
}

// pullTrail drives RemoteTrailGet for the remote trail, flagged for a full
// sync, with the origin answering the trail document with doc.
func (f *pullFixture) pullTrail(t *testing.T, trail *core.Record, doc map[string]any) {
	t.Helper()
	trail.Set("needs_full_sync", true)
	if err := f.app.Save(trail); err != nil {
		t.Fatal(err)
	}
	doc["id"] = trail.Id
	f.origin.paths["/api/v1/trail/"+trail.Id] = f.mustJSON(doc)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/trail/"+trail.Id+"?handle=alice@remote.example", nil)
	request.SetPathValue("id", trail.Id)
	response := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.App, e.Request, e.Response = f.app, request, response
	if err := RemoteTrailGet(e); err != nil {
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatal(err)
		}
		t.Fatalf("RemoteTrailGet failed with %d: %v", apiErr.Status, err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", response.Code, response.Body.String())
	}
}

func TestPullSyncCommentsRefuseLocalAndForeignObjects(t *testing.T) {
	f := setupPullFixture(t)

	a := f.row(t, f.comments, f.lou, f.localT, "https://local.example/api/v1/comment/a", "a original")
	b := f.row(t, f.comments, f.lou, f.remoteR, "https://local.example/api/v1/comment/b", "b original")
	c := f.row(t, f.comments, f.alice, f.remoteR2, pullOrigin+"/api/v1/comment/c", "c original")
	d := f.row(t, f.comments, f.carol, f.remoteR, pullThird+"/api/v1/comment/d", "d original")
	e := f.row(t, f.comments, f.alice, f.remoteR, pullOrigin+"/api/v1/comment/e", "e original")

	f.pullComments(t, f.remoteR, []map[string]any{
		{"id": "a", "iri": a.GetString("iri"), "text": "a pwned", "expand": authorExpand(f.mallory)},
		{"id": "b", "iri": b.GetString("iri"), "text": "b pwned", "expand": authorExpand(f.alice)},
		{"id": "c", "iri": c.GetString("iri"), "text": "c pwned", "expand": authorExpand(f.alice)},
		{"id": "d", "iri": d.GetString("iri"), "text": "d pwned", "expand": authorExpand(f.carol)},
		{"id": "e", "iri": e.GetString("iri"), "text": "e pwned", "expand": authorExpand(f.mallory)},
		{"id": "f", "iri": pullOrigin + "/api/v1/comment/f", "text": "f pwned", "expand": authorExpand(f.carol)},
	})

	f.assertUntouched(t, "a (local comment on a local trail)", a, "a original")
	f.assertUntouched(t, "b (local comment on the synced trail)", b, "b original")
	f.assertUntouched(t, "c (comment on another remote trail)", c, "c original")
	f.assertUntouched(t, "d (comment on a third host)", d, "d original")
	f.assertUntouched(t, "e (comment of another author)", e, "e original")

	if got, err := f.app.FindFirstRecordByData("comments", "iri", pullOrigin+"/api/v1/comment/f"); err == nil {
		t.Errorf("f: comment authored by a third-host actor was stored (author %s)", got.GetString("author"))
	}
}

func TestPullSyncCommentsStillImportOriginComments(t *testing.T) {
	f := setupPullFixture(t)

	existing := f.row(t, f.comments, f.alice, f.remoteR, pullOrigin+"/api/v1/comment/n2", "old")

	f.pullComments(t, f.remoteR, []map[string]any{
		{"id": "n1", "text": "new", "expand": authorExpand(f.alice)},
		{"id": "n2", "iri": existing.GetString("iri"), "text": "updated", "expand": authorExpand(f.alice)},
	})

	created, err := f.app.FindFirstRecordByData("comments", "iri", pullOrigin+"/api/v1/comment/n1")
	if err != nil {
		t.Fatalf("new origin comment was not stored: %v", err)
	}
	if created.GetString("text") != "new" || created.GetString("author") != f.alice.Id || created.GetString("trail") != f.remoteR.Id {
		t.Errorf("new comment = text %q author %s trail %s", created.GetString("text"), created.GetString("author"), created.GetString("trail"))
	}

	updated, err := f.app.FindRecordById("comments", existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.GetString("text") != "updated" {
		t.Errorf("existing origin comment text = %q, want updated", updated.GetString("text"))
	}
}

func TestPullSyncSummitLogsAndWaypointsRefuseLocalObjects(t *testing.T) {
	f := setupPullFixture(t)

	sl := f.row(t, f.summits, f.lou, f.localT, "https://local.example/api/v1/summit-log/s1", "s original")
	wp := f.row(t, f.waypoints, f.lou, f.localT, "https://local.example/api/v1/waypoint/w1", "w original")

	f.pullTrail(t, f.remoteR, map[string]any{
		"name": "remote trail",
		"expand": map[string]any{
			"summit_logs_via_trail": []any{
				map[string]any{"id": "s1", "iri": sl.GetString("iri"), "text": "s pwned", "expand": authorExpand(f.mallory)},
			},
			"waypoints_via_trail": []any{
				map[string]any{"id": "w1", "iri": wp.GetString("iri"), "name": "w pwned"},
			},
		},
	})

	f.assertUntouched(t, "summit log", sl, "s original")
	f.assertUntouched(t, "waypoint", wp, "w original")
}

func TestPullSyncStillImportsOriginSummitLogsAndWaypoints(t *testing.T) {
	f := setupPullFixture(t)

	f.pullTrail(t, f.remoteR, map[string]any{
		"name": "remote trail",
		"expand": map[string]any{
			"summit_logs_via_trail": []any{
				map[string]any{"id": "s9", "text": "summit", "expand": authorExpand(f.alice)},
			},
			"waypoints_via_trail": []any{
				map[string]any{"id": "w9", "name": "waypoint"},
			},
		},
	})

	sl, err := f.app.FindFirstRecordByData("summit_logs", "iri", pullOrigin+"/api/v1/summit-log/s9")
	if err != nil {
		t.Fatalf("origin summit log was not stored: %v", err)
	}
	if sl.GetString("text") != "summit" || sl.GetString("author") != f.alice.Id || sl.GetString("trail") != f.remoteR.Id {
		t.Errorf("summit log = text %q author %s trail %s", sl.GetString("text"), sl.GetString("author"), sl.GetString("trail"))
	}

	wp, err := f.app.FindFirstRecordByData("waypoints", "iri", pullOrigin+"/api/v1/waypoint/w9")
	if err != nil {
		t.Fatalf("origin waypoint was not stored: %v", err)
	}
	if wp.GetString("name") != "waypoint" || wp.GetString("author") != f.alice.Id || wp.GetString("trail") != f.remoteR.Id {
		t.Errorf("waypoint = name %q author %s trail %s", wp.GetString("name"), wp.GetString("author"), wp.GetString("trail"))
	}
}
