package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// A deleted summit log takes the photos along that nothing else links to,
// whoever added them. The trail reindex and the federation delete run after
// the commit as well; a failing reindex is still reported, but must not leave
// the photos behind.
func TestSummitLogDeleteRemovesOrphanedAssets(t *testing.T) {
	for _, tc := range []struct {
		name       string
		searchDown bool
	}{
		{"search available", false},
		{"search unavailable", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			other := save("activitypub_actors", map[string]any{"preferred_username": "bob",
				"iri": "https://b.example/u/bob", "is_local": false})
			trail := save("trails", map[string]any{"name": "Ridge", "author": actor.Id})
			log := save("summit_logs", map[string]any{"trail": trail.Id, "author": actor.Id, "date": "2026-01-01 00:00:00.000Z"})
			orphan := save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id})
			shared := save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id})
			foreign := save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": other.Id})
			save("summit_log_assets", map[string]any{"asset": orphan.Id, "summit_log": log.Id})
			save("summit_log_assets", map[string]any{"asset": shared.Id, "summit_log": log.Id})
			save("trail_assets", map[string]any{"asset": shared.Id, "trail": trail.Id})
			save("summit_log_assets", map[string]any{"asset": foreign.Id, "summit_log": log.Id})

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tc.searchDown {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"message":"down","code":"internal","type":"internal","link":""}`))
					return
				}
				if r.Method == http.MethodGet {
					w.Write([]byte(`{"uid":1,"status":"succeeded"}`))
					return
				}
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
			}))
			t.Cleanup(server.Close)
			setupEventHandlers(&pocketbase.PocketBase{App: app}, meilisearch.New(server.URL))

			err := app.Delete(log)
			if !tc.searchDown && err != nil {
				t.Fatal(err)
			}

			if _, err := app.FindRecordById("summit_logs", log.Id); err == nil {
				t.Fatal("summit log was not deleted")
			}
			exists := func(id string) bool {
				_, err := app.FindRecordById("assets", id)
				return err == nil
			}
			if exists(orphan.Id) {
				t.Errorf("orphaned asset of the summit log survived its deletion (delete error: %v)", err)
			}
			if exists(foreign.Id) {
				t.Errorf("another author's asset survived the deletion of the summit log it was linked to (delete error: %v)", err)
			}
			if !exists(shared.Id) {
				t.Error("asset still linked to the trail was deleted")
			}
		})
	}
}
