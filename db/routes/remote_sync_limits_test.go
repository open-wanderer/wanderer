package routes

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type limitedSyncTransport struct {
	actorIRI string
	entered  chan struct{}
	release  chan struct{}
	calls    atomic.Int32
	active   atomic.Int32
	maximum  atomic.Int32
}

func (s *limitedSyncTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls.Add(1)
	active := s.active.Add(1)
	for {
		old := s.maximum.Load()
		if active <= old || s.maximum.CompareAndSwap(old, active) {
			break
		}
	}
	defer s.active.Add(-1)
	s.entered <- struct{}{}
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-s.release:
	}
	id := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
	return (richTextSyncTransport{map[string]any{"id": id, "iri": req.URL.Scheme + "://" + req.URL.Host + req.URL.Path, "name": "Remote copy", "expand": map[string]any{"author": map[string]any{"iri": s.actorIRI}}}}).RoundTrip(req)
}

func TestRemoteSyncSharesConcurrentCopiesBeforeFetching(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			col := f.trails
			if kind == "list" {
				col = f.lists
			}
			transport := &limitedSyncTransport{actorIRI: f.remoteActor.GetString("iri"), entered: make(chan struct{}, 10), release: make(chan struct{})}
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: transport} }
			var wg sync.WaitGroup
			results := make(chan error, 15)
			const copies = 4
			for i := 0; i < copies; i++ {
				for j := 0; j < 3; j++ {
					record := core.NewRecord(col)
					record.Set("iri", fmt.Sprintf("https://remote.example/api/v1/%s/copy%d", kind, i))
					record.Set("author", f.remoteActor.Id)
					wg.Add(1)
					go func() {
						defer wg.Done()
						var result *core.Record
						var err error
						if kind == "trail" {
							result, err = performFullSync(f.app, t.Context(), nil, record)
						} else {
							result, err = performFullListSync(f.app, t.Context(), nil, record)
						}
						if err == nil && (result == nil || result.Id == "") {
							err = fmt.Errorf("shared sync returned no persisted copy")
						}
						results <- err
					}()
				}
			}
			select {
			case <-transport.entered:
			case <-time.After(3 * time.Second):
				t.Fatal("no sync request")
			}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				remoteSyncFlights.Lock()
				count := len(remoteSyncFlights.calls)
				remoteSyncFlights.Unlock()
				if count == copies {
					break
				}
				time.Sleep(time.Millisecond)
			}
			// Allow the other callers to join the flights while the first fetch is held.
			time.Sleep(30 * time.Millisecond)
			if transport.calls.Load() != 1 {
				t.Errorf("fetched %d documents concurrently", transport.calls.Load())
			}
			close(transport.release)
			wg.Wait()
			close(results)
			for err := range results {
				if err != nil {
					t.Error(err)
				}
			}
			if got := transport.calls.Load(); got != copies {
				t.Fatalf("document requests=%d, want one per IRI (%d)", got, copies)
			}
			if got := transport.maximum.Load(); got != 1 {
				t.Fatalf("concurrent fetches=%d", got)
			}
		})
	}
}

func TestRemoteSyncCancelledWaiterDoesNotStartWork(t *testing.T) {
	f := newRemoteSyncFixture(t)
	record := core.NewRecord(f.trails)
	record.Set("iri", "https://remote.example/api/v1/trail/waiting")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	if _, err := limitedRemoteSync(f.app, ctx, "trails", record, func(context.Context) (*core.Record, error) { called = true; return record, nil }); err == nil || called {
		t.Fatalf("cancelled sync started: called=%v err=%v", called, err)
	}
}

func TestRemoteSyncCapsEachChildCollectionBeforePreparation(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			col := f.trails
			if kind == "list" {
				col = f.lists
				col.Fields.GetByName("trails").(*core.RelationField).MaxSelect = remoteSyncChildLimit
				if err := f.app.Save(col); err != nil {
					t.Fatal(err)
				}
			}
			local := core.NewRecord(col)
			local.Set("iri", "https://remote.example/api/v1/"+kind+"/large")
			local.Set("author", f.remoteActor.Id)
			relations := []string{"waypoints_via_trail", "summit_logs_via_trail"}
			if kind == "list" {
				relations = []string{"trails"}
			}
			expand := map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}
			for _, relation := range relations {
				children := make([]any, remoteSyncChildLimit+25)
				for i := range children {
					children[i] = map[string]any{"id": fmt.Sprintf("child%04d", i)}
				}
				expand[relation] = children
			}
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "large", "iri": local.GetString("iri"), "name": "Large remote copy", "expand": expand}}}
			}
			var err error
			if kind == "trail" {
				_, err = performFullSync(f.app, t.Context(), nil, local)
			} else {
				_, err = performFullListSync(f.app, t.Context(), nil, local)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, collection := range map[string][]string{"trail": {"waypoints", "summit_logs"}, "list": {"trails"}}[kind] {
				n, err := f.app.CountRecords(collection)
				if err != nil || n != remoteSyncChildLimit {
					t.Fatalf("%s stored=%d want%d: %v", collection, n, remoteSyncChildLimit, err)
				}
			}
		})
	}
}

func TestRemoteSyncCappedListKeepsUnprocessedRelations(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.lists.Fields.GetByName("trails").(*core.RelationField).MaxSelect = remoteSyncChildLimit + 100
	if err := f.app.Save(f.lists); err != nil {
		t.Fatal(err)
	}
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "actor"})
	if err := f.app.Save(shares); err != nil {
		t.Fatal(err)
	}
	outside := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/outside", "author": f.remoteActor.Id, "name": "Later trail"})
	removed := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/removed", "author": f.remoteActor.Id})
	local := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/local", "author": f.localActor.Id, "public": false})
	otherActor := f.save(f.actors, map[string]any{"iri": "https://other.example/api/v1/actor/sam"})
	other := f.save(f.trails, map[string]any{"iri": "https://other.example/api/v1/trail/other", "author": otherActor.Id, "public": false})
	localAuthor := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/local-author", "author": f.localActor.Id, "public": false})
	public := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/public", "author": f.localActor.Id, "public": true})
	shared := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/shared", "author": f.localActor.Id, "public": false})
	for _, trail := range []*core.Record{local, other, localAuthor, shared} {
		share := f.save(shares, map[string]any{"trail": trail.Id, "actor": f.remoteActor.Id})
		if trail != shared {
			if err := f.app.Delete(share); err != nil {
				t.Fatal(err)
			}
		}
	}
	previousTrails := []string{outside.Id, removed.Id, local.Id, other.Id, localAuthor.Id, public.Id, shared.Id}
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/large", "author": f.remoteActor.Id, "trails": previousTrails})
	children := make([]any, remoteSyncChildLimit)
	for i := range children {
		children[i] = map[string]any{"id": fmt.Sprintf("child%04d", i)}
	}
	for _, trail := range []*core.Record{outside, local, other, localAuthor, public, shared} {
		children = append(children, map[string]any{"id": "later", "iri": trail.GetString("iri")})
	}
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "large", "iri": list.GetString("iri"), "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "trails": children}}}}
	}
	got, err := performFullListSync(f.app, t.Context(), nil, list)
	if err != nil {
		t.Fatal(err)
	}
	ids := got.GetStringSlice("trails")
	for _, trail := range []*core.Record{outside, public, shared} {
		if !slices.Contains(ids, trail.Id) {
			t.Errorf("capped refresh removed an accessible named relation: %s", trail.GetString("iri"))
		}
	}
	for _, trail := range []*core.Record{removed, local, other, localAuthor} {
		if slices.Contains(ids, trail.Id) {
			t.Errorf("capped refresh retained an omitted or unavailable relation: %s", trail.GetString("iri"))
		}
	}
	if len(ids) != remoteSyncChildLimit+3 {
		t.Fatalf("processed list entries missing or extra: %d", len(ids))
	}
}

func TestRemoteSyncInvalidChildIDKeepsOnlyNamedRelations(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	named := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/named", "author": f.remoteActor.Id})
	omitted := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/omitted", "author": f.remoteActor.Id})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/copy", "author": f.remoteActor.Id, "trails": []string{named.Id, omitted.Id}})
	maxID := f.trails.Fields.GetByName("id").(*core.TextField).Max
	children := []any{map[string]any{"id": strings.Repeat("a", maxID+1), "iri": named.GetString("iri")}, map[string]any{"id": "valid"}}
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "copy", "iri": list.GetString("iri"), "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "trails": children}}}}
	}
	got, err := performFullListSync(f.app, t.Context(), nil, list)
	if err != nil {
		t.Fatal(err)
	}
	if ids := got.GetStringSlice("trails"); len(ids) != 2 || !slices.Contains(ids, named.Id) || slices.Contains(ids, omitted.Id) {
		t.Fatalf("invalid child id retained the wrong earlier relations: %v", ids)
	}
}

func TestRemoteSyncLocalCopiesDoNotWaitForRemoteSlot(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		for _, iri := range []string{"https://local.example/api/v1/", "https://previous.example/api/v1/"} {
			t.Run(kind+"/"+iri, func(t *testing.T) {
				f := newRemoteSyncFixture(t)
				col := f.trails
				if kind == "list" {
					col = f.lists
				}
				record := f.save(col, map[string]any{"iri": iri + kind + "/local", "author": f.localActor.Id, "needs_full_sync": true})
				for i := 0; i < remoteSyncConcurrency; i++ {
					remoteSyncSlots <- struct{}{}
				}
				defer func() {
					for i := 0; i < remoteSyncConcurrency; i++ {
						<-remoteSyncSlots
					}
				}()
				ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
				defer cancel()
				var got *core.Record
				var err error
				if kind == "trail" {
					got, err = performFullSync(f.app, ctx, nil, record)
				} else {
					got, err = performFullListSync(f.app, ctx, nil, record)
				}
				if err != nil || got == nil || got.GetBool("needs_full_sync") {
					t.Fatalf("local copy waited for remote slot: %v %v", got, err)
				}
			})
		}
	}
}
