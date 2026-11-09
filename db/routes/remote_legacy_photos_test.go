package routes

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestLegacyRemoteRetryKeepsUnrequestedChildPhotos(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		for _, tc := range []struct {
			name   string
			photos []any
		}{
			{"unrequested", []any{}},
			{"view-model-url", []any{"/api/v1/assets/modern-photo/file"}},
		} {
			t.Run(target+"/"+tc.name, func(t *testing.T) {
				f := newRemoteChildCleanupFixture(t, target)
				child := f.child("https://a.example/api/v1/"+target+"/retained", f.remoteActor.Id, f.trail.Id)
				photo := f.photo(child, "retained.jpg")
				previous := newRemoteSyncHTTPClient
				t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
				requests := 0
				newRemoteSyncHTTPClient = func() *http.Client {
					return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
						requests++
						if strings.Contains(request.URL.Query().Get("expand"), "_assets_via_") {
							return &http.Response{StatusCode: http.StatusRequestHeaderFieldsTooLarge, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("unsupported asset expands")), Request: request}, nil
						}
						relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]
						return richTextSyncTransport{map[string]any{"id": "trail", "public": true, "name": "Origin", "expand": map[string]any{
							relation: []any{map[string]any{"id": "retained", "iri": child.GetString("iri"), "name": "Point", "text": "Summit", "photos": tc.photos}},
						}}}.RoundTrip(request)
					}}}
				}
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
				if requests != 2 {
					t.Fatalf("legacy retry requests=%d, want 2", requests)
				}
				if _, err := f.app.FindRecordById(f.assets, photo.Id); err != nil {
					t.Fatalf("legacy view model deleted the cached child photo: %v", err)
				}
				links, err := f.app.CountRecords(f.links.Id, dbx.HashExp{target: child.Id, "asset": photo.Id})
				if err != nil || links != 1 {
					t.Fatalf("legacy view model unlinked the cached child photo: links=%d error=%v", links, err)
				}
				f.assertFile(photo, true)
			})
		}
	}
}
