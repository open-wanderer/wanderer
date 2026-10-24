package federation

import (
	"fmt"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ProcessInboxActivity dispatches an activity after its sender has been
// authenticated by the HTTP inbox or by the local delivery binding.
func ProcessInboxActivity(app core.App, actor, recipient *core.Record, activity pub.Activity) error {
	return ProcessReceivedInboxActivity(app, actor, recipient, activity, time.Now())
}

// ProcessReceivedInboxActivity dispatches a delivery using the time it reached
// the inbox, before any wait for the sender's key.
func ProcessReceivedInboxActivity(app core.App, actor, recipient *core.Record, activity pub.Activity, received time.Time) error {
	if actor != nil && !actor.GetBool("is_local") {
		return processOrderedInboxActivity(app, actor, recipient, activity, received)
	}
	return dispatchInboxActivity(app, actor, recipient, activity)
}

func dispatchInboxActivity(app core.App, actor, recipient *core.Record, activity pub.Activity) error {
	switch activity.Type {
	case pub.FollowType:
		return ProcessFollowActivity(app, actor, activity)
	case pub.AcceptType:
		return ProcessAcceptActivity(app, actor, activity)
	case pub.UndoType:
		return ProcessUndoActivity(app, actor, activity)
	case pub.UpdateType, pub.CreateType:
		return ProcessCreateOrUpdateActivity(app, actor, recipient, activity)
	case pub.DeleteType:
		return ProcessDeleteActivity(app, actor, activity)
	case pub.AnnounceType:
		return ProcessAnnounceActivity(app, actor, activity)
	case pub.LikeType:
		return ProcessLikeActivity(app, actor, activity)
	}
	return nil
}

func deliverLocalActivity(app core.App, sender *core.Record, inbox string, body []byte) (bool, error) {
	recipient, err := app.FindFirstRecordByFilter("activitypub_actors", "is_local=true && inbox={:inbox}", dbx.Params{"inbox": inbox})
	if err != nil {
		return false, nil
	}
	// A caller-supplied actor object cannot establish local sender identity.
	stored, err := app.FindRecordById("activitypub_actors", sender.Id)
	if err != nil || !stored.GetBool("is_local") || stored.GetString("iri") != sender.GetString("iri") || stored.GetString("private_key") != sender.GetString("private_key") {
		return true, fmt.Errorf("local delivery sender is not bound to a local account")
	}
	activity, err := UnmarshalInboxActivity(body)
	if err != nil {
		return true, err
	}
	if activity.Actor == nil || activity.Actor.GetID().String() != stored.GetString("iri") {
		return true, fmt.Errorf("local delivery actor does not match its sender")
	}
	return true, ProcessInboxActivity(app, stored, recipient, activity)
}
