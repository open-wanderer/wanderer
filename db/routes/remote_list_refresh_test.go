package routes

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteListRefreshKeepsReferencesUntilAuthorConfirmation(t *testing.T) {
	for _, failure := range []string{"unavailable", "timeout"} {
		t.Run(failure, func(t *testing.T) {
			f, list := topLevelAuthorFixture(t, "list", true)
			contributor := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/contributor", "public_key": "", "last_fetched": time.Time{}})
			existing := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/existing", "name": "Cached", "author": contributor.Id, "public": true})
			unlinked := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/unlinked", "name": "Unlinked", "author": contributor.Id, "public": true})
			list.Set("trails", []string{existing.Id})
			if err := f.app.Save(list); err != nil {
				t.Fatal(err)
			}
			children := []any{}
			for _, id := range []string{"existing", "new", "unlinked"} {
				children = append(children, map[string]any{"id": id, "iri": "https://remote.example/api/v1/trail/" + id, "name": "Refreshed", "public": true, "expand": map[string]any{"author": map[string]any{"iri": contributor.GetString("iri")}}})
			}
			payload := topLevelAuthorPayload(list, f.remoteActor.GetString("iri"))
			payload["expand"].(map[string]any)["trails"] = children
			serveTopLevelAuthorPayload(t, payload)
			previous, previousTimeout := remoteSyncGetActorByIRI, sameOriginAuthorTimeout
			t.Cleanup(func() { remoteSyncGetActorByIRI, sameOriginAuthorTimeout = previous, previousTimeout })
			sameOriginAuthorTimeout = 40 * time.Millisecond
			remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, iri string, follows bool) (*core.Record, error) {
				if failure == "timeout" {
					<-ctx.Done()
					return contributor, ctx.Err()
				}
				return contributor, fmt.Errorf("origin unavailable")
			}
			if _, err := syncTopLevelAuthorFixture(t, f, "list", list); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(f.lists, list.Id)
			if err != nil || !slices.Equal(stored.GetStringSlice("trails"), []string{existing.Id}) {
				t.Fatalf("unconfirmed refresh changed list references: record=%v error=%v", stored, err)
			}
			f.assertUnchanged(existing, "name", "author", "iri")
			f.assertUnchanged(unlinked, "name", "author", "iri")
			if count, err := f.app.CountRecords("trails"); err != nil || count != 2 {
				t.Fatalf("unconfirmed new trail saved: count=%d error=%v", count, err)
			}
			contributor.Set("public_key", "refreshed-key")
			contributor.Set("last_fetched", time.Now())
			if err := f.app.Save(contributor); err != nil {
				t.Fatal(err)
			}
			remoteSyncGetActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) { return contributor, nil }
			if _, err := syncTopLevelAuthorFixture(t, f, "list", stored); err != nil {
				t.Fatal(err)
			}
			if len(stored.GetStringSlice("trails")) != 3 {
				t.Fatalf("confirmed refresh did not restore all references: %v", stored.GetStringSlice("trails"))
			}
		})
	}
}
