package routes

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestInboxKnownKeySelfDeletionKeepsConfirmationBudget(t *testing.T) {
	for _, state := range []string{"unavailable", "missing"} {
		t.Run(state, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			handler := knownKeyInboxRouter(t, w)
			previousClient := newRemoteSyncHTTPClient
			newRemoteSyncHTTPClient = func() *http.Client { return w.client }
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			for index := 0; index < 20; index++ {
				name := fmt.Sprintf("sender%02d", index)
				actor := w.sender(t, name, "remote.example", true)
				iri, path := actor.GetString("iri"), "/api/v1/activitypub/user/"+name
				w.origin.mu.Lock()
				w.origin.unavailable[path] = state == "unavailable"
				delete(w.origin.documents, path)
				w.origin.mu.Unlock()
				for _, key := range []*rsa.PrivateKey{nil, w.otherKey} {
					for delivery := 0; delivery < 31; delivery++ {
						if status := knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, key, nil)); status != http.StatusUnauthorized {
							t.Fatalf("actor %d delivery %d: %d", index, delivery, status)
						}
					}
				}
				if got := w.origin.count(path); got != 1 {
					t.Fatalf("actor %d requests before authenticated deletion: %d", index, got)
				}
				stored, err := w.app.FindRecordById("activitypub_actors", actor.Id)
				if err != nil || stored.GetString("legacy_public_key") != w.previousPublicKey {
					t.Fatalf("actor %d changed after refused deliveries: %v", index, err)
				}
				w.origin.mu.Lock()
				w.origin.unavailable[path] = false
				w.origin.mu.Unlock()
				if status := knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, w.previousKey, nil)); status != http.StatusOK {
					t.Fatalf("actor %d authenticated deletion: %d", index, status)
				}
				if got := w.origin.count(path); got != 2 {
					t.Fatalf("actor %d origin requests in one window: %d", index, got)
				}
				if _, err := w.app.FindRecordById("activitypub_actors", actor.Id); err == nil {
					t.Fatalf("actor %d remains after confirmed deletion", index)
				}
			}
		})
	}
}

func TestInboxKnownKeySharedFailureDoesNotRepeatConfirmation(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	iri, path := actor.GetString("iri"), "/api/v1/activitypub/user/sender"
	release, started := make(chan struct{}), make(chan struct{})
	w.origin.unavailable[path] = true
	w.origin.held[path], w.origin.started[path] = release, started
	var once sync.Once
	defer once.Do(func() { close(release) })
	previousClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client { return w.client }
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
	handler := knownKeyInboxRouter(t, w)
	requests := make([]*http.Request, 31)
	for i := range requests {
		requests[i] = w.inboxRouterRequest(t, iri, nil, nil)
	}
	responses := make(chan int, len(requests))
	for _, request := range requests {
		go func(request *http.Request) { responses <- knownKeyRouterStatus(handler, request) }(request)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("actor refresh did not start")
	}
	select {
	case status := <-responses:
		t.Fatalf("delivery ended before the refresh: %d", status)
	case <-time.After(100 * time.Millisecond):
	}
	once.Do(func() { close(release) })
	for range requests {
		select {
		case status := <-responses:
			if status != http.StatusUnauthorized {
				t.Errorf("shared failure status: %d", status)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("waiting delivery did not finish")
		}
	}
	if got := w.origin.count(path); got != 1 {
		t.Fatalf("shared refresh requests: %d", got)
	}
	if _, err := w.app.FindRecordById("activitypub_actors", actor.Id); err != nil {
		t.Fatalf("shared failure removed actor: %v", err)
	}
}

func TestInboxKnownKeySelfDeletionBoundsConfirmationAttempts(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	iri, path := actor.GetString("iri"), "/api/v1/activitypub/user/sender"
	w.origin.unavailable[path] = true
	previousClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client { return w.client }
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
	handler := knownKeyInboxRouter(t, w)
	for delivery := 0; delivery < 31; delivery++ {
		if status := knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, w.previousKey, nil)); status != http.StatusUnauthorized {
			t.Fatalf("unavailable origin delivery %d: %d", delivery, status)
		}
	}
	if got := w.origin.count(path); got != 2 {
		t.Fatalf("origin requests in one window: %d", got)
	}
	if _, err := w.app.FindRecordById("activitypub_actors", actor.Id); err != nil {
		t.Fatalf("failed confirmation removed actor: %v", err)
	}
}
