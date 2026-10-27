package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/util"
)

func TestRemoteSyncGroupsRegistrableDomains(t *testing.T) {
	for _, tc := range []struct{ host, domain string }{
		{"a.routes.example.org", "example.org"}, {"B.ROUTES.EXAMPLE.ORG.", "example.org"},
		{"a.example.co.uk", "example.co.uk"}, {"b.example.co.uk", "example.co.uk"},
		{"one.github.io", "one.github.io"}, {"two.github.io", "two.github.io"},
		{"a.bücher.de", "xn--bcher-kva.de"}, {"b.xn--bcher-kva.de", "xn--bcher-kva.de"},
		{"::ffff:192.0.2.1", "192.0.2.1"}, {"2001:0db8::1", "2001:db8::1"},
		{"localhost", "localhost"},
	} {
		if got := remoteSyncDomain(tc.host); got != tc.domain {
			t.Errorf("%s grouped as %s, want %s", tc.host, got, tc.domain)
		}
	}
}

func TestRemoteSyncMediaPlansHonorRecordIDField(t *testing.T) {
	f := newRemoteSyncFixture(t)
	id := strings.Repeat("r", 16)
	data := map[string]any{"photos": []any{"first.jpg", "second.jpg"}}
	collector := remoteMediaCollector{seen: map[string]bool{}}
	collector.record(f.app, "waypoint", id, "https://remote.example", "https://remote.example/api/v1/waypoint/"+id, data)
	if len(collector.requests) != 0 {
		t.Fatal("record id outside the collection contract reached the media plan")
	}
	field := f.waypoints.Fields.GetByName("id").(*core.TextField)
	field.Max = 20
	if err := f.app.Save(f.waypoints); err != nil {
		t.Fatal(err)
	}
	collector.record(f.app, "waypoint", id, "https://remote.example", "https://remote.example/api/v1/waypoint/"+id, data)
	if len(collector.requests) == 0 {
		t.Fatal("configured record id maximum was ignored")
	}
}

func TestRemoteSyncChecksIdentityBeforeMediaPreparation(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, variant := range []string{"record-id", "object-iri", "author-iri"} {
		t.Run(variant, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			record := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "name": "Saved copy", "author": f.remoteActor.Id, "needs_full_sync": true, "full_sync_completed": true})
			payload := map[string]any{"id": "copy", "iri": record.GetString("iri"), "name": "Updated copy", "photos": []any{"first.jpg"}, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}
			switch variant {
			case "record-id":
				payload["id"] = strings.Repeat("r", 64<<10)
			case "object-iri":
				payload["iri"] = "https://remote.example/api/v1/trail/another"
			case "author-iri":
				payload["expand"].(map[string]any)["author"] = map[string]any{"iri": f.localActor.GetString("iri")}
			}
			previousClient, previousMedia := newRemoteSyncHTTPClient, fetchRemoteSyncMedia
			defer func() { newRemoteSyncHTTPClient, fetchRemoteSyncMedia = previousClient, previousMedia }()
			newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
			var mediaCalls atomic.Int32
			fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
				mediaCalls.Add(1)
				return nil, func() {}, errors.New("unexpected file request")
			}
			if _, err := performFullSync(f.app, t.Context(), nil, record.Fresh()); err == nil {
				t.Fatal("invalid identity was accepted")
			}
			if mediaCalls.Load() != 0 {
				t.Fatal("media were prepared before the identity check")
			}
			f.assertUnchanged(record, "iri", "name", "author", "needs_full_sync", "full_sync_completed")
		})
	}
}

func TestRemoteSyncRetainsListReferencesWithInvalidChildIDs(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	previous := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/previous", "name": "Saved child", "author": f.remoteActor.Id})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/copy", "author": f.remoteActor.Id, "trails": []string{previous.Id}})
	bad := map[string]any{"id": strings.Repeat("r", 64<<10), "iri": previous.GetString("iri"), "name": "Updated child"}
	payload := map[string]any{"id": "copy", "iri": list.GetString("iri"), "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "trails": []any{bad, map[string]any{"id": "valid", "name": "Valid child"}}}}
	old := newRemoteSyncHTTPClient
	defer func() { newRemoteSyncHTTPClient = old }()
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
	result, err := performFullListSync(f.app, t.Context(), nil, list)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(result.GetStringSlice("trails"), previous.Id) || len(result.GetStringSlice("trails")) != 2 {
		t.Fatalf("previous reference was lost: %v", result.GetStringSlice("trails"))
	}
	f.assertUnchanged(previous, "iri", "name", "author")
}

func TestRemoteSyncBoundsNetworkAndStagingSeparately(t *testing.T) {
	f := newRemoteSyncFixture(t)
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), remoteSyncStagingKey{}, true))
	defer cancel()
	for i := range remoteSyncFetchConcurrency {
		release, err := acquireRemoteSyncFetch(ctx, fmt.Sprintf("https://target%d.example/file", i))
		if err != nil {
			t.Fatal(err)
		}
		defer release()
	}
	waitCtx, stop := context.WithTimeout(ctx, 10*time.Millisecond)
	defer stop()
	if release, err := acquireRemoteSyncFetch(waitCtx, "https://another.example/file"); !errors.Is(err, context.DeadlineExceeded) {
		if release != nil {
			release()
		}
		t.Fatalf("network capacity ignored: %v", err)
	}
	if len(remoteSyncSlots) != 0 {
		t.Fatal("network permits consumed processing capacity")
	}
	// A queued stage has no response, file or map yet.
	for range remoteSyncNetworkConcurrency {
		remoteSyncNetworkSlots <- struct{}{}
	}
	defer func() {
		for range remoteSyncNetworkConcurrency {
			<-remoteSyncNetworkSlots
		}
	}()
	record := core.NewRecord(f.trails)
	record.Set("iri", "https://remote.example/api/v1/trail/copy")
	called := false
	_, err := limitedRemoteSyncWithin(f.app, t.Context(), "trails", record, 10*time.Millisecond, func(context.Context) (*core.Record, error) { called = true; return nil, nil })
	if !errors.Is(err, context.DeadlineExceeded) || called {
		t.Fatalf("staging capacity ignored: %v called%v", err, called)
	}
}

type stagedSyncTransport func(*http.Request) (*http.Response, error)

func (f stagedSyncTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRemoteSyncSubdomainsLeaveCapacityForFirstViews(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, count := range []int{4, 8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			entered := make(chan struct{}, count)
			var calls atomic.Int32
			previous := newRemoteSyncHTTPClient
			defer func() { newRemoteSyncHTTPClient = previous }()
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: stagedSyncTransport(func(request *http.Request) (*http.Response, error) {
					if strings.HasSuffix(request.URL.Hostname(), ".routes.example.org") {
						calls.Add(1)
						entered <- struct{}{}
						<-request.Context().Done()
						return nil, request.Context().Err()
					}
					return (richTextSyncTransport{map[string]any{"id": "first", "iri": "https://remote.example/api/v1/trail/first", "name": "First view", "public": true, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}}).RoundTrip(request)
				})}
			}
			var wait sync.WaitGroup
			for i := range count {
				record := core.NewRecord(f.trails)
				record.Set("iri", fmt.Sprintf("https://sub%d.routes.example.org/api/v1/trail/copy", i))
				record.Set("author", f.remoteActor.Id)
				wait.Add(1)
				go func() { defer wait.Done(); _, _ = performFullSync(f.app, ctx, nil, record) }()
			}
			defer wait.Wait()
			defer cancel()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("first origin fetch did not start")
			}
			deadline := time.Now().Add(time.Second)
			for {
				remoteSyncHosts.Lock()
				host := remoteSyncHosts.hosts[f.app.DataDir()+"\x00example.org"]
				users := 0
				if host != nil {
					users = host.users
				}
				remoteSyncHosts.Unlock()
				if users == count {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("registered %d of %d domain waiters", users, count)
				}
				time.Sleep(time.Millisecond)
			}
			first := core.NewRecord(f.trails)
			first.Set("iri", "https://remote.example/api/v1/trail/first")
			first.Set("author", f.remoteActor.Id)
			viewCtx, stop := context.WithTimeout(t.Context(), time.Second)
			defer stop()
			started := time.Now()
			result, err := performFullSync(f.app, viewCtx, nil, first)
			if err != nil || result == nil || result.GetString("name") != "First view" {
				t.Fatalf("unrelated first view delayed %s: result%v error%v", time.Since(started), result, err)
			}
			if got := calls.Load(); got != 1 {
				t.Errorf("one domain occupied %d network places", got)
			}
		})
	}
}

func TestRemoteSyncNetworkPhasesReleaseProcessingCapacity(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, phase := range []string{"document", "author", "contribution", "media"} {
		t.Run(phase, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			entered := make(chan struct{}, 8)
			block := func(ctx context.Context) error { entered <- struct{}{}; <-ctx.Done(); return ctx.Err() }
			previousClient, previousActor, previousMedia := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI, fetchRemoteSyncMedia
			defer func() {
				newRemoteSyncHTTPClient, remoteSyncGetActorByIRI, fetchRemoteSyncMedia = previousClient, previousActor, previousMedia
			}()
			payload := map[string]any{"id": "copy", "iri": "https://remote.example/api/v1/trail/copy", "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}
			if phase == "author" {
				payload["expand"].(map[string]any)["author"] = map[string]any{"iri": "https://remote.example/api/v1/actor/second"}
				remoteSyncGetActorByIRI = func(_ core.App, ctx context.Context, _ string, _ bool) (*core.Record, error) { return nil, block(ctx) }
			}
			if phase == "contribution" {
				payload["expand"].(map[string]any)["summit_logs_via_trail"] = []any{map[string]any{"id": "visit", "iri": "https://contributor.example/api/v1/summit-log/visit"}}
			}
			if phase == "media" {
				payload["gpx"] = "route.gpx"
				fetchRemoteSyncMedia = func(ctx context.Context, _, _, _ string, _ int64) (*filesystem.File, func(), error) {
					return nil, func() {}, block(ctx)
				}
			}
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: stagedSyncTransport(func(request *http.Request) (*http.Response, error) {
					if phase == "document" || request.URL.Hostname() == "contributor.example" {
						return nil, block(request.Context())
					}
					return (richTextSyncTransport{payload}).RoundTrip(request)
				})}
			}
			record := core.NewRecord(f.trails)
			record.Set("iri", payload["iri"])
			record.Set("author", f.remoteActor.Id)
			done := make(chan error, 1)
			go func() { _, err := performFullSync(f.app, ctx, nil, record); done <- err }()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("network phase never started")
			}
			if got := len(remoteSyncSlots); got != 0 {
				t.Errorf("network wait retained %d processing permits", got)
			}
			cancel()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("canceled network phase did not finish")
			}
		})
	}
}

func TestRemoteSyncDocumentStagingCleansTemporaryFiles(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)
	for _, variant := range []string{"complete", "too-large", "invalid", "canceled", "read-error"} {
		t.Run(variant, func(t *testing.T) {
			body := io.ReadCloser(io.NopCloser(strings.NewReader(`{"id":"copy"}`)))
			if variant == "too-large" {
				body = io.NopCloser(strings.NewReader(strings.Repeat(" ", int(util.FederationSyncBodyLimit)+1)))
			}
			if variant == "invalid" {
				body = io.NopCloser(strings.NewReader(`{`))
			}
			if variant == "read-error" {
				body = io.NopCloser(stagingReadError{})
			}
			client := &http.Client{Transport: stagedSyncTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: body, Header: http.Header{}}, nil
			})}
			remoteURL, _ := url.Parse("https://remote.example/api/v1/trail/copy")
			func() {
				document, err := fetchRemoteSyncDocument(t.Context(), client, remoteURL, "trail")
				if err != nil {
					if variant != "too-large" && variant != "read-error" {
						t.Fatal(err)
					}
					return
				}
				defer document.close()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if variant == "canceled" {
					cancel()
				}
				_, err = document.process(ctx, func(data map[string]any) (*core.Record, error) { _, err := json.Marshal(data); return nil, err })
				if variant == "complete" && err != nil {
					t.Fatal(err)
				}
				if (variant == "invalid" || variant == "canceled") && err == nil {
					t.Fatal("invalid document was processed")
				}
			}()
			files, err := filepath.Glob(filepath.Join(directory, "wanderer-sync-document-*"))
			if err != nil || len(files) != 0 {
				t.Fatalf("temporary documents left behind: %v %v", files, err)
			}
		})
	}
}

type stagingReadError struct{}

func (stagingReadError) Read([]byte) (int, error) { return 0, errors.New("response interrupted") }

func TestRemoteSyncNetworkPlanDoesNotRetainMetadata(t *testing.T) {
	f := newRemoteSyncFixture(t)
	file, err := os.CreateTemp(t.TempDir(), "document-")
	if err != nil {
		t.Fatal(err)
	}
	document := &remoteSyncDocument{file: file}
	defer document.close()
	payload := map[string]any{"id": "copy", "description": strings.Repeat("details", 1000), "extra": map[string]any{"nested": []any{map[string]any{"ignored": "value"}}}, "photos": []any{"first.jpg"}, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri"), "summary": "Full actor"}, "summit_logs_via_trail": []any{map[string]any{"id": "visit", "text": "Full visit", "iri": "https://remote.example/api/v1/summit-log/visit"}}, "trails": []any{map[string]any{"id": "unrequested"}}}}
	if err := json.NewEncoder(file).Encode(payload); err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(f.trails)
	record.Set("iri", "https://remote.example/api/v1/trail/copy")
	plan, err := document.networkPlan(f.app, t.Context(), record, "https://remote.example", "trail")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(plan.data)
	for _, excluded := range []string{"description", "extra", "nested", "photos", "summary", "text", "unrequested", "Full"} {
		if strings.Contains(string(encoded), excluded) {
			t.Errorf("network plan retained %s: %s", excluded, encoded)
		}
	}
	if len(plan.media.requests) != 1 || plan.media.requests[0].name != "first.jpg" {
		t.Fatalf("file plan changed: %+v", plan.media.requests)
	}
	_, err = document.process(t.Context(), func(data map[string]any) (*core.Record, error) {
		if data["description"] != payload["description"] {
			t.Error("parked metadata was lost")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
