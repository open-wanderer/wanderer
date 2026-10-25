package main

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// The instance actor federates the instance and is not a profile, so neither
// the startup rebuild nor the nightly repair may write it to the actors index.
func TestActorSearchSourceSkipsInstanceActor(t *testing.T) {
	app := newSearchInitApp(t)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "actor_type"},
		&core.BoolField{Name: "is_local"},
	)

	person := core.NewRecord(actors)
	person.Id = "person"
	person.Set("preferred_username", "instance")
	person.Set("actor_type", "person")
	instance := core.NewRecord(actors)
	instance.Id = "instance"
	instance.Set("preferred_username", "instance")
	instance.Set("actor_type", "instance")
	instance.Set("is_local", true)

	var source searchIndexSource
	for _, s := range searchIndexSources(app) {
		if s.index == "actors" {
			source = s
		}
	}

	documents := searchDocuments(app, source, []*core.Record{person, instance})
	if len(documents) != 1 || documents[0]["id"] != person.Id {
		t.Fatalf("documents = %v; want only the person actor", documents)
	}
}
