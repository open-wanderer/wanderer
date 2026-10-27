package routes

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestInboxOtherSoftwareWaitersRequireStoredSignature(t *testing.T) {
	w := newKnownKeyWorld(t, 4*time.Second)
	actor := w.sender(t, "sender", "remote.example", true)
	iri, path := "https://remote.example/users/sender", "/users/sender"
	actor.Set("iri", iri)
	actor.Set("inbox", iri+"/inbox")
	if err := w.app.SaveNoValidate(actor); err != nil {
		t.Fatal(err)
	}
	previousClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client { return w.client }
	t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	w.origin.mu.Lock()
	w.origin.unavailable[path] = true
	w.origin.held[path], w.origin.started[path] = release, started
	w.origin.mu.Unlock()
	handler := knownKeyInboxRouter(t, w)
	responses := make(chan int, 3)
	go func() { responses <- knownKeyRouterStatus(handler, w.inboxRouterRequest(t, iri, nil, nil)) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("actor refresh did not start")
	}
	unsigned := w.inboxRouterRequest(t, iri, nil, nil)
	differentKey := w.inboxRouterRequest(t, iri, w.otherKey, nil)
	go func() { responses <- knownKeyRouterStatus(handler, unsigned) }()
	go func() { responses <- knownKeyRouterStatus(handler, differentKey) }()
	waitForInboxActorReloads(t, cap(responses))
	// The active refresh has already observed a temporary failure. A second
	// request would observe a gone response, so any confirmation is visible.
	w.origin.mu.Lock()
	w.origin.unavailable[path] = false
	w.origin.statuses = map[string]int{path: http.StatusGone}
	w.origin.mu.Unlock()
	once.Do(func() { close(release) })
	for i := 0; i < cap(responses); i++ {
		select {
		case status := <-responses:
			if status != http.StatusUnauthorized {
				t.Errorf("waiting request: %d, want %d", status, http.StatusUnauthorized)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("waiting request did not finish")
		}
	}
	if got := w.origin.count(path); got != 1 {
		t.Errorf("origin requests: %d, want 1", got)
	}
	if _, err := w.app.FindRecordById("activitypub_actors", actor.Id); err != nil {
		t.Errorf("waiting request removed actor: %v", err)
	}
}
