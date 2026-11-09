package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// The editor saves the trail before it switches the thumbnail, and the switch
// only updates trail_assets links, so those updates have to reindex the trail.
func TestTrailThumbnailSwitchReindexesTrail(t *testing.T) {
	app := newRealSchemaApp(t)
	save := func(name string, values map[string]any) *core.Record {
		t.Helper()
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		record := core.NewRecord(collection)
		record.Load(values)
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatal(name, err)
		}
		return record
	}
	user := save("users", map[string]any{"email": "alice@a.example", "username": "alice"})
	actor := save("activitypub_actors", map[string]any{"user": user.Id, "preferred_username": "alice",
		"iri": "https://a.example/api/v1/activitypub/user/alice", "is_local": true})
	trail := save("trails", map[string]any{"name": "Ridge", "author": actor.Id})
	first := save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id, "file": "first.jpg"})
	second := save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id, "file": "second.jpg"})
	firstLink := save("trail_assets", map[string]any{"asset": first.Id, "trail": trail.Id, "is_thumbnail": true})
	secondLink := save("trail_assets", map[string]any{"asset": second.Id, "trail": trail.Id})

	var mu sync.Mutex
	var thumbnails []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/indexes/trails/documents" {
			var documents []map[string]any
			_ = json.NewDecoder(r.Body).Decode(&documents)
			mu.Lock()
			for _, document := range documents {
				if document["id"] == trail.Id {
					thumbnail, _ := document["thumbnail"].(string)
					thumbnails = append(thumbnails, thumbnail)
				}
			}
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(server.Close)
	setupEventHandlers(&pocketbase.PocketBase{App: app}, meilisearch.New(server.URL))

	// PATCH /api/v1/trail/{id}/thumbnail updates every link whose mark changes.
	firstLink.Set("is_thumbnail", false)
	if err := app.Save(firstLink); err != nil {
		t.Fatal(err)
	}
	secondLink.Set("is_thumbnail", true)
	if err := app.Save(secondLink); err != nil {
		t.Fatal(err)
	}

	want := "/api/v1/files/" + second.Collection().Id + "/" + second.Id + "/second.jpg"
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		indexed := slices.Clone(thumbnails)
		mu.Unlock()
		if slices.Contains(indexed, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("indexed thumbnails = %q; want a reindex with %s", indexed, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
