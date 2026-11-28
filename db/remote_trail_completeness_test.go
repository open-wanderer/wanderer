package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"pocketbase/routes"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type trailProofLog struct {
	mu sync.Mutex
	bytes.Buffer
}

func (w *trailProofLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Buffer.Write(p)
}

func (w *trailProofLog) text() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Buffer.String()
}

// Only direct child RecordQuery calls are captured. The embedded app's normal
// record/enrichment methods still run against the real, fully migrated SQLite DB.
type trailProofApp struct {
	core.App
	mu         sync.Mutex
	queries    []*dbx.SelectQuery
	failQuery  string
	requestCtx string
	log        trailProofLog
}

func (app *trailProofApp) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&app.log, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (app *trailProofApp) RecordQuery(c any) *dbx.SelectQuery {
	q := app.App.RecordQuery(c)
	collection, ok := c.(*core.Collection)
	if ok && (collection.Name == "waypoints" || collection.Name == "summit_logs") {
		app.mu.Lock()
		defer app.mu.Unlock()
		app.queries = append(app.queries, q)
		if app.failQuery == collection.Name {
			// A real SQLite query failure, without modifying any stored data.
			q.AndWhere(dbx.NewExp("missing_proof_column = 1"))
		}
	}
	return q
}

func (app *trailProofApp) takeQueries() []*dbx.SelectQuery {
	app.mu.Lock()
	defer app.mu.Unlock()
	queries := app.queries
	app.queries = nil
	return queries
}

type trailProofFixture struct {
	t                 *testing.T
	app               *trailProofApp
	mux               http.Handler
	owner, shared     *core.Record
	outsider          *core.Record
	actor             *core.Record
	public, private   *core.Record
	share             string
	permissionRequest int
}

func newTrailProofFixture(t *testing.T) *trailProofFixture {
	t.Helper()
	f := &trailProofFixture{t: t, app: &trailProofApp{App: newRealSchemaApp(t)}, share: strings.Repeat("a", 32)}
	newUser := func(name string) *core.Record {
		r := f.save("users", map[string]any{"email": name + "@example.com", "password": "proof-test-password"})
		return r
	}
	f.owner, f.shared, f.outsider = newUser("owner"), newUser("shared"), newUser("outsider")
	f.actor = f.save("activitypub_actors", map[string]any{"user": f.owner.Id, "is_local": true, "domain": "example.com", "preferred_username": "owner", "iri": "https://example.com/api/v1/actor/owner"})
	sharedActor := f.save("activitypub_actors", map[string]any{"user": f.shared.Id, "is_local": true, "domain": "example.com", "preferred_username": "shared", "iri": "https://example.com/api/v1/actor/shared"})
	f.public = f.save("trails", map[string]any{"author": f.actor.Id, "name": "Public empty", "public": true})
	f.private = f.save("trails", map[string]any{"author": f.actor.Id, "name": "Private empty"})
	f.save("trail_share", map[string]any{"trail": f.private.Id, "actor": sharedActor.Id, "permission": "view"})
	f.save("trail_link_share", map[string]any{"trail": f.private.Id, "token": f.share})
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/remote/trail/{id}", func(e *core.RequestEvent) error {
		if f.app.requestCtx != "" {
			e.Set(core.RequestEventKeyInfoContext, f.app.requestCtx)
		}
		return routes.RemoteTrailGet(e)
	})
	f.mux, err = router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *trailProofFixture) save(name string, data map[string]any) *core.Record {
	f.t.Helper()
	c, err := f.app.FindCollectionByNameOrId(name)
	if err != nil {
		f.t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(data)
	if err := f.app.SaveNoValidate(r); err != nil {
		f.t.Fatal(err)
	}
	return r
}

func (f *trailProofFixture) rules(name string, list, view *string) {
	f.t.Helper()
	c, err := f.app.FindCollectionByNameOrId(name)
	if err != nil {
		f.t.Fatal(err)
	}
	c.ListRule, c.ViewRule = list, view
	if err := f.app.SaveNoValidate(c); err != nil {
		f.t.Fatal(err)
	}
}

func (f *trailProofFixture) request(path string, auth *core.Record) *httptest.ResponseRecorder {
	f.t.Helper()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	// Permission-parity tests isolate the DB/rules from the separate list timing
	// guard. The latency regression test instead uses one real TCP client/IP.
	f.permissionRequest++
	r.RemoteAddr = fmt.Sprintf("192.0.2.%d:40000", f.permissionRequest)
	r.Header.Set("X-Proof", "scope")
	if auth != nil {
		token, err := auth.NewAuthToken()
		if err != nil {
			f.t.Fatal(err)
		}
		r.Header.Set("Authorization", token)
	}
	w := httptest.NewRecorder()
	f.mux.ServeHTTP(w, r)
	return w
}

func trailProofQueryParams(trailID, share string) url.Values {
	q := url.Values{"page": {"1"}, "perPage": {"1"}, "fields": {"id"}, "skipTotal": {"true"}, "filter": {"trail = " + strconv.Quote(trailID)}}
	if share != "" {
		q.Set("share", share)
	}
	return q
}

func trailProofExpands(t *testing.T, w *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct{ Expand map[string]json.RawMessage }
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Expand
}

func TestRemoteTrailChildProofPermissionsMatchRealListAPI(t *testing.T) {
	f := newTrailProofFixture(t)
	for _, tc := range []struct {
		name  string
		trail *core.Record
		auth  *core.Record
		share string
		code  int
	}{
		{"anonymous public", f.public, nil, "", 200},
		{"owner private", f.private, f.owner, "", 200},
		{"explicit shared user", f.private, f.shared, "", 200},
		{"valid share", f.private, nil, f.share, 200},
		{"wrong share public", f.public, nil, "wrong", 200},
		{"wrong share private", f.private, nil, "wrong", 403},
		{"outsider private", f.private, f.outsider, "", 403},
		{"anonymous private", f.private, nil, "", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.app.takeQueries()
			q := url.Values{"expand": {"waypoints_via_trail,summit_logs_via_trail"}}
			if tc.share != "" {
				q.Set("share", tc.share)
			}
			w := f.request("/remote/trail/"+tc.trail.Id+"?"+q.Encode(), tc.auth)
			if w.Code != tc.code {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.code, w.Body.String())
			}
			queries := f.app.takeQueries()
			if tc.code != 200 {
				if len(queries) != 0 {
					t.Fatal("child proof ran before parent authorization")
				}
				return
			}
			expands := trailProofExpands(t, w)
			if len(queries) != 2 {
				t.Fatalf("proof queries=%d want 2", len(queries))
			}
			for _, name := range []string{"waypoints", "summit_logs"} {
				list := f.request("/api/collections/"+name+"/records?"+trailProofQueryParams(tc.trail.Id, tc.share).Encode(), tc.auth)
				var result struct{ Items []json.RawMessage }
				if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &result) != nil || len(result.Items) != 0 {
					t.Fatalf("list parity failed: %s", list.Body.String())
				}
				if string(expands[name+"_via_trail"]) != "[]" {
					t.Fatalf("missing empty %s: %s", name, w.Body.String())
				}
			}
			for _, query := range queries {
				built := query.Build()
				if !strings.Contains(built.SQL(), "LIMIT 1") || strings.Contains(built.SQL(), ".*") || strings.Contains(built.SQL(), "ORDER BY") || strings.Contains(built.SQL(), "COUNT(") {
					t.Fatalf("unbounded or wide proof: %s", built.SQL())
				}
				var plan []struct{ Detail string }
				if err := f.app.ConcurrentDB().NewQuery("EXPLAIN QUERY PLAN " + built.SQL()).Bind(built.Params()).All(&plan); err != nil {
					t.Fatal(err)
				}
				indexed := false
				for _, row := range plan {
					indexed = indexed || (strings.Contains(row.Detail, "USING INDEX") && strings.Contains(row.Detail, "(trail=?)"))
				}
				if !indexed {
					t.Fatalf("proof did not use a trail index: %+v", plan)
				}
			}
		})
	}
}

func TestRemoteTrailChildProofFiltersBeforeLimitAndUsesStoredParent(t *testing.T) {
	f := newTrailProofFixture(t)
	// The raw first row is hidden, but a later row is list-visible. Neither can
	// expand because the ViewRule is stricter than the independent ListRule.
	f.rules("waypoints", types.Pointer("name = 'visible' && @request.auth.id = '"+f.owner.Id+"'"), types.Pointer("id = ''"))
	hidden := f.save("waypoints", map[string]any{"trail": f.public.Id, "name": "hidden"})
	visible := f.save("waypoints", map[string]any{"trail": f.public.Id, "name": "visible"})
	f.save("waypoints", map[string]any{"trail": f.private.Id, "name": "visible"})
	var first string
	if err := f.app.App.RecordQuery("waypoints").Select("id").Where(dbx.HashExp{"trail": f.public.Id}).Limit(1).Row(&first); err != nil || first != hidden.Id {
		t.Fatalf("raw first child=%s want hidden=%s error=%v", first, hidden.Id, err)
	}
	for _, auth := range []*core.Record{f.owner, f.outsider} {
		w := f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail", auth)
		expands := trailProofExpands(t, w)
		list := f.request("/api/collections/waypoints/records?"+trailProofQueryParams(f.public.Id, "").Encode(), auth)
		var body struct{ Items []struct{ ID string } }
		if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &body) != nil {
			t.Fatalf("list failed: %s", list.Body.String())
		}
		if auth.Id == f.owner.Id {
			if len(body.Items) != 1 || body.Items[0].ID != visible.Id || expands["waypoints_via_trail"] != nil {
				t.Fatalf("list-visible second row was mistaken for empty: list=%s trail=%s", list.Body.String(), w.Body.String())
			}
		} else if len(body.Items) != 0 || string(expands["waypoints_via_trail"]) != "[]" {
			t.Fatalf("proof/list auth mismatch: list=%s trail=%s", list.Body.String(), w.Body.String())
		}
	}
	visible.Set("trail", f.private.Id)
	if err := f.app.SaveNoValidate(visible); err != nil {
		t.Fatal(err)
	}
	w := f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail&filter=trail='"+f.private.Id+"'&sort=-name", f.owner)
	if string(trailProofExpands(t, w)["waypoints_via_trail"]) != "[]" {
		t.Fatalf("caller filter or another trail contaminated proof: %s", w.Body.String())
	}
}

func TestRemoteTrailChildProofVisibleRowsMatchRealListAPI(t *testing.T) {
	f := newTrailProofFixture(t)
	for _, name := range []string{"waypoints", "summit_logs"} {
		c, err := f.app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		// Leave the actual migrated ListRule intact, but make expansion fail.
		f.rules(name, c.ListRule, nil)
		for _, trail := range []*core.Record{f.public, f.private} {
			f.save(name, map[string]any{"trail": trail.Id, "author": f.actor.Id, "name": "Child", "text": "Child"})
		}
	}
	for _, tc := range []struct {
		name  string
		trail *core.Record
		auth  *core.Record
		share string
	}{
		{"anonymous public", f.public, nil, ""},
		{"owner private", f.private, f.owner, ""},
		{"shared user private", f.private, f.shared, ""},
		{"share token private", f.private, nil, f.share},
		{"wrong share public", f.public, f.outsider, "wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{"expand": {"waypoints_via_trail,summit_logs_via_trail"}, "share": {tc.share}}
			w := f.request("/remote/trail/"+tc.trail.Id+"?"+q.Encode(), tc.auth)
			expands := trailProofExpands(t, w)
			for _, name := range []string{"waypoints", "summit_logs"} {
				list := f.request("/api/collections/"+name+"/records?"+trailProofQueryParams(tc.trail.Id, tc.share).Encode(), tc.auth)
				var body struct{ Items []json.RawMessage }
				if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &body) != nil || len(body.Items) != 1 || expands[name+"_via_trail"] != nil {
					t.Fatalf("list-visible child was marked empty: list=%s trail=%s", list.Body.String(), w.Body.String())
				}
			}
		})
	}
}

func TestRemoteTrailChildProofMatchesChildRequestRuleVariables(t *testing.T) {
	f := newTrailProofFixture(t)
	f.save("waypoints", map[string]any{"trail": f.public.Id, "name": "visible"})
	filter := "trail = " + strconv.Quote(f.public.Id)
	rule := "@request.context = 'default' && @request.method = 'GET' && @request.auth.id = '" + f.owner.Id + "' && @request.headers.x_proof = 'scope' && @request.query.share = 'context-share' && @request.query.page = '1' && @request.query.perPage = '1' && @request.query.fields = 'id' && @request.query.skipTotal = 'true' && @request.query.filter = " + strconv.Quote(filter) + " && @request.query.expand:isset = false && @request.query.sort:isset = false && @request.query.handle:isset = false && @request.query.extra:isset = false"
	f.rules("waypoints", &rule, types.Pointer("id = ''"))
	f.app.requestCtx = "node"
	for _, share := range []string{"context-share", "wrong"} {
		q := url.Values{"expand": {"waypoints_via_trail.author"}, "share": {share}, "page": {"5"}, "perPage": {"999"}, "skipTotal": {"false"}, "filter": {"id = ''"}, "sort": {"-name"}, "extra": {"ignored"}}
		w := f.request("/remote/trail/"+f.public.Id+"?"+q.Encode(), f.owner)
		expands := trailProofExpands(t, w)
		list := f.request("/api/collections/waypoints/records?"+trailProofQueryParams(f.public.Id, share).Encode(), f.owner)
		var body struct{ Items []json.RawMessage }
		if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &body) != nil {
			t.Fatalf("list request failed: %s", list.Body.String())
		}
		if share == "context-share" {
			if len(body.Items) != 1 || expands["waypoints_via_trail"] != nil {
				t.Fatalf("child request context mismatch: list=%s trail=%s", list.Body.String(), w.Body.String())
			}
		} else if len(body.Items) != 0 || string(expands["waypoints_via_trail"]) != "[]" {
			t.Fatalf("wrong-share parity mismatch: list=%s trail=%s", list.Body.String(), w.Body.String())
		}
	}
	// A parent-only context must not grant child list access.
	f.rules("waypoints", types.Pointer("@request.context = 'node'"), nil)
	w := f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail", f.owner)
	if string(trailProofExpands(t, w)["waypoints_via_trail"]) != "[]" {
		t.Fatalf("parent node context leaked into child ListRule: %s", w.Body.String())
	}
	// An explicitly empty share is still present in the SDK query; an absent
	// share is not. Preserve :isset semantics, not only the string value.
	f.rules("waypoints", types.Pointer("@request.query.share:isset = true && @request.query.share = ''"), nil)
	for _, present := range []bool{false, true} {
		q := trailProofQueryParams(f.public.Id, "")
		remote := "/remote/trail/" + f.public.Id + "?expand=waypoints_via_trail"
		if present {
			q.Set("share", "")
			remote += "&share="
		}
		w := f.request(remote, f.owner)
		list := f.request("/api/collections/waypoints/records?"+q.Encode(), f.owner)
		var body struct{ Items []json.RawMessage }
		if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &body) != nil || (len(body.Items) == 1) != present || (trailProofExpands(t, w)["waypoints_via_trail"] == nil) != present {
			t.Fatalf("share presence parity mismatch: present=%v list=%s trail=%s", present, list.Body.String(), w.Body.String())
		}
	}
}

func TestRemoteTrailChildProofFailsClosedIndependently(t *testing.T) {
	f := newTrailProofFixture(t)
	for _, tc := range []struct {
		name string
		rule *string
		db   bool
	}{
		{"locked nil", nil, false},
		{"malformed syntax", types.Pointer("id ="), false},
		{"resolver failure", types.Pointer("missing_rule_field = true"), false},
		{"SQLite query failure", types.Pointer(""), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.rules("waypoints", tc.rule, nil)
			f.app.failQuery = ""
			if tc.db {
				f.app.failQuery = "waypoints"
			}
			w := f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail,summit_logs_via_trail", nil)
			expands := trailProofExpands(t, w)
			if expands["waypoints_via_trail"] != nil || string(expands["summit_logs_via_trail"]) != "[]" {
				t.Fatalf("failed proof must remain missing without suppressing its sibling: %s", w.Body.String())
			}
		})
	}
	f.app.failQuery = ""
	// A locked rule denies access; an invalid rule prevents the check from completing.
	for _, tc := range []struct {
		name   string
		rule   *string
		status int
		share  string
	}{
		{"parent locked nil", nil, http.StatusForbidden, ""},
		{"parent missing field", types.Pointer("missing_parent_field = true"), http.StatusInternalServerError, f.share},
		{"parent malformed syntax", types.Pointer("id ="), http.StatusInternalServerError, f.share},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.app.takeQueries()
			f.rules("trails", nil, tc.rule)
			path := "/remote/trail/" + f.public.Id + "?expand=waypoints_via_trail,summit_logs_via_trail"
			if tc.share != "" {
				path += "&share=" + url.QueryEscape(tc.share)
			}
			w := f.request(path, nil)
			if queries := len(f.app.takeQueries()); queries != 0 {
				t.Errorf("child proof ran %d queries after parent access failed", queries)
			}
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.status, w.Body.String())
			}
			if tc.share != "" {
				var body struct {
					Data map[string]json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode parent access error: %v; body=%s", err, w.Body.String())
				}
				if _, present := body.Data["share"]; present {
					t.Fatalf("parent access error carries a share denial marker: %s", w.Body.String())
				}
			}
		})
	}
	f.rules("trails", types.Pointer(""), types.Pointer(""))
	f.rules("waypoints", nil, nil)
	superuser := f.save(core.CollectionNameSuperusers, map[string]any{"email": "proof-admin@example.com", "password": "proof-test-password"})
	w := f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail", superuser)
	if trailProofExpands(t, w)["waypoints_via_trail"] != nil {
		t.Fatal("conservative locked-collection policy also applies to superusers")
	}
	f.app.takeQueries()
	id := f.app.OnRecordEnrich("trails").BindFunc(func(e *core.RecordEnrichEvent) error { return errors.New("enrichment failed") })
	defer f.app.OnRecordEnrich("trails").Unbind(id)
	w = f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail,summit_logs_via_trail", nil)
	if w.Code != 500 || len(f.app.takeQueries()) != 0 {
		t.Fatalf("proof ran after failed parent enrichment: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRemoteTrailChildProofHonorsExpandAndProjectionGuards(t *testing.T) {
	f := newTrailProofFixture(t)
	for _, tc := range []struct {
		query    string
		waypoint bool
		logs     bool
	}{
		{"", false, false},
		{"expand=author", false, false},
		{"expand=waypoints_via_trail_other", false, false},
		{"expand=waypoints_via_trail.author", true, false},
		{"expand=" + url.QueryEscape(" summit_logs_via_trail.author,waypoints_via_trail.author "), true, true},
		{"expand=waypoints_via_trail,summit_logs_via_trail&fields=", false, false},
		{"expand=waypoints_via_trail,summit_logs_via_trail&fields=id,name", false, false},
		{"expand=waypoints_via_trail,summit_logs_via_trail&fields=expand", false, false},
	} {
		t.Run(tc.query, func(t *testing.T) {
			f.app.takeQueries()
			w := f.request("/remote/trail/"+f.public.Id+"?"+tc.query, nil)
			expands := trailProofExpands(t, w)
			if (string(expands["waypoints_via_trail"]) == "[]") != tc.waypoint || (string(expands["summit_logs_via_trail"]) == "[]") != tc.logs {
				t.Fatalf("expand/projection guard mismatch: %s", w.Body.String())
			}
			wantQueries := 0
			if tc.waypoint {
				wantQueries++
			}
			if tc.logs {
				wantQueries++
			}
			if len(f.app.takeQueries()) != wantQueries {
				t.Fatal("unrequested/projected relation was queried")
			}
			if strings.HasSuffix(tc.query, "fields=id,name") {
				var body map[string]any
				if json.Unmarshal(w.Body.Bytes(), &body) != nil || len(body) != 2 || body["id"] != f.public.Id || body["name"] != f.public.GetString("name") {
					t.Fatalf("response projection changed: %s", w.Body.String())
				}
			}
		})
	}
	// A less restrictive ViewRule may already have expanded records even when
	// ListRule is locked/stricter. Those expansions must never be overwritten.
	f.save("waypoints", map[string]any{"trail": f.public.Id, "name": "Existing"})
	f.rules("waypoints", nil, types.Pointer(""))
	f.app.takeQueries()
	w := f.request("/remote/trail/"+f.public.Id+"?expand=waypoints_via_trail,summit_logs_via_trail", nil)
	expands := trailProofExpands(t, w)
	var children []json.RawMessage
	if json.Unmarshal(expands["waypoints_via_trail"], &children) != nil || len(children) != 1 || len(f.app.takeQueries()) != 1 {
		t.Fatalf("existing expansion overwritten or unnecessarily queried: %s", w.Body.String())
	}
}
