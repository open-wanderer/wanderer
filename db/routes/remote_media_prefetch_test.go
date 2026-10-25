package routes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"pocketbase/util"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestRemoteSyncDownloadsBeforeTakingWriterTransaction(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			collection := f.trails
			field := "gpx"
			if kind == "list" {
				collection = f.lists
				field = "avatar"
				collection.Fields.Add(&core.TextField{Name: "iri"}, &core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"})
			}
			collection.Fields.Add(&core.FileField{Name: field, MaxSelect: 1, MaxSize: remoteGPXMaxBytes})
			if err := f.app.Save(collection); err != nil {
				t.Fatal(err)
			}
			local := f.save(collection, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/remote", "author": f.remoteActor.Id, "needs_full_sync": true})
			started, release := make(chan struct{}), make(chan struct{})
			sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				<-release
				_, _ = w.Write([]byte("downloaded-file"))
			}))
			defer sink.Close()
			previousClient, previousFetch := newRemoteSyncHTTPClient, fetchRemoteSyncMedia
			t.Cleanup(func() { newRemoteSyncHTTPClient, fetchRemoteSyncMedia = previousClient, previousFetch })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "remote", field: "file.gpx", "author": f.remoteActor.Id, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}}}
			}
			var fetches atomic.Int32
			fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, filename string, maxBytes int64) (*filesystem.File, func(), error) {
				fetches.Add(1)
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, sink.URL, nil)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return nil, func() {}, err
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
				if err != nil {
					return nil, func() {}, err
				}
				file, err := filesystem.NewFileFromBytes(body, filename)
				return file, func() {}, err
			}
			finished := make(chan error, 1)
			go func() {
				var err error
				if kind == "trail" {
					_, err = performFullSync(f.app, t.Context(), nil, local)
				} else {
					_, err = performFullListSync(f.app, t.Context(), nil, local)
				}
				finished <- err
			}()
			select {
			case <-started:
			case err := <-finished:
				t.Fatalf("sync ended before download: %v", err)
			case <-time.After(3 * time.Second):
				t.Fatal("no media request")
			}
			writer := make(chan error, 1)
			go func() {
				writer <- f.app.RunInTransaction(func(app core.App) error {
					record := core.NewRecord(f.settingsCollection(t))
					record.Set("user", "independent-writer")
					return app.Save(record)
				})
			}()
			writerBlocked := false
			select {
			case err := <-writer:
				if err != nil {
					t.Error(err)
				}
			case <-time.After(250 * time.Millisecond):
				writerBlocked = true
			}
			close(release)
			if err := <-finished; err != nil {
				t.Fatal(err)
			}
			if writerBlocked {
				if err := <-writer; err != nil {
					t.Error(err)
				}
				t.Fatal("media download held the database writer")
			}
			if fetches.Load() != 1 {
				t.Fatalf("downloaded file %d times", fetches.Load())
			}
			stored, err := f.app.FindRecordById(collection, local.Id)
			if err != nil || stored.GetString(field) == "" || !stored.GetBool("full_sync_completed") {
				t.Fatalf("prepared file was not persisted: %v %v", stored, err)
			}
		})
	}
}

func (f *remoteSyncFixture) settingsCollection(t *testing.T) *core.Collection {
	t.Helper()
	collection, err := f.app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	return collection
}

func TestRemoteMediaPrefetchBudgetsAndDeadline(t *testing.T) {
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	requests := []remoteMediaRequest{}
	for i := 0; i < 3; i++ {
		requests = append(requests, remoteMediaRequest{url: "https://peer.example/" + strings.Repeat("x", i), origin: "https://peer.example", name: "file", maxBytes: remoteMediaMaxBytes})
	}
	var calls int
	fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, filename string, maxBytes int64) (*filesystem.File, func(), error) {
		calls++
		if maxBytes != remoteMediaMaxBytes {
			t.Fatalf("wrong remaining budget %d", maxBytes)
		}
		return &filesystem.File{Size: remoteMediaMaxBytes}, func() {}, nil
	}
	ctx, cleanup, err := prepareRemoteMedia(t.Context(), requests)
	defer cleanup()
	if err != nil || calls != 1 {
		t.Fatalf("byte cap failed: calls=%d error=%v", calls, err)
	}
	if _, err := util.PreparedRemoteSyncFile(ctx, requests[1].url, requests[1].origin, 1); err == nil {
		t.Fatal("budget-omitted file was available")
	}
	fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, filename string, maxBytes int64) (*filesystem.File, func(), error) {
		<-ctx.Done()
		return nil, func() {}, ctx.Err()
	}
	short, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if _, _, err := prepareRemoteMedia(short, requests); !errors.Is(err, errRemoteUnavailable) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline did not stop sync: %v", err)
	}
}

func TestRemoteMediaCollectorBoundsFilesAndNewPhotosPerTarget(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	collector := remoteMediaCollector{seen: map[string]bool{}}
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		photos := make([]any, 150)
		for i := range photos {
			photos[i] = strings.Repeat("a", i+1) + ".png"
		}
		collector.record(f.app, target, "remote", "https://remote.example", "https://remote.example/api/v1/"+target+"/remote", map[string]any{"photos": photos})
	}
	want := remoteMediaMaxFiles
	if len(collector.requests) != want {
		t.Fatalf("new photo requests=%d, expected %d", len(collector.requests), want)
	}
	for i := 0; i < 150; i++ {
		collector.add(remoteMediaRequest{url: "https://remote.example/" + strings.Repeat("b", i+1), origin: "https://remote.example", name: "photo", maxBytes: 1})
	}
	if len(collector.requests) != remoteMediaMaxFiles {
		t.Fatalf("global file cap=%d", len(collector.requests))
	}
}

func TestRemoteMediaCollectorUsesLegacyFieldLimits(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.trails.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 3, MaxSize: 1024})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "trail", "remote", "https://remote.example", "https://remote.example/api/v1/trail/remote", map[string]any{"photos": []any{"one.jpg", true, "two.jpg", "three.jpg", "four.jpg", "../skip.jpg"}})
	if len(collector.requests) != 3 {
		t.Fatalf("legacy photo requests=%d, want3", len(collector.requests))
	}
	for _, request := range collector.requests {
		if request.maxBytes != 1024 {
			t.Fatalf("field limit=%d, want1024", request.maxBytes)
		}
	}
}

func TestRemoteMediaPreparedFilesAreCleanedOnCompletionAndCancellation(t *testing.T) {
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "cancelled"}[cancelled], func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls, closes := 0, 0
			fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
				calls++
				if calls == 2 && cancelled {
					cancel()
				}
				return &filesystem.File{Size: 1}, func() { closes++ }, nil
			}
			requests := []remoteMediaRequest{{url: "https://remote.example/a", origin: "https://remote.example", maxBytes: 2}, {url: "https://remote.example/b", origin: "https://remote.example", maxBytes: 2}}
			_, cleanup, err := prepareRemoteMedia(ctx, requests)
			if cancelled {
				if !errors.Is(err, context.Canceled) || closes != 2 {
					t.Fatalf("cancel cleanup: closes=%d err=%v", closes, err)
				}
			} else {
				if err != nil || closes != 0 {
					t.Fatalf("files released before transaction: closes=%d err=%v", closes, err)
				}
				cleanup()
				if closes != 2 {
					t.Fatalf("prepared cleanup count=%d", closes)
				}
			}
		})
	}
}
