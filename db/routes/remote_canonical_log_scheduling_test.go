package routes

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCanonicalLogSchedulingFairShareAndThroughput(t *testing.T) {
	for _, test := range []struct {
		name        string
		sizes, want [3]int
	}{
		{"empty", [3]int{}, [3]int{}},
		{"competing-refreshes", [3]int{0, 40, 36}, [3]int{0, 10, 20}},
		{"new-listed-priority", [3]int{40, 40, 36}, [3]int{28, 1, 1}},
		{"changed-listed-priority", [3]int{5, 40, 36}, [3]int{5, 10, 15}},
		{"only-omitted", [3]int{0, 40, 0}, [3]int{0, 30, 0}},
		{"only-known-listed", [3]int{0, 0, 36}, [3]int{0, 0, 30}},
		{"only-new-listed", [3]int{40, 0, 0}, [3]int{30, 0, 0}},
		{"lend-to-omitted", [3]int{0, 40, 2}, [3]int{0, 28, 2}},
		{"lend-to-listed", [3]int{0, 2, 36}, [3]int{0, 2, 28}},
		{"below-limit", [3]int{2, 3, 4}, [3]int{2, 3, 4}},
	} {
		t.Run(test.name, func(t *testing.T) {
			priorities := [3][]canonicalLogRequest{}
			for priority, size := range test.sizes {
				for i := 0; i < size; i++ {
					priorities[priority] = append(priorities[priority], canonicalLogRequest{iri: fmt.Sprintf("%d/%d", priority, i)})
				}
			}
			seen := map[string]bool{}
			key := t.TempDir()
			for round := 0; round < 40; round++ {
				jobs := selectCanonicalLogRequests(key, priorities)
				counts := [3]int{}
				for index, job := range jobs {
					priority := int(job.iri[0] - '0')
					counts[priority]++
					seen[job.iri] = true
					if index < test.want[0] && priority != 0 {
						t.Fatal("new/changed listed entries lost their leading priority")
					}
				}
				if counts != test.want {
					t.Fatalf("selected=%v want=%v", counts, test.want)
				}
				if test.want[2] > 0 && jobs[test.want[0]].iri[0] != '2' {
					t.Fatal("known listed refreshes queued behind omitted probes")
				}
				index := test.want[0]
				for offset := 0; offset < max(test.want[1], test.want[2]); offset++ {
					for _, priority := range []int{2, 1} {
						if offset < test.want[priority] {
							if int(jobs[index].iri[0]-'0') != priority {
								t.Fatal("listed/omitted refreshes lost deterministic alternation")
							}
							index++
						}
					}
				}
			}
			if len(seen) != test.sizes[0]+test.sizes[1]+test.sizes[2] {
				t.Fatalf("rotating windows left entries unvisited: %d", len(seen))
			}
		})
	}
}

func TestRemoteOmittedAndKnownLogsShareHourlyRefreshes(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now()})
	entries := []any{}
	for i := 0; i < 36; i++ {
		id := fmt.Sprintf("listed%d", i)
		iri := "https://a.example:8443/api/v1/summit-log/" + id
		f.child(iri, author.Id, f.trail.Id)
		entries = append(entries, map[string]any{"id": id, "iri": iri, "updated": "same-hint"})
	}
	for i := 0; i < 40; i++ {
		f.child(fmt.Sprintf("https://a.example:8443/api/v1/summit-log/omitted%d", i), author.Id, f.trail.Id)
	}
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	var lock sync.Mutex
	seen := map[string]bool{}
	counts := [2]int{}
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if request.URL.Host == "a.example" {
				return richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"summit_logs_via_trail": entries}}}.RoundTrip(request)
			}
			id := strings.TrimPrefix(request.URL.Path, "/api/v1/summit-log/")
			class := 0
			if strings.HasPrefix(id, "omitted") {
				class = 1
			}
			lock.Lock()
			counts[class]++
			seen[id] = true
			lock.Unlock()
			if id == "listed35" {
				return &http.Response{StatusCode: http.StatusGone, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			}
			return richTextSyncTransport{map[string]any{"id": id, "iri": "https://a.example:8443" + request.URL.Path, "text": "Confirmed " + id,
				"expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
		}}}
	}
	for round := 0; round < 4; round++ {
		counts = [2]int{}
		if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
			t.Fatal(err)
		}
		if counts != [2]int{20, 10} {
			t.Fatalf("hourly refresh %d: listed/omitted=%v want [20 10]", round, counts)
		}
		// Every ordinary hourly refresh sees both classes due again.
		canonicalRemoteLogCache.Lock()
		for key, value := range canonicalRemoteLogCache.entries {
			if strings.HasPrefix(key, f.app.DataDir()+"\x00") {
				value.checked = time.Now().Add(-remoteSyncThreshold - time.Second)
				canonicalRemoteLogCache.entries[key] = value
			}
		}
		canonicalRemoteLogCache.Unlock()
	}
	if len(seen) != 76 {
		t.Fatalf("hourly rotations did not visit all contributions: %d/76", len(seen))
	}
	for i := 0; i < 36; i++ {
		iri := fmt.Sprintf("https://a.example:8443/api/v1/summit-log/listed%d", i)
		log, err := f.app.FindFirstRecordByData(f.children, "iri", iri)
		if i == 35 {
			if err == nil {
				t.Fatal("known listed deletion was starved")
			}
		} else if err != nil || log.GetString("text") != fmt.Sprintf("Confirmed listed%d", i) {
			t.Fatalf("known listed data was starved: %s error=%v", iri, err)
		}
	}
}

func TestCanonicalListedLogsProgressWhileOmittedHostsExhaustBudget(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	slowAuthor := f.save(f.actors, map[string]any{"iri": "https://slow.example/users/carol", "last_fetched": time.Now()})
	healthyAuthor := f.save(f.actors, map[string]any{"iri": "https://healthy.example/users/carol", "last_fetched": time.Now()})
	for i := 0; i < 12; i++ {
		f.child(fmt.Sprintf("https://slow.example/api/v1/summit-log/omitted%d", i), slowAuthor.Id, f.trail.Id)
	}
	entries := []any{}
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("listed%d", i)
		iri := "https://healthy.example/api/v1/summit-log/" + id
		f.child(iri, healthyAuthor.Id, f.trail.Id)
		entries = append(entries, map[string]any{"id": id, "iri": iri})
	}
	previousClient := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
	var listed, omitted atomic.Int32
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if request.URL.Host == "slow.example" {
				omitted.Add(1)
				<-request.Context().Done()
				return nil, request.Context().Err()
			}
			listed.Add(1)
			id := strings.TrimPrefix(request.URL.Path, "/api/v1/summit-log/")
			return richTextSyncTransport{map[string]any{"id": id, "iri": "https://healthy.example" + request.URL.Path, "text": "Confirmed",
				"expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": healthyAuthor.GetString("iri")}}}}.RoundTrip(request)
		}}}
	}
	// Shorten only the optional object budget, leaving the caller and author
	// resolution alive just as with the production ten-second budget.
	canonical := fetchCanonicalRemoteSummitLogsWithin(f.app, t.Context(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}, f.trail, "https://a.example", true, 250*time.Millisecond)
	if omitted.Load() == 0 || listed.Load() == 0 || len(canonical) == 0 {
		t.Fatalf("slow omitted jobs blocked all listed work: listed=%d omitted=%d confirmed=%d", listed.Load(), omitted.Load(), len(canonical))
	}
	if err := syncSummitLogs(f.app, t.Context(), f.trail, "https://a.example", entries, canonical); err != nil {
		t.Fatal(err)
	}
	for iri := range canonical {
		log, err := f.app.FindFirstRecordByData(f.children, "iri", iri)
		if err != nil || log.GetString("text") != "Confirmed" {
			t.Fatalf("listed progress was not saved after budget exhaustion: %s error=%v", iri, err)
		}
	}
}
