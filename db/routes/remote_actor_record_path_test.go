package routes

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestGoneActorPreservesOwnRecordPaths(t *testing.T) {
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
				f := setupHandleLookupTest(t, "https://local.example", http.StatusGone)
				for _, name := range []string{"trails", "lists", "comments", "summit_logs", "waypoints"} {
					c, err := f.app.FindCollectionByNameOrId(name)
					if err != nil {
						c = core.NewBaseCollection(name)
						c.Fields.Add(&core.TextField{Name: "iri"})
					}
					c.Fields.Add(&core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1, CascadeDelete: true})
					if err := f.app.Save(c); err != nil {
						t.Fatal(err)
					}
				}
				actor := f.remoteActor(t, "alice", "remote.example")
				record := f.record(t, collection, map[string]any{"author": actor.Id})
				id := record.Id
				if !state.own {
					id = "remote000000001"
				}
				record.Set("iri", state.origin+"/api/v1/"+kind+"/"+id)
				if err := f.app.Save(record); err != nil {
					t.Fatal(err)
				}
				iri := pub.IRI(actor.GetString("iri"))
				gone, err := reconcileGoneInboxActor(f.app, context.Background(), actor, pub.Activity{Type: pub.DeleteType, Actor: iri, Object: iri}, nil)
				preserve := state.own
				if !gone || (err != nil) != preserve {
					t.Fatalf("reconciliation gone=%v error=%v, preserve=%v", gone, err, preserve)
				}
				if requests := f.transport.seen(); len(requests) != 1 || requests[0] != iri.String() {
					t.Fatalf("origin probes=%v, want only %s", requests, iri)
				}
				for _, stored := range []*core.Record{actor, record} {
					_, err := f.app.FindRecordById(stored.Collection(), stored.Id)
					if (err == nil) != preserve {
						t.Fatalf("%s preserved=%v, want %v", stored.Collection().Name, err == nil, preserve)
					}
				}
			})
		}
	}
}

func TestGoneActorPreservesMixedRecordPaths(t *testing.T) {
	for collection, kind := range map[string]string{"trails": "trail", "lists": "list", "comments": "comment", "summit_logs": "summit-log", "waypoints": "waypoint"} {
		for ownPosition := 0; ownPosition < 3; ownPosition++ {
			t.Run(fmt.Sprintf("%s/own-at-%d", collection, ownPosition), func(t *testing.T) {
				f := setupHandleLookupTest(t, "https://local.example", http.StatusGone)
				for _, name := range []string{"trails", "lists", "comments", "summit_logs", "waypoints"} {
					c, err := f.app.FindCollectionByNameOrId(name)
					if err != nil {
						c = core.NewBaseCollection(name)
						c.Fields.Add(&core.TextField{Name: "iri"})
					}
					c.Fields.Add(&core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1, CascadeDelete: true})
					if err := f.app.Save(c); err != nil {
						t.Fatal(err)
					}
				}
				actor := f.remoteActor(t, "alice", "remote.example")
				for i := 0; i < 3; i++ {
					f.record(t, collection, map[string]any{"author": actor.Id})
				}
				// Place the own record in each position of the guard's actual
				// query order so checking only its first result cannot pass.
				records, err := f.app.FindRecordsByFilter(collection, "author={:actor}", "", -1, 0, dbx.Params{"actor": actor.Id})
				if err != nil || len(records) != 3 {
					t.Fatalf("records=%v error=%v", records, err)
				}
				for i, r := range records {
					id := fmt.Sprintf("remote%09d", i)
					if i == ownPosition {
						id = r.Id
					}
					r.Set("iri", "https://earlier.example/api/v1/"+kind+"/"+id)
					if err := f.app.Save(r); err != nil {
						t.Fatal(err)
					}
				}
				ordered, err := f.app.FindRecordsByFilter(collection, "author={:actor}", "", -1, 0, dbx.Params{"actor": actor.Id})
				if err != nil || len(ordered) != 3 {
					t.Fatalf("records=%v error=%v", ordered, err)
				}
				for i, r := range ordered {
					if r.Id != records[i].Id {
						t.Fatal("record order changed before deletion")
					}
				}
				iri := pub.IRI(actor.GetString("iri"))
				gone, err := reconcileGoneInboxActor(f.app, context.Background(), actor, pub.Activity{Type: pub.DeleteType, Actor: iri, Object: iri}, nil)
				if !gone || err == nil {
					t.Fatalf("mixed record deletion gone=%v error=%v", gone, err)
				}
				if requests := f.transport.seen(); len(requests) != 1 || requests[0] != iri.String() {
					t.Fatalf("origin probes=%v, want only %s", requests, iri)
				}
				for _, r := range append(records, actor) {
					if _, err := f.app.FindRecordById(r.Collection(), r.Id); err != nil {
						t.Fatalf("%s %s disappeared: %v", r.Collection().Name, r.Id, err)
					}
				}
			})
		}
	}
}
