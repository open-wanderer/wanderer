package routes

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/hooks"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func newRemoteReindexFixture(t *testing.T) *remoteChildFixture {
	t.Helper()
	f := newRemoteCleanupFixture(t, "summit_log")
	collection := func(name string, fields ...core.Field) *core.Collection {
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := f.app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	tags := collection("tags", &core.TextField{Name: "name"})
	category := collection("categories", &core.TextField{Name: "name"})
	f.trails.Fields.Add(&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 100}, &core.RelationField{Name: "category", CollectionId: category.Id, MaxSelect: 1})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"trail_share", "trail_like"} {
		collection(name, &core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1}, &core.RelationField{Name: "actor", CollectionId: f.actors.Id, MaxSelect: 1})
	}
	collection("activitypub_activities", &core.TextField{Name: "type"}, &core.JSONField{Name: "object"}, &core.BoolField{Name: "hidden"})
	return f
}

func TestRemoteCleanupFullReindexFlushesOnEveryCommittedExit(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, scenario := range []string{"success", "final-save-error", "index-warning", "final-save-and-index-error", "early-child-error", "rollback", "omitted-404", "omitted-404-rollback"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteReindexFixture(t)
			omitted404 := scenario == "omitted-404" || scenario == "omitted-404-rollback"
			rollback := scenario == "rollback" || scenario == "omitted-404-rollback"
			var removed []*core.Record
			for i := 0; i < 32; i++ {
				removed = append(removed, f.child(fmt.Sprintf("https://a.example/api/v1/summit-log/old%d", i), f.remoteActor.Id, f.trail.Id))
			}
			// Unlisted contributor logs are independently confirmed at their
			// canonical origin. Their deletions must join the same reindex work.
			foreign := f.child("https://c.example/api/v1/summit-log/foreign", f.remoteActor.Id, f.trail.Id)
			removed = append(removed, foreign)
			logs := []any{}
			if scenario == "early-child-error" {
				logs = append(logs, map[string]any{"id": "foreign", "iri": foreign.GetString("iri")})
				for i := 1; i <= remoteSyncWriteBatch; i++ {
					text := "Present"
					if i == remoteSyncWriteBatch {
						text = "Fail second group"
					}
					logs = append(logs, map[string]any{"id": fmt.Sprint(i), "text": text})
				}
			}
			payload := map[string]any{"id": "trail", "public": true, "name": "Refreshed", "expand": map[string]any{"summit_logs_via_trail": logs}}
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(r *http.Request) (*http.Response, error) {
					if r.URL.Host == "c.example" && r.URL.Path == "/api/v1/summit-log/foreign" {
						status := http.StatusGone
						if omitted404 {
							status = http.StatusNotFound
						}
						return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody, Request: r}, nil
					}
					if r.URL.Host != "a.example" || r.URL.Path != "/api/v1/trail/trail" {
						return nil, fmt.Errorf("unexpected remote request %s", r.URL)
					}
					return richTextSyncTransport{payload}.RoundTrip(r)
				}}}
			}
			key := canonicalLogCacheKey(f.app, f.trail, foreign.GetString("iri"))
			var proof canonicalLogAbsenceProbe
			if omitted404 {
				if first := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), payload, f.trail, "https://a.example"); len(first) != 0 {
					t.Fatal("first omitted 404 must not prepare a deletion")
				}
				canonicalRemoteLogAbsenceCache.Lock()
				entry := canonicalRemoteLogAbsenceCache.entries[key]
				if entry != nil {
					entry.first404 = entry.first404.Add(-remoteSyncThreshold - time.Second)
				}
				canonicalRemoteLogAbsenceCache.Unlock()
				if entry == nil {
					t.Fatal("first omitted 404 did not record evidence")
				}
				proof = canonicalLogAbsenceProbe{key: key, entry: entry}
				canonicalRemoteLogCache.invalidate(key)
			}
			var posts, puts, afterSuccess atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/indexes/trails/documents" {
					t.Errorf("unexpected fake index request %s", r.URL)
				}
				if r.Method == http.MethodPost {
					posts.Add(1)
					if afterSuccess.Load() == 0 {
						t.Error("reindexed before any committed delete success")
					}
					if scenario == "index-warning" || scenario == "final-save-and-index-error" || scenario == "early-child-error" {
						http.Error(w, "fake index unavailable", 500)
						return
					}
				} else if r.Method == http.MethodPut {
					puts.Add(1)
				} else {
					t.Errorf("unexpected method %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				fmt.Fprint(w, `{"taskUid":1,"status":"enqueued"}`)
			}))
			t.Cleanup(server.Close)
			client := meilisearch.New(server.URL)
			f.app.OnRecordAfterDeleteSuccess("summit_logs").BindFunc(hooks.DeleteSummitLogHandler(client))
			f.app.OnRecordAfterDeleteSuccess("summit_logs").BindFunc(func(e *core.RecordEvent) error {
				if e.App.TxInfo() != nil {
					t.Error("after-success hook ran inside transaction")
				}
				afterSuccess.Add(1)
				return e.Next()
			})
			f.app.OnRecordAfterUpdateSuccess("trails").BindFunc(hooks.UpdateTrailHandler(client, hooks.NewFederationUpdates(f.app)))
			abort := errors.New("original sync error")
			if scenario == "final-save-error" || scenario == "final-save-and-index-error" {
				f.app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
					if e.Record.GetBool("full_sync_completed") {
						return abort
					}
					return e.Next()
				})
			}
			if scenario == "early-child-error" {
				f.app.OnRecordCreate("summit_logs").BindFunc(func(e *core.RecordEvent) error {
					if e.Record.GetString("text") == "Fail second group" {
						return abort
					}
					return e.Next()
				})
			}
			if rollback {
				var attempts int
				f.app.OnRecordDelete("summit_logs").BindFunc(func(e *core.RecordEvent) error {
					attempts++
					if attempts == 2 {
						return abort
					}
					return e.Next()
				})
			}
			_, err := performFullSync(f.app, t.Context(), nil, f.trail)
			wantError := scenario != "success" && scenario != "index-warning" && scenario != "omitted-404"
			if wantError && !errors.Is(err, abort) || !wantError && err != nil {
				t.Fatalf("index warning changed original sync result: %v", err)
			}
			wantPosts, wantDeletes, wantPuts := int64(1), int64(len(removed)), int64(2)
			if rollback {
				wantPosts, wantDeletes, wantPuts = 0, 0, 1
			} else if scenario == "early-child-error" {
				wantDeletes, wantPuts = 1, 1
			}
			if posts.Load() != wantPosts || afterSuccess.Load() != wantDeletes || puts.Load() != wantPuts {
				t.Fatalf("POST=%d PUT=%d delete hooks=%d, want POST=%d PUT=%d hooks=%d", posts.Load(), puts.Load(), afterSuccess.Load(), wantPosts, wantPuts, wantDeletes)
			}
			t.Logf("cleanup requests: full POST=%d partial PUT=%d committed delete hooks=%d", posts.Load(), puts.Load(), afterSuccess.Load())
			if omitted404 {
				canonicalRemoteLogAbsenceCache.Lock()
				remaining := canonicalRemoteLogAbsenceCache.entries[key]
				canonicalRemoteLogAbsenceCache.Unlock()
				if rollback {
					if remaining != proof.entry || !canonicalRemoteLogAbsenceCache.ready(proof, canonicalLogAbsenceSnapshotOf(foreign, f.trail)) || canonicalRemoteLogCache.skip(key, "canonical-absence", foreign.GetString("updated"), true) {
						t.Fatal("coalesced rollback consumed proof or retained a negative-cache entry")
					}
				} else if remaining != nil {
					t.Fatal("coalesced deletion bypassed matching-generation forgetAfterCommit")
				}
			}
			if rollback {
				for _, record := range removed {
					if _, err := f.app.FindRecordById("summit_logs", record.Id); err != nil {
						t.Fatalf("cleanup committed a deletion before rollback: %v", err)
					}
				}
			}
		})
	}
}
