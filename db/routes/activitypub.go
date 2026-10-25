package routes

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"pocketbase/federation"
	"pocketbase/util"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/sync/singleflight"
)

// inboxPathPattern matches the canonical local ActivityPub inbox path that the
// trusted SvelteKit frontend forwards via X-Forwarded-Path. Anything else is
// rejected before it can influence recipient lookup or signature verification.
//
// The username segment mirrors the users collection (pattern `^[\w][\w.\-]*$`,
// 3-150 characters, ASCII), lowercased the same way ActorFromUser lowercases it
// when minting the actor's inbox IRI. Dots and hyphens are therefore legal in a
// local username and must be accepted here; because the first character class
// excludes them, "." and ".." remain unmatchable.
var inboxPathPattern = regexp.MustCompile(`^/api/v1/activitypub/user/[a-z0-9_][a-z0-9_.-]{2,149}/inbox$`)

func ActivitypubActor(e *core.RequestEvent) error {
	resource := e.Request.URL.Query().Get("resource")
	resource = strings.TrimPrefix(resource, "acct:")

	iri := e.Request.URL.Query().Get("iri")
	follows := e.Request.URL.Query().Get("follows") == "true"

	var userActor *core.Record
	var err error
	if e.Auth != nil {
		userActor, err = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}
	}
	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	var actor *core.Record
	if resource != "" {
		actor, err = federation.GetActorByHandle(e.App, ctx, resource, follows)
	} else {
		actor, err = federation.GetActorByIRI(e.App, ctx, iri, follows)
	}
	if err != nil && actor == nil {
		if errors.Is(err, federation.ErrActorNotFound) || strings.HasPrefix(err.Error(), "webfinger") {
			return e.NotFoundError("Not found", err)
		}
		return err
	} else if err != nil && actor != nil {
		if errors.Is(err, federation.ErrProfilePrivate) {
			// this is our own profile
			if e.Auth != nil && actor.GetString("user") == e.Auth.Id {
				return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})
			} else {
				return e.JSON(http.StatusNotFound, map[string]any{"error": "profile is private"})
			}
		}
		// we could not fetch the remote actor so we return our local cached copy
		return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})
}

var getInboxActorByIRI = federation.GetActorByIRI

var knownInboxKeyReloads singleflight.Group

// Share an actor's first key refresh across its recipients. Each delivery
// waits only as long as its own request permits and reads the stored actor
// after the shared operation completes.
func reloadKnownInboxActor(app core.App, ctx context.Context, actor *core.Record) (*core.Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bucket := "known-actor:" + app.DataDir() + ":" + actor.Id
	fetched, claimRefused, invoked := false, false, false
	result := knownInboxKeyReloads.DoChan(bucket, func() (value any, err error) {
		invoked = true
		defer func() {
			if recovered := recover(); recovered != nil {
				app.Logger().Error("actor refresh did not complete", "value", recovered, "stack", string(debug.Stack()))
				value = nil
				err = &knownInboxActorReloadPanicError{}
			}
		}()
		stored, err := app.FindRecordById("activitypub_actors", actor.Id)
		if err != nil {
			return nil, err
		}
		if !actorNeedsKnownKeyReload(stored) {
			return nil, nil
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := util.KnownActorKeyReloads.CheckRateLimit(bucket, ""); err != nil {
			claimRefused = true
			return nil, err
		}
		fetched = true
		_, err = getInboxActorByIRI(app, ctx, stored.GetString("iri"), false)
		return nil, err
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case completed := <-result:
		var reloadPanic *knownInboxActorReloadPanicError
		if errors.As(completed.Err, &reloadPanic) {
			return nil, reloadPanic
		}
		stored, err := app.FindRecordById("activitypub_actors", actor.Id)
		if err != nil {
			return nil, err
		}
		if completed.Err != nil {
			return stored, &knownInboxActorReloadError{err: completed.Err, fetched: fetched, claimRefused: claimRefused, waited: !invoked}
		}
		return stored, nil
	}
}

func ActivitypubActivityProcess(e *core.RequestEvent) error {
	received := time.Now()
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	// This endpoint is internal: it may only be invoked by the trusted SvelteKit
	// frontend, which authenticates the hop with a shared secret. Because both the
	// recipient lookup and the HTTP-signature context are derived from the
	// X-Forwarded-Path header (which only the frontend can set correctly), any
	// caller that cannot present the secret must be rejected before that header is
	// used. Fail closed if the secret is not configured.
	secret := os.Getenv("POCKETBASE_PROXY_SECRET")
	if secret == "" {
		return e.UnauthorizedError("POCKETBASE_PROXY_SECRET not configured", nil)
	}
	if subtle.ConstantTimeCompare([]byte(e.Request.Header.Get("X-Internal-Secret")), []byte(secret)) != 1 {
		return e.UnauthorizedError("Invalid internal secret", nil)
	}

	forwardedPath := e.Request.Header.Get("X-Forwarded-Path")
	if !inboxPathPattern.MatchString(forwardedPath) {
		return e.UnauthorizedError("Invalid forwarded path", nil)
	}

	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	activity, err := federation.UnmarshalInboxActivity(body)
	if err != nil {
		return err
	}

	inbox := fmt.Sprintf("%s%s", origin, forwardedPath)

	recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)
	if err != nil {
		return err
	}

	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", activity.Actor.GetID().String())
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	knownKeyReload := actorNeedsKnownKeyReload(actor)
	if err == sql.ErrNoRows || knownKeyReload || (actor != nil && !actor.GetBool("is_local") && !federation.CachedActorIdentityMatchesOrigin(actor)) {
		ctx, err := util.GetSafeActorContext(e.Request, recipient)
		if err != nil {
			return err
		}
		storedActor := actor
		if knownKeyReload {
			// A known actor's first key refresh has its own budget; the
			// recipient's other lookups cannot prevent this delivery.
			bucket := "known-actor:" + e.App.DataDir() + ":" + storedActor.Id
			ctx = util.WithRateLimitBucket(ctx, bucket)
			actor, err = reloadKnownInboxActor(e.App, ctx, storedActor)
		} else {
			actor, err = getInboxActorByIRI(e.App, ctx, activity.Actor.GetID().String(), false)
		}
		var reloadPanic *knownInboxActorReloadPanicError
		if errors.As(err, &reloadPanic) {
			return e.InternalServerError("Unable to refresh actor", nil)
		}
		if err != nil {
			if inboxActorGoneCheckAllowed(e.App, ctx, e.Request, body, storedActor, activity, err, knownKeyReload) {
				legacySigned := func(stored *core.Record) bool { return legacySelfDeletionSigned(e.App, e.Request, body, stored) }
				if gone, deletionError := reconcileGoneInboxActor(e.App, ctx, storedActor, activity, legacySigned); gone {
					if deletionError != nil {
						e.App.Logger().Warn("remote account cleanup failed", "error", deletionError)
					}
					return e.JSON(http.StatusOK, deletionError)
				}
			}
			return e.UnauthorizedError("Unable to verify actor identity", err)
		}
	}

	if actor == nil || (!actor.GetBool("is_local") && !federation.CachedActorIdentityMatchesOrigin(actor)) {
		return e.UnauthorizedError("Invalid actor identity", nil)
	}
	verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"), body, activity.Actor.GetID().String())
	if err != nil || !verified {
		e.App.Logger().Warn("refused activity with invalid HTTP signature", "error", err)
		return e.UnauthorizedError("Invalid http signature", err)
	}

	err = federation.ProcessReceivedInboxActivity(e.App, actor, recipient, activity, received)
	if err != nil {
		message := "ActivityPub activity could not be processed"
		if errors.Is(err, util.ErrNotificationEmailDelivery) {
			message = "ActivityPub notification email could not be delivered"
		}
		e.App.Logger().Warn(message, "type", activity.Type, "actor", actor.Id, "recipient", recipient.Id, "error", err)
	}
	return e.JSON(http.StatusOK, err)
}

// actorNeedsKnownKeyReload identifies a remote key retained by the cache
// refresh. The stored host must name the actor's own origin; the earlier key
// itself does not authenticate a delivery.
func actorNeedsKnownKeyReload(actor *core.Record) bool {
	if actor == nil || actor.GetBool("is_local") || strings.TrimSpace(actor.GetString("legacy_public_key")) == "" || strings.TrimSpace(actor.GetString("public_key")) != "" {
		return false
	}
	parsed, err := url.Parse(actor.GetString("iri"))
	domain := strings.TrimSpace(actor.GetString("domain"))
	return err == nil && domain != "" && strings.EqualFold(strings.TrimPrefix(parsed.Hostname(), "www."), domain)
}

func ActivitypubActorFollow(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	followType := e.Request.PathValue("follow")
	if followType != "followers" && followType != "following" {
		return e.NotFoundError("unknown type: "+followType, nil)
	}
	page := e.Request.URL.Query().Get("page")
	intPage := 0

	if page != "" {
		var err error
		intPage, err = strconv.Atoi(page)
		if err != nil {
			return err
		}
	}

	actor, err := e.App.FindRecordById("activitypub_actors", id)
	if err != nil {
		return err
	}

	var userActor *core.Record
	if e.Auth != nil {
		userActor, err = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	url := actor.GetString(followType)

	if url == "" {
		return e.BadRequestError("unknown type: "+followType, nil)
	}
	var collection *pub.OrderedCollectionPage
	if cursor := e.Request.URL.Query().Get("cursor"); cursor != "" {
		collection, err = federation.FetchCollectionCursor(e.App, ctx, url, cursor)
	} else {
		collection, err = federation.FetchCollectionPage(e.App, ctx, url, intPage)
	}
	if err != nil {
		if errors.Is(err, federation.ErrInvalidCursor) {
			return e.BadRequestError("Invalid cursor", err)
		} else if errors.Is(err, federation.ErrProfilePrivate) {
			return e.JSON(http.StatusNotFound, map[string]any{"error": "profile is private"})
		} else if errors.Is(err, util.ErrRateLimited) {
			return e.TooManyRequestsError("Too many requests", err)
		}
		return err
	}
	if next := federation.CollectionNext(url, collection, intPage); next != "" {
		collection.Next = pub.IRI(next)
	} else {
		collection.Next = nil
	}
	return e.JSON(http.StatusOK, collection)
}

func ActivitypubTrail(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	trail, err := e.App.FindRecordById("trails", id)
	if err != nil {
		return e.NotFoundError("trail not found", nil)
	}

	reqInfo, err := e.RequestInfo()
	if err != nil {
		return err
	}
	canAccess, err := e.App.CanAccessRecord(trail, reqInfo, trail.Collection().ViewRule)
	if err != nil || !canAccess {
		return e.NotFoundError("trail not found", nil)
	}

	trailObject, err := util.ObjectFromTrail(e.App, trail, nil)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, trailObject)
}

func ActivitypubComment(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	comment, err := e.App.FindRecordById("comments", id)
	if err != nil {
		return e.NotFoundError("comment not found", nil)
	}

	reqInfo, err := e.RequestInfo()
	if err != nil {
		return err
	}
	canAccess, err := e.App.CanAccessRecord(comment, reqInfo, comment.Collection().ViewRule)
	if err != nil || !canAccess {
		return e.NotFoundError("comment not found", nil)
	}

	commentObject, err := util.ObjectFromComment(e.App, comment, nil)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, commentObject)
}
