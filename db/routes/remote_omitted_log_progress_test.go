package routes

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRemoteOmittedContributionsProgressAcrossHourlyRefreshes(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, state := range []string{"omitted-and-late-deletion"} {
		t.Run(state, func(t *testing.T) {
			f := newRemoteCleanupFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now()})
			entries := []any{}
			for i := 0; i < 36; i++ {
				iri := fmt.Sprintf("https://a.example:8443/api/v1/summit-log/entry%d", i)
				entries = append(entries, map[string]any{"id": fmt.Sprintf("entry%d", i), "iri": iri, "updated": "same-origin-hint"})
				if state == "omitted-and-late-deletion" {
					f.child(iri, author.Id, f.trail.Id)
				}
			}
			var omittedID string
			if state == "omitted-and-late-deletion" {
				omittedID = f.child("https://a.example:8443/api/v1/summit-log/omitted", author.Id, f.trail.Id).Id
			}
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			var requests atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "a.example" {
						return richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"summit_logs_via_trail": entries}}}.RoundTrip(request)
					}
					requests.Add(1)
					id := strings.TrimPrefix(request.URL.Path, "/api/v1/summit-log/")
					if state == "omitted-and-late-deletion" && (id == "omitted" || id == "entry35") {
						return &http.Response{StatusCode: http.StatusGone, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
					}
					return richTextSyncTransport{map[string]any{"id": id, "iri": "https://a.example:8443/api/v1/summit-log/" + id, "text": "Confirmed " + id, "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}}.RoundTrip(request)
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
				if round == 0 && omittedID != "" {
					if _, err := f.app.FindRecordById(f.children, omittedID); err == nil {
						t.Fatal("omission probe was starved behind the normal refresh window")
					}
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
			if state == "new-contributors" {
				records, err := f.app.FindAllRecords(f.children)
				if err != nil || len(records) != 36 {
					t.Fatalf("later contributions still starved: count%d error%v", len(records), err)
				}
			} else {
				if _, err := f.app.FindFirstRecordByData(f.children, "iri", "https://a.example:8443/api/v1/summit-log/entry35"); err == nil {
					t.Fatal("late listed deletion never reached its canonical origin")
				}
			}
		})
	}
}
