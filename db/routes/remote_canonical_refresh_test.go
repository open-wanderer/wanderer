package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestCanonicalFetchesHaveIndependentTimeoutsParallelCapacityAndNegativeCache(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
	previousClient, previousTimeout := newRemoteSyncHTTPClient, canonicalRemoteFetchTimeout
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient; canonicalRemoteFetchTimeout = previousTimeout })
	canonicalRemoteFetchTimeout = 30 * time.Millisecond
	var active, peak, requests atomic.Int32
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			requests.Add(1)
			if strings.HasPrefix(request.URL.Path, "/api/v1/summit-log/slow") {
				current := active.Add(1)
				defer active.Add(-1)
				for old := peak.Load(); current > old; old = peak.Load() {
					if peak.CompareAndSwap(old, current) {
						break
					}
				}
				<-request.Context().Done()
				return nil, request.Context().Err()
			}
			return richTextSyncTransport{map[string]any{"id": "fast", "iri": "https://third.example/api/v1/summit-log/fast", "text": "Canonical fast", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
		}}}
	}
	entries := []any{}
	for i := 0; i < 4; i++ {
		entries = append(entries, map[string]any{"id": fmt.Sprintf("slow%d", i), "iri": fmt.Sprintf("https://third.example/api/v1/summit-log/slow%d", i), "updated": "one"})
	}
	entries = append(entries, map[string]any{"id": "fast", "iri": "https://third.example/api/v1/summit-log/fast"})
	data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}
	result := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
	if peak.Load() != 4 || len(result) != 1 || result["https://third.example/api/v1/summit-log/fast"].author != author.Id {
		t.Fatalf("slow hosts displaced fast entry or lost bounded parallelism: peak%d results%v", peak.Load(), result)
	}
	before := requests.Load()
	result = fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
	if requests.Load() != before+1 || len(result) != 1 {
		t.Fatalf("failed references refetched immediately: before%d after%d results%v", before, requests.Load(), result)
	}
	entries[0].(map[string]any)["updated"] = "two"
	_ = fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
	if requests.Load() != before+2 {
		t.Fatal("unconfirmed updated hint changed the negative cache")
	}
}

func TestCanonicalCacheDoesNotHideRolledBackRefreshOrDeletion(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
	iri := "https://third.example/api/v1/summit-log/original"
	log := f.child(iri, author.Id, f.trail.Id)
	previousClient := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
	requests, gone := 0, false
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			requests++
			if gone {
				return &http.Response{StatusCode: http.StatusGone, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			}
			return richTextSyncTransport{map[string]any{"id": "original", "iri": iri, "text": "Canonical", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
		}}}
	}
	entries := []any{map[string]any{"id": "copy", "iri": iri, "updated": "one"}}
	data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}
	sentinel := errors.New("later transaction failure")
	for _, deletion := range []bool{false, true} {
		gone = deletion
		entries[0].(map[string]any)["updated"] = fmt.Sprint(deletion)
		canonical := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
		err := f.app.RunInTransaction(func(tx core.App) error {
			if err := syncSummitLogs(tx, t.Context(), f.trail, "https://remote.example", entries, canonical); err != nil {
				return err
			}
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatal(err)
		}
		stored, err := f.app.FindRecordById(f.children, log.Id)
		if err != nil || stored.GetString("text") != "Original" {
			t.Fatalf("rollback changed canonical copy: %v %v", stored, err)
		}
		before := requests
		_ = fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
		if requests != before+1 {
			t.Fatalf("rollback was hidden by cache (deletion=%t)", deletion)
		}
	}
}

func TestCanonicalKnownCopiesRefreshOnChangedHintsAndRemoveConfirmedGoneEntries(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, status := range []int{http.StatusNotFound, http.StatusGone} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "https://third.example/api/v1/summit-log/original"
			log := f.child(iri, author.Id, f.trail.Id)
			hint, text, gone := "one", "Canonical one", false
			var canonicalRequests int
			previousClient := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "remote.example" {
						return richTextSyncTransport{map[string]any{"id": "trail", "name": "Remote", "public": true, "expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"id": "copy", "iri": iri, "updated": hint, "text": "Untrusted copy"}}}}}.RoundTrip(request)
					}
					canonicalRequests++
					if gone {
						return &http.Response{StatusCode: status, Body: http.NoBody, Header: make(http.Header), Request: request}, nil
					}
					return richTextSyncTransport{map[string]any{"id": "original", "iri": iri, "text": text, "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
				}}}
			}
			sync := func() {
				t.Helper()
				if _, err := performFullSync(f.app, context.Background(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
			}
			sync()
			sync()
			if canonicalRequests != 1 {
				t.Fatalf("unchanged known copy fetched%d times", canonicalRequests)
			}
			hint, text = "two", "Canonical two"
			sync()
			current, err := f.app.FindRecordById(f.children, log.Id)
			if err != nil || current.GetString("text") != text || canonicalRequests != 2 {
				t.Fatalf("known origin update lost: %v requests%d err%v", current, canonicalRequests, err)
			}
			hint, gone = "three", true
			sync()
			if _, err := f.app.FindRecordById(f.children, log.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("confirmed missing origin copy survived: %v", err)
			}
		})
	}
}

func TestCanonicalFailedFirstSyncDoesNotSuppressAnotherUnsavedTrail(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
	first, second := core.NewRecord(f.trails), core.NewRecord(f.trails)
	first.Set("iri", "https://remote.example/api/v1/trail/first")
	second.Set("iri", "https://remote.example/api/v1/trail/second")
	iri := "https://third.example/api/v1/summit-log/new"
	data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"iri": iri}}}}
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	requests := 0
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			}
			return richTextSyncTransport{map[string]any{"id": "new", "iri": iri, "text": "Confirmed", "expand": map[string]any{"trail": map[string]any{"iri": second.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
		}}}
	}
	if len(fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, first, "https://remote.example")) != 0 {
		t.Fatal("failed first sync imported a log")
	}
	result := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, second, "https://remote.example")
	if requests != 2 || len(result) != 1 || result[iri].author != author.Id {
		t.Fatalf("another trail's negative cache suppressed first sync: requests=%d results=%v", requests, result)
	}
}

func TestCanonicalMissingCopiesUseFreshTrailVisibility(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, test := range []struct {
		name       string
		status     int
		public     bool
		omitPublic bool
		remove     bool
	}{
		{name: "private-not-found", status: http.StatusNotFound},
		{name: "public-not-found", status: http.StatusNotFound, public: true, remove: true},
		{name: "private-gone", status: http.StatusGone, remove: true},
		{name: "public-gone", status: http.StatusGone, public: true, remove: true},
		{name: "private-forbidden", status: http.StatusForbidden},
		{name: "public-forbidden", status: http.StatusForbidden, public: true},
		{name: "unavailable", status: http.StatusServiceUnavailable, public: true},
		{name: "unspecified-not-found", status: http.StatusNotFound, omitPublic: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "https://third.example/api/v1/summit-log/original"
			log := f.child(iri, author.Id, f.trail.Id)
			// The freshly fetched visibility decides; the stored trail may be older.
			f.trail.Set("public", !test.public)
			if err := f.app.Save(f.trail); err != nil {
				t.Fatal(err)
			}
			entries := []any{map[string]any{"id": "copy", "iri": iri, "text": "Remote copy"}}
			data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}
			if !test.omitPublic {
				data["public"] = test.public
			}
			previousClient := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			var requests atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					requests.Add(1)
					return &http.Response{StatusCode: test.status, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
				}}}
			}
			canonical := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
			if requests.Load() != 1 {
				t.Fatalf("canonical requests = %d, want 1", requests.Load())
			}
			if err := syncSummitLogs(f.app, t.Context(), f.trail, "https://remote.example", entries, canonical); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(f.children, log.Id)
			if test.remove {
				if !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("confirmed removed copy remains: %v", err)
				}
			} else if err != nil || stored.GetString("text") != "Original" || stored.GetString("author") != author.Id {
				t.Fatalf("unconfirmed copy changed: %v, %v", stored, err)
			}
		})
	}
}
