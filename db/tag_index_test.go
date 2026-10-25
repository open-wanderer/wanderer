package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestTagRenameUpdatesAllReferencedSearchDocuments(t *testing.T) {
	// Cross the production batch boundary, including one unrelated trail.
	fixture := newTagIndexFixture(t, 201)
	fixture.tag.Set("name", "panorama")
	if err := fixture.app.Save(fixture.tag); err != nil {
		t.Fatal(err)
	}
	for _, trail := range fixture.trails {
		document := fixture.index.document(trail.Id)
		if got := stringValues(document["tags"]); !slices.Equal(got, []string{"forest", "panorama"}) {
			t.Fatalf("trail %s tags = %v; want forest and panorama", trail.Id, got)
		}
		if document["author_name"] != "Search author" || document["name"] != trail.GetString("name") || !slices.Equal(stringValues(document["shares"]), []string{fixture.recipient.Id}) {
			t.Errorf("tag rename changed unrelated search fields: %v", document)
		}
	}
	if got := fixture.index.document(fixture.unrelated.Id); !reflect.DeepEqual(got, fixture.unrelatedDocument) {
		t.Errorf("unrelated trail changed: got %v; want %v", got, fixture.unrelatedDocument)
	}
	if _, patched, added := fixture.index.counts(); patched != len(fixture.trails) || added != 0 {
		t.Errorf("document writes: patched %d, added %d; want %d tag patches only", patched, added, len(fixture.trails))
	}
}

type tagIndexFixture struct {
	app               *core.BaseApp
	tag, recipient    *core.Record
	trails            []*core.Record
	unrelated         *core.Record
	unrelatedDocument map[string]any
	index             *tagTestIndex
}

func newTagIndexFixture(t *testing.T, count int) *tagIndexFixture {
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
	actors.Fields.Add(&core.TextField{Name: "preferred_username"}, &core.BoolField{Name: "is_local"})
	categories := core.NewBaseCollection("categories")
	categories.Fields.Add(&core.TextField{Name: "name"})
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
	for _, collection := range []*core.Collection{tags, actors, categories, trails, shares, likes} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	save := func(collection *core.Collection, values map[string]any) *core.Record {
		t.Helper()
		record := core.NewRecord(collection)
		record.Load(values)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	tag := save(tags, map[string]any{"name": "scenic"})
	otherTag := save(tags, map[string]any{"name": "forest"})
	author := save(actors, map[string]any{"preferred_username": "Search author", "is_local": true})
	recipient := save(actors, map[string]any{"preferred_username": "Reader", "is_local": true})
	index := &tagTestIndex{documents: make(map[string]map[string]any)}
	fixture := &tagIndexFixture{app: app, tag: tag, recipient: recipient, index: index}
	for i := 0; i < count; i++ {
		trail := save(trails, map[string]any{"name": fmt.Sprintf("Trail %03d", i), "author": author.Id, "tags": []string{tag.Id, otherTag.Id}})
		save(shares, map[string]any{"trail": trail.Id, "actor": recipient.Id})
		fixture.trails = append(fixture.trails, trail)
		index.documents[trail.Id] = map[string]any{"id": trail.Id, "name": trail.GetString("name"), "author": author.Id, "author_name": "Search author", "shares": []string{recipient.Id}, "tags": []string{"scenic", "forest"}}
	}
	fixture.unrelated = save(trails, map[string]any{"name": "Unrelated", "author": author.Id, "tags": []string{otherTag.Id}})
	fixture.unrelatedDocument = map[string]any{"id": fixture.unrelated.Id, "name": "Unrelated", "tags": []string{"forest"}}
	index.documents[fixture.unrelated.Id] = cloneTagDocument(fixture.unrelatedDocument)
	storedTag, err := app.FindRecordById("tags", tag.Id)
	if err != nil {
		t.Fatal(err)
	}
	fixture.tag = storedTag
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index.serveHTTP(t, w, r)
	}))
	t.Cleanup(server.Close)
	client := meilisearch.New(server.URL)
	// Seed records before registering unrelated creation and federation hooks.
	// Use production registration so a missing tag hook fails the regression test.
	setupEventHandlers(&pocketbase.PocketBase{App: app}, client)
	return fixture
}

type tagTestIndex struct {
	mu             sync.Mutex
	documents      map[string]map[string]any
	requests       int
	patched, added int
	failReads      bool
}

func (index *tagTestIndex) document(id string) map[string]any {
	index.mu.Lock()
	defer index.mu.Unlock()
	return cloneTagDocument(index.documents[id])
}

func (index *tagTestIndex) counts() (requests, patched, added int) {
	index.mu.Lock()
	defer index.mu.Unlock()
	return index.requests, index.patched, index.added
}

func cloneTagDocument(document map[string]any) map[string]any {
	clone := make(map[string]any, len(document))
	for key, value := range document {
		clone[key] = value
	}
	return clone
}

func (index *tagTestIndex) serveHTTP(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	index.mu.Lock()
	defer index.mu.Unlock()
	index.requests++
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/indexes/trails/documents/") {
		if index.failReads {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"message": "unavailable", "code": "internal", "type": "internal"})
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/indexes/trails/documents/")
		document, found := index.documents[id]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "missing document", "code": "document_not_found", "type": "invalid_request"})
			return
		}
		json.NewEncoder(w).Encode(document)
		return
	}
	if r.URL.Path != "/indexes/trails/documents" || (r.Method != http.MethodPut && r.Method != http.MethodPost) {
		t.Errorf("unexpected search request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", http.StatusBadRequest)
		return
	}
	var documents []map[string]any
	if err := json.NewDecoder(r.Body).Decode(&documents); err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, document := range documents {
		id, _ := document["id"].(string)
		if id == "" {
			t.Error("search document has no id")
		}
		if r.Method == http.MethodPut {
			if len(document) != 2 || document["tags"] == nil {
				t.Errorf("tag update must only patch id and tags: %v", document)
			}
			if index.documents[id] == nil {
				t.Errorf("partial update would create incomplete document %s", id)
				index.documents[id] = map[string]any{}
			}
			for key, value := range document {
				index.documents[id][key] = value
			}
			index.patched++
		} else {
			index.documents[id] = document
			index.added++
		}
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{"taskUid": 1, "indexUid": "trails", "status": "enqueued", "type": "documentAdditionOrUpdate", "enqueuedAt": "2026-09-19T00:00:00Z"})
}

func stringValues(value any) []string {
	var result []string
	switch values := value.(type) {
	case []string:
		result = slices.Clone(values)
	case []any:
		for _, value := range values {
			if text, ok := value.(string); ok {
				result = append(result, text)
			}
		}
	}
	slices.Sort(result)
	return result
}

func TestTagRenameRebuildsMissingSearchDocument(t *testing.T) {
	fixture := newTagIndexFixture(t, 1)
	trail := fixture.trails[0]
	delete(fixture.index.documents, trail.Id)
	fixture.tag.Set("name", "panorama")
	if err := fixture.app.Save(fixture.tag); err != nil {
		t.Fatal(err)
	}
	document := fixture.index.document(trail.Id)
	if document["name"] != trail.GetString("name") || document["author"] != trail.GetString("author") || document["author_name"] != "Search author" {
		t.Errorf("missing search document was not fully rebuilt: %v", document)
	}
	if !slices.Equal(stringValues(document["tags"]), []string{"forest", "panorama"}) || !slices.Equal(stringValues(document["shares"]), []string{fixture.recipient.Id}) {
		t.Errorf("rebuilt document is missing current tags or shares: %v", document)
	}
	if public, ok := document["public"].(bool); !ok || public {
		t.Errorf("rebuilt private trail must include public=false: %v", document)
	}
	for _, field := range []string{"likes", "like_count", "_geo", "category_id"} {
		if _, ok := document[field]; !ok {
			t.Errorf("rebuilt document is missing %s", field)
		}
	}
	if _, patched, added := fixture.index.counts(); added != 1 || patched != 0 {
		t.Errorf("document writes: added %d, patched %d; want one complete document", added, patched)
	}
}

func TestTagChangesWithoutSuccessfulRenameDoNotTouchSearch(t *testing.T) {
	for _, operation := range []string{"unchanged name", "rejected rename", "rolled back rename"} {
		t.Run(operation, func(t *testing.T) {
			fixture := newTagIndexFixture(t, 1)
			switch operation {
			case "unchanged name":
				if err := fixture.app.Save(fixture.tag); err != nil {
					t.Fatal(err)
				}
			case "rejected rename":
				rejected := errors.New("reject tag rename")
				fixture.app.OnRecordUpdateExecute("tags").BindFunc(func(e *core.RecordEvent) error {
					return rejected
				})
				fixture.tag.Set("name", "panorama")
				if err := fixture.app.Save(fixture.tag); !errors.Is(err, rejected) {
					t.Fatalf("save error = %v; want intentional rejection", err)
				}
			case "rolled back rename":
				rollback := errors.New("roll back tag rename")
				err := fixture.app.RunInTransaction(func(txApp core.App) error {
					fixture.tag.Set("name", "panorama")
					if err := txApp.Save(fixture.tag); err != nil {
						return err
					}
					return rollback
				})
				if !errors.Is(err, rollback) {
					t.Fatalf("transaction error = %v; want intentional rollback", err)
				}
			}
			if requests, _, _ := fixture.index.counts(); requests != 0 {
				t.Errorf("unsuccessful or unchanged rename made %d search requests", requests)
			}
			stored, err := fixture.app.FindRecordById("tags", fixture.tag.Id)
			if err != nil {
				t.Fatal(err)
			}
			if stored.GetString("name") != "scenic" {
				t.Errorf("stored name = %q; want scenic", stored.GetString("name"))
			}
		})
	}
}

func TestTagRenameSearchReadFailureDoesNotCreatePartialDocument(t *testing.T) {
	fixture := newTagIndexFixture(t, 1)
	fixture.index.failReads = true
	fixture.tag.Set("name", "panorama")
	if err := fixture.app.Save(fixture.tag); err == nil {
		t.Fatal("search failure was not propagated")
	}
	if _, patched, added := fixture.index.counts(); patched != 0 || added != 0 {
		t.Errorf("search read failure caused %d patches and %d replacements", patched, added)
	}
}
