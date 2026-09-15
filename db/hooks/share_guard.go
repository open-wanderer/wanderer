package hooks

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// Two rules protect private trails and lists from being exposed through
// shares. Both are enforced here, in request hooks, because the collection
// rules cannot express them:
//
//  1. A share stays bound to its target. PocketBase checks update rules
//     against the stored record before loading request data, so changing the
//     target could grant access to another owner's private trail or list.
//  2. Private objects may only be shared with actors on this instance. The
//     share dialog in the web frontend already refuses a cross-instance share
//     of a private object, but that check lives in the browser only; the
//     trail_share / list_share endpoints and the announce federation accept
//     the record regardless. The receiving instance would then store the
//     private object as a public one.

// UpdateShareHandler rejects an update that changes the share's target or
// hands a private object to a remote actor. objectCollection is "trails" or
// "lists", objectField the share's relation field ("trail" or "list").
// Unlike creation, updating a share does not announce it.
func UpdateShareHandler(objectCollection, objectField string) func(*core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if e.Record.GetString(objectField) != e.Record.Original().GetString(objectField) {
			return e.BadRequestError(fmt.Sprintf("The %s of an existing share cannot be changed.", objectField), nil)
		}
		if err := ensureShareAllowed(e, objectCollection, objectField); err != nil {
			return err
		}
		return e.Next()
	}
}

// ensureShareAllowed rejects a share request of a private object with a remote
// actor. Link shares carry no actor and unknown references are left to the
// regular record validation.
func ensureShareAllowed(e *core.RecordRequestEvent, objectCollection, objectField string) error {
	actorId := e.Record.GetString("actor")
	if actorId == "" {
		return nil
	}
	object, err := e.App.FindRecordById(objectCollection, e.Record.GetString(objectField))
	if err != nil {
		return nil
	}
	actor, err := e.App.FindRecordById("activitypub_actors", actorId)
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

// crossInstanceShareForbidden reports whether sharing object with actor must
// be rejected: the actor lives on another instance and the object is private.
func crossInstanceShareForbidden(object, actor *core.Record) bool {
	return !actor.GetBool("is_local") && !object.GetBool("public")
}
