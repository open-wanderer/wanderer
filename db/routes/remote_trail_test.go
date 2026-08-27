package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteTrailSyncQueryAddsRequiredExpands(t *testing.T) {
	reqURL, err := url.Parse("https://local.example/api/v1/trail/abc?handle=@alice@example.test&expand=tags,waypoints_via_trail&foo=bar")
	if err != nil {
		t.Fatal(err)
	}

	query := remoteTrailSyncQuery(reqURL, true)

	if got := query.Get("handle"); got != "" {
		t.Fatalf("handle query should not be forwarded, got %q", got)
	}
	if got := query.Get("foo"); got != "bar" {
		t.Fatalf("expected unrelated query param to be preserved, got %q", got)
	}

	expands := expandSet(query.Get("expand"))
	for _, required := range allRemoteTrailSyncExpandPaths() {
		if !expands[required] {
			t.Fatalf("expected required expand %q in %q", required, query.Get("expand"))
		}
	}
	if countExpand(query.Get("expand"), "waypoints_via_trail") != 1 {
		t.Fatalf("expected waypoints_via_trail to be deduplicated, got %q", query.Get("expand"))
	}
}

func TestRemoteTrailSyncQueryWithNilURL(t *testing.T) {
	query := remoteTrailSyncQuery(nil, true)

	expands := expandSet(query.Get("expand"))
	for _, required := range allRemoteTrailSyncExpandPaths() {
		if !expands[required] {
			t.Fatalf("expected required expand %q in %q", required, query.Get("expand"))
		}
	}
}

func TestRemoteTrailSyncQueryWithoutAssetExpands(t *testing.T) {
	reqURL, err := url.Parse("https://local.example/api/v1/trail/abc?expand=tags,trail_assets_via_trail.asset,waypoints_via_trail.waypoint_assets_via_waypoint.asset")
	if err != nil {
		t.Fatal(err)
	}

	query := remoteTrailSyncQuery(reqURL, false)

	expands := expandSet(query.Get("expand"))
	for _, removed := range remoteTrailSyncAssetExpandPaths {
		if expands[removed] {
			t.Fatalf("did not expect asset expand %q in %q", removed, query.Get("expand"))
		}
	}
	for _, required := range remoteTrailSyncCoreExpandPaths {
		if !expands[required] {
			t.Fatalf("expected core expand %q in %q", required, query.Get("expand"))
		}
	}
	if !expands["tags"] {
		t.Fatalf("expected caller expand tags to be preserved, got %q", query.Get("expand"))
	}
}

func TestLegacyRecordPhotosBuildsFederatedPhotos(t *testing.T) {
	photos := legacyRecordPhotos("https://remote.example", "trail", "trail1", map[string]any{
		"photos":    []any{"first.jpg", "second.jpg"},
		"thumbnail": float64(1),
	})

	if len(photos) != 2 {
		t.Fatalf("got %d photos, want 2", len(photos))
	}
	if photos[0].FileURL != "https://remote.example/api/v1/files/trails/trail1/first.jpg" {
		t.Fatalf("unexpected first photo url %q", photos[0].FileURL)
	}
	if photos[0].IsThumbnail {
		t.Fatal("first photo should not be thumbnail")
	}
	if !photos[1].IsThumbnail {
		t.Fatal("second photo should be thumbnail")
	}
}

func TestSyncRecordPhotoListSkipsFallbackWithoutAssetExpandsOrLegacyPhotos(t *testing.T) {
	photos, authoritative := syncRecordPhotoList("trail", "trail1", "https://remote.example", map[string]any{}, false)

	if authoritative {
		t.Fatal("fallback response without asset expands or legacy photos should not be authoritative")
	}
	if len(photos) != 0 {
		t.Fatalf("got %d photos, want 0", len(photos))
	}
}

func TestSyncRecordPhotoListTreatsPrimaryEmptyAssetExpandsAsAuthoritative(t *testing.T) {
	photos, authoritative := syncRecordPhotoList("trail", "trail1", "https://remote.example", map[string]any{}, true)

	if !authoritative {
		t.Fatal("primary response with requested asset expands should be authoritative")
	}
	if len(photos) != 0 {
		t.Fatalf("got %d photos, want 0", len(photos))
	}
}

func TestSyncRecordPhotoListTreatsLegacyEmptyPhotosAsAuthoritative(t *testing.T) {
	photos, authoritative := syncRecordPhotoList("trail", "trail1", "https://remote.example", map[string]any{
		"photos": []any{},
	}, false)

	if !authoritative {
		t.Fatal("legacy photos field should make fallback response authoritative")
	}
	if len(photos) != 0 {
		t.Fatalf("got %d photos, want 0", len(photos))
	}
}

func TestShouldRetryRemoteFetchWithoutAssetExpands(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "bad request", err: &remoteFetchStatusError{StatusCode: http.StatusBadRequest}, want: true},
		{name: "uri too long", err: &remoteFetchStatusError{StatusCode: http.StatusRequestURITooLong}, want: true},
		{name: "headers too large", err: &remoteFetchStatusError{StatusCode: http.StatusRequestHeaderFieldsTooLarge}, want: true},
		{name: "unprocessable entity", err: &remoteFetchStatusError{StatusCode: http.StatusUnprocessableEntity}, want: true},
		{name: "rate limited", err: &remoteFetchStatusError{StatusCode: http.StatusTooManyRequests}, want: false},
		{name: "server error", err: &remoteFetchStatusError{StatusCode: http.StatusBadGateway}, want: false},
		{name: "network error", err: errors.New("connection reset"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetryRemoteFetchWithoutAssetExpands(tt.err); got != tt.want {
				t.Fatalf("retry = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncRecordFilesSkipsEmptyGPX(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	record := core.NewRecord(core.NewBaseCollection("summit_logs"))

	syncRecordFiles(
		t.Context(),
		record,
		"summit_logs",
		"9300091648c5ddc",
		server.URL,
		map[string]any{"gpx": ""},
	)

	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Fatalf("empty gpx triggered %d file requests, want 0", got)
	}
}

func expandSet(value string) map[string]bool {
	result := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result[part] = true
		}
	}
	return result
}

func countExpand(value string, needle string) int {
	count := 0
	for _, part := range strings.Split(value, ",") {
		if strings.TrimSpace(part) == needle {
			count++
		}
	}
	return count
}

func allRemoteTrailSyncExpandPaths() []string {
	paths := append([]string{}, remoteTrailSyncCoreExpandPaths...)
	return append(paths, remoteTrailSyncAssetExpandPaths...)
}
