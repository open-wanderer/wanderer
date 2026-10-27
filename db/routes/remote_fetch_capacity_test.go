package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteFetchCapacityWaitPreservesErrorCauses(t *testing.T) {
	for _, queue := range []string{"domain", "global"} {
		t.Run(queue, func(t *testing.T) {
			ctx := context.WithValue(t.Context(), remoteSyncStagingKey{}, true)
			releases := []func(){}
			defer func() {
				for _, release := range releases {
					release()
				}
			}()
			count := remoteSyncFetchDomainConcurrency
			if queue == "global" {
				count = remoteSyncFetchConcurrency
			}
			for i := range count {
				target := "https://reply.example/file"
				if queue == "global" {
					target = fmt.Sprintf("https://target%d.example/file", i)
				}
				release, err := acquireRemoteSyncFetch(ctx, target)
				if err != nil {
					t.Fatal(err)
				}
				releases = append(releases, release)
			}
			waitCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
			defer cancel()
			target, _ := url.Parse("https://reply.example/file")
			_, err := fetchRemoteDocument(waitCtx, &http.Client{}, target, "summit log")
			if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, errRemoteUnavailable) || !strings.Contains(err.Error(), "remote fetch capacity unavailable") {
				t.Fatalf("capacity refusal lost a cause: %v", err)
			}
		})
	}
}

func holdRemoteContributorCapacity(t *testing.T, f *remoteSyncFixture) func() {
	t.Helper()
	entered := make(chan struct{}, remoteSyncFetchDomainConcurrency)
	old := remoteSyncGetActorByIRI
	remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, iri string, follows bool) (*core.Record, error) {
		if strings.Contains(iri, "/actor/wait") {
			entered <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return app.FindFirstRecordByData("activitypub_actors", "iri", iri)
	}
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), remoteSyncStagingKey{}, true))
	references := make([]string, remoteSyncFetchDomainConcurrency)
	for i := range references {
		references[i] = fmt.Sprintf("https://reply.example/actor/wait%d", i)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		resolveRemoteAuthorsWithin(f.app, ctx, references, time.Minute, time.Minute)
	}()
	var once sync.Once
	stop := func() { once.Do(func() { cancel(); <-done }) }
	t.Cleanup(func() { stop(); remoteSyncGetActorByIRI = old })
	for range references {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			stop()
			t.Fatal("contributor lookups did not occupy the domain capacity")
		}
	}
	return stop
}

func TestRemoteSummitCapacityWaitDoesNotDelayNextRefresh(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	author := f.save(f.actors, map[string]any{"iri": "https://reply.example/users/contributor", "last_fetched": time.Now()})
	iri := "https://reply.example/api/v1/summit-log/reply"
	oldTimeout := canonicalRemoteFetchTimeout
	canonicalRemoteFetchTimeout = 30 * time.Millisecond
	t.Cleanup(func() { canonicalRemoteFetchTimeout = oldTimeout })
	var requests atomic.Int32
	oldClient := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = oldClient })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "reply.example" {
				requests.Add(1)
				return richTextSyncTransport{map[string]any{"id": "reply", "iri": iri, "text": "Confirmed reply", "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}}.RoundTrip(req)
			}
			return richTextSyncTransport{map[string]any{"id": "trail", "name": "Synced", "public": true, "expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"id": "reply", "iri": iri}}}}}.RoundTrip(req)
		}}}
	}
	release := holdRemoteContributorCapacity(t, f.remoteSyncFixture)
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	key := canonicalLogCacheKey(f.app, f.trail, iri)
	canonicalRemoteLogCache.Lock()
	_, remembered := canonicalRemoteLogCache.entries[key]
	canonicalRemoteLogCache.Unlock()
	if remembered {
		t.Error("local fetch wait was remembered as an origin failure")
	}
	if requests.Load() != 0 {
		t.Fatalf("queued confirmation sent %d requests", requests.Load())
	}
	canonicalRemoteFetchTimeout = oldTimeout
	release()
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	record, err := f.app.FindFirstRecordByData("summit_logs", "iri", iri)
	if err != nil || record.GetString("text") != "Confirmed reply" || requests.Load() != 1 {
		t.Fatalf("next refresh did not import the entry: record=%v error=%v requests=%d", record, err, requests.Load())
	}
}

func TestRemoteAuthorCapacityWaitIsReported(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	author := f.save(f.actors, map[string]any{"iri": "https://reply.example/users/contributor"})
	release := holdRemoteContributorCapacity(t, f)
	ctx, refusals := withRemoteAuthorRefusals(context.WithValue(t.Context(), remoteSyncStagingKey{}, true))
	iri := author.GetString("iri")
	authors := resolveRemoteAuthorsWithin(f.app, ctx, []string{iri}, 20*time.Millisecond, time.Second)
	if len(authors) != 0 || !refusals.refused(iri) {
		t.Errorf("capacity wait was not reported: authors=%v refused=%v", authors, refusals.refused(iri))
	}
	release()
	authors = resolveRemoteAuthorsWithin(f.app, ctx, []string{iri}, time.Second, time.Second)
	if authors[iri] == nil || authors[iri].Id != author.Id {
		t.Fatal("next author lookup did not use the released capacity")
	}
}
