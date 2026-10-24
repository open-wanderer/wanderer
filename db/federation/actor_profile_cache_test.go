package federation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"pocketbase/util"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type actorProfileCacheFixture struct {
	app                *pbtests.TestApp
	actorRequests      int
	collectionRequests map[string]int
	collectionFailures map[string]string
	followingCount     int
}

func newActorProfileCacheFixture(t *testing.T) *actorProfileCacheFixture {
	t.Helper()
	f := &actorProfileCacheFixture{
		app: setupActorLookupTestApp(t), collectionRequests: make(map[string]int),
		collectionFailures: make(map[string]string), followingCount: 8,
	}
	originalActor, originalCollection := newActorFetchHTTPClient, newHTTPClient
	t.Cleanup(func() { newActorFetchHTTPClient, newHTTPClient = originalActor, originalCollection })
	newActorFetchHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			f.actorRequests++
			iri := request.URL.String()
			body := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":%q,"outbox":%q,"followers":%q,"following":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"origin-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"/followers", iri+"/following", iri+"#main-key", iri)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
	}
	newHTTPClient = func() *http.Client {
		return &http.Client{Timeout: 100 * time.Millisecond, Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			collection := path.Base(request.URL.Path)
			f.collectionRequests[collection]++
			failure := f.collectionFailures[collection]
			status := http.StatusOK
			switch failure {
			case "unauthorized":
				status = http.StatusUnauthorized
			case "forbidden":
				status = http.StatusForbidden
			case "not-found":
				status = http.StatusNotFound
			case "timeout":
				<-request.Context().Done()
				return nil, request.Context().Err()
			}
			if status != http.StatusOK {
				return &http.Response{StatusCode: status, Header: make(http.Header), Request: request, Body: http.NoBody}, nil
			}
			count := 0
			if collection == "following" {
				count = f.followingCount
			}
			body := fmt.Sprintf(`{"id":%q,"type":"OrderedCollectionPage","totalItems":%d,"orderedItems":[]}`, request.URL.String(), count)
			if failure == "oversize" {
				body += strings.Repeat(" ", int(util.FederationObjectBodyLimit))
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
	}
	return f
}

func TestActorProfileCacheLoadsEachPendingCountOnce(t *testing.T) {
	for _, identityOnly := range []bool{false, true} {
		for _, failure := range []string{"none", "unauthorized", "forbidden", "not-found", "timeout", "oversize"} {
			failedCollections := []string{"followers", "following", "both"}
			if failure == "none" {
				failedCollections = []string{"none"}
			}
			for _, failed := range failedCollections {
				t.Run(fmt.Sprintf("identity-only=%v/%s/%s", identityOnly, failure, failed), func(t *testing.T) {
					f := newActorProfileCacheFixture(t)
					if failed == "followers" || failed == "both" {
						f.collectionFailures["followers"] = failure
					}
					if failed == "following" || failed == "both" {
						f.collectionFailures["following"] = failure
					}
					ctx := context.Background()
					if identityOnly {
						ctx = WithActorIdentityLookup(ctx)
					}
					iri := "https://remote.example/users/alice"
					actor, err := GetActorByIRI(f.app, ctx, iri, false)
					if err != nil || actor == nil {
						t.Fatalf("initial actor lookup: actor=%v err=%v", actor, err)
					}
					fetchedAt := actor.GetString("last_fetched")
					for window := range 2 {
						for call := range 3 {
							actor, err := GetActorByIRI(f.app, t.Context(), iri, true)
							if err != nil || actor == nil {
								t.Fatalf("profile lookup %d: actor=%v err=%v", call+1, actor, err)
							}
							stored, err := f.app.FindRecordById("activitypub_actors", actor.Id)
							if err != nil {
								t.Fatal(err)
							}
							for _, collection := range []struct{ name, count, pending string }{
								{"followers", "follower_count", "follower_count_pending"},
								{"following", "following_count", "following_count_pending"},
							} {
								wantRequests := window + 1
								wantCount := 0
								if collection.name == "following" && f.collectionFailures[collection.name] == "" {
									wantCount = f.followingCount
								}
								if f.collectionRequests[collection.name] != wantRequests || stored.GetBool(collection.pending) || stored.GetInt(collection.count) != wantCount {
									t.Fatalf("window %d profile %d %s: requests=%d want=%d pending=%v count=%d want=%d", window+1, call+1, collection.name, f.collectionRequests[collection.name], wantRequests, stored.GetBool(collection.pending), stored.GetInt(collection.count), wantCount)
								}
							}
							if f.actorRequests != window+1 {
								t.Fatalf("cached profile fetched actor %d times", f.actorRequests)
							}
							if window == 1 && call == 0 {
								fetchedAt = stored.GetString("last_fetched")
							}
							if stored.GetString("last_fetched") != fetchedAt {
								t.Fatalf("profile lookup %d extended cache window", call+1)
							}
						}
						if window == 0 {
							actor, err = f.app.FindFirstRecordByData("activitypub_actors", "iri", iri)
							if err != nil {
								t.Fatal(err)
							}
							actor.Set("last_fetched", time.Now().Add(-3*time.Hour))
							if err := f.app.Save(actor); err != nil {
								t.Fatal(err)
							}
							f.followingCount = 9
						}
					}
				})
			}
		}
	}
}

func TestActorProfileCacheRetainsPreviousCountsAfterCollectionFailures(t *testing.T) {
	for _, identityOnly := range []bool{false, true} {
		for _, failure := range []string{"unauthorized", "forbidden", "not-found", "timeout", "oversize"} {
			t.Run(fmt.Sprintf("identity-only=%v/%s", identityOnly, failure), func(t *testing.T) {
				f := newActorProfileCacheFixture(t)
				f.collectionFailures["followers"], f.collectionFailures["following"] = failure, failure
				iri := "https://remote.example/users/alice"
				collection, err := f.app.FindCollectionByNameOrId("activitypub_actors")
				if err != nil {
					t.Fatal(err)
				}
				actor := core.NewRecord(collection)
				actor.Set("iri", iri)
				actor.Set("domain", "remote.example")
				actor.Set("public_key", "cached-key")
				actor.Set("last_fetched", time.Now().Add(-3*time.Hour))
				actor.Set("followers", iri+"/followers")
				actor.Set("following", iri+"/following")
				actor.Set("follower_count", 23)
				actor.Set("following_count", 41)
				if err := f.app.Save(actor); err != nil {
					t.Fatal(err)
				}
				assertPreviousCounts := func() *core.Record {
					t.Helper()
					stored, err := f.app.FindRecordById("activitypub_actors", actor.Id)
					if err != nil {
						t.Fatal(err)
					}
					if stored.GetInt("follower_count") != 23 || stored.GetInt("following_count") != 41 {
						t.Fatalf("previous counts were overwritten: followers=%d following=%d", stored.GetInt("follower_count"), stored.GetInt("following_count"))
					}
					return stored
				}
				ctx := context.Background()
				if identityOnly {
					ctx = WithActorIdentityLookup(ctx)
				}
				if _, err := GetActorByIRI(f.app, ctx, iri, false); err != nil {
					t.Fatal(err)
				}
				stored := assertPreviousCounts()
				fetchedAt := stored.GetString("last_fetched")
				for window := range 2 {
					for call := range 3 {
						if _, err := GetActorByIRI(f.app, t.Context(), iri, true); err != nil {
							t.Fatal(err)
						}
						stored = assertPreviousCounts()
						if f.actorRequests != window+1 || f.collectionRequests["followers"] != window+1 || f.collectionRequests["following"] != window+1 || stored.GetBool("follower_count_pending") || stored.GetBool("following_count_pending") {
							t.Fatalf("window %d profile %d repeated collection attempt: requests=%d/%v pending=%v/%v", window+1, call+1, f.actorRequests, f.collectionRequests, stored.GetBool("follower_count_pending"), stored.GetBool("following_count_pending"))
						}
						if window == 1 && call == 0 {
							fetchedAt = stored.GetString("last_fetched")
						}
						if stored.GetString("last_fetched") != fetchedAt {
							t.Fatal("failed collection attempt extended actor cache window")
						}
					}
					if window == 0 {
						stored.Set("last_fetched", time.Now().Add(-3*time.Hour))
						if err := f.app.Save(stored); err != nil {
							t.Fatal(err)
						}
					}
				}
			})
		}
	}
}

func TestActorProfileCacheKeepsExistingZeroCountsAndRefreshesAfterExpiry(t *testing.T) {
	f := newActorProfileCacheFixture(t)
	iri := "https://remote.example/users/alice"
	collection, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(collection)
	actor.Set("iri", iri)
	actor.Set("domain", "remote.example")
	actor.Set("public_key", "cached-key")
	actor.Set("last_fetched", time.Now())
	actor.Set("followers", iri+"/followers")
	actor.Set("following", iri+"/following")
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		cached, err := GetActorByIRI(f.app, t.Context(), iri, true)
		if err != nil || cached == nil || cached.GetInt("follower_count") != 0 || cached.GetInt("following_count") != 0 {
			t.Fatalf("existing zero counts changed: actor=%v err=%v", cached, err)
		}
	}
	if f.actorRequests != 0 || len(f.collectionRequests) != 0 {
		t.Fatalf("existing profile cache made requests: actor=%d collections=%v", f.actorRequests, f.collectionRequests)
	}
	actor.Set("last_fetched", time.Now().Add(-3*time.Hour))
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	actor, err = GetActorByIRI(f.app, WithActorIdentityLookup(t.Context()), iri, false)
	if err != nil || actor == nil || !actor.GetBool("follower_count_pending") || !actor.GetBool("following_count_pending") || f.actorRequests != 1 || len(f.collectionRequests) != 0 {
		t.Fatalf("identity refresh did not defer counts: actor=%v err=%v requests=%d/%v", actor, err, f.actorRequests, f.collectionRequests)
	}
	actor.Set("last_fetched", time.Now().Add(-time.Hour))
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	fetchedAt := actor.GetString("last_fetched")
	for range 3 {
		actor, err = GetActorByIRI(f.app, t.Context(), iri, true)
		if err != nil || actor == nil || actor.GetInt("follower_count") != 0 || actor.GetInt("following_count") != 8 || actor.GetString("last_fetched") != fetchedAt || f.actorRequests != 1 || f.collectionRequests["followers"] != 1 || f.collectionRequests["following"] != 1 {
			t.Fatalf("deferred profile counts: actor=%v err=%v requests=%d/%v", actor, err, f.actorRequests, f.collectionRequests)
		}
	}
	actor.Set("last_fetched", time.Now().Add(-3*time.Hour))
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	f.followingCount = 9
	for range 3 {
		actor, err = GetActorByIRI(f.app, t.Context(), iri, true)
		if err != nil || actor == nil || actor.GetInt("following_count") != 9 || f.actorRequests != 2 || f.collectionRequests["followers"] != 2 || f.collectionRequests["following"] != 2 {
			t.Fatalf("expired profile counts: actor=%v err=%v requests=%d/%v", actor, err, f.actorRequests, f.collectionRequests)
		}
	}
}
