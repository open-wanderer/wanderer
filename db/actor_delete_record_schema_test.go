package main

import (
	"fmt"
	"testing"

	"pocketbase/federation"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestActorCleanupPreservesOwnRecordsInMigratedSchema(t *testing.T) {
	for _, origin := range []string{"https://example.com", "https://earlier.example", "https://remote.example"} {
		t.Run(origin, func(t *testing.T) {
			app := newRealSchemaApp(t)
			local := saveInboundRecord(t, app, "activitypub_actors", map[string]any{
				"username": "Local", "preferred_username": "local", "is_local": true,
				"iri": "https://example.com/api/v1/activitypub/user/local", "inbox": "https://example.com/inbox", "public_key": "local-key",
			})
			for i, collection := range []string{"trails", "lists", "comments", "summit_logs", "waypoints"} {
				for _, own := range []bool{true, false} {
					t.Run(fmt.Sprintf("%s/own=%t", collection, own), func(t *testing.T) {
						name := fmt.Sprintf("remote%d%t", i, own)
						actor := saveInboundRecord(t, app, "activitypub_actors", map[string]any{
							"username": name, "preferred_username": name, "iri": "https://remote.example/api/v1/activitypub/user/" + name,
							"inbox": "https://remote.example/inbox", "public_key": "remote-key",
						})
						parent := saveInboundRecord(t, app, "trails", map[string]any{"author": local.Id, "name": "Parent", "iri": "https://example.com/api/v1/trail/parent-" + name})
						initialAuthor := actor.Id
						if own {
							initialAuthor = local.Id
						}
						record := saveInboundRecord(t, app, collection, map[string]any{"author": initialAuthor, "trail": parent.Id, "name": "Own record", "text": "Own entry"})
						kind := map[string]string{"trails": "trail", "lists": "list", "comments": "comment", "summit_logs": "summit-log", "waypoints": "waypoint"}[collection]
						id := "remote000000001"
						if own {
							id = record.Id
						}
						record.Set("iri", origin+"/api/v1/"+kind+"/"+id)
						record.Set("author", actor.Id)
						if err := app.SaveNoValidate(record); err != nil {
							t.Fatal(err)
						}
						author, ok := record.Collection().Fields.GetByName("author").(*core.RelationField)
						if !ok || author.CollectionId != actor.Collection().Id || !author.CascadeDelete {
							t.Fatal("migrated record author must cascade from the actor")
						}
						iri := pub.IRI(actor.GetString("iri"))
						err := federation.ProcessDeleteActivity(app, actor, pub.Activity{Type: pub.DeleteType, Actor: iri, Object: iri})
						if (err != nil) != own {
							t.Fatalf("cleanup error=%v, preserve=%t", err, own)
						}
						for _, stored := range []*core.Record{actor, record} {
							_, err := app.FindRecordById(stored.Collection(), stored.Id)
							if (err == nil) != own {
								t.Fatalf("%s preserved=%t, want %t", stored.Collection().Name, err == nil, own)
							}
						}
						if _, err := app.FindRecordById("trails", parent.Id); err != nil {
							t.Fatalf("local parent disappeared: %v", err)
						}
					})
				}
			}
		})
	}
}
