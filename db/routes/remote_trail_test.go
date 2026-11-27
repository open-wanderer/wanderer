package routes

import (
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteTrailSyncRequestsChildrenWithoutCallerProjections(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, projected := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "projection"}[projected], func(t *testing.T) {
			f := newRemoteSyncQueryFixture(t)
			contributor := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/casey", "last_fetched": time.Now()})
			trail := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/trail", "author": f.remoteActor.Id})
			requested := mockRemoteSyncQueryDocument(t, trail.GetString("iri"), func(query url.Values) map[string]any {
				return remoteTrailSyncQueryPayload(query, f.remoteActor.GetString("iri"), contributor.GetString("iri"))
			})
			var requestURL *url.URL
			share := ""
			if projected {
				requestURL = &url.URL{RawQuery: "fields=id,name&expand=waypoints_via_trail.author,trail_share_via_trail.actor&foo=bar&handle=riley&share=" + remoteSyncTestShare}
				share = remoteSyncTestShare
			}
			if _, err := performFullSync(f.app, t.Context(), requestURL, trail); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(f.trails, trail.Id)
			if err != nil {
				t.Fatal(err)
			}
			assertRemoteSyncQueryTrailMetadata(t, f, stored, f.remoteActor.Id)
			assertRemoteTrailSyncQueryChildren(t, f, stored, contributor.Id)
			if !stored.GetBool("full_sync_completed") || stored.GetBool("needs_full_sync") {
				t.Fatal("trail did not complete its full sync")
			}
			assertRemoteSyncDocumentQuery(t, requested(), remoteTrailSyncTestExpand, share)
		})
	}
}

func TestRemoteTrailGetSyncKeepsCallerProjection(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncQueryFixture(t)
	contributor := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/casey", "last_fetched": time.Now()})
	const iri = "https://remote.example/api/v1/trail/trail"
	requested := mockRemoteSyncQueryDocument(t, iri, func(query url.Values) map[string]any {
		return remoteTrailSyncQueryPayload(query, f.remoteActor.GetString("iri"), contributor.GetString("iri"))
	})
	// No cached trail exists: the handler must resolve a shell and block on sync.
	id := remoteSyncProjectedGet(t, f, "trail", "Origin trail")
	assertRemoteSyncDocumentQuery(t, requested(), remoteTrailSyncTestExpand, remoteSyncTestShare)
	stored, err := f.app.FindRecordById(f.trails, id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("iri") != iri || !stored.GetBool("full_sync_completed") || stored.GetBool("needs_full_sync") {
		t.Fatal("handler did not persist a completed remote trail sync")
	}
	assertRemoteSyncQueryTrailMetadata(t, f, stored, f.remoteActor.Id)
	assertRemoteTrailSyncQueryChildren(t, f, stored, contributor.Id)
}

func remoteTrailSyncQueryPayload(query url.Values, authorIRI, contributorIRI string) map[string]any {
	payload := remoteSyncQueryTrailPayload(query, "", authorIRI)
	expand := payload["expand"].(map[string]any)
	paths := strings.Split(query.Get("expand"), ",")
	if query.Get("fields") == "" {
		if slices.Contains(paths, "waypoints_via_trail") {
			expand["waypoints_via_trail"] = []any{map[string]any{"id": "point", "name": "Origin waypoint"}}
		}
		if slices.Contains(paths, "summit_logs_via_trail") {
			log := map[string]any{"id": "log", "text": "Origin summit log"}
			if slices.Contains(paths, "summit_logs_via_trail.author") {
				log["expand"] = map[string]any{"author": map[string]any{"iri": contributorIRI}}
			}
			expand["summit_logs_via_trail"] = []any{log}
		}
	}
	return payload
}

func assertRemoteTrailSyncQueryChildren(t *testing.T, f *remoteSyncFixture, trail *core.Record, contributor string) {
	t.Helper()
	for _, child := range []struct {
		collection, field, value, iri, author string
	}{
		{"waypoints", "name", "Origin waypoint", "https://remote.example/api/v1/waypoint/point", trail.GetString("author")},
		{"summit_logs", "text", "Origin summit log", "https://remote.example/api/v1/summit-log/log", contributor},
	} {
		records, err := f.app.FindAllRecords(child.collection)
		if err != nil || len(records) != 1 {
			t.Fatalf("readable %s missing from synchronized trail: count=%d error=%v", child.collection, len(records), err)
		}
		record := records[0]
		if record.GetString(child.field) != child.value || record.GetString("iri") != child.iri || record.GetString("author") != child.author || record.GetString("trail") != trail.Id {
			t.Fatalf("incomplete stored %s: %v", child.collection, record.PublicExport())
		}
	}
}
