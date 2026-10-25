package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteCommentSyncRemovesOnlyCanonicallyGoneCopiesAndCompleteOmissions(t *testing.T) {
	for _, kind := range []string{"comment", "note"} {
		for _, state := range []string{"listed-gone", "omitted-gone", "omitted-404", "unavailable", "available", "complete-30", "complete-45", "partial-list", "missing-list", "malformed-list", "owner-changed", "historical-local", "listed-historical-id"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				f := newRemoteCommentFixture(t)
				origin := f.save("activitypub_actors", map[string]any{"iri": "https://origin.example/users/alice", "domain": "origin.example", "public_key": "test-key", "last_fetched": time.Now()})
				author := f.save("activitypub_actors", map[string]any{"iri": "https://third.example/users/carol", "domain": "third.example", "public_key": "test-key", "last_fetched": time.Now()})
				trail := f.save("trails", map[string]any{"name": "Remote", "iri": "https://origin.example/api/v1/trail/original", "author": origin.Id, "public": true})
				iri := "https://third.example/api/v1/comment/entry"
				if kind == "note" {
					iri = "https://third.example/notes/entry"
				}
				comment := f.save("comments", map[string]any{"iri": iri, "trail": trail.Id, "author": author.Id, "text": "Stored reply"})
				if state == "historical-local" {
					local := f.localActor("local")
					comment.Set("author", local.Id)
					comment.Set("iri", "https://third.example/api/v1/comment/"+comment.Id)
					iri = comment.GetString("iri")
					if err := f.app.Save(comment); err != nil {
						t.Fatal(err)
					}
				}
				if state == "listed-historical-id" {
					comment.Set("iri", "https://third.example/api/v1/comment/"+comment.Id)
					iri = comment.GetString("iri")
					if err := f.app.Save(comment); err != nil {
						t.Fatal(err)
					}
				}
				requests := 0
				useRemoteCommentClient(t, &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "origin.example" {
						items := []any{}
						if state == "listed-gone" || state == "owner-changed" || state == "listed-historical-id" {
							items = append(items, map[string]any{"iri": iri})
						}
						if state == "malformed-list" {
							items = append(items, map[string]any{"text": "missing identity"})
						}
						if state == "complete-30" || state == "complete-45" {
							count := 30
							if state == "complete-45" {
								count = 45
							}
							for i := 0; i < count; i++ {
								items = append(items, map[string]any{"iri": fmt.Sprintf("https://origin.example/api/v1/comment/current%d", i), "text": "Current origin comment"})
							}
						}
						payload := map[string]any{"items": items}
						if state == "missing-list" {
							delete(payload, "items")
						}
						if state == "partial-list" {
							payload["totalPages"] = 2
							payload["totalItems"] = 40
						}
						if state == "complete-30" || state == "complete-45" {
							payload["totalPages"] = 1
							payload["totalItems"] = len(items)
						}
						return richTextSyncTransport{payload}.RoundTrip(request)
					}
					requests++
					if state == "owner-changed" {
						stored, _ := f.app.FindRecordById("comments", comment.Id)
						stored.Set("author", origin.Id)
						if err := f.app.Save(stored); err != nil {
							t.Fatal(err)
						}
					}
					if state == "available" {
						payload := map[string]any{"id": "entry", "iri": iri, "text": "Current reply", "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": trail.GetString("iri")}}}
						if kind == "note" {
							payload = map[string]any{"id": iri, "type": "Note", "content": "Current reply", "attributedTo": author.GetString("iri"), "inReplyTo": trail.GetString("iri"), "to": []any{"Public"}}
						}
						return richTextSyncTransport{payload}.RoundTrip(request)
					}
					status := http.StatusGone
					if state == "omitted-404" {
						status = http.StatusNotFound
					}
					if state == "unavailable" {
						status = http.StatusServiceUnavailable
					}
					return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
				}}})
				event := &core.RequestEvent{App: f.app}
				event.Request = httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
				event.Response = httptest.NewRecorder()
				if err := syncRemoteComments(event, trail); err != nil {
					t.Fatal(err)
				}
				_, err := f.app.FindRecordById("comments", comment.Id)
				deleted := state == "listed-gone" || state == "omitted-gone" || state == "omitted-404" || state == "complete-30" || state == "complete-45"
				if (err != nil) != deleted {
					t.Fatalf("comment deletion=%v state=%s", err != nil, state)
				}
				wantRequests := 1
				if state == "partial-list" || state == "missing-list" || state == "malformed-list" || state == "historical-local" || state == "listed-historical-id" {
					wantRequests = 0
				}
				if requests != wantRequests {
					t.Fatalf("canonical probes=%d want%d", requests, wantRequests)
				}
			})
		}
	}
}
