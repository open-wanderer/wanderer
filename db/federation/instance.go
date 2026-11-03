package federation

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"pocketbase/util"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// InstanceActorUsername is the preferred_username of the local instance actor.
//
// The users username pattern requires a word character first, so no account can
// ever hold a name starting with "-" and the (preferred_username, domain) unique
// index cannot collide with a user. util.SplitHandle splits only on @ and keeps
// the name whole, and a mention tokenizer either takes the whole token or none
// of it.
// WebFinger in Wanderer resolves user IRIs only, and person lookups exclude
// instance actors by type anyway.
const InstanceActorUsername = "-instance"

// FindLocalInstanceActor returns the local instance actor: the activitypub_actors
// row with actor_type instance and is_local true. It is the single definition of
// that row. Never find it by IRI or username: the IRI follows ORIGIN, and an
// existing actor may still carry an earlier username until InitInstanceActor
// runs. It returns sql.ErrNoRows when there is none.
//
// When more than one row matches, the oldest wins and an Error line names both
// ids, so every consumer uses the same row. Oldest is ascending @rowid, which is
// insertion order: a millisecond created value cannot order two rows created in
// the same millisecond, and with @rowid the Go and web lookups agree.
func FindLocalInstanceActor(app core.App) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(
		"activitypub_actors",
		"actor_type = {:t} && is_local = true",
		"@rowid", 2, 0,
		dbx.Params{"t": "instance"},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(records) > 1 {
		app.Logger().Error("more than one local instance actor, using the oldest",
			"id", records[0].Id, "other", records[1].Id)
	}
	return records[0], nil
}

// InitInstanceActor makes sure the one local Application-type instance actor
// exists and matches the current ORIGIN. It must run at startup before requests
// are served.
//
// The actor is found by type and locality (FindLocalInstanceActor). When ORIGIN
// changed, the existing row follows it in place: preferred_username, username,
// domain, iri, inbox and outbox are updated and the key pair is kept, so the
// record id, the follows pointing at it and the signing key survive. An actor
// still named "instance" is renamed to InstanceActorUsername on the same path. With no instance actor a new one is created.
//
// It only saves the row and sends nothing (no Update, no Move, no alias):
// connected peers are not notified after an ORIGIN change and the admin
// reconnects them from /federation/. Other stored IRIs (follows, activities, user
// actors, content) are not rewritten.
//
// If the save fails because a second local instance actor already owns the new
// IRI, the error names both record ids and nothing is deleted or merged. The
// database then holds a second local instance actor from a development build:
// delete the newer one in the PocketBase admin and restart.
func InitInstanceActor(app core.App) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	iri := origin + instanceActorPath

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("parsing ORIGIN: %w", err)
	}
	domain := strings.TrimPrefix(parsedOrigin.Hostname(), "www.")

	existing, err := FindLocalInstanceActor(app)
	if err == nil {
		return updateInstanceActorIdentity(app, existing, iri, domain)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("finding the local instance actor: %w", err)
	}

	priv, pubKey, err := util.GenerateRSAKeyPair()
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privEncrypted, err := security.Encrypt(privBytes, encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypting private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}
	pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		return fmt.Errorf("finding activitypub_actors collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("actor_type", "instance")
	record.Set("preferred_username", InstanceActorUsername)
	record.Set("username", fmt.Sprintf("Wanderer at %s", domain))
	record.Set("domain", domain)
	record.Set("iri", iri)
	record.Set("inbox", iri+"/inbox")
	record.Set("outbox", iri+"/outbox")
	record.Set("is_local", true)
	record.Set("public_key", string(pubPem))
	record.Set("private_key", privEncrypted)
	record.Set("last_fetched", time.Now())

	return app.Save(record)
}

// updateInstanceActorIdentity brings the six identity fields of the existing
// local instance actor in line with ORIGIN. The key pair and last_fetched are
// never touched, and nothing is saved when nothing differs.
func updateInstanceActorIdentity(app core.App, actor *core.Record, iri, domain string) error {
	want := map[string]string{
		"preferred_username": InstanceActorUsername,
		"username":           fmt.Sprintf("Wanderer at %s", domain),
		"domain":             domain,
		"iri":                iri,
		"inbox":              iri + "/inbox",
		"outbox":             iri + "/outbox",
	}
	changed := false
	for field, value := range want {
		if actor.GetString(field) != value {
			actor.Set(field, value)
			changed = true
		}
	}
	if !changed {
		return nil
	}

	if err := app.Save(actor); err != nil {
		// A second local instance actor that already owns the target IRI makes
		// the unique index refuse the move. Report both ids; never delete,
		// merge or rename the other row.
		owner, findErr := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
		if findErr == nil && owner != nil && owner.Id != actor.Id {
			app.Logger().Error("cannot move the local instance actor to the current ORIGIN: another actor owns its IRI",
				"actor", actor.Id, "owner", owner.Id, "iri", iri, "error", err)
		}
		return fmt.Errorf("updating the local instance actor %s: %w", actor.Id, err)
	}
	return nil
}

// instanceInboxPath is the only forwarded path the instance inbox accepts.
const instanceInboxPath = "/api/v1/activitypub/instance/inbox"

// instanceContentFetchWarnThreshold is the number of content fetches per peer
// host and minute before signature verification above which a warning is logged.
const instanceContentFetchWarnThreshold = 300

// instanceContentFetchCounter counts those fetches per host;
// instanceContentFetchWarnSampler limits the warning to one per host and minute.
var (
	instanceContentFetchCounter     = util.NewRateLimiter(instanceContentFetchWarnThreshold, time.Minute)
	instanceContentFetchWarnSampler = util.NewRateLimiter(1, time.Minute)
)

// noteContentPrefetch counts one content fetch for host and logs a warning,
// once per host and minute, when the threshold is exceeded. It returns true when
// it logged. It never refuses a request.
func noteContentPrefetch(app core.App, host string) bool {
	host = strings.ToLower(host)
	if instanceContentFetchCounter.CheckRateLimit("inbox-content-fetch", host) == nil {
		return false
	}
	if instanceContentFetchWarnSampler.CheckRateLimit("inbox-content-fetch-warn", host) != nil {
		return false
	}
	app.Logger().Warn("instance inbox: many pre-verification actor fetches for a peer host",
		"host", host, "threshold", instanceContentFetchWarnThreshold)
	return true
}

// instanceActorPath is the path of a Wanderer instance actor on its origin.
const instanceActorPath = "/api/v1/activitypub/instance"

// isInstanceActorIRI reports whether iri is exactly the canonical instance actor
// IRI of a Wanderer origin: http or https on the default port, path
// instanceActorPath, no query, fragment, userinfo or trailing-dot host. It also
// returns the normalized scheme://hostname+path, used as a rate-limit key.
func isInstanceActorIRI(iri string) (string, bool) {
	if iri == "" || strings.ContainsAny(iri, "?#") {
		return "", false
	}
	u, err := url.Parse(iri)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if u.User != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.HasSuffix(host, ".") {
		return "", false
	}
	defaultPort := "443"
	if scheme == "http" {
		defaultPort = "80"
	}
	if port := u.Port(); port != "" && port != defaultPort {
		return "", false
	}
	if u.EscapedPath() != instanceActorPath {
		return "", false
	}
	return scheme + "://" + host + instanceActorPath, true
}

// precheckSignature runs the signature checks that need no key, so an unknown
// actor is only fetched for a request that could still verify: signed-header
// coverage, body digest, signature time, and a key id on the actor's host.
func precheckSignature(req *http.Request, body []byte, actorIRI string) error {
	signature, err := util.CheckRequestSignature(req, body, time.Now())
	if err != nil {
		return err
	}
	if !util.SignatureKeyIsOnActorHost(signature.KeyID, actorIRI) {
		return fmt.Errorf("signature key id is not on the activity actor's host")
	}
	return nil
}

// InboxSenderResolver returns the stored, origin-confirmed sender of a signed
// delivery, fetching it with the context fetchContext prepares when needed.
// handled reports that the resolver has already answered the request.
type InboxSenderResolver func(e *core.RequestEvent, body []byte, activity pub.Activity, fetchContext func() (context.Context, error)) (actor *core.Record, handled bool, err error)

// NewInstanceInboxHandler returns the PocketBase route handler for
// POST /activitypub/instance/inbox. It dispatches Follow/Accept/Reject/Undo for
// the follow lifecycle and Create/Update/Delete/Like for content.
//
// The request must carry the internal proxy secret, the forwarded path must be
// instanceInboxPath, and the recipient is always the instance actor. After the
// HTTP signature is verified, content is accepted only from accepted peer hosts;
// lifecycle activities concerning the instance actor are accepted from anyone.
// User inboxes are not subject to this peer gate.
//
// resolveSender is the user inboxes' sender resolution, so that both kinds of
// inbox reload retained keys, refetch unconfirmed cached identities and
// reconcile deleted accounts the same way.
func NewInstanceInboxHandler(resolveSender InboxSenderResolver) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return handleInstanceInbox(e, resolveSender)
	}
}

func handleInstanceInbox(e *core.RequestEvent, resolveSender InboxSenderResolver) error {
	received := time.Now()
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	if err := util.RequireInternalProxy(e.Request); err != nil {
		if errors.Is(err, util.ErrProxySecretNotConfigured) {
			return e.UnauthorizedError("POCKETBASE_PROXY_SECRET not configured", nil)
		}
		return e.UnauthorizedError("Invalid internal secret", nil)
	}

	if e.Request.Header.Get("X-Forwarded-Path") != instanceInboxPath {
		return e.UnauthorizedError("Invalid forwarded path", nil)
	}

	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}

	var activity pub.Activity
	if err = activity.UnmarshalJSON(body); err != nil {
		return e.BadRequestError("Invalid activity", err)
	}
	if activity.Actor == nil || activity.Actor.GetID().String() == "" {
		return e.BadRequestError("Missing actor", nil)
	}
	actorIRI := activity.Actor.GetID().String()

	// The recipient is always the local instance actor, never derived from a header.
	recipient, err := FindLocalInstanceActor(e.App)
	if err != nil {
		return err
	}
	instanceIRI := recipient.GetString("iri")

	// Refuse content from hosts that are not accepted peers before fetching the
	// actor.
	if !isLifecycleActivity(activity) {
		ok, err := IsAcceptedPeerHost(e.App, actorIRI)
		if err != nil {
			return err
		}
		if !ok {
			return e.ForbiddenError("Not an accepted peer", nil)
		}
	}

	// Every wanderer has the same instance inbox path, so a delivery must be
	// bound to this instance and this inbox, not only to its body and time.
	signature, err := util.CheckRequestSignature(e.Request, body, time.Now())
	if err == nil && (!signature.Covers("(request-target)") || !signature.Covers("host")) {
		err = fmt.Errorf("signature must cover (request-target) and host")
	}
	if err != nil {
		e.App.Logger().Warn("instance inbox: request refused before actor lookup",
			"actor", actorIRI, "err", err)
		return e.UnauthorizedError("Invalid http signature", err)
	}

	// The sender may have to be fetched before the signature can be verified.
	// First run the checks that need no key. Content fetches are keyed per actor
	// IRI, lifecycle fetches per host, except for the canonical instance actor IRI,
	// which has its own budget. The fetch is still signed with the instance actor's
	// key.
	fetchContext := func() (context.Context, error) {
		if precheckErr := precheckSignature(e.Request, body, actorIRI); precheckErr != nil {
			e.App.Logger().Warn("instance inbox: request refused before actor fetch",
				"actor", actorIRI, "err", precheckErr)
			return nil, e.UnauthorizedError("Invalid http signature", precheckErr)
		}
		ctx, err := util.GetSafeActorContext(e.Request, recipient)
		if err != nil {
			return nil, err
		}
		if isLifecycleActivity(activity) {
			if normalized, ok := isInstanceActorIRI(actorIRI); ok {
				return util.WithRateLimitBucket(ctx, "inbox-instance-actor:"+normalized), nil
			}
			return util.WithRateLimitBucket(ctx, "inbox-lifecycle:"+recipient.Id), nil
		}
		if u, perr := url.Parse(actorIRI); perr == nil {
			noteContentPrefetch(e.App, u.Host)
		}
		return util.WithRateLimitBucket(ctx, "inbox-actor:"+actorIRI), nil
	}
	actor, handled, err := resolveSender(e, body, activity, fetchContext)
	if err != nil || handled {
		return err
	}

	// Verify HTTP signature against the sender actor's stored public key.
	verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"), body, actorIRI)
	if err != nil || !verified {
		e.App.Logger().Error("instance inbox: invalid http signature", "err", err)
		return e.UnauthorizedError("Invalid http signature", err)
	}

	// A valid signature proves key ownership, not a peer relationship.
	if err := authorizeInstanceActivity(e.App, actor, activity, instanceIRI); err != nil {
		e.App.Logger().Warn("instance inbox: activity refused",
			"signer", actor.GetString("iri"), "type", string(activity.Type), "err", err)
		switch {
		case errors.Is(err, errNotAcceptedPeer):
			return e.ForbiddenError("Not an accepted peer", nil)
		case errors.Is(err, errLifecycleNotForInstance):
			return e.ForbiddenError("Activity does not concern the instance actor", nil)
		default:
			return err
		}
	}

	// Processing errors and unknown activity types are returned as 400.
	switch activity.Type {
	case pub.RejectType:
		if err := ProcessRejectActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Reject activity", err)
		}
	case pub.FollowType, pub.AcceptType, pub.UndoType, pub.CreateType, pub.UpdateType, pub.DeleteType, pub.LikeType:
		// Ordered with the deliveries to user inboxes: a Delete or Undo that
		// arrives before its Create or Like still retracts it.
		err := ProcessReceivedInboxActivity(e.App, actor, recipient, activity, received)
		// A peer forwards every like and unlike; the receiver may not hold
		// the trail or the like, or the twin delivery to a user inbox removed
		// it first. A like carries no content to attach anywhere else.
		if errors.Is(err, sql.ErrNoRows) && (activity.Type == pub.LikeType || activity.Type == pub.UndoType) {
			err = nil
		}
		if err != nil {
			return e.BadRequestError(fmt.Sprintf("Failed to process %s activity", activity.Type), err)
		}
	default:
		return e.BadRequestError("Unsupported activity type", nil)
	}

	return e.JSON(http.StatusOK, map[string]bool{"success": true})
}
