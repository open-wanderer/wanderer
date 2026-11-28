package main

import (
	"encoding/json"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAssetReindexUsesOnlySearchThumbnailInputsAndDirectTrailLinks(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("index-owner")
	direct := f.save("trails", map[string]any{"name": "Direct", "author": owner.Id})
	destination := f.save("trails", map[string]any{"name": "Destination", "author": owner.Id})
	childTrail := f.save("trails", map[string]any{"name": "Child-only", "author": owner.Id})
	point := f.save("waypoints", map[string]any{"name": "Point", "author": owner.Id, "trail": childTrail.Id})
	photo := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": "first.jpg"})
	second := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": "second.jpg"})
	childPhoto := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": "child.jpg"})
	link := f.save("trail_assets", map[string]any{"trail": direct.Id, "asset": photo.Id, "is_thumbnail": true})
	f.save("trail_assets", map[string]any{"trail": direct.Id, "asset": second.Id})
	f.save("waypoint_assets", map[string]any{"waypoint": point.Id, "asset": childPhoto.Id})
	// Public writes load a fresh record; newly constructed fixture records
	// intentionally have no Original snapshot until they are read from SQL.
	photo, err := f.app.FindRecordById("assets", photo.Id)
	if err != nil {
		t.Fatal(err)
	}
	childPhoto, err = f.app.FindRecordById("assets", childPhoto.Id)
	if err != nil {
		t.Fatal(err)
	}
	link, err = f.app.FindRecordById("trail_assets", link.Id)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	documents := map[string]string{}
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/indexes/trails/documents" {
			var batch []map[string]any
			_ = json.NewDecoder(r.Body).Decode(&batch)
			mu.Lock()
			writes++
			for _, doc := range batch {
				id, _ := doc["id"].(string)
				thumbnail, _ := doc["thumbnail"].(string)
				documents[id] = thumbnail
			}
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	defer server.Close()
	setupEventHandlers(&pocketbase.PocketBase{App: f.app}, meilisearch.New(server.URL))
	defer func() { time.Sleep(time.Second) }()
	photo.Load(map[string]any{"lat": 46, "lon": 8, "taken_at": "2026-10-01T08:00:00Z", "metadata": map[string]any{"exif": map[string]any{"camera": "changed"}, "content_hash": "hash"}})
	if err := f.app.SaveNoValidate(photo); err != nil {
		t.Fatal(err)
	}
	childPhoto.Set("file", "child-replacement.jpg")
	if err := f.app.SaveNoValidate(childPhoto); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if writes != 0 {
		t.Errorf("unrelated asset fields/child-only gallery caused %d search writes", writes)
	}
	mu.Unlock()
	if t.Failed() {
		return
	}
	photo.Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
	if err := f.app.SaveNoValidate(photo); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := documents[direct.Id]
	count := writes
	mu.Unlock()
	if count != 1 || !strings.Contains(got, "/"+second.Id+"/second.jpg") {
		t.Fatal("generated classification did not refresh real-photo preference", count, got)
	}
	photo.Set("metadata", map[string]any{})
	photo.Set("file", "replacement.jpg")
	if err := f.app.SaveNoValidate(photo); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got = documents[direct.Id]
	count = writes
	mu.Unlock()
	if count != 2 || !strings.Contains(got, "/"+photo.Id+"/replacement.jpg") {
		t.Fatal("physical filename change did not update search URL", count, got)
	}
	mu.Lock()
	documents = map[string]string{}
	mu.Unlock()
	link.Set("trail", destination.Id)
	if err := f.app.SaveNoValidate(link); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		oldURL, oldSeen := documents[direct.Id]
		newURL, newSeen := documents[destination.Id]
		mu.Unlock()
		if oldSeen && newSeen {
			if !strings.Contains(oldURL, "/"+second.Id+"/second.jpg") || !strings.Contains(newURL, "/"+photo.Id+"/replacement.jpg") {
				t.Fatal("moving direct link indexed stale thumbnails", oldURL, newURL)
			}
			break
		}
		if time.Now().After(until) {
			t.Fatal("moving a direct link failed to update old and new trail")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
