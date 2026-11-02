package util

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestIsInstanceActor(t *testing.T) {
	collection := core.NewBaseCollection("activitypub_actors")
	collection.Fields.Add(&core.TextField{Name: "actor_type"})

	build := func(actorType string) *core.Record {
		r := core.NewRecord(collection)
		r.Set("actor_type", actorType)
		return r
	}

	cases := []struct {
		name   string
		record *core.Record
		want   bool
	}{
		{"nil record", nil, false},
		{"instance", build("instance"), true},
		{"person", build("person"), false},
		{"empty", build(""), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsInstanceActor(tc.record); got != tc.want {
				t.Fatalf("IsInstanceActor = %v; want %v", got, tc.want)
			}
		})
	}
}
