package util

import "github.com/pocketbase/pocketbase/core"

// IsInstanceActor reports whether an activitypub_actors record is an instance
// (Application) actor. Instance actors are excluded from search.
func IsInstanceActor(r *core.Record) bool {
	return r != nil && r.GetString("actor_type") == "instance"
}

// IsLocalInstanceActor reports whether an activitypub_actors record is this
// instance's own actor: an instance actor with is_local set. It recognises the
// actor on a loaded record whatever its IRI, which follows ORIGIN and can be stale.
func IsLocalInstanceActor(r *core.Record) bool {
	return IsInstanceActor(r) && r.GetBool("is_local")
}
