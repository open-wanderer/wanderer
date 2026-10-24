package routes

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteCommentSyncBoundsTheWholeCanonicalAuthorLookupPhase(t *testing.T) {
	f := newRemoteCommentFixture(t)
	origin := f.save("activitypub_actors", map[string]any{"iri": "https://origin.example/users/alice", "domain": "origin.example", "public_key": "test-key", "last_fetched": time.Now()})
	trail := f.save("trails", map[string]any{"name": "Remote", "author": origin.Id, "public": true, "iri": "https://origin.example/api/v1/trail/original"})
	network := &canonicalCommentNetwork{replies: map[string]canonicalCommentReply{}}
	for i := 0; i < 25; i++ {
		iri := fmt.Sprintf("https://notes.example/notes/reply%d", i)
		network.list = append(network.list, map[string]any{"iri": iri})
		network.replies[fmt.Sprintf("notes.example/notes/reply%d", i)] = canonicalCommentReply{status: http.StatusOK, body: map[string]any{
			"id": iri, "type": "Note", "attributedTo": fmt.Sprintf("https://notes.example/users/author%d", i), "inReplyTo": trail.GetString("iri"),
			"to": "https://www.w3.org/ns/activitystreams#Public", "content": "Confirmed object whose author stalls",
		}}
	}
	useRemoteCommentClient(t, &http.Client{Transport: network})
	previousGetter, previousTimeout := remoteSyncGetActorByIRI, canonicalRemoteFetchTimeout
	t.Cleanup(func() { remoteSyncGetActorByIRI, canonicalRemoteFetchTimeout = previousGetter, previousTimeout })
	canonicalRemoteFetchTimeout = 2 * time.Second
	var started atomic.Int32
	remoteSyncGetActorByIRI = func(_ core.App, ctx context.Context, _ string, includeFollows bool) (*core.Record, error) {
		if includeFollows {
			t.Error("canonical author lookup fetched social collections")
		}
		started.Add(1)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	start := time.Now()
	runConfirmedCommentPolicySync(t, f, trail)
	elapsed := time.Since(start)
	if elapsed > 12*time.Second || started.Load() > 24 {
		t.Fatalf("canonical author phase escaped its ten-second budget: elapsed=%s started=%d", elapsed, started.Load())
	}
	if elapsed < 8*time.Second || started.Load() < 16 || network.count("notes.example/notes/") != 25 {
		t.Fatalf("fixture did not exercise an exhausted author phase: elapsed=%s started=%d fetched=%d", elapsed, started.Load(), network.count("notes.example/notes/"))
	}
	comments, err := f.app.FindAllRecords("comments")
	if err != nil || len(comments) != 0 {
		t.Fatalf("unconfirmed authors were imported: comments=%v error=%v", comments, err)
	}
}
