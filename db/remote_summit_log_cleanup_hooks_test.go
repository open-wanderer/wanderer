package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteSummitLogCleanupWithProductionHooks(t *testing.T) {
	f := newRecordHooksFixture(t)
	remote := f.save("activitypub_actors", map[string]any{"iri": "https://remote.example/users/alice", "preferred_username": "alice"})
	local := f.localActor("local")
	parents := []*core.Record{
		f.save("trails", map[string]any{"author": remote.Id, "name": "Remote A", "iri": "https://remote.example/api/v1/trail/a"}),
		f.save("trails", map[string]any{"author": remote.Id, "name": "Remote B", "iri": "https://remote.example/api/v1/trail/b"}),
	}
	localTrail := f.save("trails", map[string]any{"author": local.Id, "name": "Local"})
	var records, activities []*core.Record
	for i := 0; i < 40; i++ {
		parent := parents[0]
		if i >= 32 {
			parent = parents[1]
		}
		log := f.save("summit_logs", map[string]any{"author": remote.Id, "trail": parent.Id, "iri": fmt.Sprintf("https://remote.example/api/v1/summit-log/%d", i)})
		records = append(records, log)
		activities = append(activities, f.save("activitypub_activities", map[string]any{
			"type": "Create", "iri": fmt.Sprintf("https://remote.example/activity/%d", i), "actor": remote.GetString("iri"),
			"object": map[string]any{"id": log.GetString("iri"), "type": "Note", "content": "Private snapshot", "inReplyTo": parent.GetString("iri")},
		}))
	}
	localLog := f.save("summit_logs", map[string]any{"author": local.Id, "trail": localTrail.Id})
	nestedLocalLog := f.save("summit_logs", map[string]any{"author": local.Id, "trail": localTrail.Id})
	rolledBack := f.save("summit_logs", map[string]any{"author": remote.Id, "trail": parents[0].Id, "iri": "https://remote.example/api/v1/summit-log/rollback"})
	dependents := core.NewBaseCollection("cleanup_dependents")
	dependents.Fields.Add(&core.RelationField{Name: "log", CollectionId: records[0].Collection().Id, MaxSelect: 1, Required: true, CascadeDelete: true})
	if err := f.app.Save(dependents); err != nil {
		t.Fatal(err)
	}
	dependent := f.save(dependents.Name, map[string]any{"log": records[0].Id})
	var posts atomic.Int64
	var mu sync.Mutex
	indexed := map[string]map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/indexes/trails/documents" || r.Method != http.MethodPost {
			t.Errorf("unexpected index request %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
			return
		}
		var docs []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&docs); err != nil {
			t.Error(err)
		}
		mu.Lock()
		for _, doc := range docs {
			if _, full := doc["shares"]; !full {
				t.Error("cleanup must emit a full index document, not a partial update")
			}
			indexed[doc["id"].(string)] = doc
		}
		mu.Unlock()
		posts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"taskUid":1,"status":"enqueued"}`)
	}))
	t.Cleanup(server.Close)
	client := meilisearch.New(server.URL)
	setupEventHandlers(&pocketbase.PocketBase{App: f.app}, client)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := federation.WaitForDeliveries(ctx); err != nil {
			t.Error(err)
		}
	})

	entered, release := make(chan struct{}), make(chan struct{})
	releaseHooks := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseHooks)
	var afterSuccess atomic.Int64
	f.app.OnRecordDelete("summit_logs").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == records[0].Id {
			// A nested ordinary delete must not inherit the batch context.
			if err := e.App.Delete(nestedLocalLog); err != nil {
				return err
			}
		}
		return e.Next()
	})
	var cascaded atomic.Int64
	f.app.OnRecordAfterDeleteSuccess(dependents.Name).BindFunc(func(e *core.RecordEvent) error {
		if util.CollectRemoteSummitLogReindex(e.Context, parents[0].Id, client) {
			t.Error("cascade inherited the cleanup marker")
		}
		cascaded.Add(1)
		return e.Next()
	})
	f.app.OnRecordAfterDeleteSuccess("summit_logs").BindFunc(func(e *core.RecordEvent) error {
		if e.App.TxInfo() != nil {
			t.Error("after-success hook ran before commit")
		}
		if e.Context.Done() != nil {
			t.Error("marker unexpectedly changed app.Delete cancellation semantics")
		}
		afterSuccess.Add(1)
		if e.Record.Id == records[0].Id {
			close(entered)
			<-release
		}
		return e.Next()
	})
	batch := util.NewRemoteSummitLogCleanup(f.app)
	done := make(chan error, 1)
	go func() {
		done <- f.app.RunInTransaction(func(tx core.App) error {
			for _, record := range records {
				if err := batch.Delete(tx, record); err != nil {
					return err
				}
			}
			if posts.Load() != 0 || afterSuccess.Load() != 0 {
				t.Error("cleanup indexed or ran after-success hooks before commit")
			}
			return nil
		})
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("marked transaction failed before after-success hook: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("marked after-success hook did not run")
	}
	// This unmarked local deletion overlaps the marked after-commit hooks. It
	// must still emit its own Delete activity and immediate ordinary reindex.
	localErr := f.app.Delete(localLog)
	releaseHooks()
	if err := <-done; err != nil || localErr != nil {
		t.Fatalf("marked delete=%v concurrent ordinary delete=%v", err, localErr)
	}
	if posts.Load() != 2 || afterSuccess.Load() != int64(len(records)+2) || cascaded.Load() != 1 {
		t.Fatalf("only unmarked deletions should have indexed: posts=%d afterSuccess=%d cascaded=%d", posts.Load(), afterSuccess.Load(), cascaded.Load())
	}
	if _, err := f.app.FindRecordById(dependents.Name, dependent.Id); err == nil {
		t.Fatal("marked deletion bypassed a normal cascade")
	}
	// The eventual reindex must read the fresh committed parent, not capture
	// a transaction app or an earlier record snapshot in a deferred closure.
	if _, err := f.app.DB().NewQuery("UPDATE trails SET name='Fresh committed name' WHERE id={:id}").Bind(map[string]any{"id": parents[0].Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	batch.Flush()
	batch.Flush() // Drained work must not be emitted twice.
	if posts.Load() != 4 {
		t.Fatalf("want one full reindex per remote parent plus ordinary deletion, got %d", posts.Load())
	}
	mu.Lock()
	freshName := indexed[parents[0].Id]["name"]
	mu.Unlock()
	if freshName != "Fresh committed name" {
		t.Fatalf("indexed a stale parent: %v", freshName)
	}
	for _, activity := range activities {
		stored, err := f.app.FindRecordById("activitypub_activities", activity.Id)
		if err != nil {
			t.Fatal(err)
		}
		var object map[string]any
		if err := stored.UnmarshalJSONField("object", &object); err != nil || object["type"] != "Tombstone" || object["content"] != nil || !stored.GetBool("hidden") {
			t.Fatalf("per-record activity cleanup was lost: %v, %v", object, err)
		}
	}
	deletes, err := f.app.FindRecordsByFilter("activitypub_activities", "type='Delete'", "", 0, 0)
	if err != nil || len(deletes) != 2 {
		t.Fatalf("only the ordinary local deletions should emit Delete: %v, %v", deletes, err)
	}
	for _, activity := range deletes {
		if activity.GetString("actor") != local.GetString("iri") {
			t.Fatal("remote copy deletion emitted a local Delete activity")
		}
	}

	abort := errors.New("rollback")
	rollbackBatch := util.NewRemoteSummitLogCleanup(f.app)
	err = f.app.RunInTransaction(func(tx core.App) error {
		if err := rollbackBatch.Delete(tx, rolledBack); err != nil {
			return err
		}
		return abort
	})
	rollbackBatch.Flush()
	if !errors.Is(err, abort) || posts.Load() != 4 || afterSuccess.Load() != int64(len(records)+2) {
		t.Fatalf("rolled-back cleanup ran hooks/reindexed: error=%v posts=%d hooks=%d", err, posts.Load(), afterSuccess.Load())
	}
	if _, err := f.app.FindRecordById("summit_logs", rolledBack.Id); err != nil {
		t.Fatalf("rolled-back log disappeared: %v", err)
	}
}
