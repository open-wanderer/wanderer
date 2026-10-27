package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func setChildAuthorLimits(t *testing.T, short, perActor, budget time.Duration) {
	t.Helper()
	oldShort, oldActor, oldBudget := canonicalRemoteFetchTimeout, sameOriginAuthorTimeout, sameOriginAuthorBudget
	t.Cleanup(func() {
		canonicalRemoteFetchTimeout, sameOriginAuthorTimeout, sameOriginAuthorBudget = oldShort, oldActor, oldBudget
	})
	canonicalRemoteFetchTimeout, sameOriginAuthorTimeout, sameOriginAuthorBudget = short, perActor, budget
}

func childAuthorContribution(id, author string) map[string]any {
	return map[string]any{"id": id, "name": "Edited", "text": "Edited", "public": true, "expand": map[string]any{"author": map[string]any{"iri": author}}}
}

func TestRemoteChildAuthorsAllowOriginResponseTime(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, known := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/known=%t", kind, known), func(t *testing.T) {
				t.Setenv("ORIGIN", "https://local.example")
				setChildAuthorLimits(t, 5*time.Millisecond, 200*time.Millisecond, 250*time.Millisecond)
				f := newRemoteSyncFixture(t)
				parentCollection, childCollection, relation, childKind := f.trails, f.logs, "summit_logs_via_trail", "summit-log"
				if kind == "list" {
					parentCollection, childCollection, relation, childKind = f.lists, f.trails, "trails", "trail"
				}
				parent := f.save(parentCollection, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/parent", "author": f.remoteActor.Id, "name": "Original", "needs_full_sync": true})
				authorIRI := "https://remote.example/users/contributor"
				children := []any{childAuthorContribution("new", authorIRI)}
				var existing *core.Record
				if known {
					actor := f.save(f.actors, map[string]any{"iri": authorIRI, "public_key": "", "last_fetched": time.Time{}})
					existing = f.save(childCollection, map[string]any{"iri": "https://remote.example/api/v1/" + childKind + "/existing", "author": actor.Id, "trail": parent.Id, "name": "Original", "text": "Original"})
					children = append(children, childAuthorContribution("existing", authorIRI))
					if kind == "list" {
						parent.Set("trails", []string{existing.Id})
						if err := f.app.Save(parent); err != nil {
							t.Fatal(err)
						}
					}
				}
				oldClient, oldActor := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI
				t.Cleanup(func() { newRemoteSyncHTTPClient, remoteSyncGetActorByIRI = oldClient, oldActor })
				payload := map[string]any{"id": "parent", "name": "Synced", "public": true, "expand": map[string]any{relation: children}}
				newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
				var calls atomic.Int32
				remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, iri string, _ bool) (*core.Record, error) {
					calls.Add(1)
					timer := time.NewTimer(30 * time.Millisecond)
					defer timer.Stop()
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-timer.C:
					}
					actor, _ := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
					if actor == nil {
						actor = core.NewRecord(f.actors)
					}
					actor.Load(map[string]any{"iri": iri, "domain": "remote.example", "public_key": "confirmed-key", "last_fetched": time.Now()})
					return actor, app.Save(actor)
				}
				var err error
				if kind == "trail" {
					_, err = performFullSync(f.app, t.Context(), nil, parent)
				} else {
					_, err = performFullListSync(f.app, t.Context(), nil, parent)
				}
				if err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 1 {
					t.Fatalf("author lookup count %d, want 1", calls.Load())
				}
				imported, err := f.app.FindFirstRecordByData(childCollection, "iri", "https://remote.example/api/v1/"+childKind+"/new")
				if err != nil {
					t.Fatalf("origin contributor skipped: %v", err)
				}
				if kind == "list" {
					saved, _ := f.app.FindRecordById(parentCollection, parent.Id)
					found := false
					for _, id := range saved.GetStringSlice("trails") {
						found = found || id == imported.Id
					}
					if !found {
						t.Fatal("imported trail missing from list")
					}
				}
				if existing != nil {
					saved, _ := f.app.FindRecordById(childCollection, existing.Id)
					field := "text"
					if kind == "list" {
						field = "name"
					}
					if saved.GetString(field) != "Edited" {
						t.Fatalf("stored contribution not refreshed: %q", saved.GetString(field))
					}
				}
			})
		}
	}
}

func TestRemoteChildAuthorsKeepQueuedConfirmedIdentity(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	setChildAuthorLimits(t, 500*time.Millisecond, 60*time.Millisecond, 60*time.Millisecond)
	f := newRemoteSyncFixture(t)
	parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/parent", "author": f.remoteActor.Id, "name": "Original", "needs_full_sync": true})
	known := "https://remote.example/users/known"
	f.save(f.actors, map[string]any{"iri": known, "last_fetched": time.Now().Add(-3 * time.Hour)})
	children := []any{}
	for i := range 4 {
		children = append(children, childAuthorContribution(fmt.Sprintf("waiting%d", i), fmt.Sprintf("https://remote.example/users/waiting%d", i)))
	}
	children = append(children, childAuthorContribution("known", known))
	oldClient, oldActor := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI
	t.Cleanup(func() { newRemoteSyncHTTPClient, remoteSyncGetActorByIRI = oldClient, oldActor })
	payload := map[string]any{"id": "parent", "name": "Synced", "public": true, "expand": map[string]any{"summit_logs_via_trail": children}}
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
	var started, knownStarted atomic.Int32
	remoteSyncGetActorByIRI = func(_ core.App, ctx context.Context, iri string, _ bool) (*core.Record, error) {
		if iri == known {
			knownStarted.Add(1)
			return nil, errors.New("unexpected queued lookup")
		}
		started.Add(1)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 400*time.Millisecond)
	defer cancel()
	if _, err := performFullSync(f.app, ctx, nil, parent); err != nil {
		t.Fatal(err)
	}
	if started.Load() != 4 || knownStarted.Load() != 0 {
		t.Fatalf("lookup ordering: started%d known%d", started.Load(), knownStarted.Load())
	}
	if _, err := f.app.FindFirstRecordByData(f.logs, "iri", "https://remote.example/api/v1/summit-log/known"); err != nil {
		t.Fatalf("queued confirmed contributor skipped: %v", err)
	}
}
