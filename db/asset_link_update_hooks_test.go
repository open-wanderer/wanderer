package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestChildAssetLinkReplacementFederatesWithoutSearchReindexThroughMainBindings(t *testing.T) {
	for _, target := range []string{"waypoint", "summit_log"} {
		t.Run(target, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			t.Setenv("ORIGIN", "https://a.example")
			author := f.localActor("alice")
			trail := f.save("trails", map[string]any{"name": "Ridge", "author": author.Id, "public": true, "iri": "https://a.example/api/v1/trail/ridge"})
			collection := map[string]string{"waypoint": "waypoints", "summit_log": "summit_logs"}[target]
			child := f.save(collection, map[string]any{"trail": trail.Id, "author": author.Id, "iri": "https://a.example/api/v1/" + target + "/child", "name": "Point", "text": "Summit", "date": "2026-01-01 00:00:00.000Z"})
			first := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id, "file": "first.jpg"})
			second := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id, "file": "replacement.jpg"})
			linkCollection := map[string]string{"waypoint": "waypoint_assets", "summit_log": "summit_log_assets"}[target]
			link := f.save(linkCollection, map[string]any{target: child.Id, "asset": first.Id})
			var indexed atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/indexes/trails/documents" {
					indexed.Add(1)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
			}))
			t.Cleanup(server.Close)
			setupEventHandlers(&pocketbase.PocketBase{App: f.app}, meilisearch.New(server.URL))
			// Asset merge changes this existing link in place. There is no
			// create/delete event to accidentally satisfy either assertion.
			link.Set("asset", second.Id)
			if err := f.app.Save(link); err != nil {
				t.Fatal(err)
			}
			time.Sleep(750 * time.Millisecond)
			if indexed.Load() != 0 {
				t.Fatal("child gallery change reindexed an unchanged search thumbnail")
			}
			if err := f.app.OnTerminate().Trigger(&core.TerminateEvent{App: f.app}); err != nil {
				t.Fatal(err)
			}
			activities := federatedActivities(t, f.app)
			if len(activities) != 1 || !strings.HasPrefix(activities[0], "Update with ") {
				t.Fatalf("main bindings did not announce updated child photo link: %v", activities)
			}
		})
	}
}
