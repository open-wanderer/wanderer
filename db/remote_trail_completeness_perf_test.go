package main

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// Exercise real PB HTTP/SQLite on one TCP client/IP, at normal hover/view rates.
// Timing is evidence, not a brittle assertion: captured logs and API hook counts
// prove that fixed in-process proofs avoid the generic list throttle, while a
// control confirms that PocketBase's countermeasure and route limits still work.
func TestRemoteTrailChildProofAvoidsFilteredListThrottle(t *testing.T) {
	f := newTrailProofFixture(t)
	if f.app.Settings().RateLimits.Enabled || len(f.app.Settings().TrustedProxy.Headers) != 0 {
		t.Fatal("probe requires unmodified PB default rate/proxy settings")
	}
	var listRequests atomic.Int32
	f.app.OnRecordsListRequest("waypoints", "summit_logs").BindFunc(func(e *core.RecordsListRequestEvent) error {
		listRequests.Add(1)
		return e.Next()
	})
	server := httptest.NewServer(f.mux)
	defer server.Close()
	client := server.Client()
	client.Timeout = 5 * time.Second
	do := func(path string) (time.Duration, int, []byte) {
		t.Helper()
		start := time.Now()
		// No XFF, fake IPs, superuser tokens or rate-limit exclusions.
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return time.Since(start), response.StatusCode, body
	}
	f.app.takeQueries()
	var timings []float64
	const views = 12
	ticker := time.NewTicker(time.Second / 5)
	defer ticker.Stop()
	q := url.Values{"expand": {"category,subcategory,subcategory.category,waypoints_via_trail,summit_logs_via_trail,summit_logs_via_trail.author,trail_share_via_trail.actor,trail_like_via_trail,tags,author"}}
	path := "/remote/trail/" + f.public.Id + "?" + q.Encode()
	for i := 0; i < views; i++ {
		if i > 0 {
			<-ticker.C
		}
		duration, code, body := do(path)
		var result struct{ Expand map[string]json.RawMessage }
		if code != 200 || json.Unmarshal(body, &result) != nil || string(result.Expand["waypoints_via_trail"]) != "[]" || string(result.Expand["summit_logs_via_trail"]) != "[]" {
			t.Fatalf("proof view failed: status=%d body=%s", code, body)
		}
		timings = append(timings, float64(duration)/float64(time.Millisecond))
	}
	queries := f.app.takeQueries()
	if len(queries) != 2*views || listRequests.Load() != 0 || strings.Contains(f.app.log.text(), "Randomized throttle") {
		t.Fatalf("proof used the filtered-list timing path: queries=%d listRequests=%d logs=%s", len(queries), listRequests.Load(), f.app.log.text())
	}
	slices.Sort(timings)
	t.Logf("same-IP default PB: views=%d rate=5/s in-process-proofs=%d list-HTTP-requests=0 randomized-throttle-logs=0 median=%.2fms p95=%.2fms max=%.2fms", views, len(queries), timings[len(timings)/2], timings[int(math.Ceil(.95*float64(len(timings))))-1], timings[len(timings)-1])
	// A finite positive control proves we did not disable or evade the generic
	// list timing guard. This uses the exact former SDK request query shape.
	for i := 0; i < 5; i++ {
		_, code, body := do("/api/collections/waypoints/records?" + trailProofQueryParams(f.public.Id, "").Encode())
		if code != 200 {
			t.Fatalf("generic-list control failed: status=%d body=%s", code, body)
		}
	}
	throttles := strings.Count(f.app.log.text(), "Randomized throttle because of too many failed searches")
	if listRequests.Load() != 5 || throttles == 0 {
		t.Fatalf("generic list timing countermeasure was not exercised: lists=%d logs=%s", listRequests.Load(), f.app.log.text())
	}
	t.Logf("same-IP generic-list positive control: requests=5 randomized-throttle-logs=%d (guard intact)", throttles)
	for _, line := range strings.Split(f.app.log.text(), "\n") {
		if strings.Contains(line, "Randomized throttle") {
			t.Logf("captured PB log: %s", line)
		}
	}
	f.app.Settings().RateLimits.Enabled = true
	f.app.Settings().RateLimits.Rules = []core.RateLimitRule{{Label: "GET /remote/trail/", MaxRequests: 1, Duration: 60}}
	for _, want := range []int{http.StatusOK, http.StatusTooManyRequests} {
		_, code, body := do(path)
		if code != want {
			t.Fatalf("configured route limit changed: status=%d want=%d body=%s", code, want, body)
		}
	}
	t.Log("configured /remote/trail route limit: 200 then 429 (retained)")
}
