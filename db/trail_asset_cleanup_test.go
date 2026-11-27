package main

import (
	"errors"
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

// A deleted trail takes the photos along that nothing else links to, whoever
// added them, and its waypoints and summit logs take theirs. The hooks of that
// cascade run after the commit and report their errors through the trail's
// delete, which then skips the trail's own after-delete hooks; its photos have
// to go all the same. A trail merge deletes its source inside a transaction.
func TestTrailDeleteRemovesOrphanedAssets(t *testing.T) {
	for _, tc := range []struct {
		name         string
		cascadeFails bool
		inTx         bool
	}{
		{name: "hooks succeed"},
		{name: "cascade hook fails", cascadeFails: true},
		{name: "inside a transaction", inTx: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			alice := f.localActor("alice")
			carol := f.localActor("carol")
			bob := f.save("activitypub_actors", map[string]any{"preferred_username": "bob",
				"iri": "https://b.example/api/v1/activitypub/user/bob", "is_local": false})
			trail := f.save("trails", map[string]any{"name": "Ridge", "author": alice.Id})
			other := f.save("trails", map[string]any{"name": "Valley", "author": alice.Id})
			waypoint := f.save("waypoints", map[string]any{"name": "Hut", "trail": trail.Id, "author": alice.Id})
			carolLog := f.save("summit_logs", map[string]any{"trail": trail.Id, "author": carol.Id, "date": "2026-01-01 00:00:00.000Z"})
			bobLog := f.save("summit_logs", map[string]any{"trail": trail.Id, "author": bob.Id, "date": "2026-01-02 00:00:00.000Z",
				"iri": "https://b.example/api/v1/summit-log/ridge"})
			otherLog := f.save("summit_logs", map[string]any{"trail": other.Id, "author": carol.Id, "date": "2026-01-03 00:00:00.000Z"})

			// Added by Carol, who may edit the trail.
			trailPhoto := f.photo(carol)
			waypointPhoto := f.photo(alice)
			carolPhoto := f.photo(carol)
			bobPhoto := f.federatedPhoto(bob, "log")
			sharedTrailPhoto := f.photo(alice)
			sharedLogPhoto := f.photo(carol)
			f.save("trail_assets", map[string]any{"asset": trailPhoto.Id, "trail": trail.Id})
			f.save("waypoint_assets", map[string]any{"asset": waypointPhoto.Id, "waypoint": waypoint.Id})
			f.save("summit_log_assets", map[string]any{"asset": carolPhoto.Id, "summit_log": carolLog.Id})
			f.save("summit_log_assets", map[string]any{"asset": bobPhoto.Id, "summit_log": bobLog.Id})
			f.save("trail_assets", map[string]any{"asset": sharedTrailPhoto.Id, "trail": trail.Id})
			f.save("trail_assets", map[string]any{"asset": sharedTrailPhoto.Id, "trail": other.Id})
			f.save("waypoint_assets", map[string]any{"asset": sharedLogPhoto.Id, "waypoint": waypoint.Id})
			f.save("summit_log_assets", map[string]any{"asset": sharedLogPhoto.Id, "summit_log": otherLog.Id})
			f.bindHooks()
			if tc.cascadeFails {
				f.app.OnRecordAfterDeleteSuccess("waypoints").BindFunc(func(e *core.RecordEvent) error {
					return errors.New("waypoint hook failed")
				})
			}

			var err error
			if tc.inTx {
				err = f.app.RunInTransaction(func(txApp core.App) error {
					return txApp.Delete(trail)
				})
			} else {
				err = f.app.Delete(trail)
			}
			if tc.cascadeFails != (err != nil) {
				t.Fatalf("delete error = %v", err)
			}

			if f.exists(trail) {
				t.Fatal("trail was not deleted")
			}
			for name, photo := range map[string]*core.Record{
				"trail":                trailPhoto,
				"waypoint":             waypointPhoto,
				"local summit log":     carolPhoto,
				"federated summit log": bobPhoto,
			} {
				if f.exists(photo) {
					t.Errorf("photo of the %s survived the trail's deletion", name)
				}
			}
			if !f.exists(sharedTrailPhoto) {
				t.Error("photo still linked to another trail was deleted")
			}
			if !f.exists(sharedLogPhoto) {
				t.Error("photo still linked to another trail's summit log was deleted")
			}
		})
	}
}
