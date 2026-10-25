package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type listReferenceFixture struct {
	*remoteSyncFixture
	list     *core.Record
	lock     sync.Mutex
	requests []string
	entries  []any
	trails   map[string]map[string]any
	authors  map[string]string
	origin   func(*http.Request)
}

func newListReferenceFixture(t *testing.T) *listReferenceFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	f := &listReferenceFixture{remoteSyncFixture: newRemoteSyncFixture(t), trails: map[string]map[string]any{}, authors: map[string]string{}}
	f.remoteSyncFixture.trails.Fields.Add(&core.NumberField{Name: "distance"}, &core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	if err := f.app.Save(f.remoteSyncFixture.trails); err != nil {
		t.Fatal(err)
	}
	f.list = f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/list-id", "author": f.remoteActor.Id, "needs_full_sync": true})
	f.authors["remote.example"] = f.remoteActor.GetString("iri")
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: f.serve}}
	}
	return f
}

func (f *listReferenceFixture) serve(request *http.Request) (*http.Response, error) {
	f.lock.Lock()
	f.requests = append(f.requests, request.URL.Host+request.URL.Path)
	f.lock.Unlock()
	respond := func(status int, payload any) (*http.Response, error) {
		body, err := json.Marshal(payload)
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, err
	}
	if author := f.authors[request.URL.Host]; author != "" && request.URL.Path == "/api/v1/list/list-id" {
		return respond(http.StatusOK, map[string]any{"id": "list-id", "iri": "https://" + request.URL.Host + "/api/v1/list/list-id", "name": "List", "public": true,
			"expand": map[string]any{"author": map[string]any{"iri": author}, "trails": f.entries}})
	}
	if f.origin != nil {
		f.origin(request)
	}
	f.lock.Lock()
	payload, found := f.trails[request.URL.Host+request.URL.Path]
	f.lock.Unlock()
	if !found || request.URL.Query().Get("fields") == "" {
		return respond(http.StatusNotFound, map[string]any{"message": "not found"})
	}
	return respond(http.StatusOK, payload)
}

func (f *listReferenceFixture) sync(t *testing.T) []string {
	t.Helper()
	f.lock.Lock()
	f.requests = nil
	f.lock.Unlock()
	if _, err := performFullListSync(f.app, t.Context(), &url.URL{}, f.list); err != nil {
		t.Fatal(err)
	}
	stored, err := f.app.FindRecordById(f.lists, f.list.Id)
	if err != nil {
		t.Fatal(err)
	}
	f.list = stored
	f.lock.Lock()
	defer f.lock.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *listReferenceFixture) third(id, name string, distance float64, author string) string {
	iri := "https://third.example/api/v1/trail/" + id
	f.trails["third.example/api/v1/trail/"+id] = map[string]any{"id": id, "iri": iri, "name": name, "public": true, "distance": distance,
		"expand": map[string]any{"author": map[string]any{"iri": author}}}
	return iri
}

func TestRemoteListLoadsUnknownForeignTrailsAtTheirOrigin(t *testing.T) {
	f := newListReferenceFixture(t)
	carol := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	riley := f.remoteActor.GetString("iri")
	first := f.third("first-trail", "Origin first", 12000, carol.GetString("iri"))
	second := f.third("second-trail", "Origin second", 8000, carol.GetString("iri"))
	// Older origins answer with an empty iri for their own trails.
	f.trails["third.example/api/v1/trail/second-trail"]["iri"] = ""
	f.entries = []any{
		map[string]any{"id": "own-one", "iri": "https://remote.example/api/v1/trail/own-one", "name": "Own one", "public": true, "expand": map[string]any{"author": map[string]any{"iri": riley}}},
		map[string]any{"id": "copy-one", "iri": first, "name": "Listed copy", "distance": 99000.0, "public": true, "expand": map[string]any{"author": map[string]any{"iri": riley}}},
		map[string]any{"id": "own-two", "iri": "https://remote.example/api/v1/trail/own-two", "name": "Own two", "public": true, "expand": map[string]any{"author": map[string]any{"iri": riley}}},
		map[string]any{"id": "copy-two", "iri": second, "name": "Listed copy", "distance": 99000.0, "public": true, "expand": map[string]any{"author": map[string]any{"iri": carol.GetString("iri")}}},
	}
	requests := f.sync(t)
	got := []string{}
	for _, id := range f.list.GetStringSlice("trails") {
		trail, err := f.app.FindRecordById(f.remoteSyncFixture.trails, id)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, trail.GetString("name"))
		if strings.HasPrefix(trail.GetString("iri"), "https://third.example/") &&
			(trail.GetString("author") != carol.Id || !trail.GetBool("needs_full_sync") || trail.GetFloat("distance") == 99000) {
			t.Fatalf("foreign trail not taken from its origin: %v", trail)
		}
	}
	if want := []string{"Own one", "Origin first", "Own two", "Origin second"}; !slices.Equal(got, want) {
		t.Fatalf("list trails = %v, want %v", got, want)
	}
	thirdRequests := 0
	for _, request := range requests {
		if strings.HasPrefix(request, "third.example/") {
			thirdRequests++
		}
	}
	if thirdRequests != 2 {
		t.Fatalf("origin requests = %v", requests)
	}
	// Stored references are refreshed through their own sync, not the list.
	if requests := f.sync(t); slices.ContainsFunc(requests, func(request string) bool { return strings.HasPrefix(request, "third.example/") }) || len(f.list.GetStringSlice("trails")) != 4 {
		t.Fatalf("refresh requests = %v trails = %v", requests, f.list.GetStringSlice("trails"))
	}
}

func TestRemoteListRejectsForeignReferencesTheOriginDoesNotConfirm(t *testing.T) {
	for _, state := range []string{"private", "missing", "other-author-host", "other-object", "local-author"} {
		t.Run(state, func(t *testing.T) {
			f := newListReferenceFixture(t)
			carol := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
			iri := f.third("trail-id", "Origin", 1000, carol.GetString("iri"))
			payload := f.trails["third.example/api/v1/trail/trail-id"]
			switch state {
			case "private":
				payload["public"] = false
			case "missing":
				delete(f.trails, "third.example/api/v1/trail/trail-id")
			case "other-author-host":
				payload["expand"] = map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}
			case "other-object":
				payload["iri"] = "https://third.example/api/v1/trail/other"
			case "local-author":
				payload["expand"] = map[string]any{"author": map[string]any{"iri": f.localActor.GetString("iri")}}
			}
			f.entries = []any{map[string]any{"id": "copy", "iri": iri, "name": "Listed copy", "public": true, "expand": map[string]any{"author": map[string]any{"iri": carol.GetString("iri")}}}}
			f.sync(t)
			if count, _ := f.app.CountRecords(f.remoteSyncFixture.trails); count != 0 || len(f.list.GetStringSlice("trails")) != 0 {
				t.Fatalf("unconfirmed reference stored: count=%d trails=%v", count, f.list.GetStringSlice("trails"))
			}
			if requests := f.sync(t); slices.ContainsFunc(requests, func(request string) bool { return strings.HasPrefix(request, "third.example/") }) {
				t.Fatalf("failed reference requested again before the next refresh: %v", requests)
			}
		})
	}
}

func TestRemoteListBoundsForeignReferenceRequests(t *testing.T) {
	f := newListReferenceFixture(t)
	f.entries = []any{
		map[string]any{"iri": "https://local.example/api/v1/trail/unknown"},
		map[string]any{"iri": "https://local.example:8443/api/v1/trail/unknown"},
		map[string]any{"iri": "http://local.example/api/v1/trail/unknown"},
	}
	for i := range 1000 {
		f.entries = append(f.entries, map[string]any{"iri": fmt.Sprintf("https://peer%d.example/api/v1/trail/trail-%d", i, i)})
	}
	seen := map[string]bool{}
	for round := range 3 {
		requests := f.sync(t)
		peers := 0
		for _, request := range requests {
			if strings.HasPrefix(request, "local.example") {
				t.Fatalf("own host requested: %s", request)
			}
			if !strings.HasPrefix(request, "peer") {
				continue
			}
			peers++
			if seen[request] {
				t.Fatalf("round %d repeated failed reference %s", round, request)
			}
			seen[request] = true
		}
		if peers != remoteListReferenceLimit {
			t.Fatalf("round %d requested %d references, want %d", round, peers, remoteListReferenceLimit)
		}
	}
	if count, _ := f.app.CountRecords(f.remoteSyncFixture.trails); count != 0 {
		t.Fatalf("unconfirmed references stored: %d", count)
	}
}

func TestRemoteListStoresForeignReferencesOnlyForTheirOriginAuthor(t *testing.T) {
	f := newListReferenceFixture(t)
	carol := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	iri := "https://third.example/api/v1/trail/trail-id"
	for _, author := range []*core.Record{f.localActor, f.remoteActor, carol} {
		ctx := context.WithValue(t.Context(), remoteListReferencesKey{}, remoteListReferenceState{confirmed: map[string]remoteListReference{iri: {data: map[string]any{"name": "Origin", "public": true}, iri: iri, author: author.Id}}})
		trail, err := remoteListReferenceRecord(f.app, ctx, f.remoteSyncFixture.trails, iri, nil)
		if err != nil {
			t.Fatal(err)
		}
		if (trail != nil) != (author == carol) {
			t.Fatalf("reference stored for author %s: %v", author.GetString("iri"), trail)
		}
	}
}

func TestRemoteListSharesConcurrentForeignReferenceRequests(t *testing.T) {
	f := newListReferenceFixture(t)
	carol := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	dana := f.save(f.actors, map[string]any{"iri": "https://other.example/api/v1/actor/dana", "last_fetched": time.Now()})
	f.authors["other.example"] = dana.GetString("iri")
	other := f.save(f.lists, map[string]any{"iri": "https://other.example/api/v1/list/list-id", "author": dana.Id, "needs_full_sync": true})
	iri := f.third("trail-id", "Origin", 1000, carol.GetString("iri"))
	f.entries = []any{map[string]any{"iri": iri}}
	arrived, release := make(chan struct{}, 2), make(chan struct{})
	f.origin = func(*http.Request) {
		arrived <- struct{}{}
		<-release
	}
	var wait sync.WaitGroup
	for _, list := range []*core.Record{f.list, other} {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := performFullListSync(f.app, t.Context(), &url.URL{}, list); err != nil {
				t.Error(err)
			}
		}()
	}
	select {
	case <-arrived:
		time.Sleep(200 * time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Error("reference was not requested at its origin")
	}
	close(release)
	wait.Wait()
	if len(arrived) != 0 {
		t.Fatal("concurrent lists requested the same reference twice")
	}
	trail, err := f.app.FindFirstRecordByData(f.remoteSyncFixture.trails, "iri", iri)
	if err != nil {
		t.Fatal(err)
	}
	for _, list := range []*core.Record{f.list, other} {
		stored, err := f.app.FindRecordById(f.lists, list.Id)
		if err != nil || !slices.Equal(stored.GetStringSlice("trails"), []string{trail.Id}) {
			t.Fatalf("list %s trails = %v, error %v", list.GetString("iri"), stored.GetStringSlice("trails"), err)
		}
	}
}

func TestRemoteListReferenceAnswerMustNameItsOwnAuthorAndObject(t *testing.T) {
	for _, kind := range []string{"public", "author-host", "local-author", "object"} {
		t.Run(kind, func(t *testing.T) {
			f := newListReferenceFixture(t)
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
			iri := f.third("trail-id", "Origin", 1000, author.GetString("iri"))
			payload := f.trails["third.example/api/v1/trail/trail-id"]
			switch kind {
			case "public":
				payload["public"] = false
			case "author-host":
				payload["expand"].(map[string]any)["author"] = map[string]any{"iri": f.remoteActor.GetString("iri")}
			case "local-author":
				payload["expand"].(map[string]any)["author"] = map[string]any{"iri": "https://local.example/api/v1/actor/anna"}
			case "object":
				payload["iri"] = "https://third.example/api/v1/trail/other"
			}
			result := loadRemoteListReference(t.Context(), newRemoteSyncHTTPClient(), iri, remoteListReferenceFields)
			if result.answer != remoteListReferenceRefused {
				t.Fatalf("unconfirmed answer accepted: %+v", result)
			}
		})
	}
}
