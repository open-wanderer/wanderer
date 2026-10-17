package federation

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// A share's Announce embeds the whole trail. For a private trail it must not
// show up in the public listing of activitypub_activities.
func TestAnnounceOfPrivateTrailIsHidden(t *testing.T) {
	for _, public := range []bool{false, true} {
		t.Run(map[bool]string{false: "private", true: "public"}[public], func(t *testing.T) {
			f := setupAddressingTestApp(t)
			activities, err := f.app.FindCollectionByNameOrId("activitypub_activities")
			if err != nil {
				t.Fatal(err)
			}
			activities.Fields.Add(&core.BoolField{Name: "hidden"})
			if err := f.app.Save(activities); err != nil {
				t.Fatal(err)
			}
			actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
			if err != nil {
				t.Fatal(err)
			}
			trails, err := f.app.FindCollectionByNameOrId("trails")
			if err != nil {
				t.Fatal(err)
			}
			shares := core.NewBaseCollection("trail_share")
			shares.Fields.Add(
				&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
				&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
				&core.TextField{Name: "permission"},
			)
			if err := f.app.Save(shares); err != nil {
				t.Fatal(err)
			}

			author := f.actor(t, "author", true)
			sharee := f.actor(t, "sharee", false)
			trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": public, "name": "Trail"}, "trail")
			share := f.record(t, "trail_share", map[string]any{"actor": sharee.Id, "trail": trail.Id, "permission": "view"})

			if err := CreateAnnounceActivity(f.app, share, TrailAnnounceType); err != nil {
				t.Fatal(err)
			}

			announce, err := f.app.FindFirstRecordByData("activitypub_activities", "type", "Announce")
			if err != nil {
				t.Fatal(err)
			}
			if got := announce.GetBool("hidden"); got != !public {
				t.Fatalf("hidden = %v for a trail with public = %v", got, public)
			}
		})
	}
}
