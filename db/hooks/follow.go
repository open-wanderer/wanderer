package hooks

import (
	"context"
	"fmt"
	"os"
	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

// guardInstanceFollowRequest refuses records-API writes that create or change
// a follows row involving the local instance actor, unless the caller is a
// superuser. For updates the stored record is checked too.
func guardInstanceFollowRequest(e *core.RecordRequestEvent) error {
	if e.HasSuperuserAuth() {
		return nil
	}
	if isInstanceFollow(e.App, e.Record) {
		return e.ForbiddenError("only a superuser can manage instance follows", nil)
	}
	if !e.Record.IsNew() {
		if original := e.Record.Original(); original != nil && isInstanceFollow(e.App, original) {
			return e.ForbiddenError("only a superuser can manage instance follows", nil)
		}
	}
	return nil
}

// CreateFollowHandler handles OnRecordCreateRequest("follows"). Instance follows
// are superuser-only; user-level follows are delivered after the record is saved.
func CreateFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		// Instance follows are delivered by InstanceFollowCreateHandler.
		if isInstanceFollow(e.App, e.Record) {
			if err := guardInstanceFollowRequest(e); err != nil {
				return err
			}
			return e.Next()
		}
		// Don't deliver when the save failed.
		if err := e.Next(); err != nil {
			return err
		}
		federation.CreateFollowActivity(e.App, e.Record)

		return nil
	}
}

// UpdateFollowRequestHandler handles OnRecordUpdateRequest("follows") and makes
// updates involving the local instance actor superuser-only.
func UpdateFollowRequestHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if err := guardInstanceFollowRequest(e); err != nil {
			return err
		}
		return e.Next()
	}
}

func DeleteFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		// Instance follows are handled by InstanceFollowDeleteHandler.
		if isInstanceFollow(e.App, e.Record) {
			return e.Next()
		}
		federation.CreateUnfollowActivity(e.App, e.Record)

		return e.Next()
	}
}

// isOutboundInstanceFollow reports whether the local instance actor is the
// follower in the follows record.
func isOutboundInstanceFollow(app core.App, follow *core.Record) bool {
	instanceIRI := os.Getenv("ORIGIN") + "/api/v1/activitypub/instance"
	followerActor, err := app.FindRecordById("activitypub_actors", follow.GetString("follower"))
	if err != nil {
		return false
	}
	return followerActor.GetString("iri") == instanceIRI
}

// isInstanceFollow reports whether the follower or followee of the follows
// record is the local instance actor, compared by IRI.
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

// instanceFollowAction maps a status change to "accept", "reject" or "".
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

// InstanceFollowCreateHandler handles OnRecordAfterCreateSuccess("follows") and
// delivers a Follow when the local instance actor is the follower. The remote
// followee is fetched if it is not cached.
func InstanceFollowCreateHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		// Only outbound instance follows.
		if !isOutboundInstanceFollow(e.App, e.Record) {
			return e.Next()
		}

		// Make sure the remote followee is cached locally.
		followeeID := e.Record.GetString("followee")
		followeeActor, err := e.App.FindRecordById("activitypub_actors", followeeID)
		if err != nil || followeeActor == nil {
			// Missing followee: log and don't block the save.
			e.App.Logger().Error(fmt.Sprintf("instance follow create: followee actor %s not found: %v", followeeID, err))
			return e.Next()
		}

		// Refresh the followee's inbox and public key.
		followeeIRI := followeeActor.GetString("iri")
		if followeeIRI != "" {
			ctx := context.Background()
			_, refreshErr := federation.GetActorByIRI(e.App, ctx, followeeIRI, false)
			if refreshErr != nil {
				// Fall back to the stored actor record.
				e.App.Logger().Error(fmt.Sprintf("instance follow create: GetActorByIRI refresh failed: %v", refreshErr))
			}
		}

		if err := federation.CreateFollowActivity(e.App, e.Record); err != nil {
			e.App.Logger().Error(fmt.Sprintf("instance follow create: CreateFollowActivity failed: %v", err))
		}

		return e.Next()
	}
}

// InstanceFollowUpdateHandler handles OnRecordAfterUpdateSuccess("follows") and
// delivers Accept or Reject when an inbound instance follow changes to accepted
// or rejected.
func InstanceFollowUpdateHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		// Only inbound instance follows (the local instance is the followee).
		if !isInstanceFollow(e.App, e.Record) || isOutboundInstanceFollow(e.App, e.Record) {
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
// It delivers Undo{Follow} when an outbound instance follow (the local instance
// actor is the follower) is deleted; other deletions send nothing.
func InstanceFollowDeleteHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if !isOutboundInstanceFollow(e.App, e.Record) {
			return e.Next()
		}

		if err := federation.CreateUnfollowActivity(e.App, e.Record); err != nil {
			e.App.Logger().Error(fmt.Sprintf("instance follow delete: CreateUnfollowActivity failed: %v", err))
		}

		return e.Next()
	}
}
