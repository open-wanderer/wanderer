package routes

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteChildAuthorsLoadNewIdentityBeforeStoredContributors(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			testRemoteChildAuthorProgress(t, kind, 8, 1)
		})
	}
}

func TestRemoteChildAuthorsProgressAcrossExpiredIdentityWindows(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			testRemoteChildAuthorProgress(t, kind, 0, 3)
		})
	}
}

func testRemoteChildAuthorProgress(t *testing.T, kind string, storedCount, refreshes int) {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	setChildAuthorLimits(t, 5*time.Millisecond, time.Second, time.Second)
	f := newRemoteSyncFixture(t)
	parents, children, relation, childKind := f.trails, f.logs, "summit_logs_via_trail", "summit-log"
	if kind == "list" {
		parents, children, relation, childKind = f.lists, f.trails, "trails", "trail"
	}
	parent := f.save(parents, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/parent", "author": f.remoteActor.Id, "name": "Original", "needs_full_sync": true})
	expandedChildren := []any{}
	for i := range 9 {
		iri := fmt.Sprintf("https://remote.example/users/contributor%d", i)
		expandedChildren = append(expandedChildren, childAuthorContribution(fmt.Sprintf("entry%d", i), iri))
		if i < storedCount {
			f.save(f.actors, map[string]any{"iri": iri, "last_fetched": time.Now().Add(-3 * time.Hour)})
		}
	}
	oldClient, oldActor := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI
	t.Cleanup(func() { newRemoteSyncHTTPClient, remoteSyncGetActorByIRI = oldClient, oldActor })
	payload := map[string]any{"id": "parent", "name": "Synced", "public": true, "expand": map[string]any{relation: expandedChildren}}
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
	remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, iri string, _ bool) (*core.Record, error) {
		// Every contributor answers within its own limit, but only one group
		// of four fits in a step. All stored identities are expired below.
		timer := time.NewTimer(3 * sameOriginAuthorBudget / 5)
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
	for refresh := range refreshes {
		// Put every saved identity beyond the actor cache window before
		// the next refresh, as with profile refreshes three hours apart.
		actors, err := f.app.FindRecordsByFilter("activitypub_actors", "is_local=false", "", -1, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, actor := range actors {
			actor.Set("last_fetched", time.Now().Add(-3*time.Hour))
			if err := f.app.Save(actor); err != nil {
				t.Fatal(err)
			}
		}
		parent, err = f.app.FindRecordById(parents, parent.Id)
		if err != nil {
			t.Fatal(err)
		}
		if kind == "trail" {
			_, err = performFullSync(f.app, t.Context(), nil, parent)
		} else {
			_, err = performFullListSync(f.app, t.Context(), nil, parent)
		}
		if err != nil {
			t.Fatalf("refresh %d: %v", refresh+1, err)
		}
	}
	parent, err := f.app.FindRecordById(parents, parent.Id)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 9 {
		iri := fmt.Sprintf("https://remote.example/api/v1/%s/entry%d", childKind, i)
		child, err := f.app.FindFirstRecordByData(children, "iri", iri)
		if err != nil {
			t.Errorf("entry %d missing after %d refreshes: %v", i, refreshes, err)
			continue
		}
		if kind == "list" {
			linked := false
			for _, id := range parent.GetStringSlice("trails") {
				linked = linked || id == child.Id
			}
			if !linked {
				t.Errorf("entry %d missing from list", i)
			}
		}
	}
}
