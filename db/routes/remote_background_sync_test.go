package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

type backgroundSyncTestApp struct{ core.App }

func waitForBackgroundSync(record *core.Record, timeout time.Duration) error {
	syncing := &trailSyncing
	if record.Collection().Name == "lists" {
		syncing = &listSyncing
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		if _, active := syncing.Load(record.GetString("iri")); !active {
			return nil
		}
		select {
		case <-deadline.C:
			return fmt.Errorf("background sync did not finish")
		case <-tick.C:
		}
	}
}

func (app *backgroundSyncTestApp) CanAccessRecord(record *core.Record, info *core.RequestInfo, rule *string) (bool, error) {
	// Finish the background mutation before the handler reads its response.
	// Sharing the record then fails deterministically, without racing JSON export.
	if err := waitForBackgroundSync(record, 5*time.Second); err != nil {
		return false, err
	}
	return app.App.CanAccessRecord(record, info, rule)
}

type backgroundSyncTransport struct{}

func (backgroundSyncTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	body, err := json.Marshal(map[string]any{
		"id": path.Base(request.URL.Path), "iri": request.URL.Scheme + "://" + request.URL.Host + request.URL.Path,
		"name": "Refreshed", "public": true,
		"expand": map[string]any{"author": map[string]any{"iri": "https://remote.example/actor/alice"}},
	})
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Request:    request,
	}, nil
}

func TestRemoteGetBackgroundSyncKeepsResponse(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	originalClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: backgroundSyncTransport{}}
	}
	t.Cleanup(func() { newRemoteSyncHTTPClient = originalClient })

	for _, tt := range []struct {
		collection string
		path       string
		handler    func(*core.RequestEvent) error
	}{
		{"trails", "trail", RemoteTrailGet},
		{"lists", "list", RemoteListGet},
	} {
		t.Run(tt.collection, func(t *testing.T) {
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

			if tt.collection == "lists" {
				save(core.NewBaseCollection("trails"))
			}

			collection := core.NewBaseCollection(tt.collection)
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
			record.Set("iri", "https://remote.example/api/v1/"+tt.path+"/"+record.Id)
			save(record)

			t.Cleanup(func() {
				if err := waitForBackgroundSync(record, remoteSyncBudget+time.Second); err != nil {
					t.Error(err)
				}
			})

			e := &core.RequestEvent{App: &backgroundSyncTestApp{App: app}}
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
				t.Fatalf("response = %d %s, want original cached record", response.Code, response.Body.String())
			}
			updated, err := app.FindRecordById(tt.collection, record.Id)
			if err != nil {
				t.Fatal(err)
			}
			if updated.GetString("name") != "Refreshed" {
				t.Fatalf("stored name = %q, want Refreshed", updated.GetString("name"))
			}
		})
	}
}
