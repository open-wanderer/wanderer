package routes

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// requireLogged fails the test unless the captured log holds every fragment.
func requireLogged(t *testing.T, logged string, fragments ...string) {
	t.Helper()
	for _, want := range fragments {
		if !strings.Contains(logged, want) {
			t.Fatalf("log does not contain %q:\n%s", want, logged)
		}
	}
}

// poisonedJobs is more malformed objects than a pool has workers. A panic that
// ended its worker would leave the pool without workers before every one of
// them was reached.
const poisonedJobs = canonicalRemoteFetchParallelism + 2

// poisonTransport panics for every object whose path names it poison, as a
// malformed remote response that reaches a parser bug would, and answers the
// others. It counts the poison objects that were reached.
type poisonTransport struct {
	reached *atomic.Int32
	good    func(*http.Request) (*http.Response, error)
}

func (p poisonTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if strings.Contains(request.URL.Path, "poison") {
		p.reached.Add(1)
		panic("probe: malformed remote object " + request.URL.Path)
	}
	return p.good(request)
}

// requireAllPoisonReached fails unless every poison object was handed to a
// worker: with one worker lost per panic, at most the pool size would be.
func requireAllPoisonReached(t *testing.T, reached *atomic.Int32) {
	t.Helper()
	if got := reached.Load(); got != poisonedJobs {
		t.Fatalf("poison objects reached = %d, want %d: a panic ended its worker", got, poisonedJobs)
	}
}

// A panic in a worker of a joined pool is not covered by PocketBase's request
// recover, so each pool must recover and log it per job and carry on with the
// rest of the queue. Each pool gets more poison objects than workers plus one
// good object, which must still be processed. The log is read right after the
// pool function returned: the recover runs before the WaitGroup completes,
// which -race checks.
func TestRemoteSyncWorkersRecoverFromPanic(t *testing.T) {
	t.Run("authors", func(t *testing.T) {
		t.Setenv("ORIGIN", "https://local.example")
		f := newRemoteChildFixture(t, "summit_log")
		good := f.save(f.actors, map[string]any{"iri": "https://third.example/users/good", "last_fetched": time.Now()})
		original := remoteSyncGetActorByIRI
		t.Cleanup(func() { remoteSyncGetActorByIRI = original })
		var reached atomic.Int32
		remoteSyncGetActorByIRI = func(app core.App, _ context.Context, iri string, _ bool) (*core.Record, error) {
			if strings.Contains(iri, "poison") {
				reached.Add(1)
				panic("probe: malformed remote actor")
			}
			return app.FindFirstRecordByData("activitypub_actors", "iri", iri)
		}
		buf, logger := newPanicLog()
		app := &panicLogApp{App: f.app, logger: logger}
		iris := []string{good.GetString("iri")}
		for i := 0; i < poisonedJobs; i++ {
			iris = append(iris, fmt.Sprintf("https://third.example/users/poison%d", i))
		}

		got := resolveRemoteAuthorsWithin(app, t.Context(), iris, time.Second, 5*time.Second)
		requireAllPoisonReached(t, &reached)
		if len(got) != 1 || got[good.GetString("iri")] == nil {
			t.Fatalf("authors = %v, want only the good one", got)
		}
		requireLogged(t, buf.String(), "background sync of a remote object panicked", "activitypub_actors", "https://third.example/users/poison0", "probe: malformed remote actor")
	})

	t.Run("comments", func(t *testing.T) {
		t.Setenv("ORIGIN", "https://local.example")
		f := newRemoteChildFixture(t, "summit_log")
		author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
		buf, logger := newPanicLog()
		app := &panicLogApp{App: f.app, logger: logger}
		var reached atomic.Int32
		client := &http.Client{Transport: poisonTransport{reached: &reached, good: func(request *http.Request) (*http.Response, error) {
			return richTextSyncTransport{map[string]any{"id": "good", "iri": "https://third.example/api/v1/comment/good", "text": "Canonical good", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
		}}}
		references := []canonicalCommentReference{{iri: "https://third.example/api/v1/comment/good"}}
		for i := 0; i < poisonedJobs; i++ {
			references = append(references, canonicalCommentReference{iri: fmt.Sprintf("https://third.example/api/v1/comment/poison%d", i)})
		}

		got := fetchCanonicalRemoteComments(app, t.Context(), client, f.trail, references)
		requireAllPoisonReached(t, &reached)
		if len(got) != 1 || got["https://third.example/api/v1/comment/good"].author != author.Id {
			t.Fatalf("comments = %v, want only the good one", got)
		}
		requireLogged(t, buf.String(), "background sync of a remote object panicked", "collection=comments", "https://third.example/api/v1/comment/poison0", "probe: malformed remote object")
		// A panicking object is a failed one: it is not asked for again at once.
		for i := 0; i < poisonedJobs; i++ {
			key := canonicalLogCacheKey(app, f.trail, fmt.Sprintf("https://third.example/api/v1/comment/poison%d", i))
			if !canonicalRemoteLogCache.skip(key, "", "", false) {
				t.Fatalf("poison comment %d was not recorded as failed", i)
			}
		}
	})

	t.Run("summit-logs", func(t *testing.T) {
		t.Setenv("ORIGIN", "https://local.example")
		f := newRemoteChildFixture(t, "summit_log")
		author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
		original := newRemoteSyncHTTPClient
		t.Cleanup(func() { newRemoteSyncHTTPClient = original })
		var reached atomic.Int32
		newRemoteSyncHTTPClient = func() *http.Client {
			return &http.Client{Transport: poisonTransport{reached: &reached, good: func(request *http.Request) (*http.Response, error) {
				return richTextSyncTransport{map[string]any{"id": "good", "iri": "https://third.example/api/v1/summit-log/good", "text": "Canonical good", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
			}}}
		}
		buf, logger := newPanicLog()
		app := &panicLogApp{App: f.app, logger: logger}
		entries := []any{map[string]any{"iri": "https://third.example/api/v1/summit-log/good"}}
		for i := 0; i < poisonedJobs; i++ {
			entries = append(entries, map[string]any{"iri": fmt.Sprintf("https://third.example/api/v1/summit-log/poison%d", i)})
		}

		got := fetchCanonicalRemoteSummitLogs(app, t.Context(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}, f.trail, "https://remote.example")
		requireAllPoisonReached(t, &reached)
		if len(got) != 1 || got["https://third.example/api/v1/summit-log/good"].author != author.Id {
			t.Fatalf("summit logs = %v, want only the good one", got)
		}
		requireLogged(t, buf.String(), "background sync of a remote object panicked", "collection=summit_logs", "https://third.example/api/v1/summit-log/poison0", "probe: malformed remote object")
		for i := 0; i < poisonedJobs; i++ {
			key := canonicalLogCacheKey(app, f.trail, fmt.Sprintf("https://third.example/api/v1/summit-log/poison%d", i))
			if !canonicalRemoteLogCache.skip(key, "", "", false) {
				t.Fatalf("poison summit log %d was not recorded as failed", i)
			}
		}
	})

	t.Run("list-references", func(t *testing.T) {
		t.Setenv("ORIGIN", "https://local.example")
		f := newListViewFixture(t)
		good := f.trail("good", "Good")
		original := newRemoteSyncHTTPClient
		t.Cleanup(func() { newRemoteSyncHTTPClient = original })
		// The fixture installed its own client; wrap it after the fact.
		var reached atomic.Int32
		newRemoteSyncHTTPClient = func() *http.Client {
			return &http.Client{Transport: poisonTransport{reached: &reached, good: f.serve}}
		}
		lists, err := f.app.FindCollectionByNameOrId("lists")
		if err != nil {
			t.Fatal(err)
		}
		list := core.NewRecord(lists)
		list.Set("iri", "https://origin.example/api/v1/list/list-id")
		buf, logger := newPanicLog()
		app := &panicLogApp{App: f.app, logger: logger}
		refs := []any{good}
		for i := 0; i < poisonedJobs; i++ {
			refs = append(refs, map[string]any{"iri": fmt.Sprintf("https://trail.example/api/v1/trail/poison%d", i)})
		}

		ctx := prepareRemoteListReferences(app, t.Context(), list, map[string]any{"expand": map[string]any{"trails": refs}}, "https://origin.example")
		requireAllPoisonReached(t, &reached)
		state := remoteListReferenceStateOf(ctx)
		if _, loaded := state.confirmed[good["iri"].(string)]; !loaded || len(state.confirmed) != 1 {
			t.Fatalf("confirmed references = %v, want only the good one", state.confirmed)
		}
		requireLogged(t, buf.String(), "background sync of a remote object panicked", "collection=trails", "https://trail.example/api/v1/trail/poison0", "probe: malformed remote object")
		// A panicking reference is a refused one: it is not asked for again, and
		// does not keep the list waiting for another sync.
		for i := 0; i < poisonedJobs; i++ {
			if !remoteListReferenceFailed(app, fmt.Sprintf("https://trail.example/api/v1/trail/poison%d", i)) {
				t.Fatalf("poison reference %d was not recorded as failed", i)
			}
		}
	})

	t.Run("list-visibility", func(t *testing.T) {
		f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
		original := newRemoteSyncHTTPClient
		t.Cleanup(func() { newRemoteSyncHTTPClient = original })
		author := f.remoteActor(t, "bob", "third.example")
		var reached atomic.Int32
		newRemoteSyncHTTPClient = func() *http.Client {
			return &http.Client{Transport: poisonTransport{reached: &reached, good: func(request *http.Request) (*http.Response, error) {
				return richTextSyncTransport{map[string]any{"id": "good", "iri": "https://third.example/api/v1/trail/good", "public": true, "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
			}}}
		}
		stored := func(id string) *core.Record {
			trail := f.object(t, "trails", author.Id, "https://third.example", id, false, false)
			trail.Set("public", false)
			if err := f.app.Save(trail); err != nil {
				t.Fatal(err)
			}
			return trail
		}
		good := stored("good")
		refs := []any{map[string]any{"iri": good.GetString("iri")}}
		for i := 0; i < poisonedJobs; i++ {
			refs = append(refs, map[string]any{"iri": stored(fmt.Sprintf("poison%d", i)).GetString("iri")})
		}
		buf, logger := newPanicLog()
		app := &panicLogApp{App: f.app, logger: logger}

		ctx := prepareRemoteListVisibility(app, t.Context(), map[string]any{"expand": map[string]any{"trails": refs}}, "https://remote.example")
		requireAllPoisonReached(t, &reached)
		confirmed, _ := ctx.Value(remoteListVisibilityKey{}).(map[string]remoteTrailVisibility)
		if _, found := confirmed[good.Id]; !found || len(confirmed) != 1 {
			t.Fatalf("visibility confirmed = %v, want only the good trail", confirmed)
		}
		requireLogged(t, buf.String(), "background sync of a remote object panicked", "https://third.example/api/v1/trail/poison0", "probe: malformed remote object")
	})
}
