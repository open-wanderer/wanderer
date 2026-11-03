package hooks

import (
	"errors"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
)

const guardInstanceIRI = "https://trails.example.com/api/v1/activitypub/instance"

type guardFixture struct {
	app      core.App
	instance *core.Record
	alice    *core.Record
	bob      *core.Record
	remote   *core.Record
}

func newGuardFixture(t *testing.T) *guardFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	app := newFollowTestApp(t)
	return &guardFixture{
		app:      app,
		instance: createFollowTestActor(t, app, guardInstanceIRI, "instance", true),
		alice:    createFollowTestActor(t, app, "https://trails.example.com/api/v1/activitypub/user/alice", "person", true),
		bob:      createFollowTestActor(t, app, "https://trails.example.com/api/v1/activitypub/user/bob", "person", true),
		remote:   createFollowTestActor(t, app, "https://remote.example.com/api/v1/activitypub/instance", "instance", false),
	}
}

func (f *guardFixture) regularAuth() *core.Record {
	return core.NewRecord(core.NewAuthCollection("users"))
}

func (f *guardFixture) superuserAuth(t *testing.T) *core.Record {
	t.Helper()
	col, err := f.app.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("find superusers collection: %v", err)
	}
	return core.NewRecord(col)
}

func (f *guardFixture) newFollow(t *testing.T, follower, followee string) *core.Record {
	t.Helper()
	col, err := f.app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("follower", follower)
	r.Set("followee", followee)
	r.Set("status", "accepted")
	return r
}

// runGuard runs handler as the only binding of a hook and reports whether the
// chain reached its end (the handler called e.Next()).
func runGuard(handler func(e *core.RecordRequestEvent) error, app core.App, auth, record *core.Record) (reachedNext bool, err error) {
	e := new(core.RecordRequestEvent)
	e.RequestEvent = &core.RequestEvent{App: app, Auth: auth}
	e.Record = record

	h := &hook.Hook[*core.RecordRequestEvent]{}
	h.BindFunc(handler)
	err = h.Trigger(e, func(*core.RecordRequestEvent) error {
		reachedNext = true
		return nil
	})
	return reachedNext, err
}

func assertForbidden(t *testing.T, reachedNext bool, err error) {
	t.Helper()
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("want 403 ApiError, got %v", err)
	}
	if reachedNext {
		t.Fatal("handler must not call e.Next() when it refuses")
	}
}

func TestCreateFollowHandlerRejectsInstanceFollowForNonSuperuser(t *testing.T) {
	f := newGuardFixture(t)

	// a regular user inserting its own actor as an accepted follower of the instance actor
	rec := f.newFollow(t, f.alice.Id, f.instance.Id)
	reached, err := runGuard(CreateFollowHandler(), f.app, f.regularAuth(), rec)
	assertForbidden(t, reached, err)

	// the other direction involves the instance actor too
	rec = f.newFollow(t, f.instance.Id, f.remote.Id)
	reached, err = runGuard(CreateFollowHandler(), f.app, f.regularAuth(), rec)
	assertForbidden(t, reached, err)
}

func TestCreateFollowHandlerAllowsInstanceFollowForSuperuser(t *testing.T) {
	f := newGuardFixture(t)

	rec := f.newFollow(t, f.instance.Id, f.remote.Id)
	reached, err := runGuard(CreateFollowHandler(), f.app, f.superuserAuth(t), rec)
	if err != nil {
		t.Fatalf("superuser create: %v", err)
	}
	if !reached {
		t.Fatal("superuser create must continue the chain")
	}
}

func TestUpdateFollowRequestHandlerRejectsChangeToInstanceFollowee(t *testing.T) {
	f := newGuardFixture(t)
	stored := createFollowRecord(t, f.app, f.alice.Id, f.bob.Id, "accepted")

	rec, err := f.app.FindRecordById("follows", stored.Id)
	if err != nil {
		t.Fatalf("reload follow: %v", err)
	}
	rec.Set("followee", f.instance.Id)

	reached, err := runGuard(UpdateFollowRequestHandler(), f.app, f.regularAuth(), rec)
	assertForbidden(t, reached, err)

	// a superuser may make the same change
	reached, err = runGuard(UpdateFollowRequestHandler(), f.app, f.superuserAuth(t), rec)
	if err != nil || !reached {
		t.Fatalf("superuser update: reached=%v err=%v", reached, err)
	}
}

func TestUpdateFollowRequestHandlerRejectsExistingInstanceFollow(t *testing.T) {
	f := newGuardFixture(t)
	stored := createFollowRecord(t, f.app, f.remote.Id, f.instance.Id, "pending")

	rec, err := f.app.FindRecordById("follows", stored.Id)
	if err != nil {
		t.Fatalf("reload follow: %v", err)
	}
	// moving the row away from the instance actor is refused too
	rec.Set("followee", f.bob.Id)
	reached, err := runGuard(UpdateFollowRequestHandler(), f.app, f.regularAuth(), rec)
	assertForbidden(t, reached, err)

	rec, err = f.app.FindRecordById("follows", stored.Id)
	if err != nil {
		t.Fatalf("reload follow: %v", err)
	}
	rec.Set("status", "accepted")
	reached, err = runGuard(UpdateFollowRequestHandler(), f.app, f.regularAuth(), rec)
	assertForbidden(t, reached, err)
}

func TestFollowRequestHandlersIgnoreUserLevelFollows(t *testing.T) {
	f := newGuardFixture(t)

	rec := f.newFollow(t, f.alice.Id, f.bob.Id)
	reached, err := runGuard(UpdateFollowRequestHandler(), f.app, f.regularAuth(), rec)
	if err != nil || !reached {
		t.Fatalf("user-level update: reached=%v err=%v", reached, err)
	}

	stored := createFollowRecord(t, f.app, f.alice.Id, f.remote.Id, "pending")
	loaded, err := f.app.FindRecordById("follows", stored.Id)
	if err != nil {
		t.Fatalf("reload follow: %v", err)
	}
	loaded.Set("status", "accepted")
	reached, err = runGuard(UpdateFollowRequestHandler(), f.app, f.regularAuth(), loaded)
	if err != nil || !reached {
		t.Fatalf("user-level status update: reached=%v err=%v", reached, err)
	}
}
