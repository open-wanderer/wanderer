package federation

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestActorProfileCacheDoesNotClaimFollowingFromSupersededSnapshot(t *testing.T) {
	for _, scenario := range []string{"new-window", "same-window"} {
		t.Run(scenario, func(t *testing.T) {
			app, actor := newConcurrentProfileActor(t)
			actor.Set("following_count_pending", true)
			if err := app.Save(actor); err != nil {
				t.Fatal(err)
			}
			followersURL := actor.GetString("followers")
			followingURL := actor.GetString("following")
			newFollowingURL := "https://remote.example/new-following"
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			started, resume := make(chan struct{}), make(chan struct{})
			var requests, followersRequests, followingRequests, newFollowingRequests atomic.Int32
			var workers sync.WaitGroup
			originalClient := newHTTPClient
			newHTTPClient = func() *http.Client {
				return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
					requests.Add(1)
					switch request.URL.String() {
					case followersURL:
						if followersRequests.Add(1) == 1 {
							close(started)
							select {
							case <-resume:
							case <-request.Context().Done():
								return nil, request.Context().Err()
							}
						}
						return concurrentProfileCollectionResponse(request, 71), nil
					case followingURL:
						followingRequests.Add(1)
					case newFollowingURL:
						newFollowingRequests.Add(1)
					default:
						return nil, fmt.Errorf("unexpected collection: %s", request.URL)
					}
					return concurrentProfileCollectionResponse(request, 83), nil
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
			awaitConcurrentProfileEvent(t, ctx, started, "follower fetch before concurrent writer")

			updated, err := app.FindRecordById("activitypub_actors", actor.Id)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "new-window" {
				updated.Set("last_fetched", time.Now())
				updated.Set("follower_count_pending", true)
			} else {
				updated.Set("following", newFollowingURL)
			}
			updated.Set("following_count_pending", true)
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
			fresh, err := app.FindRecordById("activitypub_actors", actor.Id)
			if err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 1 || followersRequests.Load() != 1 || followingRequests.Load() != 0 || newFollowingRequests.Load() != 0 {
				t.Errorf("superseded snapshot fetched another collection: total=%d followers=%d following=%d new-following=%d", requests.Load(), followersRequests.Load(), followingRequests.Load(), newFollowingRequests.Load())
			}
			for name, record := range map[string]*core.Record{"stored": fresh, "returned": got.actor} {
				if record == nil {
					t.Fatalf("%s actor is nil", name)
				}
				if !record.GetBool("following_count_pending") {
					t.Errorf("%s actor consumed the writer's following marker", name)
				}
				if record.GetInt("following_count") != 41 {
					t.Errorf("%s actor changed the unclaimed following count: %d", name, record.GetInt("following_count"))
				}
				if record.GetString("following") != updated.GetString("following") {
					t.Errorf("%s actor lost the writer's following URL: %q", name, record.GetString("following"))
				}
			}

			profile, err := assembleActor(app, ctx, fresh, true)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := app.FindRecordById("activitypub_actors", actor.Id)
			if err != nil {
				t.Fatal(err)
			}
			for name, record := range map[string]*core.Record{"stored": stored, "returned": profile} {
				if record == nil {
					t.Fatalf("%s actor is nil", name)
				}
				if record.GetBool("following_count_pending") || record.GetInt("following_count") != 83 {
					t.Errorf("%s actor did not complete the fresh following count: pending=%t count=%d", name, record.GetBool("following_count_pending"), record.GetInt("following_count"))
				}
			}
			wantRequests, wantFollowers, wantFollowing, wantNewFollowing := int32(3), int32(2), int32(1), int32(0)
			if scenario == "same-window" {
				wantRequests, wantFollowers, wantFollowing, wantNewFollowing = 2, 1, 0, 1
			}
			if requests.Load() != wantRequests || followersRequests.Load() != wantFollowers || followingRequests.Load() != wantFollowing || newFollowingRequests.Load() != wantNewFollowing {
				t.Errorf("fresh snapshot fetched unexpected collections: total=%d followers=%d following=%d new-following=%d", requests.Load(), followersRequests.Load(), followingRequests.Load(), newFollowingRequests.Load())
			}
		})
	}
}
