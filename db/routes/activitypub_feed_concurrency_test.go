package routes

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// Create and Update of one item may leave a shared actor refresh together.
// Both deliveries succeed, while their recipient has only one feed entry.
func TestInboxConcurrentCreateAndUpdateShareFeedEntry(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, refresh := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s-refresh-%t", collection, refresh), func(t *testing.T) {
				w := newKnownKeyWorld(t, 4*time.Second)
				handler := knownKeyInboxRouter(t, w)
				actor := w.sender(t, "sender", "remote.example", refresh)
				if !refresh {
					actor.Set("public_key", w.publicKey)
					actor.Set("last_fetched", time.Now())
					if err := w.app.SaveNoValidate(actor); err != nil {
						t.Fatal(err)
					}
				}
				const path = "/api/v1/activitypub/user/sender"
				refreshStarted, refreshRelease := make(chan struct{}), make(chan struct{})
				var releaseRefresh sync.Once
				defer releaseRefresh.Do(func() { close(refreshRelease) })
				if refresh {
					w.origin.held[path], w.origin.started[path] = refreshRelease, refreshStarted
				}
				feedStarted, feedRelease := make(chan struct{}), make(chan struct{})
				var startFeed, releaseFeed sync.Once
				defer releaseFeed.Do(func() { close(feedRelease) })
				w.app.OnRecordCreate("feed").BindFunc(func(event *core.RecordEvent) error {
					startFeed.Do(func() { close(feedStarted) })
					<-feedRelease
					return event.Next()
				})
				type result struct {
					status int
					err    error
				}
				results := make(chan result, 2)
				deliver := func(kind pub.ActivityVocabularyType) {
					status, err := w.deliverFirstObject(handler, actor, w.recipient, collection, kind)
					results <- result{status, err}
				}
				go deliver(pub.CreateType)
				if refresh {
					select {
					case <-refreshStarted:
					case <-time.After(2 * time.Second):
						t.Fatal("actor refresh did not start")
					}
				} else {
					select {
					case <-feedStarted:
					case <-time.After(3 * time.Second):
						t.Fatal("first feed insert did not start")
					}
				}
				go deliver(pub.UpdateType)
				if refresh {
					waitForKnownKeyReloadCallers(t, 2)
					releaseRefresh.Do(func() { close(refreshRelease) })
				}
				waitForFeedInsertCallers(t, 2)
				releaseFeed.Do(func() { close(feedRelease) })
				for i := 0; i < 2; i++ {
					select {
					case got := <-results:
						if got.err != nil || got.status != http.StatusOK {
							t.Errorf("delivery response: %d, %v", got.status, got.err)
						}
					case <-time.After(5 * time.Second):
						t.Fatal("delivery did not finish")
					}
				}
				objects, err := w.app.FindAllRecords(collection)
				if err != nil || len(objects) != 1 {
					t.Fatalf("stored objects: %d, %v", len(objects), err)
				}
				feed, err := w.app.FindAllRecords("feed")
				if err != nil || len(feed) != 1 {
					t.Fatalf("feed entries: %d, want 1; error: %v", len(feed), err)
				}
				if feed[0].GetString("actor") != w.recipient.Id || feed[0].GetString("item") != objects[0].Id {
					t.Fatal("feed entry does not refer to the recipient and object")
				}
				wantRequests := 0
				if refresh {
					wantRequests = 1
				}
				if got := w.origin.count(path); got != wantRequests {
					t.Fatalf("actor requests: %d, want %d", got, wantRequests)
				}
			})
		}
	}
}

func waitForFeedInsertCallers(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		buffer := make([]byte, 1<<20)
		for {
			n := runtime.Stack(buffer, true)
			if n < len(buffer) {
				buffer = buffer[:n]
				break
			}
			buffer = make([]byte, 2*len(buffer))
		}
		callers := 0
		for _, stack := range strings.Split(string(buffer), "\n\n") {
			if strings.Contains(stack, "util.InsertIntoFeed(") {
				callers++
			}
		}
		if callers >= count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("feed insert callers: %d, want %d", callers, count)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
