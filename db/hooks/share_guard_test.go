package hooks

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestCrossInstanceShareForbidden(t *testing.T) {
	record := func(collection, field string, value bool) *core.Record {
		r := core.NewRecord(core.NewBaseCollection(collection))
		r.Set(field, value)
		return r
	}

	cases := []struct {
		name   string
		public bool
		local  bool
		want   bool
	}{
		{"private object, remote actor", false, false, true},
		{"public object, remote actor", true, false, false},
		{"private object, local actor", false, true, false},
		{"public object, local actor", true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			object := record("trails", "public", tc.public)
			actor := record("activitypub_actors", "is_local", tc.local)
			if got := crossInstanceShareForbidden(object, actor); got != tc.want {
				t.Fatalf("crossInstanceShareForbidden = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnsureShareAllowed(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.BoolField{Name: "is_local"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.BoolField{Name: "public"})
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(shares); err != nil {
		t.Fatal(err)
	}

	save := func(collection *core.Collection, field string, value bool) *core.Record {
		t.Helper()
		r := core.NewRecord(collection)
		r.Set(field, value)
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	localActor := save(actors, "is_local", true)
	remoteActor := save(actors, "is_local", false)
	publicTrail := save(trails, "public", true)
	privateTrail := save(trails, "public", false)

	shareEvent := func(trailID, actorID string) *core.RecordRequestEvent {
		share := core.NewRecord(shares)
		share.Set("trail", trailID)
		share.Set("actor", actorID)
		e := &core.RecordRequestEvent{RequestEvent: &core.RequestEvent{}}
		e.App = app
		e.Request = httptest.NewRequest("POST", "/api/collections/trail_share/records", nil)
		e.Response = httptest.NewRecorder()
		e.Collection = shares
		e.Record = share
		return e
	}

	t.Run("private trail with remote actor is rejected with 400", func(t *testing.T) {
		err := ensureShareAllowed(shareEvent(privateTrail.Id, remoteActor.Id), "trails", "trail")
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) || apiErr.Status != 400 {
			t.Fatalf("expected 400 ApiError, got %v", err)
		}
	})

	for _, tc := range []struct {
		name    string
		trailID string
		actorID string
	}{
		{"public trail with remote actor", publicTrail.Id, remoteActor.Id},
		{"private trail with local actor", privateTrail.Id, localActor.Id},
		{"unknown trail is left to record validation", "missing00000000", remoteActor.Id},
		{"unknown actor is left to record validation", privateTrail.Id, "missing00000000"},
	} {
		t.Run(tc.name+" passes", func(t *testing.T) {
			if err := ensureShareAllowed(shareEvent(tc.trailID, tc.actorID), "trails", "trail"); err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}
