package util

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestDocumentFromTrailRecordMissingAuthor(t *testing.T) {
	trail := newTrailRecord("trailmissing001", "actorgone000001")

	for _, includeShares := range []bool{false, true} {
		t.Run(includeSharesLabel(includeShares), func(t *testing.T) {
			document, err := documentFromTrailRecord(trail, nil, includeShares)
			if err == nil {
				t.Fatal("expected an error")
			}
			if document != nil {
				t.Fatalf("document = %#v; want nil", document)
			}
			message := err.Error()
			if !strings.Contains(message, trail.Id) || !strings.Contains(message, "actorgone000001") || !strings.Contains(message, "missing author reference") {
				t.Fatalf("error = %q; want trail id and missing author reference", message)
			}
		})
	}
}

func TestDocumentFromTrailRecordAuthors(t *testing.T) {
	scenarios := []struct {
		name          string
		username      string
		local         bool
		domain        string
		wantDomain    string
		federated     bool
		includeShares bool
	}{
		{name: "local with shares", username: "Local Rider", local: true, domain: "ignored.example", includeShares: true},
		{name: "local without shares", username: "Local Rider", local: true, domain: "ignored.example"},
		{name: "remote with shares", username: "Remote Rider", domain: "remote.example", wantDomain: "remote.example", federated: true, includeShares: true},
		{name: "remote without shares", username: "Remote Rider", domain: "remote.example", wantDomain: "remote.example", federated: true},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			author := newActorRecord("actorrecord0001", scenario.username, scenario.local, scenario.domain)
			trail := newTrailRecord("trailrecord0001", author.Id)
			trail.SetExpand(map[string]any{
				"trail_share_via_trail": []*core.Record{newActorRefRecord("trail_share", "shareactor00001")},
				"trail_like_via_trail":  []*core.Record{newActorRefRecord("trail_like", "likeactor000001")},
			})

			document, err := documentFromTrailRecord(trail, author, scenario.includeShares)
			if err != nil {
				t.Fatal(err)
			}
			if document["author"] != author.Id || document["author_name"] != scenario.username {
				t.Fatalf("author = %v %v; want %s %s", document["author"], document["author_name"], author.Id, scenario.username)
			}
			if document["domain"] != scenario.wantDomain || document["is_federated"] != scenario.federated {
				t.Fatalf("domain = %v, is_federated = %v; want %q, %t", document["domain"], document["is_federated"], scenario.wantDomain, scenario.federated)
			}
			assertShareFields(t, document, scenario.includeShares)
		})
	}
}

func TestIndexTrailsRejectsMissingAuthorBeforeSubmit(t *testing.T) {
	app, author := setupTrailIndexApp(t)
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	valid := saveRecord(t, app, trails, map[string]any{"name": "Kept", "author": author.Id})
	removed := saveRecord(t, app, actors, map[string]any{"preferred_username": "Removed author", "is_local": true})
	orphan := saveRecord(t, app, trails, map[string]any{"name": "Orphan", "author": removed.Id})
	result, err := app.NonconcurrentDB().Delete("activitypub_actors", dbx.HashExp{"id": removed.Id}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("deleted actor rows = %d, %v; want 1", affected, err)
	}
	valid, err = app.FindRecordById("trails", valid.Id)
	if err != nil {
		t.Fatal(err)
	}
	orphan, err = app.FindRecordById("trails", orphan.Id)
	if err != nil {
		t.Fatal(err)
	}
	if orphan.GetString("author") != removed.Id {
		t.Fatalf("author reference = %q; want dangling %q", orphan.GetString("author"), removed.Id)
	}

	capture := &meiliCapture{}
	server := httptest.NewServer(http.HandlerFunc(capture.serveHTTP))
	t.Cleanup(server.Close)
	err = IndexTrails(app, []*core.Record{valid, orphan}, meilisearch.New(server.URL))
	if err == nil {
		t.Fatal("expected IndexTrails to return the missing author error")
	}
	message := err.Error()
	if !strings.Contains(message, orphan.Id) || !strings.Contains(message, removed.Id) {
		t.Fatalf("error = %q; want trail id and missing author reference", message)
	}
	if requests := capture.snapshot(); len(requests) != 0 {
		t.Fatalf("submitted %d search requests; want none", len(requests))
	}
}

func TestUpdateTrailAuthor(t *testing.T) {
	app, author := setupTrailIndexApp(t)
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := saveRecord(t, app, trails, map[string]any{"name": "Updated", "author": author.Id, "public": true})
	trail, err = app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("nil author", func(t *testing.T) {
		capture := &meiliCapture{}
		server := httptest.NewServer(http.HandlerFunc(capture.serveHTTP))
		t.Cleanup(server.Close)
		err := UpdateTrail(app, trail, nil, meilisearch.New(server.URL))
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), trail.Id) || !strings.Contains(err.Error(), author.Id) {
			t.Fatalf("error = %q; want trail id and missing author reference", err.Error())
		}
		if requests := capture.snapshot(); len(requests) != 0 {
			t.Fatalf("submitted %d search requests; want none", len(requests))
		}
	})

	t.Run("valid author", func(t *testing.T) {
		capture := &meiliCapture{}
		server := httptest.NewServer(http.HandlerFunc(capture.serveHTTP))
		t.Cleanup(server.Close)
		if err := UpdateTrail(app, trail, author, meilisearch.New(server.URL)); err != nil {
			t.Fatal(err)
		}
		requests := capture.snapshot()
		if len(requests) != 1 || requests[0].method != http.MethodPut || requests[0].path != "/indexes/trails/documents" {
			t.Fatalf("requests = %+v; want one trail document update", requests)
		}
		var documents []map[string]any
		if err := json.Unmarshal(requests[0].body, &documents); err != nil {
			t.Fatal(err)
		}
		if len(documents) != 1 {
			t.Fatalf("documents = %d; want 1", len(documents))
		}
		document := documents[0]
		if document["id"] != trail.Id || document["author"] != author.Id || document["author_name"] != "Search author" {
			t.Fatalf("document author fields = id %v author %v name %v", document["id"], document["author"], document["author_name"])
		}
		if document["domain"] != "" || document["is_federated"] != false {
			t.Fatalf("domain = %v, is_federated = %v; want local author", document["domain"], document["is_federated"])
		}
		if _, ok := document["shares"]; ok {
			t.Fatalf("update included shares: %#v", document["shares"])
		}
	})
}

func includeSharesLabel(includeShares bool) string {
	if includeShares {
		return "include shares"
	}
	return "without shares"
}

func assertShareFields(t *testing.T, document map[string]any, includeShares bool) {
	t.Helper()
	if !includeShares {
		for _, key := range []string{"shares", "likes", "like_count"} {
			if _, ok := document[key]; ok {
				t.Fatalf("%s = %#v; want it omitted", key, document[key])
			}
		}
		return
	}
	shares, _ := document["shares"].([]string)
	likes, _ := document["likes"].([]string)
	if !slices.Equal(shares, []string{"shareactor00001"}) || !slices.Equal(likes, []string{"likeactor000001"}) || document["like_count"] != 1 {
		t.Fatalf("shares = %#v, likes = %#v, like_count = %#v", document["shares"], document["likes"], document["like_count"])
	}
}

func newTrailRecord(id, authorID string) *core.Record {
	actors := core.NewBaseCollection("activitypub_actors")
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "public"},
	)
	record := core.NewRecord(trails)
	record.Id = id
	record.Set("author", authorID)
	record.Set("name", "Ridge line")
	return record
}

func newActorRecord(id, username string, local bool, domain string) *core.Record {
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.TextField{Name: "icon"},
		&core.BoolField{Name: "is_local"},
	)
	record := core.NewRecord(actors)
	record.Id = id
	record.Set("preferred_username", username)
	record.Set("domain", domain)
	record.Set("icon", "icon.png")
	record.Set("is_local", local)
	return record
}

func newActorRefRecord(collection, actorID string) *core.Record {
	reference := core.NewBaseCollection(collection)
	reference.Fields.Add(&core.TextField{Name: "actor"})
	record := core.NewRecord(reference)
	record.Set("actor", actorID)
	return record
}

func setupTrailIndexApp(t *testing.T) (*core.BaseApp, *core.Record) {
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
	for _, collection := range []*core.Collection{tags, actors, categories, trails, shares, likes} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}

	author := saveRecord(t, app, actors, map[string]any{
		"preferred_username": "Search author",
		"is_local":           true,
		"domain":             "ignored.example",
	})
	return app, author
}

func saveRecord(t *testing.T, app core.App, collection *core.Collection, values map[string]any) *core.Record {
	t.Helper()
	record := core.NewRecord(collection)
	record.Load(values)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

type meiliRequest struct {
	method string
	path   string
	body   []byte
}

type meiliCapture struct {
	mu       sync.Mutex
	requests []meiliRequest
}

func (capture *meiliCapture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	capture.mu.Lock()
	capture.requests = append(capture.requests, meiliRequest{method: r.Method, path: r.URL.Path, body: body})
	capture.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"taskUid":1,"indexUid":"trails","status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2026-09-19T00:00:00Z"}`))
}

func (capture *meiliCapture) snapshot() []meiliRequest {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return slices.Clone(capture.requests)
}

// Deleting a bloated activitypub_actors row is the documented cleanup for the
// duplicate-actor migration failure, and it leaves lists pointing at an id that
// no longer resolves, exactly as it does for trails.
func TestDocumentFromListRecordMissingAuthor(t *testing.T) {
	list := newListRecord("listmissing0001", "actorgone000001")

	for _, includeShares := range []bool{false, true} {
		t.Run(includeSharesLabel(includeShares), func(t *testing.T) {
			document, err := documentFromListRecord(list, nil, includeShares)
			if err == nil {
				t.Fatal("expected an error")
			}
			if document != nil {
				t.Fatalf("document = %#v; want nil", document)
			}
			message := err.Error()
			if !strings.Contains(message, list.Id) || !strings.Contains(message, "actorgone000001") || !strings.Contains(message, "missing author reference") {
				t.Fatalf("error = %q; want list id and missing author reference", message)
			}
		})
	}
}

func TestIndexListsRejectsMissingAuthorBeforeSubmit(t *testing.T) {
	app, author := setupListIndexApp(t)
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	valid := saveRecord(t, app, lists, map[string]any{"name": "Kept", "author": author.Id})
	removed := saveRecord(t, app, actors, map[string]any{"preferred_username": "Removed author", "is_local": true})
	orphan := saveRecord(t, app, lists, map[string]any{"name": "Orphan", "author": removed.Id})
	result, err := app.NonconcurrentDB().Delete("activitypub_actors", dbx.HashExp{"id": removed.Id}).Execute()
	if err != nil {
		t.Fatal(err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("deleted actor rows = %d, %v; want 1", affected, err)
	}
	valid, err = app.FindRecordById("lists", valid.Id)
	if err != nil {
		t.Fatal(err)
	}
	orphan, err = app.FindRecordById("lists", orphan.Id)
	if err != nil {
		t.Fatal(err)
	}
	if orphan.GetString("author") != removed.Id {
		t.Fatalf("author reference = %q; want dangling %q", orphan.GetString("author"), removed.Id)
	}

	capture := &meiliCapture{}
	server := httptest.NewServer(http.HandlerFunc(capture.serveHTTP))
	t.Cleanup(server.Close)
	err = IndexLists(app, []*core.Record{valid, orphan}, meilisearch.New(server.URL))
	if err == nil {
		t.Fatal("expected IndexLists to return the missing author error")
	}
	message := err.Error()
	if !strings.Contains(message, orphan.Id) || !strings.Contains(message, removed.Id) {
		t.Fatalf("error = %q; want list id and missing author reference", message)
	}
	if requests := capture.snapshot(); len(requests) != 0 {
		t.Fatalf("submitted %d search requests; want none", len(requests))
	}
}

func newListRecord(id, authorID string) *core.Record {
	actors := core.NewBaseCollection("activitypub_actors")
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "public"},
	)
	record := core.NewRecord(lists)
	record.Id = id
	record.Set("author", authorID)
	record.Set("name", "Weekend loops")
	return record
}

func setupListIndexApp(t *testing.T) (*core.BaseApp, *core.Record) {
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

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.TextField{Name: "icon"},
		&core.BoolField{Name: "is_local"},
	)
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "name"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 100},
	)
	shares := core.NewBaseCollection("list_share")
	shares.Fields.Add(
		&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
	)
	for _, collection := range []*core.Collection{actors, trails, lists, shares} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}

	author := saveRecord(t, app, actors, map[string]any{
		"preferred_username": "Search author",
		"is_local":           true,
		"domain":             "ignored.example",
	})
	return app, author
}
