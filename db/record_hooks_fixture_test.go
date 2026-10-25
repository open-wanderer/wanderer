package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pocketbase/federation"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type recordHooksFixture struct {
	t   *testing.T
	app *core.BaseApp
}

func newRecordHooksFixture(t *testing.T) *recordHooksFixture {
	return &recordHooksFixture{t: t, app: newRealSchemaApp(t)}
}

func (f *recordHooksFixture) bindHooks() {
	f.t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"uid":1,"status":"succeeded"}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	f.t.Cleanup(server.Close)
	setupEventHandlers(&pocketbase.PocketBase{App: f.app}, meilisearch.New(server.URL))
	f.t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := federation.WaitForDeliveries(ctx); err != nil {
			f.t.Error(err)
		}
	})
}

func (f *recordHooksFixture) save(name string, values map[string]any) *core.Record {
	f.t.Helper()
	collection, err := f.app.FindCollectionByNameOrId(name)
	if err != nil {
		f.t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Load(values)
	if err := f.app.SaveNoValidate(record); err != nil {
		f.t.Fatal(name, err)
	}
	return record
}

func (f *recordHooksFixture) localActor(name string) *core.Record {
	f.t.Helper()
	user := f.save("users", map[string]any{"email": name + "@example.com", "username": name})
	return f.save("activitypub_actors", map[string]any{"user": user.Id, "preferred_username": name,
		"iri": "https://example.com/api/v1/activitypub/user/" + name, "is_local": true})
}
