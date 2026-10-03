package federation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
)

// Attachment downloads run under the inbound request's context: a cancelled
// context sends no further request and aborts an in-flight download.

func attachmentCancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func attachmentRequireCanceled(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("activity with a cancelled context was accepted")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want one that satisfies errors.Is(err, context.Canceled)", err)
	}
}

func TestTrailCreateCancelledContextFetchesNothing(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	act := attachmentTrailActivity(f,
		attachmentImage(server.URL+"/a.png"),
		attachmentImage(server.URL+"/b.png"),
		attachmentGPX(server.URL+"/t.gpx"),
	)
	err := ProcessCreateOrUpdateActivity(f.app, attachmentCancelledContext(), f.owner.author, f.owner.alice, act)

	attachmentRequireCanceled(t, err)
	if n := hits.total(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "trails", feedDedupTrailIRI); n != 0 {
		t.Errorf("trails rows = %d, want 0", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

func TestTrailCreateCancelStopsInFlightDownload(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		cancel()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(server.Close)

	act := attachmentTrailActivity(f,
		attachmentImage(server.URL+"/a.png"),
		attachmentImage(server.URL+"/b.png"),
		attachmentGPX(server.URL+"/t.gpx"),
	)

	done := make(chan error, 1)
	go func() {
		done <- ProcessCreateOrUpdateActivity(f.app, ctx, f.owner.author, f.owner.alice, act)
	}()

	select {
	case err := <-done:
		attachmentRequireCanceled(t, err)
	case <-time.After(5 * time.Second):
		t.Error("ProcessCreateOrUpdateActivity did not return within 5s after the context was cancelled")
		// Let the call finish before the app is torn down.
		<-done
	}
	if n := atomic.LoadInt32(&requests); n != 1 {
		t.Errorf("server received %d requests, want exactly 1", n)
	}
	if n := attachmentCount(t, f.app, "trails", feedDedupTrailIRI); n != 0 {
		t.Errorf("trails rows = %d, want 0", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

func TestSummitLogCreateCancelledContextFetchesNothing(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	act := attachmentSummitLogActivity(t, f,
		attachmentImage(server.URL+"/p.png"),
		attachmentGPX(server.URL+"/t.gpx"),
	)
	err := ProcessCreateOrUpdateActivity(f.app, attachmentCancelledContext(), f.y, f.owner.alice, act)

	attachmentRequireCanceled(t, err)
	if n := hits.total(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "summit_logs", ownerRemoteLogIRI); n != 0 {
		t.Errorf("summit_logs rows = %d, want 0", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

func TestListCreateCancelledContextFetchesNothing(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	act := feedDedupListCreate(feedDedupAuthorIRI, feedDedupListIRI)
	act.Object.(*pub.Object).Attachment = pub.ItemCollection{attachmentImage(server.URL + "/a.png")}
	err := ProcessCreateOrUpdateActivity(f.app, attachmentCancelledContext(), f.owner.author, f.owner.alice, act)

	attachmentRequireCanceled(t, err)
	if n := hits.total(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "lists", feedDedupListIRI); n != 0 {
		t.Errorf("lists rows = %d, want 0", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}

// A comment replying to an uncached trail fetches that trail and its
// attachments within the same inbound request.
func TestCommentReplyCancelledContextFetchesNoTrailAttachment(t *testing.T) {
	tmp := attachmentIsolatedTempDir(t)
	f := attachmentApp(t)
	attachmentUseLoopbackClient(t)
	server, hits := attachmentServer(t, attachmentSmallBody)

	const trailIRI = "https://remote.example.com/api/v1/trail/uncached"
	trailAct := feedDedupTrailCreate(ownerYIRI, trailIRI)
	trailObject := trailAct.Object.(*pub.Object)
	trailObject.AttributedTo = pub.IRI(ownerYIRI)
	trailObject.Attachment = pub.ItemCollection{attachmentGPX(server.URL + "/t.gpx")}

	trailAuthorRefusals.reset()
	orig := fetchTrailObject
	fetchTrailObject = func(ctx context.Context, iri string) (*pub.Object, error) {
		if iri != trailIRI {
			t.Errorf("fetched %q, want %q", iri, trailIRI)
		}
		return trailObject, nil
	}
	t.Cleanup(func() {
		fetchTrailObject = orig
		trailAuthorRefusals.reset()
	})

	act := ownerCommentActivity(pub.CreateType, ownerYIRI, ownerRemoteCommentIRI, trailIRI, "nice")
	err := ProcessCreateOrUpdateActivity(f.app, attachmentCancelledContext(), f.y, f.owner.alice, act)

	attachmentRequireCanceled(t, err)
	if n := hits.total(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
	if n := attachmentCount(t, f.app, "comments", ownerRemoteCommentIRI); n != 0 {
		t.Errorf("comments rows = %d, want 0", n)
	}
	if n := attachmentCount(t, f.app, "trails", trailIRI); n != 0 {
		t.Errorf("trails rows = %d, want 0", n)
	}
	attachmentRequireNoTempFiles(t, tmp)
}
