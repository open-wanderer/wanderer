package routes

import (
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteListSyncRequestsReadableTrailsWithoutCallerProjections(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, projected := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "projection"}[projected], func(t *testing.T) {
			f := newRemoteSyncQueryFixture(t)
			trailAuthor := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/casey", "last_fetched": time.Now()})
			list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/list", "author": f.remoteActor.Id})
			requested := mockRemoteSyncQueryDocument(t, list.GetString("iri"), func(query url.Values) map[string]any {
				return remoteListSyncQueryPayload(query, f.remoteActor.GetString("iri"), trailAuthor.GetString("iri"))
			})
			var requestURL *url.URL
			share := ""
			if projected {
				requestURL = &url.URL{RawQuery: "fields=id,name&expand=list_share_via_list.actor&foo=bar&handle=riley&share=" + remoteSyncTestShare}
				share = remoteSyncTestShare
			}
			if _, err := performFullListSync(f.app, t.Context(), requestURL, list); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(f.lists, list.Id)
			if err != nil {
				t.Fatal(err)
			}
			assertRemoteListSyncQueryStored(t, f, stored, trailAuthor.Id)
			assertRemoteSyncDocumentQuery(t, requested(), remoteListSyncTestExpand, share)
		})
	}
}

func TestRemoteListGetSyncKeepsCallerProjection(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncQueryFixture(t)
	trailAuthor := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/casey", "last_fetched": time.Now()})
	const iri = "https://remote.example/api/v1/list/list"
	requested := mockRemoteSyncQueryDocument(t, iri, func(query url.Values) map[string]any {
		return remoteListSyncQueryPayload(query, f.remoteActor.GetString("iri"), trailAuthor.GetString("iri"))
	})
	// No cached list exists: the handler must resolve a shell and block on sync.
	id := remoteSyncProjectedGet(t, f, "list", "Public list")
	assertRemoteSyncDocumentQuery(t, requested(), remoteListSyncTestExpand, remoteSyncTestShare)
	stored, err := f.app.FindRecordById(f.lists, id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("iri") != iri {
		t.Fatal("handler did not persist the remote list identity")
	}
	assertRemoteListSyncQueryStored(t, f, stored, trailAuthor.Id)
}

func remoteListSyncQueryPayload(query url.Values, authorIRI, trailAuthorIRI string) map[string]any {
	expand := map[string]any{}
	paths := strings.Split(query.Get("expand"), ",")
	if query.Get("fields") == "" {
		if slices.Contains(paths, "author") {
			expand["author"] = map[string]any{"iri": authorIRI}
		}
		if slices.Contains(paths, "trails") {
			expand["trails"] = []any{remoteSyncQueryTrailPayload(query, "trails.", trailAuthorIRI)}
		}
	}
	return map[string]any{"id": "list", "iri": "https://remote.example/api/v1/list/list", "name": "Public list", "public": true, "expand": expand}
}

func assertRemoteListSyncQueryStored(t *testing.T, f *remoteSyncFixture, list *core.Record, trailAuthor string) {
	t.Helper()
	if list.GetString("name") != "Public list" || !list.GetBool("public") || list.GetString("author") != f.remoteActor.Id || !list.GetBool("full_sync_completed") || list.GetBool("needs_full_sync") {
		t.Fatalf("incomplete stored list: %v", list.PublicExport())
	}
	ids := list.GetStringSlice("trails")
	if len(ids) != 1 {
		t.Fatalf("readable trail missing from synchronized list: %v", ids)
	}
	trail, err := f.app.FindRecordById(f.trails, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if trail.GetString("iri") != "https://remote.example/api/v1/trail/trail" || !trail.GetBool("needs_full_sync") || trail.GetBool("full_sync_completed") {
		t.Fatal("list import lost the trail identity or full-sync requirement")
	}
	assertRemoteSyncQueryTrailMetadata(t, f, trail, trailAuthor)
}
