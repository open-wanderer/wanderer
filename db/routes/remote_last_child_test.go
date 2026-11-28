package routes

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFullRemoteSyncRetractsTheLastExplicitlyEmptyChild(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		for _, scenario := range []string{"explicit-empty", "omitted-empty", "missing-expand", "malformed", "legacy-retry"} {
			t.Run(target+"/"+scenario, func(t *testing.T) {
				f := newRemoteChildCleanupFixture(t, target)
				kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
				child := f.child("https://a.example/api/v1/"+kind+"/last", f.remoteActor.Id, f.trail.Id)
				photo := f.photo(child, "last.jpg")
				previous := newRemoteSyncHTTPClient
				t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
				relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]
				newRemoteSyncHTTPClient = func() *http.Client {
					return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
						if scenario == "legacy-retry" && strings.Contains(request.URL.Query().Get("expand"), "_assets_via_") {
							return &http.Response{StatusCode: http.StatusRequestHeaderFieldsTooLarge, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("legacy"))}, nil
						}
						payload := map[string]any{"id": "trail", "public": true, "name": "Origin"}
						expand := map[string]any{}
						if scenario == "explicit-empty" {
							expand[relation] = []any{}
						}
						if scenario == "malformed" {
							expand[relation] = map[string]any{"id": "projection"}
						}
						if scenario != "missing-expand" {
							payload["expand"] = expand
						}
						return richTextSyncTransport{payload}.RoundTrip(request)
					}}}
				}
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
				remove := scenario == "explicit-empty"
				if record, _ := f.app.FindRecordById(f.children, child.Id); (record == nil) != remove {
					t.Fatal("last origin child's removal did not follow child-list completeness")
				}
				if asset, _ := f.app.FindRecordById(f.assets, photo.Id); (asset == nil) != remove {
					t.Fatal("last child's photo removal did not follow child-list completeness")
				}
				f.assertFile(photo, !remove)
			})
		}
	}
}
