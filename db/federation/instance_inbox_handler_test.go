package federation

import (
	"context"
	"database/sql"
	"errors"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// testInstanceInboxHandler stands in for the production handler, whose sender
// resolution lives in the routes package and cannot be imported here. It
// fetches a sender that is unknown or whose cached key and identity are not
// confirmed by its origin; the routes tests cover the shared resolution.
var testInstanceInboxHandler = NewInstanceInboxHandler(fetchUnconfirmedInboxSender)

func fetchUnconfirmedInboxSender(e *core.RequestEvent, body []byte, activity pub.Activity, fetchContext func() (context.Context, error)) (*core.Record, bool, error) {
	actorIRI := activity.Actor.GetID().String()
	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", actorIRI)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if actor != nil && (actor.GetBool("is_local") || CachedActorIdentityMatchesOrigin(actor)) {
		return actor, false, nil
	}
	ctx, err := fetchContext()
	if err != nil {
		return nil, false, err
	}
	actor, err = GetActorByIRI(e.App, ctx, actorIRI, false)
	if err != nil {
		return nil, false, err
	}
	if !actor.GetBool("is_local") && !CachedActorIdentityMatchesOrigin(actor) {
		return nil, false, e.UnauthorizedError("Invalid actor identity", nil)
	}
	return actor, false, nil
}
