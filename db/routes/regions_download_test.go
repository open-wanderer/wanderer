package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

func TestRegionArchiveDownloadRejectsResumesAcrossRebuilds(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })

	archive := regions.RegionArchivePath(app, "monaco")
	write := func(content string, modTime time.Time) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(archive, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(archive, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/regions/{id}/download", RegionArchiveDownload)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	get := func(headers map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/regions/monaco/download", nil)
		for key, value := range headers {
			request.Header.Set(key, value)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}

	write("first build of the archive", time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC))
	first := get(nil)
	etag := first.Header().Get("ETag")
	if first.Code != http.StatusOK || etag == "" || first.Header().Get("Last-Modified") == "" {
		t.Fatalf("first download: status %d, ETag %q, Last-Modified %q", first.Code, etag, first.Header().Get("Last-Modified"))
	}

	resumed := get(map[string]string{"Range": "bytes=6-", "If-Range": etag})
	if resumed.Code != http.StatusPartialContent || resumed.Body.String() != "build of the archive" {
		t.Fatalf("resume of the same build: status %d, body %q", resumed.Code, resumed.Body.String())
	}

	write("second build, replaced overnight", time.Date(2026, 10, 8, 3, 0, 0, 0, time.UTC))
	stale := get(map[string]string{"Range": "bytes=6-", "If-Range": etag})
	if stale.Code != http.StatusOK || stale.Body.String() != "second build, replaced overnight" {
		t.Fatalf("resume across a rebuild: status %d, body %q; want the whole new file", stale.Code, stale.Body.String())
	}
	if stale.Header().Get("ETag") == etag {
		t.Error("ETag did not change when the archive was rebuilt")
	}
}
