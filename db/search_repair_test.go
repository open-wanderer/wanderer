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
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

func TestRepairSearchIndexes(t *testing.T) {
	t.Run("current documents are left alone", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		search.store(t, "trails", trailDocuments(t, app, trails)...)
		search.store(t, "actors", actorDocuments(t, actors)...)

		logText := runRepair(t, app, client)
		if writes, deletes := search.changes(); len(writes) != 0 || len(deletes) != 0 {
			t.Fatalf("writes = %v, deletes = %v; want none\n%s", writes, deletes, logText)
		}
		if !strings.Contains(logText, "Search repair trails: 0 documents rewritten, 0 removed") {
			t.Fatalf("log = %s", logText)
		}
	})

	t.Run("a trail made private is rewritten", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		documents := trailDocuments(t, app, trails)
		search.store(t, "trails", documents...)
		search.store(t, "actors", actorDocuments(t, actors)...)

		trails[0].Set("public", false)
		if err := app.UnsafeWithoutHooks().Save(trails[0]); err != nil {
			t.Fatal(err)
		}

		runRepair(t, app, client)
		writes, deletes := search.changes()
		if !sameBatches(writes["trails"], []string{trails[0].Id}) || len(writes["actors"]) != 0 || len(deletes) != 0 {
			t.Fatalf("writes = %v, deletes = %v; want only %s rewritten", writes, deletes, trails[0].Id)
		}
		if public := search.document("trails", trails[0].Id)["public"]; public != false {
			t.Fatalf("stored public = %v; want false", public)
		}
	})

	t.Run("a missing document is added and an orphan removed", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		search.store(t, "trails", trailDocuments(t, app, trails[1:])...)
		search.store(t, "trails", map[string]any{"id": "deletedtrail000"})
		search.store(t, "actors", actorDocuments(t, actors)...)
		search.store(t, "actors", map[string]any{"id": "deletedactor000", "preferred_username": "gone"})

		runRepair(t, app, client)
		writes, deletes := search.changes()
		if !sameBatches(writes["trails"], []string{trails[0].Id}) || len(writes["actors"]) != 0 {
			t.Fatalf("writes = %v; want only %s added", writes, trails[0].Id)
		}
		if !slices.Equal(deletes["trails"], []string{"deletedtrail000"}) || !slices.Equal(deletes["actors"], []string{"deletedactor000"}) {
			t.Fatalf("deletes = %v; want the two orphans", deletes)
		}
	})

	t.Run("actors are compared by name only", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 1)
		search, client := newMemorySearch(t)
		search.store(t, "trails", trailDocuments(t, app, trails)...)
		stored := actorDocuments(t, actors)
		stored[0]["icon"] = "https://example.com/old-icon.png"
		search.store(t, "actors", stored...)

		runRepair(t, app, client)
		if writes, _ := search.changes(); len(writes) != 0 {
			t.Fatalf("writes = %v; want an icon difference ignored", writes)
		}

		stored[0]["preferred_username"] = "old name"
		search.store(t, "actors", stored...)
		runRepair(t, app, client)
		if writes, _ := search.changes(); !sameBatches(writes["actors"], []string{actors[0].Id}) {
			t.Fatalf("writes = %v; want the renamed actor rewritten", writes)
		}
	})

	t.Run("each batch is finished before the next is sent", func(t *testing.T) {
		app, _, actors := newSearchRepairApp(t, int(searchInitPageSize)+1)
		search, client := newMemorySearch(t)
		search.store(t, "actors", actorDocuments(t, actors)...)

		runRepair(t, app, client)
		issued, waited := search.issuedTasks(), search.waitedTasks()
		if len(issued) == 0 || !slices.Equal(waited, issued) {
			t.Fatalf("waited for tasks %v; want every issued task %v", waited, issued)
		}
		writes, _ := search.changes()
		sizes := make([]int, len(writes["trails"]))
		for i, batch := range writes["trails"] {
			sizes[i] = len(batch)
		}
		if !slices.Equal(sizes, []int{int(searchInitPageSize), 1}) {
			t.Fatalf("trail batch sizes = %v; want [%d 1]", sizes, searchInitPageSize)
		}
	})

	t.Run("a rejected document does not stop the repair", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, int(searchInitPageSize)+1)
		search, client := newMemorySearch(t)
		search.store(t, "trails", map[string]any{"id": "deletedtrail000"})
		search.store(t, "actors", actorDocuments(t, actors)...)
		bad := trails[0].Id
		search.rejected["trails"] = map[string]bool{bad: true}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		err := repairSearchIndexes(&searchInitLogApp{App: app, logger: logger}, client)
		if err == nil || !strings.Contains(err.Error(), bad) {
			t.Fatalf("repairSearchIndexes error = %v; want the rejected trail %s", err, bad)
		}
		for _, trail := range trails {
			stored := search.document("trails", trail.Id) != nil
			if stored == (trail.Id == bad) {
				t.Fatalf("trail %s stored = %v; want every trail but the rejected one", trail.Id, stored)
			}
		}
		if search.document("trails", "deletedtrail000") != nil {
			t.Fatal("orphan was not removed after the rejected batch")
		}
	})

	t.Run("an index with queued document tasks is skipped", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		search.store(t, "actors", actorDocuments(t, actors)...)
		search.pending["trails"] = 3
		drainTimeout, pollInterval := searchQueueDrainTimeout, searchQueuePollInterval
		searchQueueDrainTimeout, searchQueuePollInterval = 30*time.Millisecond, 10*time.Millisecond
		t.Cleanup(func() { searchQueueDrainTimeout, searchQueuePollInterval = drainTimeout, pollInterval })

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		err := repairSearchIndexes(&searchInitLogApp{App: app, logger: logger}, client)
		if err == nil || !strings.Contains(err.Error(), "repair search index trails: skipped") {
			t.Fatalf("repairSearchIndexes error = %v; want the skipped trails index reported", err)
		}
		if writes, _ := search.changes(); len(writes) != 0 {
			t.Fatalf("writes = %v; want trails %v left alone", writes, trails)
		}
	})

	t.Run("a panic is logged and releases the indexes", func(t *testing.T) {
		app, _, _ := newSearchRepairApp(t, 1)
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		runSearchRepair(&searchInitLogApp{App: app, logger: logger}, panickingSearchClient{})
		if !strings.Contains(buf.String(), "Search repair panicked") {
			t.Fatalf("log = %s", buf.String())
		}
		if !searchIndexLock.TryLock() {
			t.Fatal("the search indexes are still locked after the panic")
		}
		searchIndexLock.Unlock()
	})

	// makePrivate saves trail as private and applies the partial update the
	// trail hook sends, as a live edit would. The test app binds no hooks.
	makePrivate := func(t *testing.T, app core.App, search *memorySearch, trail *core.Record) {
		t.Helper()
		// Keep the new updated time apart from the one already read.
		time.Sleep(5 * time.Millisecond)
		trail.Set("public", false)
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
		document := search.documents["trails"][trail.Id]
		document["public"] = false
	}

	t.Run("a live write while the stored documents are read is kept", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		documents := trailDocuments(t, app, trails)
		documents[1]["name"] = "drifted"
		search.store(t, "trails", documents...)
		search.store(t, "actors", actorDocuments(t, actors)...)
		search.onFetch = func(index string) {
			if index == "trails" && trails[0].GetBool("public") {
				makePrivate(t, app, search, trails[0])
			}
		}

		runRepair(t, app, client)
		if public := search.document("trails", trails[0].Id)["public"]; public != false {
			t.Fatalf("stored public = %v; want the live edit kept", public)
		}
		if name := search.document("trails", trails[1].Id)["name"]; name == "drifted" {
			t.Fatal("the drifted trail on the same page was not repaired")
		}
	})

	// A like changes the trail's document but not the trail record, so only
	// reading the stored documents before the records protects it.
	t.Run("a like added while the stored documents are read is kept", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		documents := trailDocuments(t, app, trails)
		documents[1]["name"] = "drifted"
		search.store(t, "trails", documents...)
		search.store(t, "actors", actorDocuments(t, actors)...)
		likes, err := app.FindCollectionByNameOrId("trail_like")
		if err != nil {
			t.Fatal(err)
		}
		liked := false
		search.onFetch = func(index string) {
			if index != "trails" || liked {
				return
			}
			liked = true
			like := core.NewRecord(likes)
			like.Set("trail", trails[0].Id)
			like.Set("actor", actors[0].Id)
			if err := app.Save(like); err != nil {
				t.Error(err)
			}
			document := search.documents["trails"][trails[0].Id]
			document["likes"] = []any{actors[0].Id}
			document["like_count"] = 1.0
		}

		runRepair(t, app, client)
		if count := search.document("trails", trails[0].Id)["like_count"]; count != 1.0 {
			t.Fatalf("stored like_count = %v; want the live like kept", count)
		}
	})

	t.Run("a live write after the documents are built is kept", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		documents := trailDocuments(t, app, trails)
		documents[0]["name"] = "drifted"
		documents[1]["name"] = "drifted"
		search.store(t, "trails", documents...)
		search.store(t, "actors", actorDocuments(t, actors)...)
		afterSearchRepairBuild = func() {
			if trails[0].GetBool("public") {
				search.mu.Lock()
				makePrivate(t, app, search, trails[0])
				search.mu.Unlock()
			}
		}
		t.Cleanup(func() { afterSearchRepairBuild = func() {} })

		runRepair(t, app, client)
		if public := search.document("trails", trails[0].Id)["public"]; public != false {
			t.Fatalf("stored public = %v; want the live edit kept", public)
		}
		if name := search.document("trails", trails[1].Id)["name"]; name == "drifted" {
			t.Fatal("the unchanged drifted trail was not repaired")
		}
	})

	t.Run("a trail deleted after the documents are built is not written back", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		search.store(t, "actors", actorDocuments(t, actors)...)
		afterSearchRepairBuild = func() {
			if err := app.UnsafeWithoutHooks().Delete(trails[0]); err != nil {
				t.Error(err)
			}
			afterSearchRepairBuild = func() {}
		}
		t.Cleanup(func() { afterSearchRepairBuild = func() {} })

		runRepair(t, app, client)
		writes, _ := search.changes()
		for _, batch := range writes["trails"] {
			if slices.Contains(batch, trails[0].Id) {
				t.Fatalf("the deleted trail was written back in %v", batch)
			}
		}
		if search.document("trails", trails[0].Id) != nil {
			t.Fatal("the deleted trail is in the index")
		}
		if search.document("trails", trails[1].Id) == nil {
			t.Fatal("the remaining missing trail was not added")
		}
	})

	t.Run("a batch that does not finish in time stops only its index", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, int(searchInitPageSize)+1)
		search, client := newMemorySearch(t)
		search.store(t, "trails", map[string]any{"id": "deletedtrail000"})
		stored := actorDocuments(t, actors)
		stored[0]["preferred_username"] = "old name"
		search.store(t, "actors", stored...)
		search.stuck["trails"] = true
		timeout := searchRepairTaskTimeout
		searchRepairTaskTimeout = 50 * time.Millisecond
		t.Cleanup(func() { searchRepairTaskTimeout = timeout })

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		err := repairSearchIndexes(&searchInitLogApp{App: app, logger: logger}, client)
		if err == nil || !strings.Contains(err.Error(), "did not finish in time") {
			t.Fatalf("repairSearchIndexes error = %v; want the timeout", err)
		}
		writes, deletes := search.changes()
		if len(writes["trails"]) != 1 {
			t.Fatalf("trail batches = %d; want the repair to stop after the first of %d trails", len(writes["trails"]), len(trails))
		}
		if len(deletes["trails"]) != 0 {
			t.Fatalf("deletes = %v; want no orphan scan behind the backlog", deletes)
		}
		if !sameBatches(writes["actors"], []string{actors[0].Id}) {
			t.Fatalf("actor writes = %v; want the other indexes still repaired", writes["actors"])
		}
	})

	t.Run("a share revoked after the documents are built stays revoked", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		shares, err := app.FindCollectionByNameOrId("trail_share")
		if err != nil {
			t.Fatal(err)
		}
		share := core.NewRecord(shares)
		share.Set("trail", trails[0].Id)
		share.Set("actor", actors[0].Id)
		if err := app.Save(share); err != nil {
			t.Fatal(err)
		}
		search, client := newMemorySearch(t)
		documents := trailDocuments(t, app, trails)
		documents[0]["name"] = "drifted"
		search.store(t, "trails", documents...)
		search.store(t, "actors", actorDocuments(t, actors)...)
		afterSearchRepairBuild = func() {
			afterSearchRepairBuild = func() {}
			// The share hook deletes the share and patches the document.
			if err := app.Delete(share); err != nil {
				t.Error(err)
			}
			search.mu.Lock()
			search.documents["trails"][trails[0].Id]["shares"] = []any{}
			search.mu.Unlock()
		}
		t.Cleanup(func() { afterSearchRepairBuild = func() {} })

		runRepair(t, app, client)
		if shared := search.document("trails", trails[0].Id)["shares"]; len(shared.([]any)) != 0 {
			t.Fatalf("stored shares = %v; want the revoked share kept out", shared)
		}
	})

	t.Run("shares stored in hook order are current", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 1)
		actorCollection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			t.Fatal(err)
		}
		shares, err := app.FindCollectionByNameOrId("trail_share")
		if err != nil {
			t.Fatal(err)
		}
		// Shared in the reverse of the order the share hook sorts them in.
		var sharedWith []string
		for _, id := range []string{"zzzzzzzzzzzzzzz", "aaaaaaaaaaaaaaa"} {
			actor := core.NewRecord(actorCollection)
			actor.Id = id
			actor.Set("preferred_username", id)
			if err := app.Save(actor); err != nil {
				t.Fatal(err)
			}
			share := core.NewRecord(shares)
			share.Set("trail", trails[0].Id)
			share.Set("actor", id)
			if err := app.Save(share); err != nil {
				t.Fatal(err)
			}
			sharedWith = append(sharedWith, id)
		}
		search, client := newMemorySearch(t)
		document := trailDocuments(t, app, trails)[0]
		slices.Sort(sharedWith)
		document["shares"] = sharedWith
		search.store(t, "trails", document)
		all, err := app.FindAllRecords("activitypub_actors")
		if err != nil {
			t.Fatal(err)
		}
		search.store(t, "actors", actorDocuments(t, all)...)

		runRepair(t, app, client)
		if writes, _ := search.changes(); len(writes) != 0 {
			t.Fatalf("writes = %v; want sorted shares to compare equal", writes)
		}
		_ = actors
	})

	t.Run("a missing index is filled", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		search, client := newMemorySearch(t)
		search.store(t, "actors", actorDocuments(t, actors)...)
		search.missing["trails"] = true

		runRepair(t, app, client)
		for _, trail := range trails {
			if search.document("trails", trail.Id) == nil {
				t.Fatalf("trail %s was not written to the missing index", trail.Id)
			}
		}
	})

	t.Run("a rejected document is found by halving the batch", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, int(searchInitPageSize))
		search, client := newMemorySearch(t)
		search.store(t, "actors", actorDocuments(t, actors)...)
		bad := trails[37].Id
		search.rejected["trails"] = map[string]bool{bad: true}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		err := repairSearchIndexes(&searchInitLogApp{App: app, logger: logger}, client)
		if err == nil || !strings.Contains(err.Error(), bad) {
			t.Fatalf("repairSearchIndexes error = %v; want the rejected trail %s", err, bad)
		}
		// One full batch, then two halves per level down to the one document.
		if attempts := len(search.issuedTasks()); attempts > 1+2*7 {
			t.Fatalf("write attempts = %d; want the batch halved, not retried per document", attempts)
		}
		for _, trail := range trails {
			if stored := search.document("trails", trail.Id) != nil; stored == (trail.Id == bad) {
				t.Fatalf("trail %s stored = %v; want every trail but the rejected one", trail.Id, stored)
			}
		}
	})

	t.Run("lists are repaired", func(t *testing.T) {
		app, trails, actors := newSearchRepairApp(t, 2)
		lists, err := app.FindCollectionByNameOrId("lists")
		if err != nil {
			t.Fatal(err)
		}
		var saved []*core.Record
		for i := 0; i < 2; i++ {
			list := core.NewRecord(lists)
			list.Set("name", fmt.Sprintf("List %d", i))
			list.Set("author", actors[0].Id)
			list.Set("trails", []string{trails[0].Id, trails[1].Id})
			if err := app.Save(list); err != nil {
				t.Fatal(err)
			}
			saved = append(saved, list)
		}
		slices.SortFunc(saved, func(a, b *core.Record) int { return strings.Compare(a.Id, b.Id) })
		documents, errs := util.ListSearchDocuments(app, saved)
		for _, err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		documents[1]["trails"] = 1
		search, client := newMemorySearch(t)
		search.store(t, "trails", trailDocuments(t, app, trails)...)
		search.store(t, "actors", actorDocuments(t, actors)...)
		search.store(t, "lists", documents...)

		runRepair(t, app, client)
		writes, _ := search.changes()
		if !sameBatches(writes["lists"], []string{saved[1].Id}) || len(writes["trails"]) != 0 {
			t.Fatalf("writes = %v; want only list %s rewritten", writes, saved[1].Id)
		}
		if count := search.document("lists", saved[1].Id)["trails"]; count != 2.0 {
			t.Fatalf("stored trail count = %v; want 2", count)
		}
	})

	t.Run("a run is skipped while the indexes are being written", func(t *testing.T) {
		app, _, _ := newSearchRepairApp(t, 1)
		search, client := newMemorySearch(t)
		searchIndexLock.Lock()
		defer searchIndexLock.Unlock()

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		runSearchRepair(&searchInitLogApp{App: app, logger: logger}, client)
		if requests := search.requestCount(); requests != 0 {
			t.Fatalf("requests = %d; want none while locked", requests)
		}
		if !strings.Contains(buf.String(), "Search repair skipped") {
			t.Fatalf("log = %s", buf.String())
		}
	})
}

func TestReindexSearchTrails(t *testing.T) {
	app, trails, _ := newSearchRepairApp(t, 3)
	search, client := newMemorySearch(t)

	if err := reindexSearchTrails(app, client, []string{trails[1].Id, "deletedtrail000"}); err != nil {
		t.Fatal(err)
	}
	if writes, _ := search.changes(); !sameBatches(writes["trails"], []string{trails[1].Id}) {
		t.Fatalf("writes = %v; want only %s", writes, trails[1].Id)
	}
}

// panickingSearchClient panics on any call, as a nil client would.
type panickingSearchClient struct {
	meilisearch.ServiceManager
}

func sameBatches(got [][]string, want ...[]string) bool {
	return slices.EqualFunc(got, want, slices.Equal[[]string])
}

func runRepair(t *testing.T, app core.App, client meilisearch.ServiceManager) string {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	if err := repairSearchIndexes(&searchInitLogApp{App: app, logger: logger}, client); err != nil {
		t.Fatalf("repairSearchIndexes: %v\n%s", err, buf.String())
	}
	return buf.String()
}

// newSearchRepairApp seeds an author and trailCount public trails.
func newSearchRepairApp(t *testing.T, trailCount int) (*core.BaseApp, []*core.Record, []*core.Record) {
	t.Helper()
	app := newSearchInitApp(t)
	author := seedTrails(t, app, trailCount)
	trails, err := app.FindAllRecords("trails")
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(trails, func(a, b *core.Record) int { return strings.Compare(a.Id, b.Id) })
	return app, trails, []*core.Record{author}
}

func trailDocuments(t *testing.T, app core.App, trails []*core.Record) []map[string]any {
	t.Helper()
	documents := make([]map[string]any, len(trails))
	for i, trail := range trails {
		document, err := util.TrailSearchDocument(app, trail)
		if err != nil {
			t.Fatal(err)
		}
		documents[i] = document
	}
	return documents
}

func actorDocuments(t *testing.T, actors []*core.Record) []map[string]any {
	t.Helper()
	documents := make([]map[string]any, len(actors))
	for i, actor := range actors {
		document, err := util.ActorSearchDocument(actor)
		if err != nil {
			t.Fatal(err)
		}
		documents[i] = document
	}
	return documents
}

// memorySearch is an in-memory stand-in for the Meilisearch document and
// task endpoints the repair uses. Writes are applied at once, and every task
// reports success.
type memorySearch struct {
	mu        sync.Mutex
	documents map[string]map[string]map[string]any
	writes    map[string][][]string
	deletes   map[string][]string
	issued    []int64
	waited    []int64
	requests  int
	nextTask  int64
	// rejected documents make the batch carrying them fail, per index.
	rejected map[string]map[string]bool
	// pending is the number of queued document tasks reported per index.
	pending map[string]int64
	failed  map[int64]bool
	// onFetch runs, with the lock held, when documents are fetched by id.
	onFetch func(index string)
	// missing indexes do not exist until a document is written to them.
	missing map[string]bool
	// stuck indexes never finish their tasks.
	stuck     map[string]bool
	taskIndex map[int64]string
}

func newMemorySearch(t *testing.T) (*memorySearch, meilisearch.ServiceManager) {
	t.Helper()
	search := &memorySearch{
		documents: map[string]map[string]map[string]any{},
		rejected:  map[string]map[string]bool{},
		pending:   map[string]int64{},
		failed:    map[int64]bool{},
		stuck:     map[string]bool{},
		missing:   map[string]bool{},
		taskIndex: map[int64]string{},
	}
	search.reset()
	server := httptest.NewServer(http.HandlerFunc(search.serveHTTP))
	t.Cleanup(server.Close)
	return search, meilisearch.New(server.URL)
}

func (search *memorySearch) reset() {
	search.writes = map[string][][]string{}
	search.deletes = map[string][]string{}
	search.issued = nil
	search.waited = nil
	search.requests = 0
}

// store puts documents into an index as they would come back from
// Meilisearch, and clears the recorded changes.
func (search *memorySearch) store(t *testing.T, index string, documents ...map[string]any) {
	t.Helper()
	search.mu.Lock()
	defer search.mu.Unlock()
	for _, document := range documents {
		normalized, err := normalizeSearchDocument(document)
		if err != nil {
			t.Fatal(err)
		}
		search.put(index, normalized)
	}
	search.reset()
}

func (search *memorySearch) put(index string, document map[string]any) {
	if search.documents[index] == nil {
		search.documents[index] = map[string]map[string]any{}
	}
	id, _ := document["id"].(string)
	search.documents[index][id] = document
}

func (search *memorySearch) document(index, id string) map[string]any {
	search.mu.Lock()
	defer search.mu.Unlock()
	return search.documents[index][id]
}

// changes returns the ids written per batch and the ids deleted, per index,
// and clears them.
func (search *memorySearch) changes() (map[string][][]string, map[string][]string) {
	search.mu.Lock()
	defer search.mu.Unlock()
	writes, deletes := search.writes, search.deletes
	search.reset()
	return writes, deletes
}

func (search *memorySearch) issuedTasks() []int64 {
	search.mu.Lock()
	defer search.mu.Unlock()
	return slices.Clone(search.issued)
}

func (search *memorySearch) waitedTasks() []int64 {
	search.mu.Lock()
	defer search.mu.Unlock()
	return slices.Clone(search.waited)
}

func (search *memorySearch) requestCount() int {
	search.mu.Lock()
	defer search.mu.Unlock()
	return search.requests
}

func (search *memorySearch) serveHTTP(w http.ResponseWriter, r *http.Request) {
	search.mu.Lock()
	defer search.mu.Unlock()
	search.requests++
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && path == "/tasks":
		_, _ = fmt.Fprintf(w, `{"results":[],"total":%d,"limit":1,"from":null,"next":null}`, search.pending[r.URL.Query().Get("indexUids")])

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/tasks/"):
		var uid int64
		fmt.Sscan(strings.TrimPrefix(path, "/tasks/"), &uid)
		search.waited = append(search.waited, uid)
		if search.stuck[search.taskIndex[uid]] {
			_, _ = fmt.Fprintf(w, `{"uid":%d,"status":"processing","type":"documentAdditionOrUpdate","enqueuedAt":"2026-10-06T00:00:00Z"}`, uid)
			return
		}
		if search.failed[uid] {
			_, _ = fmt.Fprintf(w, `{"uid":%d,"status":"failed","type":"documentAdditionOrUpdate","error":{"message":"invalid document","code":"invalid_document_geo_field","type":"invalid_request","link":""},"enqueuedAt":"2026-10-06T00:00:00Z"}`, uid)
			return
		}
		_, _ = fmt.Fprintf(w, `{"uid":%d,"status":"succeeded","type":"documentAdditionOrUpdate","enqueuedAt":"2026-10-06T00:00:00Z"}`, uid)

	case r.Method == http.MethodPost && strings.HasSuffix(path, "/documents/fetch"):
		index := strings.TrimSuffix(strings.TrimPrefix(path, "/indexes/"), "/documents/fetch")
		var query meilisearch.DocumentsQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if search.missing[index] {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, `{"message":"Index %s not found.","code":"index_not_found","type":"invalid_request","link":""}`, index)
			return
		}
		if search.onFetch != nil && query.Ids != nil {
			search.onFetch(index)
		}
		ids := query.Ids
		if ids == nil {
			for id := range search.documents[index] {
				ids = append(ids, id)
			}
			slices.Sort(ids)
		}
		var results []map[string]any
		for _, id := range ids {
			document, ok := search.documents[index][id]
			if !ok {
				continue
			}
			if query.Fields != nil {
				projected := map[string]any{}
				for _, field := range query.Fields {
					if value, ok := document[field]; ok {
						projected[field] = value
					}
				}
				document = projected
			}
			results = append(results, document)
		}
		total := len(results)
		start := min(int(query.Offset), total)
		end := total
		if query.Limit > 0 {
			end = min(start+int(query.Limit), total)
		}
		body, _ := json.Marshal(map[string]any{"results": append([]map[string]any{}, results[start:end]...), "offset": query.Offset, "limit": query.Limit, "total": total})
		_, _ = w.Write(body)

	case r.Method == http.MethodPost && strings.HasSuffix(path, "/documents/delete-batch"):
		index := strings.TrimSuffix(strings.TrimPrefix(path, "/indexes/"), "/documents/delete-batch")
		var ids []string
		if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, id := range ids {
			delete(search.documents[index], id)
		}
		search.deletes[index] = append(search.deletes[index], ids...)
		search.acceptTask(w, index)

	case r.Method == http.MethodPost && strings.HasSuffix(path, "/documents"):
		index := strings.TrimSuffix(strings.TrimPrefix(path, "/indexes/"), "/documents")
		var documents []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&documents); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ids := make([]string, len(documents))
		reject := false
		for i, document := range documents {
			ids[i], _ = document["id"].(string)
			reject = reject || search.rejected[index][ids[i]]
		}
		if reject {
			// Meilisearch applies a batch whole or not at all.
			search.acceptTask(w, index)
			search.failed[search.nextTask] = true
			return
		}
		for _, document := range documents {
			search.put(index, document)
		}
		delete(search.missing, index)
		search.writes[index] = append(search.writes[index], ids)
		search.acceptTask(w, index)

	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"unexpected request","code":"bad_request","type":"invalid_request"}`))
	}
}

func (search *memorySearch) acceptTask(w http.ResponseWriter, index string) {
	search.nextTask++
	search.issued = append(search.issued, search.nextTask)
	search.taskIndex[search.nextTask] = index
	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintf(w, `{"taskUid":%d,"indexUid":%q,"status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2026-10-06T00:00:00Z"}`, search.nextTask, index)
}
