package main

import (
	"encoding/json"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestActorFollowCountStatusFieldsStayInternal(t *testing.T) {
	app := newRealSchemaApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"follower_count_pending", "following_count_pending"}
	for _, name := range names {
		field, ok := actors.Fields.GetByName(name).(*core.BoolField)
		if !ok || !field.Hidden {
			t.Fatalf("actor count status field %q must be a hidden boolean, got %#v", name, actors.Fields.GetByName(name))
		}
	}

	actor := core.NewRecord(actors)
	actor.Set("iri", "https://remote.example/users/schema-check")
	actor.Set("inbox", "https://remote.example/users/schema-check/inbox")
	actor.Set("public_key", "schema-public-key")
	actor.Set("preferred_username", "schema-check")
	actor.Set("username", "Schema Check")
	for _, name := range names {
		actor.Set(name, true)
	}
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	stored, err := app.FindRecordById(actors, actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if !stored.GetBool(name) {
			t.Fatalf("actor count status field %q did not persist its true value", name)
		}
	}

	data, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	var public map[string]any
	if err := json.Unmarshal(data, &public); err != nil {
		t.Fatal(err)
	}
	if public["iri"] != actor.GetString("iri") || public["id"] != actor.Id {
		t.Fatalf("actor JSON lost its public identity: %s", data)
	}
	for _, name := range names {
		if _, exists := public[name]; exists {
			t.Fatalf("actor JSON exposed internal count status %q: %s", name, data)
		}
	}
}
