package routes

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteListReferencesReadableLocalTrails(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "actor"}, &core.TextField{Name: "permission"})
	if err := f.app.Save(shares); err != nil {
		t.Fatal(err)
	}
	private := f.save(f.trails, map[string]any{"iri": "https://historical.example/api/v1/trail/private", "author": f.localActor.Id, "name": "Private", "public": false})
	foreignAuthor := f.save(f.actors, map[string]any{"iri": "https://foreign.example/api/v1/actor/bob"})
	foreign := f.save(f.trails, map[string]any{"iri": "https://foreign.example/api/v1/trail/private", "author": foreignAuthor.Id, "name": "Foreign private", "public": false})
	public := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/public", "author": f.localActor.Id, "name": "Public", "public": true})
	shared := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/shared", "author": f.localActor.Id, "name": "Shared", "public": false})
	f.save(shares, map[string]any{"trail": shared.Id, "actor": f.remoteActor.Id, "permission": "view"})
	list := f.save(f.lists, map[string]any{"author": f.remoteActor.Id})
	raw := []any{}
	for _, trail := range []*core.Record{private, foreign, public, shared} {
		// The source claims every trail is public. Only local state and
		// the list author's actual share determine reference visibility.
		raw = append(raw, map[string]any{"id": "copy", "iri": trail.GetString("iri"), "public": true})
	}
	if err := syncTrails(f.app, t.Context(), list, "https://remote.example", raw); err != nil {
		t.Fatal(err)
	}
	if got := list.GetStringSlice("trails"); !slices.Equal(got, []string{public.Id, shared.Id}) {
		t.Fatalf("private trail included in list relations: %v", got)
	}
	if err := f.app.Delete(f.mustFindShare(t, shares.Id, shared.Id)); err != nil {
		t.Fatal(err)
	}
	if err := syncTrails(f.app, t.Context(), list, "https://remote.example", raw); err != nil {
		t.Fatal(err)
	}
	if got := list.GetStringSlice("trails"); !slices.Equal(got, []string{public.Id}) {
		t.Fatalf("revoked share survived list resynchronization: %v", got)
	}
}

func (f *remoteSyncFixture) mustFindShare(t *testing.T, collection, trail string) *core.Record {
	t.Helper()
	record, err := f.app.FindFirstRecordByData(collection, "trail", trail)
	if err != nil {
		t.Fatal(err)
	}
	return record
}
