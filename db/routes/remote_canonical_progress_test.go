package routes

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCanonicalContributionBudgetMakesProgressAcrossHourlyRefreshes(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, state := range []string{"new-contributors"} {
		t.Run(state, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			entries := []any{}
			for i := 0; i < 36; i++ {
				iri := fmt.Sprintf("https://third.example/api/v1/summit-log/entry%d", i)
				entries = append(entries, map[string]any{"id": fmt.Sprintf("entry%d", i), "iri": iri, "updated": "same-origin-hint"})
			}
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			var requests atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "remote.example" {
						return richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"summit_logs_via_trail": entries}}}.RoundTrip(request)
					}
					requests.Add(1)
					id := strings.TrimPrefix(request.URL.Path, "/api/v1/summit-log/")
					return richTextSyncTransport{map[string]any{"id": id, "iri": "https://third.example/api/v1/summit-log/" + id, "text": "Confirmed " + id, "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}}.RoundTrip(request)
				}}}
			}
			for round := 0; round < 2; round++ {
				requests.Store(0)
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
				if count := requests.Load(); count > canonicalRemoteRecordLimit || count == 0 {
					t.Fatalf("refresh%d requests=%d", round, count)
				}
				// A background refresh happens after the short result cache has
				// expired. Preserve no artificially fresh success entries here.
				canonicalRemoteLogCache.Lock()
				for key, value := range canonicalRemoteLogCache.entries {
					if strings.HasPrefix(key, f.app.DataDir()+"\x00") {
						value.checked = time.Now().Add(-2 * time.Hour)
						canonicalRemoteLogCache.entries[key] = value
					}
				}
				canonicalRemoteLogCache.Unlock()
			}
			records, err := f.app.FindAllRecords(f.children)
			if err != nil || len(records) != 36 {
				t.Fatalf("later contributions were not refreshed: count%d error%v", len(records), err)
			}
		})
	}
}
