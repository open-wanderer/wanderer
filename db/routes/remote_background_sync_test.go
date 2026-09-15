package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

type backgroundSyncTestApp struct {
	core.App
	synced chan error
}

func (app *backgroundSyncTestApp) RunInTransaction(fn func(core.App) error) error {
	err := app.App.RunInTransaction(fn)
	app.synced <- err
	return err
}

func (app *backgroundSyncTestApp) CanAccessRecord(record *core.Record, info *core.RequestInfo, rule *string) (bool, error) {
	// Finish the background mutation before the handler reads its response.
	// Sharing the record then fails deterministically, without racing JSON export.
	select {
	case err := <-app.synced:
		if err != nil {
			return false, err
		}
	case <-time.After(5 * time.Second):
		return false, fmt.Errorf("background sync did not finish")
	}
	return app.App.CanAccessRecord(record, info, rule)
}

type backgroundSyncTransport struct{}

func (backgroundSyncTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"id":"remote000000001","name":"Refreshed"}`)),
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
			defer app.Cleanup()
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
				&core.TextField{Name: "iri"},
				&core.BoolField{Name: "is_local"},
				&core.DateField{Name: "last_fetched"},
			)
			save(actors)
			actor := core.NewRecord(actors)
			actor.Set("preferred_username", "alice")
			actor.Set("domain", "remote.example")
			actor.Set("iri", "https://remote.example/actor/alice")
			actor.Set("last_fetched", time.Now())
			save(actor)

			collection := core.NewBaseCollection(tt.collection)
			collection.ViewRule = types.Pointer("")
			collection.Fields.Add(
				&core.TextField{Name: "iri"},
				&core.TextField{Name: "name"},
				&core.BoolField{Name: "needs_full_sync"},
				&core.BoolField{Name: "full_sync_completed"},
				&core.DateField{Name: "updated"},
			)
			save(collection)
			record := core.NewRecord(collection)
			record.Set("name", "Cached")
			record.Set("full_sync_completed", true)
			record.Set("updated", time.Now().Add(-remoteSyncThreshold-time.Minute))
			save(record)
			record.Set("iri", "https://remote.example/api/v1/"+tt.path+"/"+record.Id)
			save(record)

			e := &core.RequestEvent{App: &backgroundSyncTestApp{App: app, synced: make(chan error, 1)}}
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
