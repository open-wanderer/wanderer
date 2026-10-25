package federation

import (
	"fmt"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestActorDeletionPreservesOwnRecordPaths(t *testing.T) {
	for collection, kind := range map[string]string{"trails": "trail", "lists": "list", "comments": "comment", "summit_logs": "summit-log", "waypoints": "waypoint"} {
		for _, state := range []struct {
			name, origin string
			own          bool
		}{
			{"current", "https://local.example", true},
			{"earlier", "https://earlier.example", true},
			{"actor-origin", "https://remote.example", true},
			{"remote-copy", "https://remote.example", false},
			{"current-copy", "https://local.example", false},
		} {
			t.Run(collection+"/"+state.name, func(t *testing.T) {
				app := setupActorDeleteTestApp(t)
				actor := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/alice", false)
				c, _ := app.FindCollectionByNameOrId(collection)
				record := core.NewRecord(c)
				record.Set("author", actor.Id)
				if err := app.Save(record); err != nil {
					t.Fatal(err)
				}
				id := record.Id
				if !state.own {
					id = "remote000000001"
				}
				record.Set("iri", state.origin+"/api/v1/"+kind+"/"+id)
				if err := app.Save(record); err != nil {
					t.Fatal(err)
				}
				err := processDeleteActorActivity(app, actor, newDeleteActivity(actor.GetString("iri"), actor.GetString("iri")))
				preserve := state.own
				if (err != nil) != preserve {
					t.Fatalf("deletion error=%v, preserve=%v", err, preserve)
				}
				for _, stored := range []*core.Record{actor, record} {
					_, err := app.FindRecordById(stored.Collection(), stored.Id)
					if (err == nil) != preserve {
						t.Fatalf("%s preserved=%v, want %v", stored.Collection().Name, err == nil, preserve)
					}
				}
			})
		}
	}
}

func TestActorDeletionPreservesMixedRecordPaths(t *testing.T) {
	for collection, kind := range map[string]string{"trails": "trail", "lists": "list", "comments": "comment", "summit_logs": "summit-log", "waypoints": "waypoint"} {
		for ownPosition := 0; ownPosition < 3; ownPosition++ {
			t.Run(fmt.Sprintf("%s/own-at-%d", collection, ownPosition), func(t *testing.T) {
				app := setupActorDeleteTestApp(t)
				actor := newTestActor(t, app, "https://remote.example/api/v1/activitypub/user/alice", false)
				c, err := app.FindCollectionByNameOrId(collection)
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 3; i++ {
					r := core.NewRecord(c)
					r.Set("author", actor.Id)
					if err := app.Save(r); err != nil {
						t.Fatal(err)
					}
				}
				// Place the own record in each position of the guard's actual
				// query order so checking only its first result cannot pass.
				records, err := app.FindRecordsByFilter(collection, "author={:actor}", "", -1, 0, dbx.Params{"actor": actor.Id})
				if err != nil || len(records) != 3 {
					t.Fatalf("records=%v error=%v", records, err)
				}
				for i, r := range records {
					id := fmt.Sprintf("remote%09d", i)
					if i == ownPosition {
						id = r.Id
					}
					r.Set("iri", "https://earlier.example/api/v1/"+kind+"/"+id)
					if err := app.Save(r); err != nil {
						t.Fatal(err)
					}
				}
				ordered, err := app.FindRecordsByFilter(collection, "author={:actor}", "", -1, 0, dbx.Params{"actor": actor.Id})
				if err != nil || len(ordered) != 3 {
					t.Fatalf("records=%v error=%v", ordered, err)
				}
				for i, r := range ordered {
					if r.Id != records[i].Id {
						t.Fatal("record order changed before deletion")
					}
				}
				if err := processDeleteActorActivity(app, actor, newDeleteActivity(actor.GetString("iri"), actor.GetString("iri"))); err == nil {
					t.Fatal("actor deletion accepted with mixed record paths")
				}
				for _, r := range append(records, actor) {
					if _, err := app.FindRecordById(r.Collection(), r.Id); err != nil {
						t.Fatalf("%s %s disappeared: %v", r.Collection().Name, r.Id, err)
					}
				}
			})
		}
	}
}
