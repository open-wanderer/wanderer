package util

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/doyensec/safeurl"
)

func TestRemoteSyncFileBoundsChunkedResponsesBeforeMaterialization(t *testing.T) {
	const limit = 5 << 20
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gpx+xml")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		length := limit
		if r.URL.Path == "/large" {
			length += 32 << 10
		}
		_, _ = io.CopyN(w, strings.NewReader(strings.Repeat("x", length)), int64(length))
	}))
	defer source.Close()
	previous := newRemoteSyncFileClient
	newRemoteSyncFileClient = func(config *safeurl.Config) HTTPDoer {
		local := *config
		local.AllowedIPs = []net.IP{net.ParseIP("127.0.0.1")}
		return safeurl.Client(&local)
	}
	t.Cleanup(func() { newRemoteSyncFileClient = previous })
	file, cleanup, err := DownloadRemoteSyncFile(t.Context(), source.URL+"/exact", source.URL, "route.gpx", limit)
	defer cleanup()
	if err != nil || file == nil || file.Size != limit {
		t.Fatalf("exact-cap response rejected: file=%v err=%v", file, err)
	}
	file, closeLarge, err := DownloadRemoteSyncFile(t.Context(), source.URL+"/large", source.URL, "route.gpx", limit)
	defer closeLarge()
	if err == nil || file != nil {
		t.Fatalf("oversized chunked response accepted: file=%v err=%v", file, err)
	}
	if federatedFileHTTPConfig.AllowedIPs != nil {
		t.Fatal("streaming fixture changed the shared file policy")
	}
}
