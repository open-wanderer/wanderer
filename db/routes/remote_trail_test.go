package routes

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestRemoteTrailSyncRequestsChildrenWithoutCallerProjections(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, projected := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "projection"}[projected], func(t *testing.T) {
			f := newRemoteChildFixture(t, "waypoint")
			f.remoteActor.Set("iri", "https://a.example/users/alice")
			if err := f.app.Save(f.remoteActor); err != nil {
				t.Fatal(err)
			}
			f.trail.Set("iri", "https://a.example/api/v1/trail/trail")
			if err := f.app.Save(f.trail); err != nil {
				t.Fatal(err)
			}
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			var requested url.Values
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					requested = request.URL.Query()
					payload := map[string]any{"id": "trail", "name": "Origin trail", "public": true,
						"expand": map[string]any{}}
					if slices.Contains(strings.Split(requested.Get("expand"), ","), "waypoints_via_trail") && requested.Get("fields") == "" {
						payload["expand"].(map[string]any)["waypoints_via_trail"] = []any{map[string]any{"id": "point", "name": "Origin waypoint"}}
					}
					return richTextSyncTransport{payload}.RoundTrip(request)
				}}}
			}
			var requestURL *url.URL
			if projected {
				requestURL = &url.URL{RawQuery: "fields=id,name&expand=trail_share_via_trail.actor&foo=bar&handle=alice&share=abcdefghijklmnopqrstuvwxyz012345"}
			}
			if _, err := performFullSync(f.app, t.Context(), requestURL, f.trail); err != nil {
				t.Fatal(err)
			}
			records, err := f.app.FindAllRecords(f.children)
			if err != nil || len(records) != 1 || records[0].GetString("name") != "Origin waypoint" {
				t.Fatalf("readable waypoint missing from synchronized trail: count=%d error=%v", len(records), err)
			}
			if requested.Get("fields") != "" || requested.Get("foo") != "" || requested.Get("handle") != "" || strings.Contains(requested.Get("expand"), "trail_share") {
				t.Fatalf("caller projection reached the trail origin: %v", requested)
			}
			for _, required := range []string{"author", "category", "subcategory", "tags", "waypoints_via_trail", "waypoints_via_trail.author", "summit_logs_via_trail", "summit_logs_via_trail.author"} {
				if !slices.Contains(strings.Split(requested.Get("expand"), ","), required) {
					t.Fatalf("origin request omitted %s: %v", required, requested)
				}
			}
			if projected && requested.Get("share") != "abcdefghijklmnopqrstuvwxyz012345" {
				t.Fatalf("origin request lost its share: %v", requested)
			}
		})
	}
}
