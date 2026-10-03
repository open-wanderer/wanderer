package util

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// All downloads of one inbound activity share one budget of attempts, stored
// bytes and time.

// attachmentBudgetServer serves body(path) and counts the requests. Paths whose
// number is a multiple of five answer 404 (they still count against the budget).
func attachmentBudgetServer(t *testing.T, body func(path string) []byte) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if strings.HasSuffix(r.URL.Path, "5") || strings.HasSuffix(r.URL.Path, "0") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body(r.URL.Path))
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

func attachmentBudgetUseLoopback(t *testing.T) {
	t.Helper()
	attachmentUseClient(t, &http.Client{Timeout: 30 * time.Second})
}

func TestRemoteAttachmentBudgetUsesDocumentedLimits(t *testing.T) {
	ctx, cancel := WithRemoteAttachmentBudget(context.Background())
	defer cancel()

	b := remoteAttachmentBudgetFrom(ctx)
	if b == nil {
		t.Fatal("context carries no attachment budget")
	}
	if b.bytesLeft != RemoteActivityAttachmentMaxBytes || b.filesLeft != RemoteActivityAttachmentMaxFiles {
		t.Errorf("budget = %d bytes, %d files; want %d bytes, %d files",
			b.bytesLeft, b.filesLeft, RemoteActivityAttachmentMaxBytes, RemoteActivityAttachmentMaxFiles)
	}
}

func TestRemoteAttachmentBudgetCountsFiles(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	attachmentBudgetUseLoopback(t)
	server, hits := attachmentBudgetServer(t, func(string) []byte { return []byte("data") })

	ctx, cancel := WithRemoteAttachmentBudget(context.Background())
	defer cancel()

	for i := 0; i < RemoteActivityAttachmentMaxFiles; i++ {
		f, cleanup, err := DownloadRemoteFile(ctx, server.URL+"/f"+strconv.Itoa(i), RemotePhotoMaxBytes, server.URL)
		if f == nil && err == nil {
			t.Fatalf("attempt %d: no file and no error", i)
		}
		if errors.Is(err, ErrRemoteAttachmentBudgetExhausted) {
			t.Fatalf("attempt %d of %d was refused by the budget", i+1, RemoteActivityAttachmentMaxFiles)
		}
		cleanup()
	}
	if n := int(atomic.LoadInt32(hits)); n != RemoteActivityAttachmentMaxFiles {
		t.Fatalf("server received %d requests, want %d", n, RemoteActivityAttachmentMaxFiles)
	}

	_, cleanup, err := DownloadRemoteFile(ctx, server.URL+"/one-too-many", RemotePhotoMaxBytes, server.URL)
	cleanup()
	if !errors.Is(err, ErrRemoteAttachmentBudgetExhausted) {
		t.Errorf("error after the budget was spent = %v, want ErrRemoteAttachmentBudgetExhausted", err)
	}
	if n := int(atomic.LoadInt32(hits)); n != RemoteActivityAttachmentMaxFiles {
		t.Errorf("server received %d requests, want no request after the budget was spent (%d)", n, RemoteActivityAttachmentMaxFiles)
	}
	if left := attachmentLeftovers(t, tmp); len(left) != 0 {
		t.Errorf("temp entries left behind: %v", left)
	}
}

func TestRemoteAttachmentBudgetCountsBytes(t *testing.T) {
	const budgetBytes = 1000
	// The byte budget is tested at a small size; the real limits are checked by
	// TestRemoteAttachmentBudgetUsesDocumentedLimits.
	t.Cleanup(SetRemoteAttachmentBudgetLimitsForTesting(budgetBytes, RemoteActivityAttachmentMaxFiles))
	attachmentBudgetUseLoopback(t)

	t.Run("a download of exactly the budget succeeds and the next is refused", func(t *testing.T) {
		tmp := attachmentIsolatedTempDir(t)
		var hits int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&hits, 1)
			_, _ = w.Write(make([]byte, budgetBytes))
		}))
		t.Cleanup(server.Close)

		ctx, cancel := WithRemoteAttachmentBudget(context.Background())
		defer cancel()

		f, cleanup, err := DownloadRemoteFile(ctx, server.URL+"/a", RemotePhotoMaxBytes, server.URL)
		if err != nil {
			t.Fatalf("download of exactly the byte budget: %v", err)
		}
		if f.Size != budgetBytes {
			t.Errorf("size = %d, want %d", f.Size, budgetBytes)
		}
		cleanup()

		_, cleanup, err = DownloadRemoteFile(ctx, server.URL+"/b", RemotePhotoMaxBytes, server.URL)
		cleanup()
		if !errors.Is(err, ErrRemoteAttachmentBudgetExhausted) {
			t.Errorf("error after the byte budget was spent = %v, want ErrRemoteAttachmentBudgetExhausted", err)
		}
		if n := atomic.LoadInt32(&hits); n != 1 {
			t.Errorf("server received %d requests, want 1", n)
		}
		if left := attachmentLeftovers(t, tmp); len(left) != 0 {
			t.Errorf("temp entries left behind: %v", left)
		}
	})

	t.Run("a body past the remaining bytes is refused and not charged", func(t *testing.T) {
		tmp := attachmentIsolatedTempDir(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/big":
				_, _ = w.Write(make([]byte, budgetBytes-10))
			case "/eleven":
				_, _ = w.Write(make([]byte, 11))
			default:
				_, _ = w.Write(make([]byte, 10))
			}
		}))
		t.Cleanup(server.Close)

		ctx, cancel := WithRemoteAttachmentBudget(context.Background())
		defer cancel()

		_, cleanup, err := DownloadRemoteFile(ctx, server.URL+"/big", 1024*1024, server.URL)
		if err != nil {
			t.Fatalf("first download: %v", err)
		}
		cleanup()

		_, cleanup, err = DownloadRemoteFile(ctx, server.URL+"/eleven", 1024, server.URL)
		cleanup()
		if err == nil {
			t.Fatal("an 11-byte body with 10 bytes of budget left was accepted")
		}
		if left := attachmentLeftovers(t, tmp); len(left) != 0 {
			t.Errorf("temp entries left behind: %v", left)
		}

		f, cleanup, err := DownloadRemoteFile(ctx, server.URL+"/ten", 1024, server.URL)
		if err != nil {
			t.Fatalf("the refused download was charged against the byte budget: %v", err)
		}
		if f.Size != 10 {
			t.Errorf("size = %d, want 10", f.Size)
		}
		cleanup()
	})
}

func TestWithRemoteAttachmentBudgetDeadlineAndNesting(t *testing.T) {
	attachmentIsolatedTempDir(t)
	attachmentBudgetUseLoopback(t)
	server, hits := attachmentBudgetServer(t, func(string) []byte { return []byte("data") })

	before := time.Now()
	outer, cancel := WithRemoteAttachmentBudget(context.Background())
	defer cancel()

	deadline, ok := outer.Deadline()
	if !ok {
		t.Fatal("the budget context has no deadline")
	}
	if limit := before.Add(RemoteActivityAttachmentTimeout); deadline.After(limit.Add(time.Second)) {
		t.Errorf("deadline %v is later than now plus RemoteActivityAttachmentTimeout (%v)", deadline, limit)
	}

	inner, innerCancel := WithRemoteAttachmentBudget(outer)
	defer innerCancel()

	for i := 0; i < RemoteActivityAttachmentMaxFiles; i++ {
		c := outer
		if i%2 == 1 {
			c = inner
		}
		_, cleanup, err := DownloadRemoteFile(c, server.URL+"/f"+strconv.Itoa(i), RemotePhotoMaxBytes, server.URL)
		cleanup()
		if errors.Is(err, ErrRemoteAttachmentBudgetExhausted) {
			t.Fatalf("attempt %d was refused early", i+1)
		}
	}
	for _, c := range []context.Context{outer, inner} {
		_, cleanup, err := DownloadRemoteFile(c, server.URL+"/over", RemotePhotoMaxBytes, server.URL)
		cleanup()
		if !errors.Is(err, ErrRemoteAttachmentBudgetExhausted) {
			t.Errorf("error after the shared budget was spent = %v, want ErrRemoteAttachmentBudgetExhausted", err)
		}
	}
	if n := int(atomic.LoadInt32(hits)); n != RemoteActivityAttachmentMaxFiles {
		t.Errorf("server received %d requests, want %d", n, RemoteActivityAttachmentMaxFiles)
	}
}

func TestRemoteAttachmentBudgetConcurrent(t *testing.T) {
	attachmentIsolatedTempDir(t)
	attachmentBudgetUseLoopback(t)
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("data"))
	}))
	t.Cleanup(server.Close)

	ctx, cancel := WithRemoteAttachmentBudget(context.Background())
	defer cancel()

	total := 2 * RemoteActivityAttachmentMaxFiles
	var refused int32
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, cleanup, err := DownloadRemoteFile(ctx, fmt.Sprintf("%s/c%d", server.URL, i), RemotePhotoMaxBytes, server.URL)
			cleanup()
			if errors.Is(err, ErrRemoteAttachmentBudgetExhausted) {
				atomic.AddInt32(&refused, 1)
			}
		}(i)
	}
	wg.Wait()

	if n := int(atomic.LoadInt32(&hits)); n != RemoteActivityAttachmentMaxFiles {
		t.Errorf("server received %d requests, want exactly %d", n, RemoteActivityAttachmentMaxFiles)
	}
	if n := int(atomic.LoadInt32(&refused)); n != total-RemoteActivityAttachmentMaxFiles {
		t.Errorf("%d downloads were refused by the budget, want %d", n, total-RemoteActivityAttachmentMaxFiles)
	}
}

func TestDownloadRemoteFileWithoutBudgetIsBounded(t *testing.T) {
	attachmentIsolatedTempDir(t)
	attachmentBudgetUseLoopback(t)
	server, _ := attachmentBudgetServer(t, func(string) []byte { return []byte("data") })

	f, cleanup, err := DownloadRemoteFile(context.Background(), server.URL+"/f1", RemotePhotoMaxBytes, server.URL)
	defer cleanup()
	if err != nil {
		t.Fatalf("download without a budget in the context: %v", err)
	}
	if f.Size != 4 {
		t.Errorf("size = %d, want 4", f.Size)
	}
}
