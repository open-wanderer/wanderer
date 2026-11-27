package routes

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestRemoteSyncConfirmsOmittedThirdPartyLogDeletionAtItsOrigin(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, scenario := range []string{"gone", "not-found", "available", "unavailable", "missing-expand", "saturated", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteChildFixture(t, "summit_log")
			f.remoteActor.Set("iri", "https://a.example/api/v1/actor/alice")
			f.remoteActor.Set("domain", "a.example")
			if err := f.app.Save(f.remoteActor); err != nil {
				t.Fatal(err)
			}
			f.trail.Set("iri", "https://a.example/api/v1/trail/trail")
			if err := f.app.Save(f.trail); err != nil {
				t.Fatal(err)
			}
			author := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now()})
			iri := "https://a.example:8443/api/v1/summit-log/omitted"
			log := f.child(iri, author.Id, f.trail.Id)
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			requests := 0
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "a.example" {
						payload := map[string]any{"id": "trail", "name": "Origin", "public": true}
						if scenario != "missing-expand" {
							entries := []any{}
							if scenario == "saturated" {
								for i := 0; i < 1000; i++ {
									entries = append(entries, map[string]any{"id": fmt.Sprint(i), "iri": fmt.Sprintf("https://a.example/api/v1/summit-log/%d", i)})
								}
							}
							if scenario == "malformed" {
								entries = []any{map[string]any{"text": "No identity"}}
							}
							payload["expand"] = map[string]any{"summit_logs_via_trail": entries}
						}
						return richTextSyncTransport{payload}.RoundTrip(request)
					}
					requests++
					status := http.StatusGone
					if scenario == "not-found" {
						status = http.StatusNotFound
					}
					if scenario == "unavailable" {
						status = http.StatusServiceUnavailable
					}
					if scenario != "available" {
						return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
					}
					return richTextSyncTransport{map[string]any{"id": "omitted", "iri": iri, "text": "Still exists", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
				}}}
			}
			if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
				t.Fatal(err)
			}
			_, err := f.app.FindRecordById(f.children, log.Id)
			deleted := scenario == "gone"
			if (err != nil) != deleted {
				t.Fatalf("omitted log existence changed without canonical proof: deleted=%t err=%v", deleted, err)
			}
			wantRequests := 1
			if scenario == "missing-expand" || scenario == "saturated" || scenario == "malformed" {
				wantRequests = 0
			}
			if requests != wantRequests {
				t.Fatalf("non-authoritative omission probed: requests=%d want=%d", requests, wantRequests)
			}
		})
	}
}
