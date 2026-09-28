package routes

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

// A copy of a remote trail or list outlives the original when the origin's
// Delete never arrived, and the origin cannot always know who holds a copy.
// Refreshing the copy is the chance to notice: an origin that answers 404 or
// 410 no longer has the object for us, so the copy goes and the request gets
// a 404 rather than a 500 or the stale copy.

// goneTransport answers every fetch with one status.
type goneTransport struct{ status int }

func (t goneTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"status":` + strconv.Itoa(t.status) + `,"message":"nope"}`)),
	}, nil
}

type goneFixture struct {
	app        *pbtests.TestApp
	collection *core.Collection
	path       string
	handler    func(*core.RequestEvent) error
}

func setupGoneTest(t *testing.T, status int, collection, path string, handler func(*core.RequestEvent) error) *goneFixture {
	t.Helper()

	t.Setenv("ORIGIN", "https://local.example")
	originalClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: goneTransport{status}}
	}
	t.Cleanup(func() { newRemoteSyncHTTPClient = originalClient })

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "is_local"},
		&core.DateField{Name: "last_fetched"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(actors)
	actor.Set("preferred_username", "alice")
	actor.Set("domain", "remote.example")
	actor.Set("iri", "https://remote.example/actor/alice")
	actor.Set("last_fetched", time.Now())
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}

	c := core.NewBaseCollection(collection)
	c.ViewRule = types.Pointer("")
	c.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
		&core.DateField{Name: "updated"},
	)
	if err := app.Save(c); err != nil {
		t.Fatal(err)
	}

	return &goneFixture{app: app, collection: c, path: path, handler: handler}
}

// cached stores a completed copy of a remote object, flagged for a blocking
// re-sync or aged past the background refresh threshold.
func (f *goneFixture) cached(t *testing.T, needsFullSync bool) *core.Record {
	t.Helper()

	record := core.NewRecord(f.collection)
	record.Set("name", "Cached")
	record.Set("public", true)
	record.Set("full_sync_completed", true)
	record.Set("needs_full_sync", needsFullSync)
	record.Set("updated", time.Now().Add(-remoteSyncThreshold-time.Minute))
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	record.Set("iri", "https://remote.example/api/v1/"+f.path+"/"+record.Id)
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func (f *goneFixture) get(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()

	e := &core.RequestEvent{App: f.app}
	e.Request = httptest.NewRequest(http.MethodGet, "/api/v1/"+f.path+"/"+id+"?handle=alice@remote.example", nil)
	e.Request.SetPathValue("id", id)
	response := httptest.NewRecorder()
	e.Response = response
	if err := f.handler(e); err != nil {
		// Route errors are rendered by PocketBase; here they surface as
		// the returned error, carrying their status.
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatal(err)
		}
		response.Code = apiErr.Status
	}
	return response
}

func (f *goneFixture) waitGone(t *testing.T, id string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := f.app.FindRecordById(f.collection.Name, id); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("copy %s still exists although its origin reports it gone", id)
}

// A 403 is what an origin answers for a trail taken private, since the sync
// is unauthenticated. The copy is kept: dropping it would cascade away the
// comments and summit logs local users left on it. What to show for such a
// copy is left to a later tombstone mechanism; here it only must survive.
func TestRemoteGetKeepsCopyOnForbidden(t *testing.T) {
	for _, needsFullSync := range []bool{true, false} {
		f := setupGoneTest(t, http.StatusForbidden, "trails", "trail", RemoteTrailGet)
		record := f.cached(t, needsFullSync)

		f.get(t, record.Id)
		time.Sleep(200 * time.Millisecond) // a background refresh would drop it asynchronously
		if _, err := f.app.FindRecordById("trails", record.Id); err != nil {
			t.Fatalf("copy dropped on a 403 (needs_full_sync=%v)", needsFullSync)
		}
	}
}

func TestRemoteGetDropsCopyReportedGone(t *testing.T) {
	for _, tt := range []struct {
		collection, path string
		handler          func(*core.RequestEvent) error
	}{
		{"trails", "trail", RemoteTrailGet},
		{"lists", "list", RemoteListGet},
	} {
		for _, status := range []int{http.StatusNotFound, http.StatusGone} {
			name := tt.collection + "/" + strconv.Itoa(status)

			t.Run(name+"/blocking sync answers 404 and drops the copy", func(t *testing.T) {
				f := setupGoneTest(t, status, tt.collection, tt.path, tt.handler)
				record := f.cached(t, true)

				if got := f.get(t, record.Id).Code; got != http.StatusNotFound {
					t.Fatalf("status = %d, want 404", got)
				}
				f.waitGone(t, record.Id)
			})

			t.Run(name+"/never cached answers 404", func(t *testing.T) {
				f := setupGoneTest(t, status, tt.collection, tt.path, tt.handler)

				if got := f.get(t, "unknown00000001").Code; got != http.StatusNotFound {
					t.Fatalf("status = %d, want 404", got)
				}
			})

			t.Run(name+"/background refresh serves the copy once more, then drops it", func(t *testing.T) {
				f := setupGoneTest(t, status, tt.collection, tt.path, tt.handler)
				record := f.cached(t, false)

				if got := f.get(t, record.Id).Code; got != http.StatusOK {
					t.Fatalf("status = %d, want 200 from the cached copy", got)
				}
				f.waitGone(t, record.Id)
			})
		}
	}
}
