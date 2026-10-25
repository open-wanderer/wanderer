package routes

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestInboxKnownKeyWaitingDeletionSharesConfirmation(t *testing.T) {
	for _, originStatus := range []int{http.StatusNotFound, http.StatusGone, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(originStatus), func(t *testing.T) {
			w := newKnownKeyWorld(t)
			w.client.Timeout = 10 * time.Second
			handler := knownKeyInboxRouter(t, w)
			previousClient := newRemoteSyncHTTPClient
			newRemoteSyncHTTPClient = func() *http.Client { return w.client }
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			w.origin.statuses = map[string]int{}
			for index, mode := range []string{"unsigned", "different-key", "retained-key"} {
				t.Run(mode, func(t *testing.T) {
					actor := w.sender(t, fmt.Sprintf("sender%d", index), "remote.example", true)
					iri := actor.GetString("iri")
					path := fmt.Sprintf("/api/v1/activitypub/user/sender%d", index)
					release, started := make(chan struct{}), make(chan struct{})
					var once sync.Once
					defer once.Do(func() { close(release) })
					w.origin.mu.Lock()
					w.origin.unavailable[path] = true
					w.origin.held[path], w.origin.started[path] = release, started
					w.origin.mu.Unlock()
					type delivery struct {
						request *http.Request
						want    int
					}
					var deliveries []delivery
					keys := []*rsa.PrivateKey{nil}
					if mode == "different-key" {
						keys = []*rsa.PrivateKey{w.otherKey}
					} else if mode == "retained-key" {
						keys = []*rsa.PrivateKey{nil, w.otherKey}
					}
					for _, key := range keys {
						for i := 0; i < 31; i++ {
							deliveries = append(deliveries, delivery{w.inboxRouterRequest(t, iri, key, nil), http.StatusUnauthorized})
						}
					}
					if mode == "retained-key" {
						want := http.StatusOK
						if originStatus == http.StatusServiceUnavailable {
							want = http.StatusUnauthorized
						}
						deliveries = append(deliveries, delivery{w.inboxRouterRequest(t, iri, w.previousKey, nil), want})
					}
					type result struct{ status, want int }
					responses := make(chan result, len(deliveries)+1)
					go func() {
						responses <- result{knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, nil, nil)), http.StatusUnauthorized}
					}()
					select {
					case <-started:
					case <-time.After(3 * time.Second):
						t.Fatal("actor refresh did not start")
					}
					for _, current := range deliveries {
						go func(current delivery) {
							responses <- result{knownKeyRouterStatus(handler, current.request), current.want}
						}(current)
					}
					waitForInboxActorReloads(t, len(deliveries)+1)
					select {
					case got := <-responses:
						t.Fatalf("delivery ended before the refresh: %d", got.status)
					default:
					}
					w.origin.mu.Lock()
					w.origin.unavailable[path] = false
					w.origin.statuses[path] = originStatus
					delete(w.origin.documents, path)
					w.origin.mu.Unlock()
					once.Do(func() { close(release) })
					for i := 0; i < cap(responses); i++ {
						select {
						case got := <-responses:
							if got.status != got.want {
								t.Errorf("waiting deletion: %d, want %d", got.status, got.want)
							}
						case <-time.After(5 * time.Second):
							t.Fatal("waiting deletion did not finish")
						}
					}
					wantRequests := 1
					if mode == "retained-key" {
						wantRequests = 2
					}
					if got := w.origin.count(path); got != wantRequests {
						t.Errorf("origin requests: %d, want %d", got, wantRequests)
					}
					_, err := w.app.FindRecordById("activitypub_actors", actor.Id)
					wantRemoved := mode == "retained-key" && originStatus != http.StatusServiceUnavailable
					if (err != nil) != wantRemoved {
						t.Errorf("actor removed=%v, want %v", err != nil, wantRemoved)
					}
					if mode == "retained-key" && originStatus == http.StatusServiceUnavailable {
						for i := 0; i < 31; i++ {
							if status := knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, w.previousKey, nil)); status != http.StatusUnauthorized {
								t.Errorf("later unavailable deletion: %d", status)
							}
						}
						if got := w.origin.count(path); got != 2 {
							t.Errorf("repeated deletion origin requests: %d", got)
						}
					}
				})
			}
		})
	}
}

func TestInboxKnownKeyRefreshFailurePrecedesDeletionConfirmation(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	iri, path := actor.GetString("iri"), "/api/v1/activitypub/user/sender"
	delete(w.origin.documents, path)
	previousClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client { return w.client }
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	getInboxActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
		close(started)
		<-release
		panic("actor refresh stopped")
	}
	handler := knownKeyInboxRouter(t, w)
	responses := make(chan int, 2)
	go func() { responses <- knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, nil, nil)) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("actor refresh did not start")
	}
	go func() { responses <- knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, w.previousKey, nil)) }()
	waitForInboxActorReloads(t, cap(responses))
	once.Do(func() { close(release) })
	for i := 0; i < cap(responses); i++ {
		select {
		case status := <-responses:
			if status != http.StatusInternalServerError {
				t.Errorf("refresh failure response: %d", status)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("waiting delivery did not finish")
		}
	}
	if got := w.origin.count(path); got != 0 {
		t.Errorf("refresh failure reached deletion confirmation %d times", got)
	}
	if _, err := w.app.FindRecordById("activitypub_actors", actor.Id); err != nil {
		t.Errorf("refresh failure removed actor: %v", err)
	}
}
