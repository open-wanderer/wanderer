package routes

import (
	"crypto/rsa"
	"net/http"
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestInboxKnownKeySelfDeletionAfterFailedRefresh(t *testing.T) {
	for _, scenario := range []string{"retained-key", "different-key", "unsigned", "origin-available", "origin-unavailable"} {
		t.Run(scenario, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			actor := w.sender(t, "sender", "remote.example", true)
			iri, path := actor.GetString("iri"), "/api/v1/activitypub/user/sender"
			w.origin.unavailable[path] = true
			if status := w.deliver(t, iri, w.previousKey); status != http.StatusUnauthorized {
				t.Fatalf("initial unavailable refresh: %d", status)
			}
			if got := w.origin.count(path); got != 1 {
				t.Fatalf("initial refresh requests: %d", got)
			}
			stored, err := w.app.FindRecordById("activitypub_actors", actor.Id)
			if err != nil || stored.GetString("public_key") != "" || stored.GetString("legacy_public_key") != w.previousPublicKey {
				t.Fatalf("retained actor after failed refresh: %v", err)
			}
			w.origin.mu.Lock()
			w.origin.unavailable[path] = scenario == "origin-unavailable"
			if scenario != "origin-available" {
				delete(w.origin.documents, path)
			}
			w.origin.mu.Unlock()
			previousClient := newRemoteSyncHTTPClient
			newRemoteSyncHTTPClient = func() *http.Client { return w.client }
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			var key *rsa.PrivateKey = w.previousKey
			if scenario == "different-key" {
				key = w.otherKey
			} else if scenario == "unsigned" {
				key = nil
			}
			want := http.StatusUnauthorized
			if scenario == "retained-key" {
				want = http.StatusOK
			}
			if status := w.deliverActivity(t, iri, key, w.recipient, pub.DeleteType, iri, nil); status != want {
				t.Fatalf("self-deletion after failed refresh: %d, want %d", status, want)
			}
			// Only a request signed with the retained key may claim an
			// independent origin check after the earlier failed refresh.
			wantRequests := 2
			if scenario == "different-key" || scenario == "unsigned" {
				wantRequests = 1
			}
			if got := w.origin.count(path); got != wantRequests {
				t.Fatalf("origin requests after self-deletion: %d", got)
			}
			_, err = w.app.FindRecordById("activitypub_actors", actor.Id)
			if (err != nil) != (want == http.StatusOK) {
				t.Fatalf("actor removed=%v, status=%d", err != nil, want)
			}
		})
	}
}
