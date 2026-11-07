package routes

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

// panicTransport panics instead of answering, as a malformed remote response
// that reaches a parser bug would.
type panicTransport struct{ value any }

func (p panicTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic(p.value)
}

func panickingClient(value any) func() *http.Client {
	return func() *http.Client { return &http.Client{Transport: panicTransport{value: value}} }
}

// panicLogApp captures what the app logs. It cannot reuse searchInitLogApp,
// which lives in package main.
type panicLogApp struct {
	core.App
	logger *slog.Logger
}

func (app *panicLogApp) Logger() *slog.Logger { return app.logger }

func newPanicLog() (*bytes.Buffer, *slog.Logger) {
	var buf bytes.Buffer
	return &buf, slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// newBackgroundPanicFixture stores a cached remote record of the given
// collection that is older than remoteSyncThreshold, so viewing it starts a
// background sync.
func newBackgroundPanicFixture(t *testing.T, collectionName, path string) (core.App, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	save := func(model core.Model) {
		t.Helper()
		if err := app.Save(model); err != nil {
			t.Fatal(err)
		}
	}

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.TextField{Name: "public_key"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "is_local"},
		&core.DateField{Name: "last_fetched"},
	)
	save(actors)
	actor := core.NewRecord(actors)
	actor.Set("preferred_username", "alice")
	actor.Set("domain", "remote.example")
	actor.Set("public_key", "cached-test-key")
	actor.Set("iri", "https://remote.example/actor/alice")
	actor.Set("last_fetched", time.Now())
	save(actor)

	if collectionName == "lists" {
		save(core.NewBaseCollection("trails"))
	}

	collection := core.NewBaseCollection(collectionName)
	collection.ViewRule = types.Pointer("public=true")
	collection.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "name"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.BoolField{Name: "public"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
		&core.DateField{Name: "updated"},
	)
	save(collection)
	record := core.NewRecord(collection)
	record.Set("name", "Cached")
	record.Set("author", actor.Id)
	record.Set("public", true)
	record.Set("full_sync_completed", true)
	record.Set("updated", time.Now().Add(-remoteSyncThreshold-time.Minute))
	save(record)
	record.Set("iri", "https://remote.example/api/v1/"+path+"/"+record.Id)
	save(record)
	return app, record
}

func TestRemoteGetBackgroundSyncRecoversFromPanic(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")

	for _, tt := range []struct {
		collection string
		path       string
		handler    func(*core.RequestEvent) error
	}{
		{"trails", "trail", RemoteTrailGet},
		{"lists", "list", RemoteListGet},
	} {
		for _, pv := range []struct {
			name  string
			value any
			want  string
		}{
			{"string", "probe: malformed remote object", "probe: malformed remote object"},
			{"nil", nil, "panic called with nil argument"},
		} {
			t.Run(tt.collection+"/"+pv.name, func(t *testing.T) {
				originalClient := newRemoteSyncHTTPClient
				newRemoteSyncHTTPClient = panickingClient(pv.value)
				t.Cleanup(func() { newRemoteSyncHTTPClient = originalClient })

				app, record := newBackgroundPanicFixture(t, tt.collection, tt.path)
				t.Cleanup(func() {
					if err := waitForBackgroundSync(record, remoteSyncBudget+time.Second); err != nil {
						t.Error(err)
					}
				})
				buf, logger := newPanicLog()

				e := &core.RequestEvent{App: &panicLogApp{App: &backgroundSyncTestApp{App: app}, logger: logger}}
				e.Request = httptest.NewRequest(http.MethodGet, "/api/v1/"+tt.path+"/"+record.Id+"?handle=alice@remote.example", nil)
				e.Request.SetPathValue("id", record.Id)
				response := httptest.NewRecorder()
				e.Response = response
				if err := tt.handler(e); err != nil {
					t.Fatal(err)
				}
				var body struct{ Name string }
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if response.Code != http.StatusOK || body.Name != "Cached" {
					t.Fatalf("response = %d %s, want the cached record", response.Code, response.Body.String())
				}

				// The iri is released only after the recover logged, so the
				// buffer is safe to read once the sync is no longer active.
				if err := waitForBackgroundSync(record, 5*time.Second); err != nil {
					t.Fatal(err)
				}
				logged := buf.String()
				for _, want := range []string{
					"background sync of a remote object panicked",
					"collection=" + tt.collection,
					record.GetString("iri"),
					pv.want,
					"stack=",
				} {
					if !strings.Contains(logged, want) {
						t.Fatalf("log does not contain %q:\n%s", want, logged)
					}
				}

				stored, err := app.FindRecordById(tt.collection, record.Id)
				if err != nil {
					t.Fatal(err)
				}
				if stored.GetString("name") != "Cached" {
					t.Fatalf("stored name = %q, want Cached", stored.GetString("name"))
				}
				syncing := &trailSyncing
				if tt.collection == "lists" {
					syncing = &listSyncing
				}
				if _, held := syncing.Load(record.GetString("iri")); held {
					t.Fatal("iri still held in the syncing map after the recovered panic")
				}
			})
		}
	}
}
