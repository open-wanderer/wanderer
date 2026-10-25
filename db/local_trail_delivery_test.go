package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/federation"
)

func TestLocalTrailEditReachesEachFollowerOnce(t *testing.T) {
	app := newRealSchemaApp(t)
	t.Setenv("ORIGIN", "https://a.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "fde406459dc1f6ca6f348e1f44a9a2af")
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"uid":1,"status":"succeeded"}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(search.Close)
	setupEventHandlers(&pocketbase.PocketBase{App: app}, meilisearch.New(search.URL))
	save := func(collection string, values map[string]any) *core.Record {
		t.Helper()
		col, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatal(err)
		}
		record := core.NewRecord(col)
		record.Load(values)
		if err := app.Save(record); err != nil {
			t.Fatal(collection, err)
		}
		return record
	}
	wait := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		if err := federation.WaitForDeliveries(ctx); err != nil {
			t.Fatal(err)
		}
	}
	actors := map[string]*core.Record{}
	for _, name := range []string{"owner", "first", "second"} {
		user := save("users", map[string]any{"username": name, "email": name + "@a.example", "password": "test-password", "verified": true})
		actor, err := app.FindFirstRecordByData("activitypub_actors", "user", user.Id)
		if err != nil {
			t.Fatal(err)
		}
		actors[name] = actor
		if name != "owner" {
			save("follows", map[string]any{"follower": actor.Id, "followee": actors["owner"].Id, "status": "accepted"})
		}
	}
	trail := save("trails", map[string]any{"name": "Before", "author": actors["owner"].Id, "public": true})
	wait()
	for _, name := range []string{"first", "second"} {
		filter := dbx.HashExp{"item": trail.Id, "actor": actors[name].Id}
		entries, err := app.FindAllRecords("feed", filter)
		if err != nil || len(entries) != 1 {
			t.Fatalf("follower %s before edit: entries=%d err=%v", name, len(entries), err)
		}
		for _, entry := range entries {
			if err := app.Delete(entry); err != nil {
				t.Fatal(err)
			}
		}
		remaining, err := app.CountRecords("feed", filter)
		if err != nil || remaining != 0 {
			t.Fatalf("follower %s before edit: feed=%d err=%v", name, remaining, err)
		}
	}
	const iri = "https://b.example/api/v1/trail/aaaaaaaaaaaaaaa"
	trail.Set("iri", iri)
	if err := app.UnsafeWithoutHooks().Save(trail); err != nil {
		t.Fatal(err)
	}
	var updates atomic.Int32
	app.OnRecordCreate("activitypub_activities").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("type") == "Update" && updates.Add(1) > 3 {
			return errors.New("activity limit")
		}
		return e.Next()
	})
	trail, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	trail.Set("name", "After")
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	wait()
	stored, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	storedUpdates, err := app.CountRecords("activitypub_activities", dbx.HashExp{"type": "Update", "actor": actors["owner"].GetString("iri")})
	if err != nil {
		t.Fatal(err)
	}
	if updates.Load() != 1 || storedUpdates != 1 || stored.GetBool("needs_full_sync") || stored.GetString("name") != "After" || stored.GetString("iri") != iri {
		t.Fatalf("unexpected stored values: attempts=%d updates=%d needs_full_sync=%t name=%q iri=%q", updates.Load(), storedUpdates, stored.GetBool("needs_full_sync"), stored.GetString("name"), stored.GetString("iri"))
	}
	for _, name := range []string{"first", "second"} {
		feed, err := app.CountRecords("feed", dbx.HashExp{"item": trail.Id, "actor": actors[name].Id})
		if err != nil || feed != 1 {
			t.Errorf("follower %s: feed=%d err=%v", name, feed, err)
		}
	}
}
