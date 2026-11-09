package util

import (
	"bytes"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFetchFederatedFileKeepsStagedPathUntilCleanup(t *testing.T) {
	body := append([]byte("GIF89a"), bytes.Repeat([]byte("x"), 4096)...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) }))
	defer server.Close()
	restore := SetRemoteAttachmentClientForTesting(server.Client())
	defer restore()
	file, cleanup, err := FetchFederatedFile(t.Context(), server.URL+"/camera.jpg", server.URL, "photo", int64(len(body)))
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	staged, ok := file.Reader.(*filesystem.PathReader)
	if !ok {
		t.Fatalf("photo buffered in memory: %T", file.Reader)
	}
	if file.Size != int64(len(body)) || MediaFileExtension("image/gif") != ".gif" || !strings.HasSuffix(file.Name, ".gif") {
		t.Fatalf("wrong staged file: %+v", file)
	}
	reader, err := file.Reader.Open()
	if err != nil {
		t.Fatal(err)
	}
	copied, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || !bytes.Equal(copied, body) {
		t.Fatalf("staged bytes unavailable: %v", err)
	}
	cleanup()
	if _, err := os.Stat(staged.Path); !os.IsNotExist(err) {
		t.Fatalf("staged file remains: %v", err)
	}
	file, cleanup, err = FetchFederatedFile(t.Context(), server.URL+"/camera.jpg", server.URL, "photo", int64(len(body)-1))
	defer cleanup()
	if err == nil || file != nil {
		t.Fatalf("oversized photo accepted: %v %v", file, err)
	}
}
