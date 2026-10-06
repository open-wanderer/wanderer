package trailmerge

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"

	"pocketbase/util"
)

func newRichTextMergeFixture(t *testing.T, description string) mergeContext {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "preferred_username"}, &core.BoolField{Name: "is_local"}, &core.TextField{Name: "domain"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "description", Max: 10000}, &core.DateField{Name: "date"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1})
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"summit_logs", "comments"} {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(&core.TextField{Name: "text"},
			&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
			&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1})
		if name == "summit_logs" {
			collection.Fields.Add(&core.DateField{Name: "date"})
		} else {
			collection.Fields.Add(&core.DateField{Name: "created"})
		}
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	// Ordinary model writes reject oversized values; only the production merge
	// functions below explicitly shorten their derived content before saving.
	app.OnRecordValidate().BindFunc(util.SanitizeHTML())
	actor := core.NewRecord(actors)
	actor.Set("preferred_username", "alice")
	actor.Set("is_local", true)
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	source := core.NewRecord(trails)
	source.Set("description", description)
	source.Set("author", actor.Id)
	source.Set("date", "2020-01-02 03:04:05.000Z")
	if err := app.Save(source); err != nil {
		t.Fatal(err)
	}
	target := core.NewRecord(trails)
	target.Set("description", "Target trail")
	target.Set("author", actor.Id)
	if err := app.Save(target); err != nil {
		t.Fatal(err)
	}
	return mergeContext{App: app, Actor: actor, ActorID: actor.Id, Source: source, Target: target}
}

func TestCreateTrailSummitLogBoundsLongDescriptionWithoutChangingSource(t *testing.T) {
	description := "<p><strong>" + strings.Repeat("x", 8976) + "</strong></p>"
	ctx := newRichTextMergeFixture(t, description)
	logID, err := createTrailSummitLog(ctx)
	if err != nil {
		t.Fatalf("valid 9000-character trail description could not be projected into a summit log: %v", err)
	}
	log, err := ctx.App.FindRecordById("summit_logs", logID)
	if err != nil {
		t.Fatal(err)
	}
	want := "<p><strong>" + strings.Repeat("x", 4976) + "</strong></p>"
	if got := log.GetString("text"); got != want || utf8.RuneCountInString(got) != 5000 {
		t.Fatalf("generated summit log did not retain balanced markup within its own limit: length=%d", utf8.RuneCountInString(got))
	}
	if log.GetString("author") != ctx.ActorID || log.GetString("trail") != ctx.Target.Id ||
		!log.GetDateTime("date").Time().Equal(ctx.Source.GetDateTime("date").Time()) {
		t.Fatal("generated summit log lost its owner, target or source date")
	}
	source, err := ctx.App.FindRecordById("trails", ctx.Source.Id)
	if err != nil {
		t.Fatal(err)
	}
	if source.GetString("description") != description {
		t.Fatal("creating the shorter summit log changed the source trail description")
	}
}

func TestMergeTrailCommentsBoundsAddedAttributionWithoutChangingSource(t *testing.T) {
	ctx := newRichTextMergeFixture(t, "Source trail")
	collection, err := ctx.App.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatal(err)
	}
	text := "<p>" + strings.Repeat("x", 4993) + "</p>"
	comment := core.NewRecord(collection)
	comment.Set("text", text)
	comment.Set("author", ctx.ActorID)
	comment.Set("trail", ctx.Source.Id)
	comment.Set("created", "2020-01-02 03:04:05.000Z")
	if err := ctx.App.Save(comment); err != nil {
		t.Fatalf("valid 5000-character source comment could not be saved: %v", err)
	}
	ids, err := mergeTrailComments(ctx)
	if err != nil {
		t.Fatalf("added attribution caused a valid source comment to fail the merge: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("created %d comments, want one", len(ids))
	}
	merged, err := ctx.App.FindRecordById("comments", ids[0])
	if err != nil {
		t.Fatal(err)
	}
	const attribution = "@alice (2020-01-02)\n\n"
	want := attribution + "<p>" + strings.Repeat("x", 5000-len(attribution)-7) + "</p>"
	if got := merged.GetString("text"); got != want || utf8.RuneCountInString(got) != 5000 {
		t.Fatalf("merged comment lost attribution or balanced markup: length=%d", utf8.RuneCountInString(got))
	}
	if merged.GetString("author") != ctx.ActorID || merged.GetString("trail") != ctx.Target.Id {
		t.Fatal("merged comment has the wrong owner or target")
	}
	source, err := ctx.App.FindRecordById("comments", comment.Id)
	if err != nil {
		t.Fatal(err)
	}
	if source.GetString("text") != text || source.GetString("trail") != ctx.Source.Id ||
		!source.GetDateTime("created").Time().Equal(comment.GetDateTime("created").Time()) {
		t.Fatal("comment projection changed the original text, trail or creation date")
	}
}
