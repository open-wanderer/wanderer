package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const remoteTrailSyncTestExpand = "author,category,subcategory,tags,waypoints_via_trail,summit_logs_via_trail,summit_logs_via_trail.author"
const remoteListSyncTestExpand = "author,trails,trails.author,trails.category,trails.subcategory,trails.tags"
const remoteSyncTestShare = "abcdefghijklmnopqrstuvwxyz012345"

func TestRemoteSyncQueriesFilterShareValues(t *testing.T) {
	valid := "abcdefghijklmnopqrstuvwxyz012345"
	for _, target := range []struct {
		name  string
		query func(*url.URL) url.Values
	}{
		{"trail", remoteTrailSyncQuery},
		{"list", remoteListSyncQuery},
	} {
		for _, scenario := range []struct {
			name string
			raw  string
			want string
		}{
			{"valid", valid, valid},
			{"invalid-character", strings.Repeat("a", 31) + "-", ""},
			{"oversized", strings.Repeat("a", 33), ""},
			{"uppercase", strings.Repeat("A", 32), ""},
			{"encoded-delimiter", strings.Repeat("a", 31) + "%26", ""},
			{"encoded-space", strings.Repeat("a", 31) + "%20", ""},
			{"encoded-query", url.QueryEscape(valid + "&fields=*"), ""},
			{"encoded-percent", url.QueryEscape("%61" + strings.Repeat("a", 29)), ""},
		} {
			t.Run(target.name+"/"+scenario.name, func(t *testing.T) {
				query := target.query(&url.URL{RawQuery: "share=" + scenario.raw})
				if query.Get("share") != scenario.want {
					t.Fatalf("forwarded share=%q, want %q", query.Get("share"), scenario.want)
				}
				if scenario.want == "" && query.Has("share") {
					t.Fatalf("invalid share parameter reached the origin: %v", query)
				}
			})
		}
	}
}

func TestRemoteSyncQueriesCannotBeInflatedByCallerParameters(t *testing.T) {
	request, err := url.Parse("https://a.example/trail?expand=" + strings.Repeat("caller.", 10_000) + "&padding=" + strings.Repeat("x", 50_000) + "&share=" + strings.Repeat("a", 10_000))
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []url.Values{remoteTrailSyncQuery(request), remoteListSyncQuery(request)} {
		if len(query.Encode()) > 1024 || query.Get("padding") != "" || query.Get("share") != "" || strings.Contains(query.Get("expand"), "caller") {
			t.Fatalf("caller input reached the sync origin: %d encoded bytes", len(query.Encode()))
		}
	}
}

func TestRemoteTrailSyncQueryOmitsUnusedWaypointAuthors(t *testing.T) {
	for _, caller := range []*url.URL{nil, {RawQuery: "expand=waypoints_via_trail.author"}} {
		paths := strings.Split(remoteTrailSyncQuery(caller).Get("expand"), ",")
		if slices.Contains(paths, "waypoints_via_trail.author") {
			t.Fatal("unused waypoint author expansion requested, even though sync inherits the trail author")
		}
		for _, required := range []string{"waypoints_via_trail", "summit_logs_via_trail.author"} {
			if !slices.Contains(paths, required) {
				t.Fatalf("consumed expansion %s omitted", required)
			}
		}
	}
}

// Extend the existing sync fixture with real metadata relations and API rules.
func newRemoteSyncQueryFixture(t *testing.T) *remoteSyncFixture {
	t.Helper()
	f := newRemoteSyncFixture(t)
	save := func(model core.Model) {
		t.Helper()
		if err := f.app.Save(model); err != nil {
			t.Fatal(err)
		}
	}
	f.actors.Fields.Add(&core.TextField{Name: "preferred_username"})
	f.actors.ViewRule = types.Pointer("is_local = false")
	save(f.actors)
	f.remoteActor.Set("preferred_username", "riley")
	save(f.remoteActor)

	categories := core.NewBaseCollection("categories")
	categories.Fields.Add(&core.TextField{Name: "name"})
	save(categories)
	subcategories := core.NewBaseCollection("subcategories")
	subcategories.Fields.Add(&core.TextField{Name: "name"}, &core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1})
	save(subcategories)
	tags := core.NewBaseCollection("tags")
	tags.Fields.Add(&core.TextField{Name: "name"})
	save(tags)
	f.trails.Fields.Add(
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1},
		&core.RelationField{Name: "subcategory", CollectionId: subcategories.Id, MaxSelect: 1},
		&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 10},
		&core.TextField{Name: "federated_category_name"}, &core.TextField{Name: "federated_subcategory_name"},
	)
	for _, collection := range []*core.Collection{f.trails, f.lists} {
		collection.ViewRule = types.Pointer("public = true")
		save(collection)
	}
	for _, collection := range []*core.Collection{f.waypoints, f.logs} {
		collection.ViewRule = types.Pointer("trail.public = true")
		save(collection)
	}
	category := f.save(categories, map[string]any{"name": "Hiking"})
	f.save(subcategories, map[string]any{"name": "Winter", "category": category.Id})
	f.save(tags, map[string]any{"name": "Scenic"})
	return f
}

func remoteSyncQueryTrailPayload(query url.Values, prefix, authorIRI string) map[string]any {
	expand := map[string]any{}
	if query.Get("fields") == "" {
		paths := strings.Split(query.Get("expand"), ",")
		for field, value := range map[string]any{
			"author":      map[string]any{"iri": authorIRI},
			"category":    map[string]any{"id": "origcategory001", "name": "Hiking"},
			"subcategory": map[string]any{"id": "origsubcat00001", "name": "Winter"},
			"tags":        []any{map[string]any{"id": "origtag00000001", "name": "Scenic"}, map[string]any{"id": "origtag00000002", "name": "Lakes"}},
		} {
			if slices.Contains(paths, prefix+field) {
				expand[field] = value
			}
		}
	}
	return map[string]any{
		"id": "trail", "iri": "https://remote.example/api/v1/trail/trail", "name": "Origin trail", "public": true,
		"description": "Complete origin description", "category": "origcategory001", "subcategory": "origsubcat00001",
		"tags": []any{"origtag00000001", "origtag00000002"}, "expand": expand,
	}
}

// Other sync work may fetch actors or references after the document. Only the
// exact document endpoint counts towards these query assertions.
func mockRemoteSyncQueryDocument(t *testing.T, iri string, payload func(url.Values) map[string]any) func() url.Values {
	t.Helper()
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	var mu sync.Mutex
	var requests []url.Values
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if request.URL.Scheme+"://"+request.URL.Host+request.URL.EscapedPath() != iri {
				return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			}
			query := request.URL.Query()
			mu.Lock()
			requests = append(requests, query)
			mu.Unlock()
			return richTextSyncTransport{payload(query)}.RoundTrip(request)
		}}}
	}
	return func() url.Values {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if len(requests) != 1 {
			t.Fatalf("origin document %s requested %d times, want 1", iri, len(requests))
		}
		return requests[0]
	}
}

func assertRemoteSyncDocumentQuery(t *testing.T, query url.Values, expand, share string) {
	t.Helper()
	want := url.Values{"expand": {expand}}
	if share != "" {
		want.Set("share", share)
	}
	if query.Encode() != want.Encode() {
		t.Fatalf("origin document query = %v, want only %v", query, want)
	}
}

func assertRemoteSyncQueryTrailMetadata(t *testing.T, f *remoteSyncFixture, trail *core.Record, author string) {
	t.Helper()
	if trail.GetString("name") != "Origin trail" || trail.GetString("description") != "Complete origin description" || !trail.GetBool("public") || trail.GetString("author") != author {
		t.Fatalf("incomplete stored trail: %v", trail.PublicExport())
	}
	for _, relation := range []struct{ field, collection, name string }{
		{"category", "categories", "Hiking"}, {"subcategory", "subcategories", "Winter"},
	} {
		record, err := f.app.FindRecordById(relation.collection, trail.GetString(relation.field))
		if err != nil || record.GetString("name") != relation.name || trail.GetString("federated_"+relation.field+"_name") != relation.name {
			t.Fatalf("stored %s relation/name missing: %v %v", relation.field, record, err)
		}
		if relation.field == "subcategory" && record.GetString("category") != trail.GetString("category") {
			t.Fatal("stored subcategory does not belong to the stored category")
		}
	}
	ids := trail.GetStringSlice("tags")
	for _, name := range []string{"Scenic", "Lakes"} {
		tag, err := f.app.FindFirstRecordByData("tags", "name", name)
		if err != nil || !slices.Contains(ids, tag.Id) {
			t.Fatalf("stored tag relation %s missing: %v %v", name, ids, err)
		}
	}
	if count, err := f.app.CountRecords("tags"); err != nil || len(ids) != 2 || count != 2 {
		t.Fatalf("tags were lost or duplicated: relations=%v count=%d error=%v", ids, count, err)
	}
}

func remoteSyncProjectedGet(t *testing.T, f *remoteSyncFixture, kind, name string) string {
	t.Helper()
	query := url.Values{"handle": {"@riley@remote.example"}, "fields": {"id,name,expand"}, "expand": {"author"}, "foo": {"bar"}, "share": {remoteSyncTestShare}}
	request := httptest.NewRequest(http.MethodGet, "/remote/"+kind+"/"+kind+"?"+query.Encode(), nil)
	request.SetPathValue("id", kind)
	originalURL := *request.URL
	response := httptest.NewRecorder()
	e := &core.RequestEvent{App: f.app}
	e.Request, e.Response = request, response
	handler := RemoteTrailGet
	if kind == "list" {
		handler = RemoteListGet
	}
	if err := handler(e); err != nil {
		t.Fatal(err)
	}
	if *request.URL != originalURL {
		t.Fatalf("handler modified caller URL: %s, want %s", request.URL, &originalURL)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", response.Code, response.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	var id, gotName string
	var expand map[string]json.RawMessage
	if err := json.Unmarshal(body["id"], &id); err != nil || id == "" {
		t.Fatalf("missing response id: %s", response.Body.String())
	}
	if err := json.Unmarshal(body["name"], &gotName); err != nil || gotName != name {
		t.Fatalf("incorrect response name: %s", response.Body.String())
	}
	if err := json.Unmarshal(body["expand"], &expand); err != nil || len(body) != 3 || len(expand) != 1 {
		t.Fatalf("caller projection leaked metadata or child expansions: %s", response.Body.String())
	}
	var author struct {
		ID     string         `json:"id"`
		IRI    string         `json:"iri"`
		Expand map[string]any `json:"expand"`
	}
	if err := json.Unmarshal(expand["author"], &author); err != nil || author.ID != f.remoteActor.Id || author.IRI != f.remoteActor.GetString("iri") || len(author.Expand) != 0 {
		t.Fatalf("caller author expansion missing or changed: %s", response.Body.String())
	}
	return id
}
