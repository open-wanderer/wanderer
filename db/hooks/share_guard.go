package hooks

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// Private trails and lists may only be shared with actors on this instance.
// The share dialog in the web frontend already refuses a cross-instance share
// of a private object, but that check lives in the browser only; the
// trail_share / list_share endpoints and the announce federation accept the
// record regardless. The receiving instance would then store the private
// object as a public one. This guard enforces the rule on the server.

// crossInstanceShareForbidden reports whether sharing object with actor must
// be rejected: the actor lives on another instance and the object is private.
func crossInstanceShareForbidden(object, actor *core.Record) bool {
	return !actor.GetBool("is_local") && !object.GetBool("public")
}

// ensureShareAllowed rejects a share request of a private object with a remote
// actor. objectCollection is "trails" or "lists", objectField the share's
// relation field ("trail" or "list"). Unknown references are left to the
// regular record validation.
func ensureShareAllowed(e *core.RecordRequestEvent, objectCollection, objectField string) error {
	object, err := e.App.FindRecordById(objectCollection, e.Record.GetString(objectField))
	if err != nil {
		return nil
	}
	actor, err := e.App.FindRecordById("activitypub_actors", e.Record.GetString("actor"))
	if err != nil {
		return nil
	}
	if crossInstanceShareForbidden(object, actor) {
		return e.BadRequestError(
			fmt.Sprintf("A %s must be public to be shared with users on other instances.", objectField),
			nil,
		)
	}
	return nil
}
