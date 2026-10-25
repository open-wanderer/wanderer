package routes

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func TestCanonicalSummitLogsBoundReferencesAndHostsAndIgnoreFailedHintChanges(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	var lock sync.Mutex
	hosts := map[string]int{}
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			lock.Lock()
			hosts[request.URL.Host]++
			lock.Unlock()
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
		}}}
	}
	entries := []any{}
	for i := 0; i < 1000; i++ {
		entries = append(entries, map[string]any{"iri": fmt.Sprintf("https://peer%d.example/api/v1/summit-log/log", i), "updated": "first"})
	}
	data := map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}
	_ = fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
	if len(hosts) != 30 {
		t.Fatalf("one sync fetched %d hosts, want 30", len(hosts))
	}
	// Keep the admitted references stable while the origin changes its hints.
	entries = entries[:30]
	for _, raw := range entries {
		raw.(map[string]any)["updated"] = "changed"
	}
	data["expand"].(map[string]any)["summit_logs_via_trail"] = entries
	_ = fetchCanonicalRemoteSummitLogs(f.app, t.Context(), data, f.trail, "https://remote.example")
	for host, count := range hosts {
		if count != 1 {
			t.Fatalf("failed %s refetched after a hint change: %d requests", host, count)
		}
	}
}
