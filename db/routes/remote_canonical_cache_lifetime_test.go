package routes

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestCanonicalConfirmationsUseSeparateRefreshIntervals(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	previousInterval := remoteSyncThreshold
	remoteSyncThreshold = 37 * time.Minute
	t.Cleanup(func() { remoteSyncThreshold = previousInterval })
	for _, failed := range []bool{false, true} {
		name := "confirmed"
		if failed {
			name = "unavailable"
		}
		t.Run(name, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "https://third.example/api/v1/summit-log/original"
			f.child(iri, author.Id, f.trail.Id)
			entries := []any{map[string]any{"id": "copy", "iri": iri, "updated": "one"}}
			data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}
			previousClient := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			var requests atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					requests.Add(1)
					if failed {
						return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
					}
					return richTextSyncTransport{map[string]any{"id": "original", "iri": iri, "text": "Confirmed", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
				}}}
			}
			refresh := func() {
				t.Helper()
				canonical := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
				if err := syncSummitLogs(f.app, t.Context(), f.trail, "https://remote.example", entries, canonical); err != nil {
					t.Fatal(err)
				}
			}
			age := func(duration time.Duration) {
				t.Helper()
				key := canonicalLogCacheKey(f.app, f.trail, iri)
				canonicalRemoteLogCache.Lock()
				defer canonicalRemoteLogCache.Unlock()
				entry, found := canonicalRemoteLogCache.entries[key]
				if !found {
					t.Fatal("confirmation result was not cached")
				}
				entry.checked = time.Now().Add(-duration)
				canonicalRemoteLogCache.entries[key] = entry
			}
			refresh()
			if requests.Load() != 1 {
				t.Fatalf("initial requests = %d, want 1", requests.Load())
			}
			for _, elapsed := range []time.Duration{61 * time.Second, 14 * time.Minute} {
				age(elapsed)
				refresh()
				if requests.Load() != 1 {
					t.Fatalf("refetched after %s: requests = %d", elapsed, requests.Load())
				}
			}
			age(16 * time.Minute)
			refresh()
			want := int32(1)
			if failed {
				want = 2
			}
			if requests.Load() != want {
				t.Fatalf("requests after 16 minutes = %d, want %d", requests.Load(), want)
			}
			if !failed {
				age(remoteSyncThreshold + time.Second)
				refresh()
				if requests.Load() != 2 {
					t.Fatalf("requests after sync interval = %d, want 2", requests.Load())
				}
			}
		})
	}
}

func TestCanonicalConfirmedCacheRechecksChangedReferences(t *testing.T) {
	for _, test := range []struct {
		name, hint, updated string
		exists              bool
	}{
		{name: "hint", hint: "two", updated: "stored", exists: true},
		{name: "stored-version", hint: "one", updated: "changed", exists: true},
		{name: "missing-copy", hint: "one", updated: "stored"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cache := &canonicalLogCache{}
			cache.success("entry", "one", "stored")
			if cache.skip("entry", test.hint, test.updated, test.exists) {
				t.Fatal("a changed reference reused its confirmation")
			}
		})
	}
}

func TestCanonicalCacheCapacityKeepsUnexpiredConfirmations(t *testing.T) {
	previousInterval := remoteSyncThreshold
	remoteSyncThreshold = 37 * time.Minute
	t.Cleanup(func() { remoteSyncThreshold = previousInterval })
	now := time.Now()
	cache := &canonicalLogCache{entries: map[string]canonicalLogCacheEntry{
		"confirmed": {checked: now.Add(-61 * time.Second)},
		"expired":   {checked: now.Add(-16 * time.Minute), failed: true},
	}}
	for i := len(cache.entries); i < canonicalRemoteLogCacheLimit; i++ {
		cache.entries[fmt.Sprint(i)] = canonicalLogCacheEntry{checked: now, failed: true}
	}
	cache.remember("new", canonicalLogCacheEntry{failed: true})
	if _, found := cache.entries["confirmed"]; !found {
		t.Fatal("capacity cleanup removed an unexpired confirmation")
	}
	if _, found := cache.entries["expired"]; found {
		t.Fatal("capacity cleanup retained an expired failure")
	}
	if len(cache.entries) != canonicalRemoteLogCacheLimit {
		t.Fatalf("cache entries = %d, want %d", len(cache.entries), canonicalRemoteLogCacheLimit)
	}
}
