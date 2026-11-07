package routes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
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

// regionListing returns the leaf entry for path from the real listing handler.
func regionListing(t *testing.T, app core.App, query, path string) map[string]any {
	t.Helper()
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/regions", RegionsList)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/regions"+query, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("listing: status %d: %s", response.Code, response.Body.String())
	}
	var entries []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry["path"] == path {
			return entry
		}
	}
	return nil
}

func enableRegion(t *testing.T, app core.App, path string, enabled bool) *core.Record {
	t.Helper()
	region, err := app.FindFirstRecordByFilter("regions", "path = {:path}", dbx.Params{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	region.Set("enabled", enabled)
	if err := app.Save(region); err != nil {
		t.Fatal(err)
	}
	return region
}

func TestRegionsListOffersArchivesOfRecoveredErrorRows(t *testing.T) {
	app := newRemoteRealSchemaApp(t)
	enableRegion(t, app, "monaco", true)
	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Load(map[string]any{
		"path": "monaco", "name": "Monaco", "status": "error", "dem_status": "error",
		"error_message": "rebuild failed", "vector_built_date": "20261001",
		"min_lon": 7.4, "min_lat": 43.7, "max_lon": 7.5, "max_lat": 43.8,
	})
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{regions.RegionArchivePath(app, "monaco"), regions.RegionDemPath(app, "monaco")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("pmtiles"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := regions.ReconcileArchives(app); err != nil {
		t.Fatal(err)
	}

	entry := regionListing(t, app, "?enabled=true", "monaco")
	if entry == nil {
		t.Fatal("monaco is not listed")
	}
	if entry["status"] != "ready" || entry["vector_url"] == nil || entry["dem_status"] != "ready" || entry["dem_url"] == nil {
		t.Errorf("listing entry = %v, want ready with both download URLs", entry)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("upstream down")
}

func TestRegionsListReportsAFirstBuildGeometryFailure(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = failingTransport{}
	t.Cleanup(func() { http.DefaultTransport = previous })

	app := newRemoteRealSchemaApp(t)
	enableRegion(t, app, "monaco", true)
	if entry := regionListing(t, app, "?enabled=true", "monaco"); entry == nil || entry["status"] != "building" {
		t.Fatalf("before the first pass: %v, want building", entry)
	}

	regions.BuildAllLocked(app)

	entry := regionListing(t, app, "?enabled=true", "monaco")
	if entry == nil {
		t.Fatal("monaco is not listed")
	}
	message, _ := entry["error"].(string)
	if entry["status"] != "error" || !strings.Contains(message, "raw.githubusercontent.com") {
		t.Errorf("listing entry = %v, want status error naming the upstreams", entry)
	}
	if _, ok := entry["bbox"]; ok {
		t.Errorf("listing entry = %v, want no bbox", entry)
	}

	// Disabled again, the failure leaves the filtered listing and stays
	// harmless in the full one.
	enableRegion(t, app, "monaco", false)
	if entry := regionListing(t, app, "?enabled=true", "monaco"); entry != nil {
		t.Errorf("disabled region in the filtered listing: %v", entry)
	}
	if entry := regionListing(t, app, "", "monaco"); entry == nil || entry["enabled"] != false || entry["vector_url"] != nil {
		t.Errorf("full listing entry = %v", entry)
	}
}

// archiveDownloadServer serves a sparse archive of the given size from a real
// HTTP server, through the same router and middleware as production. done is
// closed when the handler returns.
func archiveDownloadServer(t *testing.T, size int64, configure func(*httptest.Server)) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })

	archive := regions.RegionArchivePath(app, "monaco")
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(size); err != nil {
		t.Fatal(err)
	}
	file.Close()

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/regions/{id}/download", RegionArchiveDownload)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		mux.ServeHTTP(w, r)
	}))
	configure(server)
	t.Cleanup(server.Close)
	return server, done
}

func TestRegionArchiveDownloadOutlivesTheServerWriteTimeout(t *testing.T) {
	const size = 64 << 20 // far more than the socket buffers can absorb
	const serverTimeout = 300 * time.Millisecond

	for _, test := range []struct {
		name  string
		start func(*httptest.Server)
	}{
		{"http1", func(server *httptest.Server) { server.Start() }},
		{"http2", func(server *httptest.Server) {
			server.EnableHTTP2 = true
			server.StartTLS()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, _ := archiveDownloadServer(t, size, func(server *httptest.Server) {
				server.Config.WriteTimeout = serverTimeout
				server.Config.ReadTimeout = serverTimeout
				test.start(server)
			})

			response, err := server.Client().Get(server.URL + "/regions/monaco/download")
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()

			// Read at a pace that keeps the transfer going for several times
			// the server's timeout.
			started := time.Now()
			var received int64
			for {
				n, err := io.CopyN(io.Discard, response.Body, 1<<20)
				received += n
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("download cut after %v at %d of %d bytes: %v", time.Since(started), received, size, err)
				}
				time.Sleep(20 * time.Millisecond)
			}
			if received != size {
				t.Fatalf("received %d bytes, want %d", received, size)
			}
			if elapsed := time.Since(started); elapsed < 3*serverTimeout {
				t.Fatalf("transfer took %v, too short to prove anything against a %v timeout", elapsed, serverTimeout)
			}
		})
	}
}

func TestRegionArchiveDownloadDropsAStalledClient(t *testing.T) {
	previous := regionDownloadStallTimeout
	regionDownloadStallTimeout = 300 * time.Millisecond
	t.Cleanup(func() { regionDownloadStallTimeout = previous })

	server, done := archiveDownloadServer(t, 64<<20, func(server *httptest.Server) {
		server.Config.WriteTimeout = time.Hour
		server.Start()
	})

	response, err := server.Client().Get(server.URL + "/regions/monaco/download")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err := io.CopyN(io.Discard, response.Body, 1<<20); err != nil {
		t.Fatal(err)
	}

	// The client stops reading here, so the server's writes block.
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("a client that stopped reading still holds the download open")
	}
}

func TestRegionArchiveDownloadEndsAtTheOverallLimit(t *testing.T) {
	previous := regionDownloadMaxDuration
	regionDownloadMaxDuration = 300 * time.Millisecond
	t.Cleanup(func() { regionDownloadMaxDuration = previous })

	const size = 64 << 20
	server, done := archiveDownloadServer(t, size, func(server *httptest.Server) {
		server.Config.WriteTimeout = time.Hour
		server.Start()
	})

	response, err := server.Client().Get(server.URL + "/regions/monaco/download")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	// This client never stalls, so only the overall limit can end the transfer.
	var received int64
	for {
		n, err := io.CopyN(io.Discard, response.Body, 256<<10)
		received += n
		if err != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if received >= size {
		t.Fatalf("received all %d bytes, the limit did not apply", received)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the handler is still running after the overall limit")
	}
}
