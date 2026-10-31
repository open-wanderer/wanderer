package util

import "github.com/pocketbase/pocketbase/core"

// IsInstanceActor reports whether an activitypub_actors record is an instance
// (Application) actor. Instance actors are excluded from search.
func IsInstanceActor(r *core.Record) bool {
	return r != nil && r.GetString("actor_type") == "instance"
}
