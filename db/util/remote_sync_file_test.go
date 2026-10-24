package util

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doyensec/safeurl"
)

type syncFileResponseBody struct {
	io.Reader
	closed bool
}

func (b *syncFileResponseBody) Close() error { b.closed = true; return nil }

func TestRemoteSyncFileStreamsToTemporaryStorageAndClosesResources(t *testing.T) {
	for _, scenario := range []string{"success", "status", "size", "request"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			body := &syncFileResponseBody{Reader: strings.NewReader("route")}
			client := &remoteFileClosingClient{HTTPDoer: &http.Client{Transport: attachmentRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				if deadline, ok := request.Context().Deadline(); !ok || time.Until(deadline) > RemoteAttachmentTimeout {
					t.Error("file request has no bounded deadline")
				}
				if scenario == "request" {
					return nil, errors.New("unavailable")
				}
				status := http.StatusOK
				if scenario == "status" {
					status = http.StatusNotFound
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: body, Request: request}, nil
			})}}
			previous := newRemoteSyncFileClient
			newRemoteSyncFileClient = func(*safeurl.Config) HTTPDoer { return client }
			t.Cleanup(func() { newRemoteSyncFileClient = previous })
			limit := int64(5)
			if scenario == "size" {
				limit = 4
			}
			file, cleanup, err := DownloadRemoteSyncFile(context.Background(), "https://remote.example:8443/route.gpx", "https://remote.example:8443/api/v1/trail/one", "route.gpx", limit)
			defer cleanup()
			if scenario == "success" {
				if err != nil || file == nil || string(attachmentReadAll(t, file)) != "route" {
					t.Fatalf("file unavailable: %v %v", file, err)
				}
				matches, _ := filepath.Glob(filepath.Join(dir, "wanderer-sync-file-*"))
				if len(matches) != 1 {
					t.Fatalf("temporary directories=%v", matches)
				}
			} else if err == nil || file != nil {
				t.Fatalf("failed file fetch accepted: %v %v", file, err)
			}
			if client.closed != 1 || (scenario != "request" && !body.closed) {
				t.Fatalf("resources not closed: idle=%d body=%v", client.closed, body.closed)
			}
			cleanup()
			if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
				t.Fatalf("temporary files retained: %v %v", entries, err)
			}
		})
	}
}

func TestRemoteSyncFileAggregateBudgetCountsFailedResponses(t *testing.T) {
	previous := newRemoteSyncFileClient
	newRemoteSyncFileClient = func(*safeurl.Config) HTTPDoer { return attachmentStaticClient([]byte("1234567890")) }
	t.Cleanup(func() { newRemoteSyncFileClient = previous })
	ctx := WithFederatedMediaByteBudget(t.Context(), 12)
	_, cleanup, err := DownloadRemoteSyncFile(ctx, "https://remote.example/a", "https://remote.example", "a", 5)
	cleanup()
	if err == nil {
		t.Fatal("oversized file accepted")
	}
	_, cleanup, err = DownloadRemoteSyncFile(ctx, "https://remote.example/b", "https://remote.example", "b", 10)
	cleanup()
	if !errors.Is(err, ErrFederatedMediaBudget) {
		t.Fatalf("failed response did not consume shared budget: %v", err)
	}
}

func TestRemoteSyncFileOriginPortAndRedirectPolicy(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetRequests.Add(1); _, _ = w.Write([]byte("other")) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, target.URL+"/photo.jpg", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("photo"))
	}))
	defer source.Close()
	previous := newRemoteSyncFileClient
	newRemoteSyncFileClient = func(config *safeurl.Config) HTTPDoer {
		local := *config
		local.AllowedIPs = []net.IP{net.ParseIP("127.0.0.1")}
		return safeurl.Client(&local)
	}
	t.Cleanup(func() { newRemoteSyncFileClient = previous })
	file, cleanup, err := DownloadRemoteSyncFile(t.Context(), source.URL+"/photo.jpg", source.URL, "photo.jpg", 5)
	defer cleanup()
	if err != nil || file == nil || string(attachmentReadAll(t, file)) != "photo" {
		t.Fatalf("origin-port file failed: %v %v", file, err)
	}
	redirected, closeRedirect, err := DownloadRemoteSyncFile(t.Context(), source.URL+"/redirect", source.URL, "photo.jpg", 5)
	closeRedirect()
	if err != nil || redirected == nil || targetRequests.Load() != 1 {
		t.Fatalf("public-file redirect on another port failed: hits=%d err=%v", targetRequests.Load(), err)
	}
	if federatedFileHTTPConfig.AllowedIPs != nil {
		t.Fatal("fixture changed the shared file policy")
	}
}
