package routes

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCanonicalReferencesRejectLoosePathsAndRedirectsAndDeduplicateFailures(t *testing.T) {
	for _, scenario := range []string{"duplicate-failure", "redirect", "extra-prefix", "query", "fragment", "encoded-path"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			iri := "https://third.example/api/v1/summit-log/original"
			switch scenario {
			case "extra-prefix":
				iri = "https://third.example/extra/api/v1/summit-log/original"
			case "query":
				iri += "?secret=reference"
			case "fragment":
				iri += "#reference"
			case "encoded-path":
				iri = "https://third.example/api/v1/summit-log/%6friginal"
			}
			var requests, redirected atomic.Int32
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					requests.Add(1)
					if request.URL.Host != "third.example" {
						redirected.Add(1)
					}
					status := http.StatusServiceUnavailable
					header := make(http.Header)
					if scenario == "redirect" {
						status = http.StatusFound
						header.Set("Location", "https://elsewhere.example/api/v1/summit-log/original")
					}
					return &http.Response{StatusCode: status, Header: header, Body: http.NoBody, Request: request}, nil
				}}}
			}
			entries := []any{map[string]any{"id": "original", "iri": iri}, map[string]any{"id": "duplicate", "iri": iri}}
			result := fetchCanonicalRemoteSummitLogs(f.app, context.Background(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}, f.trail, "https://remote.example")
			wantRequests := int32(0)
			if scenario == "duplicate-failure" || scenario == "redirect" {
				wantRequests = 1
			}
			if len(result) != 0 || requests.Load() != wantRequests || redirected.Load() != 0 {
				t.Fatalf("invalid reference fetched: scenario=%s results=%v requests=%d redirects=%d", scenario, result, requests.Load(), redirected.Load())
			}
			if strings.Contains(iri, "secret") && requests.Load() != 0 {
				t.Fatal("remote query accepted")
			}
		})
	}
}
