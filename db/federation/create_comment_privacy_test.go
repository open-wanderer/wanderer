package federation

import (
	"context"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
)

// A comment on a private trail is never federated: nothing is delivered and
// no activity is recorded, since recorded activities are publicly listed on
// the author's outbox.
func TestCommentOnPrivateTrailIsNotFederated(t *testing.T) {
	ctx := context.Background()

	for _, typ := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
		t.Run(string(typ), func(t *testing.T) {
			f := setupAddressingTestApp(t)
			author := f.actor(t, "author", true)
			// A remote trail author stands in for any remote audience: on a
			// public trail they would receive the comment.
			trailAuthor := f.actor(t, "trailauthor", false)
			trail := f.record(t, "trails", map[string]any{"author": trailAuthor.Id, "public": false}, "trail")
			comment := f.record(t, "comments", map[string]any{"author": author.Id, "trail": trail.Id, "text": "geheim"}, "comment")

			if err := CreateCommentActivity(f.app, ctx, comment, typ); err != nil {
				t.Fatal(err)
			}

			if got := f.in.wait("trailauthor", 1, 300*time.Millisecond); got != 0 {
				t.Fatalf("nothing should be delivered for a private trail, got %d", got)
			}
			records, err := f.app.FindAllRecords("activitypub_activities")
			if err != nil {
				t.Fatal(err)
			}
			if len(records) != 0 {
				t.Fatalf("no activity should be recorded for a private trail, found %d", len(records))
			}
		})
	}

	t.Run("public trail still federates", func(t *testing.T) {
		f := setupAddressingTestApp(t)
		author := f.actor(t, "author", true)
		trailAuthor := f.actor(t, "trailauthor", false)
		trail := f.record(t, "trails", map[string]any{"author": trailAuthor.Id, "public": true}, "trail")
		comment := f.record(t, "comments", map[string]any{"author": author.Id, "trail": trail.Id, "text": "toll"}, "comment")

		if err := CreateCommentActivity(f.app, ctx, comment, pub.CreateType); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("trailauthor", 1, 5*time.Second); got != 1 {
			t.Fatalf("trail author should receive the comment, got %d", got)
		}
		records, err := f.app.FindAllRecords("activitypub_activities")
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 1 {
			t.Fatalf("expected the Create to be recorded, found %d", len(records))
		}
	})
}
