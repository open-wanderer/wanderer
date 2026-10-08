package hooks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func newListDeleteTestApp(t *testing.T) (*pbtests.TestApp, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	lists := core.NewBaseCollection("lists")
	rule := ""
	lists.DeleteRule = &rule
	feed := core.NewBaseCollection("feed")
	feed.Fields.Add(&core.TextField{Name: "item"})
	for _, collection := range []*core.Collection{lists, feed} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	list := core.NewRecord(lists)
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	entry := core.NewRecord(feed)
	entry.Set("item", list.Id)
	if err := app.Save(entry); err != nil {
		t.Fatal(err)
	}
	return app, list
}

func TestListDeleteResponseWaitsForSearchIndex(t *testing.T) {
	app, list := newListDeleteTestApp(t)
	secondPoll := make(chan struct{})
	releaseTask := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseTask) }) }
	var polls atomic.Int32
	var deleted atomic.Bool
	searchServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/indexes/lists/documents/"+list.Id:
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"taskUid":73,"status":"enqueued"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/73":
			poll := polls.Add(1)
			if poll == 1 {
				fmt.Fprint(w, `{"uid":73,"status":"processing"}`)
				return
			}
			if poll == 2 {
				close(secondPoll)
			}
			<-releaseTask
			deleted.Store(true)
			fmt.Fprint(w, `{"uid":73,"status":"succeeded"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/lists/search":
			if deleted.Load() {
				fmt.Fprint(w, `{"hits":[]}`)
			} else {
				fmt.Fprintf(w, `{"hits":[{"id":%q}]}`, list.Id)
			}
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(searchServer.Close)
	t.Cleanup(release)
	client := meilisearch.New(searchServer.URL)
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(DeleteListHandler(client))
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/collections/lists/records/"+list.Id, nil))
		close(done)
	}()
	t.Cleanup(func() {
		release()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("DELETE request did not finish during cleanup")
		}
	})
	select {
	case <-secondPoll:
	case <-done:
		t.Fatalf("DELETE returned before the search task completed: status %d", response.Code)
	case <-time.After(3 * time.Second):
		t.Fatal("DELETE did not poll the processing search task again")
	}
	select {
	case <-done:
		t.Fatal("DELETE returned while the search task was still pending")
	default:
	}
	pending, err := client.Index("lists").Search("", &meilisearch.SearchRequest{})
	if err != nil || len(pending.Hits) != 1 {
		t.Fatalf("search before task completion: result=%+v error=%v", pending, err)
	}
	release()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("DELETE did not return after search task completion")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE status %d: %s", response.Code, response.Body.String())
	}
	result, err := client.Index("lists").Search("", &meilisearch.SearchRequest{})
	if err != nil || len(result.Hits) != 0 {
		t.Fatalf("search immediately after DELETE: result=%+v error=%v", result, err)
	}
	if _, err := app.FindRecordById("lists", list.Id); err == nil {
		t.Fatal("list remains in the database")
	}
	entries, err := app.FindAllRecords("feed")
	if err != nil || len(entries) != 0 {
		t.Fatalf("feed cleanup: entries=%d error=%v", len(entries), err)
	}
}

// A public list exercises the post-delete work that must survive an index
// failure: announcing the deletion and removing its entries from the feed.
func newFederatedListDeleteTestApp(t *testing.T) (*pbtests.TestApp, *core.Record, *inboxCounter) {
	t.Helper()
	app := setupActorDeleteHooksTestApp(t)
	inbox := newInboxCounter(t)
	_, author := newDepartingActor(t, app, inbox)
	lists, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	rule := ""
	lists.DeleteRule = &rule
	lists.Fields.Add(&core.TextField{Name: "iri"}, &core.BoolField{Name: "public"})
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}
	feed := core.NewBaseCollection("feed")
	feed.Fields.Add(&core.TextField{Name: "item"})
	if err := app.Save(feed); err != nil {
		t.Fatal(err)
	}
	list := core.NewRecord(lists)
	list.Set("author", author.Id)
	list.Set("public", true)
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	list.Set("iri", "https://local.example/api/v1/list/"+list.Id)
	if err := app.Save(list); err != nil {
		t.Fatal(err)
	}
	entry := core.NewRecord(feed)
	entry.Set("item", list.Id)
	if err := app.Save(entry); err != nil {
		t.Fatal(err)
	}
	return app, list, inbox
}

func bindListDeleteTestHooks(app *pbtests.TestApp, client meilisearch.ServiceManager) *atomic.Bool {
	app.OnRecordDelete("lists").BindFunc(CollectListDeleteRecipientsHandler())
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(DeleteListHandler(client))
	continued := &atomic.Bool{}
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(func(e *core.RecordEvent) error {
		continued.Store(true)
		return e.Next()
	})
	return continued
}

func listDeleteTestRouter(t *testing.T, app *pbtests.TestApp) http.Handler {
	t.Helper()
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return mux
}

func assertListDeletionCompleted(t *testing.T, app *pbtests.TestApp, list *core.Record, inbox *inboxCounter, continued *atomic.Bool) {
	t.Helper()
	if _, err := app.FindRecordById("lists", list.Id); err == nil {
		t.Fatal("list remains in the database")
	}
	entries, err := app.FindAllRecords("feed")
	if err != nil || len(entries) != 0 {
		t.Fatalf("feed cleanup: entries=%d error=%v", len(entries), err)
	}
	activities, err := app.FindAllRecords("activitypub_activities")
	if err != nil || len(activities) != 1 {
		t.Fatalf("delete activities: count=%d error=%v", len(activities), err)
	}
	var object string
	if err := activities[0].UnmarshalJSONField("object", &object); err != nil {
		t.Fatal(err)
	}
	if activities[0].GetString("type") != "Delete" || object != list.GetString("iri") {
		t.Fatalf("unexpected activity: type=%q object=%q", activities[0].GetString("type"), object)
	}
	if !continued.Load() {
		t.Fatal("post-delete hook did not continue to the next handler")
	}
	if hits := inbox.waitForHits(1, 3*time.Second); hits != 1 {
		t.Fatalf("follower received %d Delete deliveries, want 1", hits)
	}
}

func TestListDeleteContinuesAfterSearchFailure(t *testing.T) {
	for _, test := range []struct {
		name, taskStatus        string
		deleteStatus, getStatus int
	}{
		{name: "enqueue rejected", deleteStatus: http.StatusServiceUnavailable},
		{name: "task lookup failed", getStatus: http.StatusServiceUnavailable},
		{name: "task failed", taskStatus: "failed"},
		{name: "task canceled", taskStatus: "canceled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, list, inbox := newFederatedListDeleteTestApp(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodDelete && r.URL.Path == "/indexes/lists/documents/"+list.Id:
					if test.deleteStatus != 0 {
						w.WriteHeader(test.deleteStatus)
						fmt.Fprint(w, `{"message":"search unavailable","code":"unavailable","type":"internal"}`)
						return
					}
					w.WriteHeader(http.StatusAccepted)
					fmt.Fprint(w, `{"taskUid":73,"status":"enqueued"}`)
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/73":
					if test.getStatus != 0 {
						w.WriteHeader(test.getStatus)
						fmt.Fprint(w, `{"message":"search unavailable","code":"unavailable","type":"internal"}`)
						return
					}
					fmt.Fprintf(w, `{"uid":73,"status":%q}`, test.taskStatus)
				default:
					http.Error(w, "unexpected request", http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)
			continued := bindListDeleteTestHooks(app, meilisearch.New(server.URL, meilisearch.DisableRetries()))
			response := httptest.NewRecorder()
			listDeleteTestRouter(t, app).ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/collections/lists/records/"+list.Id, nil))
			if response.Code != http.StatusNoContent {
				t.Fatalf("DELETE status %d: %s", response.Code, response.Body.String())
			}
			assertListDeletionCompleted(t, app, list, inbox, continued)
		})
	}
}

func TestListDeleteSearchWaitHasDeadline(t *testing.T) {
	app, list, inbox := newFederatedListDeleteTestApp(t)
	var polls atomic.Int32
	var release atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/indexes/lists/documents/"+list.Id:
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"taskUid":73,"status":"enqueued"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/73":
			polls.Add(1)
			if release.Load() {
				fmt.Fprint(w, `{"uid":73,"status":"succeeded"}`)
			} else {
				fmt.Fprint(w, `{"uid":73,"status":"processing"}`)
			}
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	continued := bindListDeleteTestHooks(app, meilisearch.New(server.URL, meilisearch.DisableRetries()))
	mux := listDeleteTestRouter(t, app)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	started := time.Now()
	go func() {
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/collections/lists/records/"+list.Id, nil))
		close(done)
	}()
	t.Cleanup(func() {
		// Release an implementation without a deadline as well, so the
		// regression can fail without leaving a request running forever.
		release.Store(true)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("DELETE request did not finish during cleanup")
		}
	})
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("DELETE waited longer than the search deadline")
	}
	if elapsed := time.Since(started); elapsed < 4*time.Second {
		t.Fatalf("DELETE returned after %s without waiting for the search deadline", elapsed)
	}
	if polls.Load() < 2 {
		t.Fatalf("processing search task was only polled %d times", polls.Load())
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE status %d: %s", response.Code, response.Body.String())
	}
	assertListDeletionCompleted(t, app, list, inbox, continued)
}

func TestListDeleteEnqueueUsesParentContext(t *testing.T) {
	app, list, inbox := newFederatedListDeleteTestApp(t)
	startedRequest := make(chan struct{})
	releaseRequest := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/indexes/lists/documents/"+list.Id {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		close(startedRequest)
		select {
		case <-r.Context().Done():
		case <-releaseRequest:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"message":"search unavailable","code":"unavailable","type":"internal"}`)
		}
	}))
	t.Cleanup(server.Close)
	continued := bindListDeleteTestHooks(app, meilisearch.New(server.URL, meilisearch.DisableRetries()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	var deleteErr error
	go func() {
		deleteErr = app.DeleteWithContext(ctx, list)
		close(done)
	}()
	t.Cleanup(func() {
		close(releaseRequest)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("context-bound deletion did not finish during cleanup")
		}
	})
	select {
	case <-startedRequest:
	case <-time.After(2 * time.Second):
		t.Fatal("search deletion was not enqueued")
	}
	cancel()
	select {
	case <-done:
		if deleteErr != nil {
			t.Fatalf("committed deletion failed after context cancellation: %v", deleteErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("search enqueue ignored the parent's cancellation")
	}
	assertListDeletionCompleted(t, app, list, inbox, continued)
}
