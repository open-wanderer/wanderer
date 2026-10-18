package permissions_test

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestTrailsBoundingBoxCombinesOwnSharedAndPublicTrails(t *testing.T) {
	app := newRulesTestApp(t)
	newActor := func(name string) *core.Record {
		t.Helper()
		user := saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com",
			"user": user.Id, "is_local": true, "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
	}
	owner, friend, empty := newActor("owner"), newActor("friend"), newActor("empty")
	newTrail := func(author *core.Record, public bool, minLat, maxLat, minLon, maxLon float64) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "trails", map[string]any{
			"name": "trail", "author": author.Id, "public": public, "distance": 1,
			"min_lat": minLat, "max_lat": maxLat, "min_lon": minLon, "max_lon": maxLon,
		})
	}
	newTrail(owner, false, 10, 11, 20, 21)
	newTrail(friend, true, 46, 47, 8, 9)
	shared := newTrail(friend, false, 50, 60, -5, 0)
	saveRulesTestRecord(t, app, "trail_share", map[string]any{
		"trail": shared.Id, "actor": owner.Id, "permission": "view",
	})

	for _, test := range []struct {
		actor                          *core.Record
		minLat, maxLat, minLon, maxLon float64
	}{
		{owner, 10, 60, -5, 21},
		{friend, 46, 60, -5, 9},
		{empty, 46, 47, 8, 9},
	} {
		var got [4]float64
		err := app.DB().
			NewQuery("SELECT min_lat, max_lat, min_lon, max_lon FROM trails_bounding_box WHERE id = {:id}").
			Bind(dbx.Params{"id": test.actor.Id}).
			Row(&got[0], &got[1], &got[2], &got[3])
		if err != nil {
			t.Fatal(err)
		}
		want := [4]float64{test.minLat, test.maxLat, test.minLon, test.maxLon}
		if got != want {
			t.Errorf("%s box = %v; want %v", test.actor.GetString("username"), got, want)
		}
	}
}
