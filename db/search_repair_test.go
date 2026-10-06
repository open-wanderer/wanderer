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
}

func newMemorySearch(t *testing.T) (*memorySearch, meilisearch.ServiceManager) {
	t.Helper()
	search := &memorySearch{documents: map[string]map[string]map[string]any{}}
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
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/tasks/"):
		var uid int64
		fmt.Sscan(strings.TrimPrefix(path, "/tasks/"), &uid)
		search.waited = append(search.waited, uid)
		_, _ = fmt.Fprintf(w, `{"uid":%d,"status":"succeeded","type":"documentAdditionOrUpdate","enqueuedAt":"2026-10-06T00:00:00Z"}`, uid)

	case r.Method == http.MethodPost && strings.HasSuffix(path, "/documents/fetch"):
		index := strings.TrimSuffix(strings.TrimPrefix(path, "/indexes/"), "/documents/fetch")
		var query meilisearch.DocumentsQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
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
		for i, document := range documents {
			search.put(index, document)
			ids[i], _ = document["id"].(string)
		}
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
	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintf(w, `{"taskUid":%d,"indexUid":%q,"status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2026-10-06T00:00:00Z"}`, search.nextTask, index)
}
