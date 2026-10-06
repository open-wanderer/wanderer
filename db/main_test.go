package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
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

	"pocketbase/util"
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
	t.Run("a trail that cannot be indexed is left out", func(t *testing.T) {
		app := newSearchInitApp(t)
		author := seedTrails(t, app, int(searchInitPageSize)+1)
		orphan := trailPage(t, app, 0)[0]
		missingAuthorID := orphanTrailAuthor(t, app, orphan)
		first := slices.DeleteFunc(trailPage(t, app, 0), func(r *core.Record) bool { return r.Id == orphan.Id })
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
		if !strings.Contains(logText, "Unable to index trails "+orphan.Id) || !strings.Contains(logText, missingAuthorID) || !strings.Contains(logText, "missing author reference") {
			t.Fatalf("log = %s; want the failed trail and author", logText)
		}
		// The nightly repair retries the trail; the rebuild itself is complete.
		if got := recordedSearchVersions(app); !maps.Equal(got, util.SearchDocumentVersions) {
			t.Fatalf("recorded versions = %v; want %v", got, util.SearchDocumentVersions)
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
		if strings.Contains(logText, "Unable to index trails") {
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

	t.Run("a rejected page leaves the rebuild incomplete", func(t *testing.T) {
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
		want := maps.Clone(util.SearchDocumentVersions)
		delete(want, "trails")
		if got := recordedSearchVersions(app); !maps.Equal(got, want) {
			t.Fatalf("recorded versions = %v; want %v", got, want)
		}
		assertLaterSearchPhases(t, logText)
	})
}

func TestInitMeilisearchDocumentsKeepsCurrentIndexes(t *testing.T) {
	trailCount := int64(searchInitPageSize) + 1
	// seedTrails stores one author actor next to the trails.
	const actorCount int64 = 1

	t.Run("current indexes are kept", func(t *testing.T) {
		app := newSearchInitApp(t)
		seedTrails(t, app, int(trailCount))
		recordSearchIndexVersions(t, app, util.SearchDocumentVersions)

		state, client, overflow := newSearchInitClient(t, nil)
		// A count short of the collection is left to the nightly repair.
		state.documentCounts["trails"] = trailCount - 1
		state.documentCounts["actors"] = actorCount
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), nil)
		for _, index := range []string{"trails", "lists", "actors"} {
			if !strings.Contains(logText, "Search index "+index+" is current") {
				t.Fatalf("log = %s; want %s kept", logText, index)
			}
		}
	})

	t.Run("an empty index is rebuilt", func(t *testing.T) {
		app := newSearchInitApp(t)
		seedTrails(t, app, int(trailCount))
		recordSearchIndexVersions(t, app, util.SearchDocumentVersions)

		state, client, overflow := newSearchInitClient(t, nil)
		state.documentCounts["trails"] = trailCount
		// Queued settings updates are not document writes and are not canceled.
		state.settingsTasks["actors"] = 1
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "actors"},
			{method: http.MethodPost, index: "actors"},
		})
		if canceled := state.canceledIndexes(); len(canceled) != 0 {
			t.Fatalf("canceled = %v; want none", canceled)
		}
	})

	t.Run("a recreated index is rebuilt although it holds documents", func(t *testing.T) {
		app := newSearchInitApp(t)
		seedTrails(t, app, int(trailCount))
		recordSearchIndexVersions(t, app, util.SearchDocumentVersions)

		state, client, overflow := newSearchInitClient(t, nil)
		state.documentCounts["trails"] = trailCount
		// A live write already put a document into the wiped index.
		state.documentCounts["actors"] = actorCount
		const recreated = "2026-10-06T04:00:00Z"
		state.createdAt["actors"] = recreated
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "actors"},
			{method: http.MethodPost, index: "actors"},
		})
		if !strings.Contains(logText, "Rebuilding search index actors: Meilisearch recreated the index") {
			t.Fatalf("log = %s", logText)
		}
		if got := readSearchIndexVersions(app)["actors"].CreatedAt; got != recreated {
			t.Fatalf("recorded creation time = %q; want %q", got, recreated)
		}
	})

	t.Run("a failed delete keeps the recorded version", func(t *testing.T) {
		app := newSearchInitApp(t)
		seedTrails(t, app, int(trailCount))
		recordSearchIndexVersions(t, app, util.SearchDocumentVersions)

		state, client, overflow := newSearchInitClient(t, nil)
		state.documentCounts["trails"] = trailCount
		state.failDeleteIndexes["actors"] = true
		logText, err := runSearchInit(t, app, client, overflow)
		if err == nil || !strings.Contains(err.Error(), "rebuild search index actors") {
			t.Fatalf("initMeilisearchDocuments error = %v; want the actors rebuild failure\n%s", err, logText)
		}
		// The index still holds its documents, so it is not marked for a
		// rebuild on the next start.
		if got := recordedSearchVersions(app); !maps.Equal(got, util.SearchDocumentVersions) {
			t.Fatalf("recorded versions = %v; want %v", got, util.SearchDocumentVersions)
		}
	})

	t.Run("an unreachable Meilisearch leaves the indexes alone", func(t *testing.T) {
		app := newSearchInitApp(t)
		seedTrails(t, app, int(trailCount))
		recordSearchIndexVersions(t, app, util.SearchDocumentVersions)

		state, client, overflow := newSearchInitClient(t, nil)
		state.failStats = true
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), nil)
		if got := recordedSearchVersions(app); !maps.Equal(got, util.SearchDocumentVersions) {
			t.Fatalf("recorded versions = %v; want %v", got, util.SearchDocumentVersions)
		}
		if !strings.Contains(logText, "leaving it for the nightly repair") {
			t.Fatalf("log = %s", logText)
		}
	})

	t.Run("an older version cancels queued tasks and rebuilds", func(t *testing.T) {
		app := newSearchInitApp(t)
		seedTrails(t, app, int(trailCount))
		versions := maps.Clone(util.SearchDocumentVersions)
		versions["actors"]--
		recordSearchIndexVersions(t, app, versions)

		state, client, overflow := newSearchInitClient(t, nil)
		state.documentCounts["trails"] = trailCount
		state.documentCounts["actors"] = actorCount
		state.pendingTasks["actors"] = 5
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		assertSearchInitCalls(t, state.calls(), []searchInitCall{
			{method: http.MethodDelete, index: "actors"},
			{method: http.MethodPost, index: "actors"},
		})
		if canceled := state.canceledIndexes(); !slices.Equal(canceled, []string{"actors"}) {
			t.Fatalf("canceled = %v; want [actors]", canceled)
		}
		if types := state.canceledTypes; len(types) != 1 || types[0] != "documentAdditionOrUpdate,documentDeletion" {
			t.Fatalf("canceled task types = %v; want document writes only", types)
		}
		if got := recordedSearchVersions(app); !maps.Equal(got, util.SearchDocumentVersions) {
			t.Fatalf("recorded versions = %v; want %v", got, util.SearchDocumentVersions)
		}
	})

	t.Run("a first start records every version", func(t *testing.T) {
		app := newSearchInitApp(t)
		_, client, overflow := newSearchInitClient(t, nil)
		logText, err := runSearchInit(t, app, client, overflow)
		if err != nil {
			t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
		}
		if got := recordedSearchVersions(app); !maps.Equal(got, util.SearchDocumentVersions) {
			t.Fatalf("recorded versions = %v; want %v", got, util.SearchDocumentVersions)
		}
	})
}

func TestInitMeilisearchDocumentsBatchesActorsByThousand(t *testing.T) {
	app := newSearchInitApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1001; i++ {
		actor := core.NewRecord(actors)
		actor.Set("preferred_username", fmt.Sprintf("actor%04d", i))
		if err := app.Save(actor); err != nil {
			t.Fatal(err)
		}
	}

	state, client, overflow := newSearchInitClient(t, nil)
	logText, err := runSearchInit(t, app, client, overflow)
	if err != nil {
		t.Fatalf("initMeilisearchDocuments: %v\n%s", err, logText)
	}
	if got := state.actorBatches(); !slices.Equal(got, []int{1000, 1}) {
		t.Fatalf("actor batch sizes = %v; want [1000 1]", got)
	}
}

// searchInitCreatedAt is the creation time the fake reports for every index
// it was not told otherwise about.
const searchInitCreatedAt = "2026-01-01T00:00:00Z"

// recordSearchIndexVersions records the given versions as built into the
// fake's indexes.
func recordSearchIndexVersions(t *testing.T, app core.App, versions map[string]int) {
	t.Helper()
	states := make(map[string]searchIndexState, len(versions))
	for index, version := range versions {
		states[index] = searchIndexState{Version: version, CreatedAt: searchInitCreatedAt}
	}
	if err := writeSearchIndexVersions(app, states); err != nil {
		t.Fatal(err)
	}
}

// recordedSearchVersions returns the recorded document version per index.
func recordedSearchVersions(app core.App) map[string]int {
	versions := map[string]int{}
	for index, state := range readSearchIndexVersions(app) {
		versions[index] = state.Version
	}
	return versions
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
	lists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 100},
	)
	// The real collections carry these, and the search repair reads updated.
	for _, collection := range []*core.Collection{actors, trails, lists} {
		collection.Fields.Add(
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
	}
	for _, collection := range []*core.Collection{tags, actors, categories, trails, shares, likes, lists} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	listShares := core.NewBaseCollection("list_share")
	listShares.Fields.Add(
		&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
	)
	for _, collection := range []*core.Collection{listShares} {
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
		OrderBy("id ASC").
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
	actorBatchSizes      []int
	rejectedTrailBatches int
	rejectTrailIDs       map[string]bool
	// pendingTasks (document writes), settingsTasks and documentCounts
	// answer the task and stats queries the rebuild check makes, per index.
	pendingTasks      map[string]int64
	settingsTasks     map[string]int64
	documentCounts    map[string]int64
	canceled          []string
	canceledTypes     []string
	failDeleteIndexes map[string]bool
	failStats         bool
	// createdAt overrides the creation time reported per index.
	createdAt map[string]string
}

func newSearchInitClient(t *testing.T, rejectTrailIDs map[string]bool) (*searchInitServer, meilisearch.ServiceManager, <-chan struct{}) {
	t.Helper()
	overflow := make(chan struct{}, 1)
	state := &searchInitServer{
		limit:             12,
		overflow:          overflow,
		rejectTrailIDs:    rejectTrailIDs,
		pendingTasks:      map[string]int64{},
		settingsTasks:     map[string]int64{},
		documentCounts:    map[string]int64{},
		failDeleteIndexes: map[string]bool{},
		createdAt:         map[string]string{},
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

func (state *searchInitServer) canceledIndexes() []string {
	state.mu.Lock()
	defer state.mu.Unlock()
	return slices.Clone(state.canceled)
}

func (state *searchInitServer) actorBatches() []int {
	state.mu.Lock()
	defer state.mu.Unlock()
	return slices.Clone(state.actorBatchSizes)
}

// serveTaskOrStats answers the task, cancel and stats requests, and reports
// whether the request was one of them.
func (state *searchInitServer) serveTaskOrStats(w http.ResponseWriter, r *http.Request) bool {
	index := r.URL.Query().Get("indexUids")
	types := r.URL.Query().Get("types")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/tasks":
		state.mu.Lock()
		var pending int64
		if types == "" || strings.Contains(types, "documentAdditionOrUpdate") {
			pending += state.pendingTasks[index]
		}
		if types == "" || strings.Contains(types, "settingsUpdate") {
			pending += state.settingsTasks[index]
		}
		state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"results":[],"total":%d,"limit":1,"from":null,"next":null}`, pending)
	case r.Method == http.MethodPost && r.URL.Path == "/tasks/cancel":
		state.mu.Lock()
		state.canceled = append(state.canceled, index)
		state.canceledTypes = append(state.canceledTypes, types)
		state.pendingTasks[index] = 0
		state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"taskUid":7,"indexUid":null,"status":"enqueued","type":"taskCancelation","enqueuedAt":"2026-09-19T00:00:00Z"}`))
	case r.Method == http.MethodGet && r.URL.Path == "/tasks/7":
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uid":7,"indexUid":null,"status":"succeeded","type":"taskCancelation","enqueuedAt":"2026-09-19T00:00:00Z"}`))
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/indexes/") && !strings.Contains(strings.TrimPrefix(r.URL.Path, "/indexes/"), "/"):
		index = strings.TrimPrefix(r.URL.Path, "/indexes/")
		if state.failStats {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"unavailable","code":"internal","type":"internal"}`))
			return true
		}
		state.mu.Lock()
		createdAt := state.createdAt[index]
		state.mu.Unlock()
		if createdAt == "" {
			createdAt = searchInitCreatedAt
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"uid":%q,"primaryKey":"id","createdAt":%q,"updatedAt":%q}`, index, createdAt, createdAt)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/indexes/") && strings.HasSuffix(r.URL.Path, "/stats"):
		index = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/indexes/"), "/stats")
		if state.failStats {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"unavailable","code":"internal","type":"internal"}`))
			return true
		}
		state.mu.Lock()
		count := state.documentCounts[index]
		state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"numberOfDocuments":%d,"isIndexing":false,"fieldDistribution":{}}`, count)
	default:
		return false
	}
	return true
}

func (state *searchInitServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if state.serveTaskOrStats(w, r) {
		return
	}
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
	if !overLimit && r.Method == http.MethodPost && indexName == "actors" {
		state.actorBatchSizes = append(state.actorBatchSizes, len(documents))
	}
	state.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if overLimit {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"too many requests","code":"too_many_requests","type":"internal"}`))
		return
	}
	if !overLimit && r.Method == http.MethodDelete && state.failDeleteIndexes[indexName] {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"delete failed","code":"internal","type":"internal"}`))
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
