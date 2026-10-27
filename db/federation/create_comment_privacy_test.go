package federation

import (
	"context"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// mentionFixture is a local trail author and commenter, plus a remote actor
// the comment mentions. The remote actor is cached, so resolving the mention
// does not go to the network.
func mentionFixture(t *testing.T) (f *addressingFixture, author, trailAuthor *core.Record) {
	t.Helper()

	f = setupAddressingTestApp(t)
	actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(&core.DateField{Name: "last_fetched"}, &core.TextField{Name: "public_key"})
	if err := f.app.Save(actors); err != nil {
		t.Fatal(err)
	}

	author = f.actor(t, "author", true)
	trailAuthor = f.actor(t, "trailauthor", true)
	mentioned := f.actor(t, "carol", false)
	mentioned.Set("domain", "remote.example")
	mentioned.Set("last_fetched", time.Now())
	mentioned.Set("public_key", "trusted cached actor key")
	if err := f.app.Save(mentioned); err != nil {
		t.Fatal(err)
	}
	return f, author, trailAuthor
}

const mentionText = `<p>Hi <a class="mention">@carol@remote.example</a></p>`

// A comment on a private trail is never federated: nothing is delivered and
// no activity is recorded, since recorded activities are publicly listed on
// the author's outbox. Remote trails are always stored as public, so the trail
// is local and the comment reaches remote actors only through mentions.
func TestCommentOnPrivateTrailIsNotFederated(t *testing.T) {
	ctx := context.Background()

	for _, typ := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
		t.Run(string(typ), func(t *testing.T) {
			f, author, trailAuthor := mentionFixture(t)
			trail := f.record(t, "trails", map[string]any{"author": trailAuthor.Id, "public": false}, "trail")
			comment := f.record(t, "comments", map[string]any{"author": author.Id, "trail": trail.Id, "text": mentionText}, "comment")

			if err := CreateCommentActivity(f.app, ctx, comment, typ); err != nil {
				t.Fatal(err)
			}

			if got := f.in.wait("carol", 1, 300*time.Millisecond); got != 0 {
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
		f, author, trailAuthor := mentionFixture(t)
		trail := f.record(t, "trails", map[string]any{"author": trailAuthor.Id, "public": true}, "trail")
		comment := f.record(t, "comments", map[string]any{"author": author.Id, "trail": trail.Id, "text": mentionText}, "comment")

		if err := CreateCommentActivity(f.app, ctx, comment, pub.CreateType); err != nil {
			t.Fatal(err)
		}

		if got := f.in.wait("carol", 1, 5*time.Second); got != 1 {
			t.Fatalf("mentioned actor should receive the comment, got %d", got)
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
