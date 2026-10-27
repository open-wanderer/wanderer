package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteSyncFetchDomainsQueueBeforeGlobalCapacity(t *testing.T) {
	ctx := context.WithValue(t.Context(), remoteSyncStagingKey{}, true)
	releaseWorkflow, err := acquireRemoteSyncHost(ctx, "example.org")
	if err != nil {
		t.Fatal(err)
	}
	defer releaseWorkflow()
	releases := []func(){}
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	for i := range remoteSyncFetchDomainConcurrency {
		release, err := acquireRemoteSyncFetch(ctx, fmt.Sprintf("https://part%d.example.org:%d/file", i, 8000+i))
		if err != nil {
			t.Fatal(err)
		}
		releases = append(releases, release)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	if release, err := acquireRemoteSyncFetch(waitCtx, "https://another.example.org/file"); !errors.Is(err, context.DeadlineExceeded) {
		if release != nil {
			release()
		}
		t.Fatalf("target domain queue ignored: %v", err)
	}
	if got := len(remoteSyncFetchSlots); got != remoteSyncFetchDomainConcurrency {
		t.Fatalf("target waiters reserved global capacity: got %d", got)
	}
	remoteSyncFetchHosts.Lock()
	host := remoteSyncFetchHosts.hosts["example.org"]
	users := 0
	if host != nil {
		users = host.users
	}
	remoteSyncFetchHosts.Unlock()
	if users != remoteSyncFetchDomainConcurrency {
		t.Fatalf("cancelled target waiter remained registered: %d", users)
	}
	release, err := acquireRemoteSyncFetch(ctx, "https://independent.example.net/file")
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestRemoteSyncSharedContributionTargetLeavesFirstViewsCapacity(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	origins := map[string]*core.Record{}
	records := []*core.Record{}
	for i := range 4 {
		host := fmt.Sprintf("parent%d.example", i)
		actor := f.save(f.actors, map[string]any{"iri": "https://" + host + "/api/v1/actor/riley"})
		origins[host] = actor
		records = append(records, f.save(f.trails, map[string]any{"iri": "https://" + host + "/api/v1/trail/copy", "author": actor.Id}))
	}
	entered := make(chan struct{}, remoteSyncFetchConcurrency)
	previous := newRemoteSyncHTTPClient
	defer func() { newRemoteSyncHTTPClient = previous }()
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: stagedSyncTransport(func(request *http.Request) (*http.Response, error) {
			if remoteSyncDomain(request.URL.Hostname()) == "example.net" {
				entered <- struct{}{}
				<-request.Context().Done()
				return nil, request.Context().Err()
			}
			actor := origins[request.URL.Hostname()]
			if actor == nil {
				actor = f.remoteActor
			}
			payload := map[string]any{"id": "copy", "iri": request.URL.Scheme + "://" + request.URL.Host + request.URL.Path, "name": "Remote copy", "public": true, "expand": map[string]any{"author": map[string]any{"iri": actor.GetString("iri")}}}
			if origins[request.URL.Hostname()] != nil {
				logs := make([]any, 30)
				for i := range logs {
					id := fmt.Sprintf("entry%010d", i)
					logs[i] = map[string]any{"id": id, "iri": fmt.Sprintf("https://contribution%d.example.net/api/v1/summit-log/%s", i, id)}
				}
				payload["expand"].(map[string]any)["summit_logs_via_trail"] = logs
			}
			return (richTextSyncTransport{payload}).RoundTrip(request)
		})}
	}
	ctx, cancel := context.WithCancel(t.Context())
	var wait sync.WaitGroup
	defer func() { cancel(); wait.Wait() }()
	for _, record := range records {
		wait.Add(1)
		go func() { defer wait.Done(); _, _ = performFullSync(f.app, ctx, nil, record) }()
	}
	// Keep the established four parallel contribution fetches available.
	for i := range canonicalRemoteFetchParallelism {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatalf("only %d parallel contribution requests admitted", i)
		}
	}
	deadline := time.After(time.Second)
	for {
		remoteSyncFetchHosts.Lock()
		host := remoteSyncFetchHosts.hosts["example.net"]
		users := 0
		if host != nil {
			users = host.users
		}
		remoteSyncFetchHosts.Unlock()
		if users == 4*canonicalRemoteFetchParallelism {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("shared-target requests did not reach the queue: %d", users)
		case <-time.After(time.Millisecond):
		}
	}
	if got := len(remoteSyncFetchSlots); got != canonicalRemoteFetchParallelism {
		t.Fatalf("shared target reserved %d global fetch slots, want %d", got, canonicalRemoteFetchParallelism)
	}
	fast := core.NewRecord(f.trails)
	fast.Set("iri", "https://remote.example/api/v1/trail/copy")
	fast.Set("author", f.remoteActor.Id)
	fastCtx, stop := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer stop()
	started := time.Now()
	result, err := performFullSync(f.app, fastCtx, nil, fast)
	t.Logf("independent first view completed in %s with %d shared-target fetches active", time.Since(started), len(remoteSyncFetchSlots))
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Id == "" || !result.GetBool("full_sync_completed") {
		t.Fatal("independent first view did not complete")
	}
}
