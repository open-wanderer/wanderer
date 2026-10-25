package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// A remote actor's Create or Update of a comment or summit log may only write
// to a record that lives on its own host and that it already authors.

const (
	ownershipLocalTrailIRI    = "https://trails.example.com/api/v1/trail/lt1"
	ownershipLocalCommentIRI  = "https://trails.example.com/api/v1/comment/c1"
	ownershipRemoteCommentIRI = "https://remote.example.com/api/v1/comment/c2"
	ownershipThirdCommentIRI  = "https://third.example.com/api/v1/comment/c3"
	ownershipLocalLogIRI      = "https://trails.example.com/api/v1/summit-log/s1"
)

func ownershipSeed(t *testing.T, app core.App, collection, iri, authorID, trailID, text string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("author", authorID)
	r.Set("trail", trailID)
	r.Set("text", text)
	if err := app.Save(r); err != nil {
		t.Fatalf("seed %s: %v", collection, err)
	}
}

func ownershipCommentUpdate(actorIRI, commentIRI, trailIRI, text string) pub.Activity {
	obj := &pub.Object{
		ID:        pub.IRI(commentIRI),
		Type:      pub.NoteType,
		To:        pub.ItemCollection{pub.PublicNS},
		Content:   ownerText(text),
		InReplyTo: pub.IRI(trailIRI),
	}
	act := pub.ActivityNew(pub.IRI("https://remote.example.com/api/v1/activitypub/activity/u1"), pub.UpdateType, obj)
	act.Actor = pub.IRI(actorIRI)
	return *act
}

func ownershipRequire(t *testing.T, app core.App, collection, iri, wantAuthor, wantText string) {
	t.Helper()
	r := attachmentStored(t, app, collection, iri)
	if got := r.GetString("author"); got != wantAuthor {
		t.Errorf("%s author = %q, want %q", collection, got, wantAuthor)
	}
	if got := r.GetString("text"); got != wantText {
		t.Errorf("%s text = %q, want %q", collection, got, wantText)
	}
}

func TestRemoteUpdateCannotOverwriteLocalComment(t *testing.T) {
	f := attachmentApp(t)
	trail := seedTrailRecord(t, f.app, ownershipLocalTrailIRI, true, f.owner.alice.Id)
	ownershipSeed(t, f.app, "comments", ownershipLocalCommentIRI, f.owner.alice.Id, trail.Id, "original")

	act := ownershipCommentUpdate(feedDedupAuthorIRI, ownershipLocalCommentIRI, ownershipLocalTrailIRI, "overwritten")
	if err := ProcessCreateOrUpdateActivity(f.app, f.owner.author, f.owner.alice, act); err == nil {
		t.Error("remote Update of a local comment was accepted")
	}
	ownershipRequire(t, f.app, "comments", ownershipLocalCommentIRI, f.owner.alice.Id, "original")
}

func TestRemoteUpdateCannotOverwriteAnotherActorsComment(t *testing.T) {
	f := attachmentApp(t)
	trail := seedTrailRecord(t, f.app, ownerRemoteTrailIRI, true, f.y.Id)
	ownershipSeed(t, f.app, "comments", ownershipRemoteCommentIRI, f.y.Id, trail.Id, "original")

	// author and y are both on remote.example.com; only y wrote the comment.
	act := ownershipCommentUpdate(feedDedupAuthorIRI, ownershipRemoteCommentIRI, ownerRemoteTrailIRI, "overwritten")
	if err := ProcessCreateOrUpdateActivity(f.app, f.owner.author, f.owner.alice, act); err == nil {
		t.Error("Update of another actor's comment was accepted")
	}
	ownershipRequire(t, f.app, "comments", ownershipRemoteCommentIRI, f.y.Id, "original")
}

func TestRemoteCreateCannotClaimCommentOnAnotherHost(t *testing.T) {
	f := attachmentApp(t)
	seedTrailRecord(t, f.app, ownerRemoteTrailIRI, true, f.y.Id)

	act := ownershipCommentUpdate(feedDedupAuthorIRI, ownershipThirdCommentIRI, ownerRemoteTrailIRI, "squatted")
	act.Type = pub.CreateType
	if err := ProcessCreateOrUpdateActivity(f.app, f.owner.author, f.owner.alice, act); err == nil {
		t.Error("Create of a comment on another host was accepted")
	}
	if n := attachmentCount(t, f.app, "comments", ownershipThirdCommentIRI); n != 0 {
		t.Errorf("comments rows = %d, want 0", n)
	}
}

func TestRemoteUpdateOfOwnCommentIsSanitized(t *testing.T) {
	f := attachmentApp(t)
	trail := seedTrailRecord(t, f.app, ownerRemoteTrailIRI, true, f.y.Id)
	ownershipSeed(t, f.app, "comments", ownershipRemoteCommentIRI, f.owner.author.Id, trail.Id, "original")

	act := ownershipCommentUpdate(feedDedupAuthorIRI, ownershipRemoteCommentIRI, ownerRemoteTrailIRI,
		`<p>edited</p><img src=x onerror="alert(document.cookie)">`)
	if err := ProcessCreateOrUpdateActivity(f.app, f.owner.author, f.owner.alice, act); err != nil {
		t.Fatalf("author's own Update was refused: %v", err)
	}
	ownershipRequire(t, f.app, "comments", ownershipRemoteCommentIRI, f.owner.author.Id, "<p>edited</p>")
}

func TestRemoteUpdateCannotOverwriteLocalSummitLog(t *testing.T) {
	f := attachmentApp(t)
	trail := seedTrailRecord(t, f.app, ownershipLocalTrailIRI, true, f.owner.alice.Id)
	ownershipSeed(t, f.app, "summit_logs", ownershipLocalLogIRI, f.owner.alice.Id, trail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, feedDedupAuthorIRI, ownershipLocalLogIRI, ownershipLocalTrailIRI, "overwritten")
	if err := ProcessCreateOrUpdateActivity(f.app, f.owner.author, f.owner.alice, act); err == nil {
		t.Error("remote Update of a local summit log was accepted")
	}
	ownershipRequire(t, f.app, "summit_logs", ownershipLocalLogIRI, f.owner.alice.Id, "original")
}

func TestRemoteUpdateCannotOverwriteAnotherActorsSummitLog(t *testing.T) {
	f := attachmentApp(t)
	trail := seedTrailRecord(t, f.app, ownerRemoteTrailIRI, true, f.y.Id)
	ownershipSeed(t, f.app, "summit_logs", ownerRemoteLogIRI, f.y.Id, trail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, feedDedupAuthorIRI, ownerRemoteLogIRI, ownerRemoteTrailIRI, "overwritten")
	if err := ProcessCreateOrUpdateActivity(f.app, f.owner.author, f.owner.alice, act); err == nil {
		t.Error("Update of another actor's summit log was accepted")
	}
	ownershipRequire(t, f.app, "summit_logs", ownerRemoteLogIRI, f.y.Id, "original")
}

func TestRemoteUpdateOfOwnSummitLogIsApplied(t *testing.T) {
	f := attachmentApp(t)
	trail := seedTrailRecord(t, f.app, ownerRemoteTrailIRI, true, f.y.Id)
	ownershipSeed(t, f.app, "summit_logs", ownerRemoteLogIRI, f.y.Id, trail.Id, "original")

	act := ownerSummitLogActivity(pub.UpdateType, ownerYIRI, ownerRemoteLogIRI, ownerRemoteTrailIRI, "edited")
	if err := ProcessCreateOrUpdateActivity(f.app, f.y, f.owner.alice, act); err != nil {
		t.Fatalf("author's own Update was refused: %v", err)
	}
	ownershipRequire(t, f.app, "summit_logs", ownerRemoteLogIRI, f.y.Id, "edited")
}
