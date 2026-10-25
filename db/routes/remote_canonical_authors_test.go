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

func TestCanonicalObjectsResolveOneNewAuthorWithAnIndependentDeadline(t *testing.T) {
	for _, kind := range []string{"summit-log", "comment"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("ORIGIN", "https://local.example")
			f := newRemoteChildFixture(t, "summit_log")
			iri := "https://third.example/users/new"
			originalClient, originalActor, originalTimeout := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI, canonicalRemoteFetchTimeout
			t.Cleanup(func() {
				newRemoteSyncHTTPClient, remoteSyncGetActorByIRI, canonicalRemoteFetchTimeout = originalClient, originalActor, originalTimeout
			})
			canonicalRemoteFetchTimeout = 80 * time.Millisecond
			var metadata, lookups atomic.Int32
			remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, authorIRI string, includeFollows bool) (*core.Record, error) {
				lookups.Add(1)
				if metadata.Load() != 4 || authorIRI != iri || includeFollows {
					t.Errorf("author resolved before all confirmed objects: metadata=%d iri=%s", metadata.Load(), authorIRI)
				}
				select {
				case <-time.After(50 * time.Millisecond):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				// Emulate another request winning the same unique actor write.
				f.save(f.actors, map[string]any{"iri": iri, "last_fetched": time.Now()})
				return nil, errors.New("UNIQUE constraint failed: activitypub_actors.iri")
			}
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					select {
					case <-time.After(50 * time.Millisecond):
					case <-request.Context().Done():
						return nil, request.Context().Err()
					}
					metadata.Add(1)
					id := request.URL.Path[len("/api/v1/"+kind+"/"):]
					payload := map[string]any{"id": id, "iri": "https://third.example" + request.URL.Path, "text": "Confirmed", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": iri}}}
					return richTextSyncTransport{payload}.RoundTrip(request)
				}}}
			}
			entries := []any{}
			for i := 0; i < 4; i++ {
				objectIRI := fmt.Sprintf("https://third.example/api/v1/%s/%d", kind, i)
				entries = append(entries, map[string]any{"iri": objectIRI})
			}
			count := 0
			if kind == "summit-log" {
				count = len(fetchCanonicalRemoteSummitLogs(f.app, t.Context(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": entries}}, f.trail, "https://remote.example"))
			} else {
				references := []canonicalCommentReference{}
				for _, entry := range entries {
					references = append(references, canonicalCommentReference{iri: entry.(map[string]any)["iri"].(string)})
				}
				count = len(fetchCanonicalRemoteComments(f.app, t.Context(), newRemoteSyncHTTPClient(), f.trail, references))
			}
			if count != 4 || lookups.Load() != 1 {
				t.Fatalf("canonical objects lost to a shared actor race/deadline: objects=%d actor lookups=%d", count, lookups.Load())
			}
		})
	}
}
