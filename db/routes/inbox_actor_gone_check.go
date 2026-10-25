package routes

import (
	"context"
	"errors"
	"net/http"
	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// A request that fetched a missing actor may confirm a deletion. A retained-key
// deletion can also confirm the origin after waiting for a failed refresh or
// after an earlier request used the refresh claim, once its signature is
// verified. All paths share one confirmation claim for the full window.
func inboxActorGoneCheckAllowed(app core.App, ctx context.Context, request *http.Request, body []byte, actor *core.Record, activity pub.Activity, err error, known bool) bool {
	if actor == nil || activity.Type != pub.DeleteType || activity.Actor == nil || activity.Object == nil || activity.Actor.GetID().String() != actor.GetString("iri") || activity.Object.GetID().String() != actor.GetString("iri") || ctx.Err() != nil {
		return false
	}
	missing := errors.Is(err, federation.ErrActorNotFound)
	if !known {
		return missing
	}
	var reloadErr *knownInboxActorReloadError
	if !errors.As(err, &reloadErr) {
		return false
	}
	fetchedMissing := reloadErr.fetched && missing
	if !fetchedMissing && !reloadErr.waited && !(reloadErr.claimRefused && errors.Is(err, util.ErrRateLimited)) {
		return false
	}
	// The retained-key exception and wanderer accounts require authentication
	// before claiming confirmation work. Other software keeps its existing
	// permanent-gone confirmation after an actual missing actor response.
	if !fetchedMissing || util.ObjectKindFromIRI(actor.GetString("iri")) == util.ObjectKindActor {
		if !legacySelfDeletionSigned(app, request, body, actor) {
			return false
		}
	}
	bucket := "known-actor-confirmation:" + app.DataDir() + ":" + actor.Id
	return util.KnownActorKeyReloads.CheckRateLimit(bucket, "") == nil
}
