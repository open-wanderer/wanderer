package routes

import (
	"net/http"
	"testing"
	"time"
)

func TestCanonicalLogRefreshFollowsOnlyItsSameHostTLSUpgrade(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, scenario := range []string{"upgrade", "host", "port", "path", "query", "downgrade", "second-hop"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "http://third.example/api/v1/summit-log/log"
			if scenario == "downgrade" {
				iri = "https://third.example/api/v1/summit-log/log"
			}
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			requests := 0
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					requests++
					if requests == 1 || scenario == "second-hop" {
						target := *request.URL
						target.Scheme = "https"
						switch scenario {
						case "host":
							target.Host = "other.example"
						case "port":
							target.Host = "third.example:8443"
						case "path":
							target.Path = "/api/v1/summit-log/another"
						case "query":
							target.RawQuery = "other=query"
						case "downgrade":
							target.Scheme = "http"
						}
						return &http.Response{StatusCode: http.StatusMovedPermanently, Header: http.Header{"Location": {target.String()}}, Body: http.NoBody, Request: request}, nil
					}
					return richTextSyncTransport{map[string]any{"id": "log", "iri": "https://third.example/api/v1/summit-log/log", "text": "Updated over TLS", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
				}}}
			}
			entries := []any{map[string]any{"iri": iri}}
			result := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}, f.trail, "https://remote.example")
			if scenario == "upgrade" {
				if requests != 2 || len(result) != 1 || result[iri].author != author.Id {
					t.Fatalf("same-host TLS upgrade did not refresh: requests=%d results=%v", requests, result)
				}
			} else {
				want := 1
				if scenario == "second-hop" {
					want = 2
				}
				if requests != want || len(result) != 0 {
					t.Fatalf("redirect outside exact TLS upgrade followed: requests=%d results=%v", requests, result)
				}
			}
		})
	}
}
