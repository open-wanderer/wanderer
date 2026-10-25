package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Exercise both post-commit hooks through PocketBase's DELETE endpoint and
// the real Meilisearch HTTP client. Search failures must not turn a committed
// deletion into a failed response or prevent the remaining hook work.
type searchDeleteFixture struct {
	app        *pbtests.TestApp
	handler    http.Handler
	path       string
	deleted    *core.Record
	actor      *core.Record
	trail      *core.Record
	feed       *core.Record
	inbox      *inboxCounter
	downstream atomic.Int32
}

func newSearchDeleteFixture(t *testing.T, kind string, search *searchDeleteServer) *searchDeleteFixture {
	t.Helper()
	f := &searchDeleteFixture{app: setupActorDeleteHooksTestApp(t), inbox: newInboxCounter(t)}
	owner, actor := newDepartingActor(t, f.app, f.inbox)
	f.actor = actor
	trails, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trails.DeleteRule = types.Pointer("")
	if err := f.app.Save(trails); err != nil {
		t.Fatal(err)
	}
	f.trail = core.NewRecord(trails)
	f.trail.Set("author", actor.Id)
	f.trail.Set("public", true)
	if err := f.app.Save(f.trail); err != nil {
		t.Fatal(err)
	}
	search.index = kind
	document := f.actor.Id
	if kind == "trails" {
		document = f.trail.Id
	}
	search.deletePath = fmt.Sprintf("/indexes/%s/documents/%s", kind, document)
	f.trail.Set("iri", "https://local.example/api/v1/trail/"+f.trail.Id)
	if err := f.app.Save(f.trail); err != nil {
		t.Fatal(err)
	}
	if kind == "trails" {
		feed := core.NewBaseCollection("feed")
		feed.Fields.Add(&core.TextField{Name: "item"})
		if err := f.app.Save(feed); err != nil {
			t.Fatal(err)
		}
		f.feed = core.NewRecord(feed)
		f.feed.Set("item", f.trail.Id)
		if err := f.app.Save(f.feed); err != nil {
			t.Fatal(err)
		}
		f.deleted = f.trail
		f.app.OnRecordDelete("trails").BindFunc(CollectTrailDeleteRecipientsHandler())
		f.app.OnRecordAfterDeleteSuccess("trails").BindFunc(DeleteTrailHandler(search.client))
	} else {
		owners, err := f.app.FindCollectionByNameOrId("owners")
		if err != nil {
			t.Fatal(err)
		}
		owners.DeleteRule = types.Pointer("")
		if err := f.app.Save(owners); err != nil {
			t.Fatal(err)
		}
		// The owner DELETE cascades through its actor and trail, as a user
		// account does. The existing fixture registers recipient collection
		// before commit and announcement before the search deletion hook.
		f.deleted = owner
		f.app.OnRecordAfterDeleteSuccess("activitypub_actors").BindFunc(DeleteActorHandler(search.client))
	}
	collection := "activitypub_actors"
	if kind == "trails" {
		collection = "trails"
	}
	f.app.OnRecordAfterDeleteSuccess(collection).BindFunc(func(e *core.RecordEvent) error {
		f.downstream.Add(1)
		return e.Next()
	})
	f.path = fmt.Sprintf("/api/collections/%s/records/%s", f.deleted.Collection().Name, f.deleted.Id)
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	f.handler, err = router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *searchDeleteFixture) delete(ctx context.Context) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, f.path, nil).WithContext(ctx)
	f.handler.ServeHTTP(response, request)
	return response
}

func (f *searchDeleteFixture) assertDeleted(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusNoContent {
		t.Fatalf("DELETE status %d, want 204: %s", response.Code, response.Body.String())
	}
	f.assertDeleteEffects(t)
}

func (f *searchDeleteFixture) assertDeleteEffects(t *testing.T) {
	t.Helper()
	for _, record := range []*core.Record{f.deleted, f.trail} {
		if _, err := f.app.FindRecordById(record.Collection().Name, record.Id); err == nil {
			t.Fatalf("%s/%s survived deletion", record.Collection().Name, record.Id)
		}
	}
	if f.feed != nil {
		if _, err := f.app.FindRecordById("feed", f.feed.Id); err == nil {
			t.Fatal("trail feed entry survived deletion")
		}
	} else if _, err := f.app.FindRecordById("activitypub_actors", f.actor.Id); err == nil {
		t.Fatal("actor survived account cascade")
	}
	if got := f.downstream.Load(); got != 1 {
		t.Fatalf("downstream hook called %d times, want 1", got)
	}
	activities, err := f.app.FindAllRecords("activitypub_activities", dbx.HashExp{"type": "Delete"})
	if err != nil || len(activities) != 1 {
		t.Fatalf("Delete activities: got %d, want 1 (error %v)", len(activities), err)
	}
	if got := f.inbox.waitForHits(1, 5*time.Second); got != 1 {
		t.Fatalf("follower received %d Delete deliveries, want 1", got)
	}
}

// The release gate also makes timeout/cancellation regressions safe to clean
// up if a context is accidentally removed: the fake server never remains
// blocked after its test ends.
type searchDeleteServer struct {
	client     meilisearch.ServiceManager
	index      string
	deletePath string
	entered    chan string
	complete   atomic.Bool
	polled     atomic.Int32
	release    chan struct{}
	close      sync.Once
}

func newSearchDeleteServer(t *testing.T, mode string) *searchDeleteServer {
	t.Helper()
	s := &searchDeleteServer{entered: make(chan string, 128), release: make(chan struct{})}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodDelete && r.URL.Path == s.deletePath) &&
			!(r.Method == http.MethodGet && r.URL.Path == "/tasks/7") {
			t.Errorf("unexpected search request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		stage := "poll"
		if r.Method == http.MethodDelete {
			stage = "enqueue"
		}
		s.entered <- stage
		if mode == "block_"+stage {
			select {
			case <-r.Context().Done():
				return
			case <-s.release:
			}
		}
		if mode == "error_"+stage {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "search unavailable", "code": "unavailable", "type": "internal", "link": "https://example.com/error"})
			return
		}
		if stage == "enqueue" {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"taskUid": 7, "indexUid": s.index, "status": "enqueued", "type": "documentDeletion"})
			return
		}
		s.polled.Add(1)
		status := "succeeded"
		if mode == "processing" && !s.complete.Load() {
			status = "processing"
		} else if mode == "failed" || mode == "canceled" {
			status = mode
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"uid": 7, "indexUid": s.index, "status": status, "type": "documentDeletion"})
	}))
	s.client = meilisearch.New(server.URL, meilisearch.DisableRetries())
	t.Cleanup(func() { s.unblock(); server.Close() })
	return s
}

func (s *searchDeleteServer) unblock() {
	s.close.Do(func() { close(s.release) })
	s.complete.Store(true)
}

func (s *searchDeleteServer) await(t *testing.T, stage string, timeout time.Duration) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case got := <-s.entered:
			if got == stage {
				return
			}
		case <-deadline.C:
			t.Fatalf("search did not reach %s within %s", stage, timeout)
		}
	}
}

func TestSearchDeleteWaitsBeforeResponding(t *testing.T) {
	for _, kind := range []string{"trails", "actors"} {
		t.Run(kind, func(t *testing.T) {
			search := newSearchDeleteServer(t, "processing")
			f := newSearchDeleteFixture(t, kind, search)
			done := make(chan *httptest.ResponseRecorder, 1)
			finished := make(chan struct{})
			t.Cleanup(func() {
				search.unblock()
				select {
				case <-finished:
				case <-time.After(2 * time.Second):
					t.Error("DELETE did not finish during test cleanup")
				}
			})
			go func() {
				defer close(finished)
				done <- f.delete(context.Background())
			}()
			search.await(t, "poll", 2*time.Second)
			// A second poll should not impose the previous 500 ms floor.
			search.await(t, "poll", 400*time.Millisecond)
			select {
			case response := <-done:
				t.Fatalf("DELETE responded before task completion: %d", response.Code)
			default:
			}
			if got := f.downstream.Load(); got != 0 {
				t.Fatalf("downstream hook ran before task completion: %d", got)
			}
			search.unblock()
			select {
			case response := <-done:
				f.assertDeleted(t, response)
			case <-time.After(2 * time.Second):
				t.Fatal("DELETE did not finish after task succeeded")
			}
		})
	}
}

func TestSearchDeleteFailuresDoNotAbortCommittedDeletion(t *testing.T) {
	for _, kind := range []string{"trails", "actors"} {
		for _, mode := range []string{"error_enqueue", "error_poll", "failed", "canceled"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				search := newSearchDeleteServer(t, mode)
				f := newSearchDeleteFixture(t, kind, search)
				f.assertDeleted(t, f.delete(context.Background()))
				if mode == "error_enqueue" && search.polled.Load() != 0 {
					t.Fatal("queried a task after enqueue failed")
				}
			})
		}
	}
}

func TestSearchDeleteHasFiveSecondBudget(t *testing.T) {
	// Cover stalled enqueue and stalled task lookup without sleeping inside
	// the hook or substituting a shorter test-only timeout.
	for _, test := range []struct{ kind, stage string }{{"trails", "enqueue"}, {"actors", "poll"}} {
		t.Run(test.kind+"/"+test.stage, func(t *testing.T) {
			search := newSearchDeleteServer(t, "block_"+test.stage)
			f := newSearchDeleteFixture(t, test.kind, search)
			done := make(chan *httptest.ResponseRecorder, 1)
			start := time.Now()
			go func() { done <- f.delete(context.Background()) }()
			search.await(t, test.stage, 2*time.Second)
			select {
			case response := <-done:
				elapsed := time.Since(start)
				if elapsed < 4500*time.Millisecond || elapsed > 7*time.Second {
					t.Fatalf("search timeout took %s, want approximately 5s", elapsed)
				}
				f.assertDeleted(t, response)
			case <-time.After(7 * time.Second):
				search.unblock()
				<-done
				t.Fatal("DELETE did not bound the stalled search request")
			}
		})
	}
}

func TestSearchDeleteHonorsParentCancellation(t *testing.T) {
	for _, kind := range []string{"trails", "actors"} {
		for _, stage := range []string{"enqueue", "poll"} {
			t.Run(kind+"/"+stage, func(t *testing.T) {
				search := newSearchDeleteServer(t, "block_"+stage)
				f := newSearchDeleteFixture(t, kind, search)
				if kind == "actors" {
					// Cascaded children also get a fresh background context,
					// so cancel a direct actor deletion rather than its owner.
					f.deleted = f.actor
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				// PocketBase's standard REST handler uses app.Delete with a
				// background context; test inherited cancellation explicitly.
				done := make(chan error, 1)
				go func() { done <- f.app.DeleteWithContext(ctx, f.deleted) }()
				search.await(t, stage, 2*time.Second)
				cancel()
				select {
				case err := <-done:
					if err != nil {
						t.Fatalf("committed deletion returned cancellation error: %v", err)
					}
					f.assertDeleteEffects(t)
				case <-time.After(2 * time.Second):
					search.unblock()
					<-done
					t.Fatal("DELETE ignored parent cancellation")
				}
			})
		}
	}
}
