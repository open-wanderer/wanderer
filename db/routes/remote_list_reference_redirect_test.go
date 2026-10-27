package routes

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRemoteListReferenceRedirectsStayOnTheirOriginalEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, from, to string
		stored         bool
	}{
		{"other host", "https://third.example/api/v1/trail/trail-id", "https://other.example/api/v1/trail/trail-id", false},
		{"other port", "https://third.example/api/v1/trail/trail-id", "https://third.example:8443/api/v1/trail/trail-id", false},
		{"other path", "https://third.example/api/v1/trail/trail-id", "https://third.example/api/v1/trail/other-id", false},
		{"scheme downgrade", "https://third.example/api/v1/trail/trail-id", "http://third.example/api/v1/trail/trail-id", false},
		{"upgrade with other host", "http://third.example/api/v1/trail/trail-id", "https://other.example/api/v1/trail/trail-id", false},
		{"upgrade with other port", "http://third.example/api/v1/trail/trail-id", "https://third.example:8443/api/v1/trail/trail-id", false},
		{"upgrade with other path", "http://third.example/api/v1/trail/trail-id", "https://third.example/api/v1/trail/other-id", false},
		{"scheme upgrade", "http://third.example/api/v1/trail/trail-id", "https://third.example/api/v1/trail/trail-id", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newListReferenceFixture(t)
			origin := strings.TrimSuffix(tc.from, "/api/v1/trail/trail-id")
			author := f.save(f.actors, map[string]any{"iri": origin + "/api/v1/actor/carol", "last_fetched": time.Now()})
			f.entries = []any{map[string]any{"iri": tc.from}}
			var redirects, targets atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					endpoint := request.URL.Scheme + "://" + request.URL.Host + request.URL.Path
					switch endpoint {
					case tc.from:
						redirects.Add(1)
						return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {tc.to + "?" + request.URL.RawQuery}}, Body: http.NoBody, Request: request}, nil
					case tc.to:
						targets.Add(1)
						return richTextSyncTransport{map[string]any{"id": "trail-id", "iri": tc.from, "public": true, "name": "Origin trail", "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
					default:
						return f.serve(request)
					}
				}}}
			}
			f.sync(t)
			if redirects.Load() != 1 {
				t.Fatalf("initial requests = %d, want 1", redirects.Load())
			}
			wantRequests := int32(0)
			if tc.stored {
				wantRequests = 1
			}
			if targets.Load() != wantRequests {
				t.Fatalf("redirect target requests = %d, want %d", targets.Load(), wantRequests)
			}
			record, err := f.app.FindFirstRecordByData("trails", "iri", tc.from)
			if !tc.stored {
				if err == nil || len(f.list.GetStringSlice("trails")) != 0 {
					t.Fatal("redirected reference was confirmed or stored")
				}
			} else if err != nil || record.GetString("author") != author.Id || record.GetString("name") != "Origin trail" || len(f.list.GetStringSlice("trails")) != 1 {
				t.Fatalf("same-endpoint upgrade was not stored: %v %v", record, err)
			}
		})
	}
}
