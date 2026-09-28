package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/auth"
)

func TestConfigureOIDCScopes(t *testing.T) {
	defaultScopes := auth.Providers["oidc"]().Scopes()

	scenarios := []struct {
		name     string
		env      map[string]string
		expected map[string][]string
	}{
		{
			name: "unset leaves the provider defaults alone",
			env:  map[string]string{},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "single scope",
			env:  map[string]string{"OIDC_SCOPES": "openid"},
			expected: map[string][]string{
				"oidc":  {"openid"},
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "comma separated list",
			env:  map[string]string{"OIDC2_SCOPES": "openid,read_prefs"},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": {"openid", "read_prefs"},
				"oidc3": defaultScopes,
			},
		},
		{
			name: "surrounding whitespace is ignored",
			env:  map[string]string{"OIDC3_SCOPES": " openid , read_prefs "},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": defaultScopes,
				"oidc3": {"openid", "read_prefs"},
			},
		},
		{
			name: "empty entries are dropped",
			env:  map[string]string{"OIDC_SCOPES": "openid,,read_prefs,"},
			expected: map[string][]string{
				"oidc":  {"openid", "read_prefs"},
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "only separators leaves the provider defaults alone",
			env:  map[string]string{"OIDC_SCOPES": ",, ,"},
			expected: map[string][]string{
				"oidc":  defaultScopes,
				"oidc2": defaultScopes,
				"oidc3": defaultScopes,
			},
		},
		{
			name: "each slot gets its own scopes",
			env: map[string]string{
				"OIDC_SCOPES":  "openid",
				"OIDC3_SCOPES": "openid,profile",
			},
			expected: map[string][]string{
				"oidc":  {"openid"},
				"oidc2": defaultScopes,
				"oidc3": {"openid", "profile"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			// restore the stock factories so each scenario starts from the defaults
			original := auth.Providers["oidc"]
			t.Cleanup(func() {
				for name := range oidcScopesEnv {
					auth.Providers[name] = original
				}
			})

			for _, env := range oidcScopesEnv {
				t.Setenv(env, s.env[env])
			}

			configureOIDCScopes()

			for name, expected := range s.expected {
				scopes := auth.Providers[name]().Scopes()
				if !slices.Equal(scopes, expected) {
					t.Fatalf("%s: expected scopes %v, got %v", name, expected, scopes)
				}
			}
		})
	}
}

// searchInitPageSize matches the trails page size in initMeilisearchDocuments.
const searchInitPageSize int64 = 100

func TestInitMeilisearchDocumentsPagination(t *testing.T) {
	t.Run("missing author on the first page", func(t *testing.T) {
		app := newSearchInitApp(t)
		author := seedTrails(t, app, int(searchInitPageSize)+1)
		first := trailPage(t, app, 0)
		second := trailPage(t, app, 1)
		if len(first) != int(searchInitPageSize) || len(second) == 0 {
			t.Fatalf("pages = %d and %d trails", len(first), len(second))
		}
		orphan := first[0]
		missingAuthorID := orphanTrailAuthor(t, app, orphan)
		first = trailPage(t, app, 0)
		second = trailPage(t, app, 1)
		if !trailPageContains(first, orphan.Id) {
			t.Fatal("orphaned trail was not on the first page")
		}

		state, client, overflow := newSearchInitClient(t, nil)
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "trails"},
			{method: http.MethodPost, index: "trails"},
			{method: http.MethodDelete, index: "lists"},
			{method: http.MethodDelete, index: "actors"},
			{method: http.MethodPost, index: "actors"},
		})
		batches, rejected := state.batches()
		if rejected != 0 {
			t.Fatalf("rejected trail batches = %d; want 0", rejected)
		}
		assertIndexedTrailPage(t, batches, second, author.Id)
		if strings.Count(logText, "Unable to index trails page 0") != 1 || strings.Contains(logText, "Unable to index trails page 1") {
			t.Fatalf("log = %s", logText)
		}
		if !strings.Contains(logText, orphan.Id) || !strings.Contains(logText, missingAuthorID) || !strings.Contains(logText, "missing author reference") {
			t.Fatalf("log = %s; want the failed trail and author", logText)
		}
		assertLaterSearchPhases(t, logText)
	})

	t.Run("all valid pages", func(t *testing.T) {
		app := newSearchInitApp(t)
		author := seedTrails(t, app, int(searchInitPageSize)+1)
		first := trailPage(t, app, 0)
		second := trailPage(t, app, 1)

		state, client, overflow := newSearchInitClient(t, nil)
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "trails"},
			{method: http.MethodPost, index: "trails"},
			{method: http.MethodPost, index: "trails"},
			{method: http.MethodDelete, index: "lists"},
			{method: http.MethodDelete, index: "actors"},
			{method: http.MethodPost, index: "actors"},
		})
		batches, rejected := state.batches()
		if rejected != 0 || len(batches) != 2 {
			t.Fatalf("trail batches = %d, rejected = %d; want 2 accepted batches", len(batches), rejected)
		}
		assertIndexedTrailPage(t, batches[:1], first, author.Id)
		assertIndexedTrailPage(t, batches[1:], second, author.Id)
		if strings.Contains(logText, "Unable to index trails page") {
			t.Fatalf("log = %s", logText)
		}
		assertLaterSearchPhases(t, logText)
	})

	t.Run("empty database", func(t *testing.T) {
		app := newSearchInitApp(t)
		state, client, overflow := newSearchInitClient(t, nil)
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "trails"},
			{method: http.MethodDelete, index: "lists"},
			{method: http.MethodDelete, index: "actors"},
		})
		batches, rejected := state.batches()
		if len(batches) != 0 || rejected != 0 || strings.Contains(logText, "Unable to index") {
			t.Fatalf("batches = %d, rejected = %d, log = %s", len(batches), rejected, logText)
		}
	})

	t.Run("rejected add advances", func(t *testing.T) {
		app := newSearchInitApp(t)
		author := seedTrails(t, app, int(searchInitPageSize)+1)
		first := trailPage(t, app, 0)
		second := trailPage(t, app, 1)
		reject := make(map[string]bool, len(first))
		for _, trail := range first {
			reject[trail.Id] = true
		}

		state, client, overflow := newSearchInitClient(t, reject)
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "trails"},
			{method: http.MethodPost, index: "trails"},
			{method: http.MethodPost, index: "trails"},
			{method: http.MethodDelete, index: "lists"},
			{method: http.MethodDelete, index: "actors"},
			{method: http.MethodPost, index: "actors"},
		})
		batches, rejected := state.batches()
		if rejected != 1 {
			t.Fatalf("rejected trail batches = %d; want 1", rejected)
		}
		assertIndexedTrailPage(t, batches, second, author.Id)
		if strings.Count(logText, "Unable to index trails page 0") != 1 {
			t.Fatalf("log = %s", logText)
		}
		assertLaterSearchPhases(t, logText)
	})
}

func assertLaterSearchPhases(t *testing.T, logText string) {
	t.Helper()
	if strings.Contains(logText, "Unable to index list page") || strings.Contains(logText, "Unable to index actor page") {
		t.Fatalf("log = %s", logText)
	}
}

func assertSearchInitCalls(t *testing.T, got, want []searchInitCall) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("requests = %+v; want %+v", got, want)
	}
}

func assertIndexedTrailPage(t *testing.T, batches [][]map[string]any, want []*core.Record, authorID string) {
	t.Helper()
	if len(batches) != 1 {
		t.Fatalf("indexed trail batches = %d; want 1", len(batches))
	}
	got := make([]string, 0, len(batches[0]))
	for _, document := range batches[0] {
		id, _ := document["id"].(string)
		got = append(got, id)
		if document["author"] != authorID || document["author_name"] != "Search author" || document["domain"] != "" || document["is_federated"] != false {
			t.Fatalf("indexed document = %#v", document)
		}
	}
	slices.Sort(got)
	wantIDs := make([]string, len(want))
	for i, trail := range want {
		wantIDs[i] = trail.Id
	}
	slices.Sort(wantIDs)
	if !slices.Equal(got, wantIDs) {
		t.Fatalf("indexed trails = %v; want %v", got, wantIDs)
	}
}

func trailPageContains(trails []*core.Record, id string) bool {
	for _, trail := range trails {
		if trail.Id == id {
			return true
		}
	}
	return false
}

type searchInitLogApp struct {
	core.App
	logger *slog.Logger
}

func (app *searchInitLogApp) Logger() *slog.Logger {
	return app.logger
}

func runSearchInit(t *testing.T, app core.App, client meilisearch.ServiceManager, overflow <-chan struct{}) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	done := make(chan error, 1)
	go func() {
		done <- initMeilisearchDocuments(&searchInitLogApp{App: app, logger: logger}, client)
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		return buf.String(), err
	case <-timer.C:
		t.Fatal("initMeilisearchDocuments did not finish before the deadline")
	case <-overflow:
		t.Fatal("initMeilisearchDocuments exceeded the search request limit")
	}
	return "", nil
}

func newSearchInitApp(t *testing.T) *core.BaseApp {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	app.Settings().Logs.MaxDays = 0

	tags := core.NewBaseCollection("tags")
	tags.Fields.Add(&core.TextField{Name: "name"})
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.TextField{Name: "icon"},
		&core.BoolField{Name: "is_local"},
	)
	categories := core.NewBaseCollection("categories")
	categories.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "icon"})
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1},
		&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 100},
	)
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
	)
	likes := core.NewBaseCollection("trail_like")
	likes.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
	)
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(&core.TextField{Name: "name"})
	for _, collection := range []*core.Collection{tags, actors, categories, trails, shares, likes, lists} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	return app
}

func seedTrails(t *testing.T, app core.App, count int) *core.Record {
	t.Helper()
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	author := core.NewRecord(actors)
	author.Set("preferred_username", "Search author")
	author.Set("is_local", true)
	if err := app.Save(author); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		trail := core.NewRecord(trails)
		trail.Set("name", fmt.Sprintf("Trail %03d", i))
		trail.Set("author", author.Id)
		trail.Set("public", true)
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
	}
	return author
}

func trailPage(t *testing.T, app core.App, page int64) []*core.Record {
	t.Helper()
	trails := []*core.Record{}
	err := app.RecordQuery("trails").
		Limit(searchInitPageSize).
		Offset(page * searchInitPageSize).
		All(&trails)
	if err != nil {
		t.Fatal(err)
	}
	return trails
}

func orphanTrailAuthor(t *testing.T, app core.App, trail *core.Record) string {
	t.Helper()
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	removed := core.NewRecord(actors)
	removed.Set("preferred_username", "Removed author")
	removed.Set("is_local", true)
	if err := app.Save(removed); err != nil {
		t.Fatal(err)
	}
	trail.Set("author", removed.Id)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	result, err := app.NonconcurrentDB().Delete("activitypub_actors", dbx.HashExp{"id": removed.Id}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("deleted actor rows = %d, %v; want 1", affected, err)
	}
	stored, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("author") != removed.Id {
		t.Fatalf("author reference = %q; want dangling %q", stored.GetString("author"), removed.Id)
	}
	return removed.Id
}

type searchInitCall struct {
	method string
	index  string
}

type searchInitServer struct {
	mu                   sync.Mutex
	limit                int
	overflow             chan struct{}
	callsLog             []searchInitCall
	trailBatches         [][]map[string]any
	rejectedTrailBatches int
	rejectTrailIDs       map[string]bool
}

func newSearchInitClient(t *testing.T, rejectTrailIDs map[string]bool) (*searchInitServer, meilisearch.ServiceManager, <-chan struct{}) {
	t.Helper()
	overflow := make(chan struct{}, 1)
	state := &searchInitServer{
		limit:          12,
		overflow:       overflow,
		rejectTrailIDs: rejectTrailIDs,
	}
	server := httptest.NewServer(http.HandlerFunc(state.serveHTTP))
	t.Cleanup(server.Close)
	return state, meilisearch.New(server.URL), overflow
}

func (state *searchInitServer) calls() []searchInitCall {
	state.mu.Lock()
	defer state.mu.Unlock()
	return slices.Clone(state.callsLog)
}

func (state *searchInitServer) batches() ([][]map[string]any, int) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return slices.Clone(state.trailBatches), state.rejectedTrailBatches
}

func (state *searchInitServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var documents []map[string]any
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&documents); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	indexName := ""
	const prefix = "/indexes/"
	const suffix = "/documents"
	if strings.HasPrefix(r.URL.Path, prefix) && strings.HasSuffix(r.URL.Path, suffix) {
		indexName = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, prefix), suffix)
	}

	state.mu.Lock()
	state.callsLog = append(state.callsLog, searchInitCall{method: r.Method, index: indexName})
	overLimit := len(state.callsLog) > state.limit
	if overLimit {
		select {
		case state.overflow <- struct{}{}:
		default:
		}
	}
	reject := false
	if !overLimit && r.Method == http.MethodPost && indexName == "trails" {
		for _, document := range documents {
			id, _ := document["id"].(string)
			if state.rejectTrailIDs[id] {
				reject = true
				break
			}
		}
		if reject {
			state.rejectedTrailBatches++
		} else {
			state.trailBatches = append(state.trailBatches, documents)
		}
	}
	state.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if overLimit {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"too many requests","code":"too_many_requests","type":"internal"}`))
		return
	}
	if indexName == "" || (r.Method != http.MethodDelete && r.Method != http.MethodPost) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"unexpected request","code":"bad_request","type":"invalid_request"}`))
		return
	}
	if reject {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"rejected documents","code":"invalid_document","type":"invalid_request"}`))
		return
	}
	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintf(w, `{"taskUid":1,"indexUid":%q,"status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2026-09-19T00:00:00Z"}`, indexName)
}
