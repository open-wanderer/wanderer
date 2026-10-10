package util

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type attachmentRoundTripFunc func(*http.Request) (*http.Response, error)

func (f attachmentRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func attachmentBodyResponse(r *http.Request, body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    r,
	}
}

// attachmentStaticClient serves body for every request without any network.
func attachmentStaticClient(body []byte) *http.Client {
	return &http.Client{Transport: attachmentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return attachmentBodyResponse(r, body), nil
	})}
}

func attachmentUseClient(t *testing.T, c HTTPDoer) {
	t.Helper()
	restore := SetRemoteAttachmentClientForTesting(c)
	t.Cleanup(restore)
}

// attachmentIsolatedTempDir points os.TempDir at a fresh directory so the test
// sees every temp file DownloadRemoteFile leaves behind.
func attachmentIsolatedTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	return dir
}

func attachmentLeftovers(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "wanderer-attachment-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func attachmentReadAll(t *testing.T, f *filesystem.File) []byte {
	t.Helper()
	r, err := f.Reader.Open()
	if err != nil {
		t.Fatalf("open file reader: %v", err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return b
}

func TestRemotePhotoLimit(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	add := func(name string, fields ...core.Field) {
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatalf("save collection %s: %v", name, err)
		}
	}
	add("limit_seven", &core.FileField{Name: "photos", MaxSelect: 7, MaxSize: RemotePhotoMaxBytes})
	add("limit_ninety_nine", &core.FileField{Name: "photos", MaxSelect: 99, MaxSize: RemotePhotoMaxBytes})
	add("limit_single", &core.FileField{Name: "photos", MaxSelect: 1, MaxSize: RemotePhotoMaxBytes})
	add("limit_no_photos", &core.TextField{Name: "name"})
	add("limit_text_photos", &core.TextField{Name: "photos"})

	cases := []struct {
		collection string
		want       int
	}{
		{"limit_seven", 7},
		{"limit_ninety_nine", 99},
		{"limit_single", 1},
		{"limit_no_photos", RemoteMaxPhotosPerObject},
		{"limit_text_photos", RemoteMaxPhotosPerObject},
		{"no_such_collection", RemoteMaxPhotosPerObject},
	}
	for _, tc := range cases {
		if got := RemotePhotoLimit(app, tc.collection); got != tc.want {
			t.Errorf("RemotePhotoLimit(%q) = %d, want %d", tc.collection, got, tc.want)
		}
	}
}

func TestDownloadRemoteFileHasDeadline(t *testing.T) {
	var deadline time.Time
	var hasDeadline bool
	attachmentUseClient(t, &http.Client{Transport: attachmentRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		deadline, hasDeadline = r.Context().Deadline()
		return attachmentBodyResponse(r, []byte("data")), nil
	})})
	attachmentIsolatedTempDir(t)

	_, cleanup, err := DownloadRemoteFile(context.Background(), "http://example.com/a.gpx", RemoteGPXMaxBytes, "")
	if err != nil {
		t.Fatalf("DownloadRemoteFile: %v", err)
	}
	defer cleanup()

	if !hasDeadline {
		t.Fatal("request context has no deadline")
	}
	if remaining := time.Until(deadline); remaining > RemoteAttachmentTimeout {
		t.Fatalf("deadline is %s away, want at most %s", remaining, RemoteAttachmentTimeout)
	}
}

func TestDownloadRemoteFileSizeLimit(t *testing.T) {
	const limit = 16
	attachmentIsolatedTempDir(t)

	attachmentUseClient(t, attachmentStaticClient(bytes.Repeat([]byte("x"), limit+1)))
	if _, cleanup, err := DownloadRemoteFile(context.Background(), "http://example.com/track.gpx", limit, ""); err == nil {
		cleanup()
		t.Fatal("body of limit+1 bytes was accepted")
	}

	exact := bytes.Repeat([]byte("y"), limit)
	attachmentUseClient(t, attachmentStaticClient(exact))
	f, cleanup, err := DownloadRemoteFile(context.Background(), "http://example.com/track.gpx", limit, "")
	if err != nil {
		t.Fatalf("body of exactly limit bytes was refused: %v", err)
	}
	defer cleanup()
	if got := attachmentReadAll(t, f); !bytes.Equal(got, exact) {
		t.Errorf("file content = %q, want %q", got, exact)
	}
	if f.Size != limit {
		t.Errorf("file size = %d, want %d", f.Size, limit)
	}
	if f.OriginalName != "track.gpx" {
		t.Errorf("OriginalName = %q, want track.gpx", f.OriginalName)
	}
	if !strings.HasPrefix(f.Name, "track_") || !strings.HasSuffix(f.Name, ".gpx") {
		t.Errorf("Name = %q, want track_<random>.gpx", f.Name)
	}
}

func TestDownloadRemoteFileStreamsToDisk(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	attachmentUseClient(t, attachmentStaticClient([]byte("gpx-body")))

	f, cleanup, err := DownloadRemoteFile(context.Background(), "http://example.com/track.gpx", RemoteGPXMaxBytes, "")
	if err != nil {
		t.Fatalf("DownloadRemoteFile: %v", err)
	}
	reader, ok := f.Reader.(*filesystem.PathReader)
	if !ok {
		cleanup()
		t.Fatalf("file reader is %T, want *filesystem.PathReader", f.Reader)
	}
	if _, err := os.Stat(reader.Path); err != nil {
		cleanup()
		t.Fatalf("downloaded file is not on disk before cleanup: %v", err)
	}
	if !strings.HasPrefix(reader.Path, tmp) {
		cleanup()
		t.Fatalf("file %s is not under the temp dir %s", reader.Path, tmp)
	}
	if got := attachmentReadAll(t, f); string(got) != "gpx-body" {
		t.Errorf("file content = %q", got)
	}

	cleanup()
	if _, err := os.Stat(reader.Path); !os.IsNotExist(err) {
		t.Errorf("file still exists after cleanup (stat error: %v)", err)
	}
	if left := attachmentLeftovers(t, tmp); len(left) != 0 {
		t.Errorf("temp entries left after cleanup: %v", left)
	}
}

func TestDownloadRemoteFileSizeLimitRemovesTemp(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	attachmentUseClient(t, attachmentStaticClient(bytes.Repeat([]byte("x"), 64)))

	f, cleanup, err := DownloadRemoteFile(context.Background(), "http://example.com/big.gpx", 16, "")
	if cleanup == nil {
		t.Fatal("cleanup must never be nil")
	}
	defer cleanup()
	if err == nil {
		t.Fatalf("oversized body was accepted (file %v)", f)
	}
	if left := attachmentLeftovers(t, tmp); len(left) != 0 {
		t.Errorf("temp entries left behind by the refused download: %v", left)
	}
}

func TestDownloadRemoteFileDefaultRefusesLoopback(t *testing.T) {
	restore := SetRemoteAttachmentClientForTesting(nil)
	defer restore()
	attachmentIsolatedTempDir(t)

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("secret"))
	}))
	defer server.Close()

	f, cleanup, err := DownloadRemoteFile(context.Background(), server.URL+"/internal.gpx", RemoteGPXMaxBytes, "")
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Fatalf("loopback URL was downloaded (file %v)", f)
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("loopback server received %d requests, want 0", n)
	}
}
