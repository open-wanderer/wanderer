package hooks

import (
	"context"
	"fmt"
	"os"
	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

func CreateFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		// CR-01: skip user-level delivery for instance follows; the AfterSuccess
		// handler (InstanceFollowCreateHandler) is the single delivery path.
		if isInstanceFollow(e.App, e.Record) {
			return e.Next()
		}
		e.Next()
		federation.CreateFollowActivity(e.App, e.Record)

		return nil
	}
}

func DeleteFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		// CR-02: skip user-level delivery for instance follows; the AfterSuccess
		// handler (InstanceFollowDeleteHandler) is the single delivery path.
		if isInstanceFollow(e.App, e.Record) {
			return e.Next()
		}
		federation.CreateUnfollowActivity(e.App, e.Record)

		return e.Next()
	}
}

// isOutboundInstanceFollow returns true only when the local instance actor is the
// FOLLOWER in the given follows record. This distinguishes admin-initiated outbound
// follows (where the local instance should sign and deliver a Follow activity) from
// inbound follows saved by ProcessFollowActivity (where the instance is the followee
// and no outgoing Follow should be sent — CR-03 fix).
func isOutboundInstanceFollow(app core.App, follow *core.Record) bool {
	instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
	followerActor, err := app.FindRecordById("activitypub_actors", follow.GetString("follower"))
	if err != nil {
		return false
	}
	return followerActor.GetString("iri") == instanceIRI
}

// isInstanceFollow returns true when either the follower or followee actor in the
// given follows record is the local instance actor (identified by IRI comparison
// against ORIGIN + "/api/v1/activitypub/instance"). IRI comparison is used because
// remote actors fetched via GetActorByIRI do not have actor_type populated locally.
func isInstanceFollow(app core.App, follow *core.Record) bool {
	instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
	for _, field := range []string{"follower", "followee"} {
		actor, err := app.FindRecordById("activitypub_actors", follow.GetString(field))
		if err != nil {
			continue
		}
		if actor.GetString("iri") == instanceIRI {
			return true
		}
	}
	return false
}

// instanceFollowAction maps an (oldStatus, newStatus) pair to a delivery action:
// "accept", "reject", or "" (no-op). This pure helper makes the status-transition
// logic unit-testable without binding a live PocketBase hook.
func instanceFollowAction(oldStatus, newStatus string) string {
	if oldStatus == newStatus {
		return ""
	}
	switch newStatus {
	case "accepted":
		return "accept"
	case "rejected":
		return "reject"
	}
	return ""
}

// InstanceFollowCreateHandler handles OnRecordAfterCreateSuccess("follows").
// When the admin creates a follows record with follower=instance actor, this handler
// delivers an outgoing Follow activity to the remote instance's inbox (FLCL-01).
// It auto-fetches the remote followee actor if not yet cached locally, so the admin
// only needs to supply the remote IRI — no pre-created actor record required.
func InstanceFollowCreateHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		// CR-03: use the directional check — only fire when the local instance actor
		// is the follower (admin-initiated outbound follow). Do NOT fire for inbound
		// follows saved by ProcessFollowActivity where the instance is the followee.
		if !isOutboundInstanceFollow(e.App, e.Record) {
			return e.Next()
		}

		// Ensure the remote followee actor is cached locally. If not found by ID,
		// attempt to resolve it from the stored IRI via a remote fetch.
		followeeID := e.Record.GetString("followee")
		followeeActor, err := e.App.FindRecordById("activitypub_actors", followeeID)
		if err != nil || followeeActor == nil {
			// The followee actor record is missing locally — log and continue so
			// the hook does not block the save.
			e.App.Logger().Error(fmt.Sprintf("instance follow create: followee actor %s not found: %v", followeeID, err))
			return e.Next()
		}

		// Re-fetch the followee via GetActorByIRI to ensure its remote fields
		// (inbox, public key) are up to date before sending the Follow activity.
		followeeIRI := followeeActor.GetString("iri")
		if followeeIRI != "" {
			ctx := context.Background()
			_, refreshErr := federation.GetActorByIRI(e.App, ctx, followeeIRI, false)
			if refreshErr != nil {
				// Non-fatal: proceed with the locally-stored actor record.
				e.App.Logger().Error(fmt.Sprintf("instance follow create: GetActorByIRI refresh failed: %v", refreshErr))
			}
		}

		if err := federation.CreateFollowActivity(e.App, e.Record); err != nil {
			e.App.Logger().Error(fmt.Sprintf("instance follow create: CreateFollowActivity failed: %v", err))
		}

		return e.Next()
	}
}

// InstanceFollowUpdateHandler handles OnRecordAfterUpdateSuccess("follows").
// When the admin changes a pending instance follow to status=accepted it delivers
// Accept{Follow} (FLCL-03); to status=rejected it delivers Reject{Follow} (FLCL-04).
// The handler is a no-op for user-level follows and for status-unchanged updates.
func InstanceFollowUpdateHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if !isInstanceFollow(e.App, e.Record) {
			return e.Next()
		}

		newStatus := e.Record.GetString("status")
		oldStatus := e.Record.Original().GetString("status")

		action := instanceFollowAction(oldStatus, newStatus)
		switch action {
		case "accept":
			if err := federation.CreateAcceptFollowActivity(e.App, e.Record); err != nil {
				e.App.Logger().Error(fmt.Sprintf("instance follow update: CreateAcceptFollowActivity failed: %v", err))
			}
		case "reject":
			if err := federation.CreateRejectFollowActivity(e.App, e.Record); err != nil {
				e.App.Logger().Error(fmt.Sprintf("instance follow update: CreateRejectFollowActivity failed: %v", err))
			}
		}

		return e.Next()
	}
}

// InstanceFollowDeleteHandler handles OnRecordAfterDeleteSuccess("follows").
// When the admin deletes an instance follow record, this handler delivers
// Undo{Follow} to the remote instance (FLCL-05).
func InstanceFollowDeleteHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if !isInstanceFollow(e.App, e.Record) {
			return e.Next()
		}

		if err := federation.CreateUnfollowActivity(e.App, e.Record); err != nil {
			e.App.Logger().Error(fmt.Sprintf("instance follow delete: CreateUnfollowActivity failed: %v", err))
		}

		return e.Next()
	}
}
