package federation

import (
	"context"
	"errors"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

const legacyThirdHostActorIRI = "https://third.example.com/api/v1/activitypub/user/commenter"

// stubTrailOrigin replaces fetchTrailObject for the test. It returns an object
// with the given id attributed to attributedTo (or err), and records every
// requested IRI. The original is restored on cleanup.
func stubTrailOrigin(t *testing.T, id, attributedTo string, err error) *[]string {
	t.Helper()
	calls := &[]string{}
	trailAuthorRefusals.reset()
	orig := fetchTrailObject
	fetchTrailObject = func(ctx context.Context, iri string) (*pub.Object, error) {
		*calls = append(*calls, iri)
		if err != nil {
			return nil, err
		}
		o := pub.ObjectNew(pub.NoteType)
		o.ID = pub.IRI(id)
		o.AttributedTo = pub.IRI(attributedTo)
		return o, nil
	}
	t.Cleanup(func() {
		fetchTrailObject = orig
		trailAuthorRefusals.reset()
	})
	return calls
}

func legacyAssertUnchanged(t *testing.T, app core.App, trail *core.Record, wantAuthor string) {
	t.Helper()
	reloaded, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatalf("reload trail: %v", err)
	}
	if got := reloaded.GetString("author"); got != wantAuthor {
		t.Errorf("trail author = %q, want %q", got, wantAuthor)
	}
	if reloaded.GetBool("needs_full_sync") {
		t.Error("needs_full_sync was set by a refused activity")
	}
}

func legacyAssertRepaired(t *testing.T, app core.App, trail *core.Record, wantAuthor string, wantSync bool) {
	t.Helper()
	reloaded, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatalf("reload trail: %v", err)
	}
	if got := reloaded.GetString("author"); got != wantAuthor {
		t.Errorf("trail author = %q, want %q", got, wantAuthor)
	}
	if wantSync && !reloaded.GetBool("needs_full_sync") {
		t.Error("needs_full_sync = false after the Update, want true")
	}
}

// A trail stored under a commenter on another host.
func TestUpdateLegacyTrailThirdHostAuthorRepaired(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	third := createTestActor(t, app, legacyThirdHostActorIRI, "person", false)
	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, third.Id)
	stubTrailOrigin(t, feedDedupTrailIRI, feedDedupAuthorIRI, nil)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI))
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act); err != nil {
		t.Fatalf("Update from the real author of a legacy trail: %v", err)
	}
	legacyAssertRepaired(t, app, trail, a.author.Id, true)
}

// A trail stored under the wrong author on the trail's own host.
func TestUpdateLegacyTrailSameHostAuthorRepaired(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, a.other.Id)
	stubTrailOrigin(t, feedDedupTrailIRI, feedDedupAuthorIRI, nil)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI))
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act); err != nil {
		t.Fatalf("Update from the real author of a legacy trail: %v", err)
	}
	legacyAssertRepaired(t, app, trail, a.author.Id, true)
}

func TestCreateDuplicateLegacyTrailRepaired(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	third := createTestActor(t, app, legacyThirdHostActorIRI, "person", false)
	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, third.Id)
	stubTrailOrigin(t, feedDedupTrailIRI, feedDedupAuthorIRI, nil)

	act := feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI)
	if err := ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act); err != nil {
		t.Fatalf("duplicate Create from the real author of a legacy trail: %v", err)
	}
	legacyAssertRepaired(t, app, trail, a.author.Id, false)
	if got := feedDedupCount(t, app, a.alice.Id, trail.Id); got != 1 {
		t.Errorf("feed rows for alice = %d, want 1", got)
	}
}

func TestUpdateTrailNonAuthorRefusedWhenOriginNamesStoredAuthor(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, a.author.Id)
	stubTrailOrigin(t, feedDedupTrailIRI, feedDedupAuthorIRI, nil)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupOtherIRI, feedDedupTrailIRI))
	authorGuardRequireRefused(t, ProcessCreateOrUpdateActivity(app, context.Background(), a.other, a.alice, act))
	legacyAssertUnchanged(t, app, trail, a.author.Id)
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestUpdateLegacyTrailRefusedWhenOriginUnreachable(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	third := createTestActor(t, app, legacyThirdHostActorIRI, "person", false)
	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, third.Id)
	stubTrailOrigin(t, "", "", errors.New("origin unreachable"))

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI))
	authorGuardRequireRefused(t, ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act))
	legacyAssertUnchanged(t, app, trail, third.Id)
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestUpdateLegacyTrailRefusedWhenOriginIDDiffers(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	third := createTestActor(t, app, legacyThirdHostActorIRI, "person", false)
	trail := seedTrailRecord(t, app, feedDedupTrailIRI, true, third.Id)
	stubTrailOrigin(t, "https://remote.example.com/api/v1/trail/other", feedDedupAuthorIRI, nil)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, feedDedupTrailIRI))
	authorGuardRequireRefused(t, ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act))
	legacyAssertUnchanged(t, app, trail, third.Id)
	authorGuardAssertNoFeed(t, app, a.alice.Id)
}

func TestUpdateForeignHostTrailNeverFetchesOrigin(t *testing.T) {
	app := authorGuardTestApp(t)
	a := feedDedupSeedActors(t, app)
	third := createTestActor(t, app, legacyThirdHostActorIRI, "person", false)
	trail := seedTrailRecord(t, app, authorGuardForeignTrailIRI, true, third.Id)
	calls := stubTrailOrigin(t, authorGuardForeignTrailIRI, feedDedupAuthorIRI, nil)

	act := authorGuardAsUpdate(feedDedupTrailCreate(feedDedupAuthorIRI, authorGuardForeignTrailIRI))
	authorGuardRequireRefused(t, ProcessCreateOrUpdateActivity(app, context.Background(), a.author, a.alice, act))
	if len(*calls) != 0 {
		t.Errorf("origin was fetched %d times for a foreign-host object, want 0", len(*calls))
	}
	legacyAssertUnchanged(t, app, trail, third.Id)
}
