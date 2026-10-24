package federation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func newConcurrentProfileActor(t *testing.T) (*pbtests.TestApp, *core.Record) {
	t.Helper()
	app := setupActorLookupTestApp(t)
	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	collection.Fields.Add(&core.TextField{Name: "inbox"})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(collection)
	iri := "https://remote.example/users/alice"
	actor.Set("iri", iri)
	actor.Set("domain", "remote.example")
	actor.Set("public_key", "cached-key")
	actor.Set("inbox", iri+"/inbox")
	actor.Set("last_fetched", time.Now().Add(-time.Hour))
	actor.Set("followers", iri+"/followers")
	actor.Set("following", iri+"/following")
	actor.Set("follower_count", 23)
	actor.Set("following_count", 41)
	actor.Set("follower_count_pending", true)
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	return app, actor
}

func concurrentProfileCollectionResponse(request *http.Request, count int) *http.Response {
	body := fmt.Sprintf(`{"id":%q,"type":"OrderedCollectionPage","totalItems":%d,"orderedItems":[]}`, request.URL.String(), count)
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}
}

func awaitConcurrentProfileEvent(t *testing.T, ctx context.Context, event <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-event:
	case <-ctx.Done():
		t.Fatalf("waiting for %s: %v", description, ctx.Err())
	}
}

func TestActorProfileCachePreservesConcurrentActorChanges(t *testing.T) {
	for _, scenario := range []struct{ outcome, writer string }{
		{"success", "same-window"},
		{"success", "new-window"},
		{"success", "new-collection"},
		{"success", "parallel-count"},
		{"forbidden", "new-window"},
		{"timeout", "new-window"},
	} {
		t.Run(scenario.outcome+"/"+scenario.writer, func(t *testing.T) {
			app, actor := newConcurrentProfileActor(t)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			started, resume := make(chan struct{}), make(chan struct{})
			var requests atomic.Int32
			var workers sync.WaitGroup
			originalClient := newHTTPClient
			newHTTPClient = func() *http.Client {
				var timeout time.Duration
				if scenario.outcome == "timeout" {
					timeout = time.Second
				}
				return &http.Client{Timeout: timeout, Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
					if requests.Add(1) == 1 {
						close(started)
					}
					select {
					case <-resume:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					switch scenario.outcome {
					case "forbidden":
						return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Request: request, Body: http.NoBody}, nil
					case "timeout":
						<-request.Context().Done()
						return nil, request.Context().Err()
					default:
						return concurrentProfileCollectionResponse(request, 73), nil
					}
				})}
			}
			defer func() {
				cancel()
				workers.Wait()
				newHTTPClient = originalClient
			}()
			type result struct {
				actor *core.Record
				err   error
			}
			completed := make(chan result, 1)
			workers.Add(1)
			go func() {
				defer workers.Done()
				profile, err := assembleActor(app, ctx, actor, true)
				completed <- result{profile, err}
			}()
			awaitConcurrentProfileEvent(t, ctx, started, "collection fetch")

			updated, err := app.FindRecordById("activitypub_actors", actor.Id)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario.writer {
			case "parallel-count":
				updated.Set("follower_count", 67)
			case "new-collection":
				updated.Set("followers", "https://remote.example/new-followers")
			default:
				updated.Set("public_key", "refreshed-key")
				updated.Set("inbox", "https://remote.example/new-inbox")
				updated.Set("following_count", 83)
				if scenario.writer == "new-window" {
					updated.Set("last_fetched", time.Now())
				}
			}
			if err := app.Save(updated); err != nil {
				t.Fatal(err)
			}
			close(resume)
			var got result
			select {
			case got = <-completed:
			case <-ctx.Done():
				t.Fatalf("profile did not finish: %v", ctx.Err())
			}
			if got.err != nil {
				t.Fatal(got.err)
			}
			workers.Wait()
			if requests.Load() != 1 {
				t.Errorf("collection requests: got %d want 1", requests.Load())
			}
			stored, err := app.FindRecordById("activitypub_actors", actor.Id)
			if err != nil {
				t.Fatal(err)
			}
			for name, record := range map[string]*core.Record{"stored": stored, "returned": got.actor} {
				if record == nil {
					t.Fatalf("%s actor is nil", name)
				}
				for _, field := range []string{"public_key", "inbox", "last_fetched", "followers"} {
					if record.GetString(field) != updated.GetString(field) {
						t.Errorf("%s actor lost concurrent %s: got %q want %q", name, field, record.GetString(field), updated.GetString(field))
					}
				}
				if record.GetInt("following_count") != updated.GetInt("following_count") {
					t.Errorf("%s actor lost concurrent following count: %d", name, record.GetInt("following_count"))
				}
				wantFollowerCount := updated.GetInt("follower_count")
				if scenario.writer == "same-window" {
					wantFollowerCount = 73
				}
				if record.GetInt("follower_count") != wantFollowerCount {
					t.Errorf("%s actor has unexpected follower count: got %d want %d", name, record.GetInt("follower_count"), wantFollowerCount)
				}
				if record.GetBool("follower_count_pending") {
					t.Errorf("%s actor restored claimed pending marker", name)
				}
			}
		})
	}
}

func TestActorProfileCacheCanceledContextDoesNotClaimCollections(t *testing.T) {
	app, actor := newConcurrentProfileActor(t)
	actor.Set("following_count_pending", true)
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var requests atomic.Int32
	originalClient := newHTTPClient
	newHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			requests.Add(1)
			return concurrentProfileCollectionResponse(request, 73), nil
		})}
	}
	defer func() { newHTTPClient = originalClient }()

	if _, err := assembleActor(app, ctx, actor, true); !errors.Is(err, context.Canceled) {
		t.Errorf("profile error: got %v want context.Canceled", err)
	}
	stored, err := app.FindRecordById("activitypub_actors", actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 {
		t.Errorf("canceled profile made %d collection requests", requests.Load())
	}
	if !stored.GetBool("follower_count_pending") || !stored.GetBool("following_count_pending") {
		t.Error("canceled profile consumed a pending marker")
	}
	if stored.GetInt("follower_count") != 23 || stored.GetInt("following_count") != 41 {
		t.Errorf("canceled profile changed counts: followers=%d following=%d", stored.GetInt("follower_count"), stored.GetInt("following_count"))
	}
}

type concurrentProfileCancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (body concurrentProfileCancelOnClose) Close() error {
	err := body.ReadCloser.Close()
	body.cancel()
	return err
}

func TestActorProfileCachePersistsCompletedCountAfterContextCancellation(t *testing.T) {
	app, actor := newConcurrentProfileActor(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var requests atomic.Int32
	originalClient := newHTTPClient
	newHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			requests.Add(1)
			response := concurrentProfileCollectionResponse(request, 73)
			response.Body = concurrentProfileCancelOnClose{ReadCloser: response.Body, cancel: cancel}
			return response, nil
		})}
	}
	defer func() { newHTTPClient = originalClient }()

	profile, err := assembleActor(app, ctx, actor, true)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("request context was not canceled after collection decode: %v", ctx.Err())
	}
	stored, err := app.FindRecordById("activitypub_actors", actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Errorf("collection requests: got %d want 1", requests.Load())
	}
	for name, record := range map[string]*core.Record{"stored": stored, "returned": profile} {
		if record == nil {
			t.Fatalf("%s actor is nil", name)
		}
		if record.GetInt("follower_count") != 73 || record.GetInt("following_count") != 41 {
			t.Errorf("%s actor lost completed count: followers=%d following=%d", name, record.GetInt("follower_count"), record.GetInt("following_count"))
		}
		if record.GetBool("follower_count_pending") {
			t.Errorf("%s actor restored claimed pending marker", name)
		}
	}
}

func TestActorProfileCacheClaimsEachCollectionOnceAcrossConcurrentProfiles(t *testing.T) {
	app, actor := newConcurrentProfileActor(t)
	actor.Set("following_count_pending", true)
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	secondSnapshot, err := app.FindRecordById("activitypub_actors", actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	followersStarted, followingStarted := make(chan struct{}), make(chan struct{})
	resumeFollowers, resumeFollowing := make(chan struct{}), make(chan struct{})
	var followersRequests, followingRequests atomic.Int32
	var workers sync.WaitGroup
	originalClient := newHTTPClient
	newHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			var count int
			var resume <-chan struct{}
			switch path.Base(request.URL.Path) {
			case "followers":
				count = 71
				if followersRequests.Add(1) == 1 {
					close(followersStarted)
					resume = resumeFollowers
				}
			case "following":
				count = 83
				if followingRequests.Add(1) == 1 {
					close(followingStarted)
					resume = resumeFollowing
				}
			default:
				return nil, fmt.Errorf("unexpected collection: %s", request.URL)
			}
			if resume != nil {
				select {
				case <-resume:
				case <-request.Context().Done():
					return nil, request.Context().Err()
				}
			}
			return concurrentProfileCollectionResponse(request, count), nil
		})}
	}
	defer func() {
		cancel()
		workers.Wait()
		newHTTPClient = originalClient
	}()
	completed := make(chan error, 2)
	startProfile := func(snapshot *core.Record) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := assembleActor(app, ctx, snapshot, true)
			completed <- err
		}()
	}
	startProfile(actor)
	awaitConcurrentProfileEvent(t, ctx, followersStarted, "first profile's follower fetch")
	startProfile(secondSnapshot)
	awaitConcurrentProfileEvent(t, ctx, followingStarted, "second profile's following fetch")
	close(resumeFollowers)
	select {
	case err := <-completed:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatalf("first profile did not finish: %v", ctx.Err())
	}
	close(resumeFollowing)
	select {
	case err := <-completed:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatalf("second profile did not finish: %v", ctx.Err())
	}
	stored, err := app.FindRecordById("activitypub_actors", actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	if followersRequests.Load() != 1 || followingRequests.Load() != 1 {
		t.Errorf("concurrent profiles repeated collection fetches: followers=%d following=%d", followersRequests.Load(), followingRequests.Load())
	}
	if stored.GetInt("follower_count") != 71 || stored.GetInt("following_count") != 83 {
		t.Errorf("concurrent count result was lost: followers=%d following=%d", stored.GetInt("follower_count"), stored.GetInt("following_count"))
	}
	if stored.GetBool("follower_count_pending") || stored.GetBool("following_count_pending") {
		t.Error("concurrent profile restored a claimed pending marker")
	}
}
