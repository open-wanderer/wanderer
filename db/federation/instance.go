package federation

import (
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
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// InitInstanceActor creates the Application-type instance actor if it does not
// exist yet. It must run at startup before requests are served. An existing
// actor and its key pair are left unchanged.
func InitInstanceActor(app core.App) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	iri := origin + "/api/v1/activitypub/instance"

	// Never regenerate the key pair of an existing actor.
	existing, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking instance actor existence: %w", err)
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("parsing ORIGIN: %w", err)
	}
	domain := strings.TrimPrefix(parsedOrigin.Hostname(), "www.")

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
	record.Set("preferred_username", "instance")
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

// InstanceInboxHandler is the PocketBase route handler for
// POST /activitypub/instance/inbox. It dispatches Follow/Accept/Reject/Undo for
// the follow lifecycle and Create/Update/Delete/Like for content.
//
// The request must carry the internal proxy secret, the forwarded path must be
// instanceInboxPath, and the recipient is always the instance actor. After the
// HTTP signature is verified, content is accepted only from accepted peer hosts;
// lifecycle activities concerning the instance actor are accepted from anyone.
// User inboxes are not subject to this peer gate.
func InstanceInboxHandler(e *core.RequestEvent) error {
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

	// The recipient is always the instance actor, never derived from a header.
	instanceIRI := origin + "/api/v1/activitypub/instance"
	recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", instanceIRI)
	if err != nil {
		return err
	}

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

	// Look up the sender actor locally; fetch remotely on cache miss.
	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", actorIRI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Unknown actor: it has to be fetched before the signature can be verified.
			// First run the checks that need no key. Content fetches are keyed per actor
			// IRI, lifecycle fetches per host, except for the canonical instance actor IRI,
			// which has its own budget. The fetch is still signed with the instance actor's
			// key.
			if precheckErr := util.PrecheckSignature(e.Request.Header, body, actorIRI, time.Now()); precheckErr != nil {
				e.App.Logger().Warn("instance inbox: request refused before actor fetch",
					"actor", actorIRI, "err", precheckErr)
				return e.UnauthorizedError("Invalid http signature", precheckErr)
			}
			ctx, ctxErr := util.GetSafeActorContext(e.Request, recipient)
			if ctxErr != nil {
				return ctxErr
			}
			if isLifecycleActivity(activity) {
				if normalized, ok := isInstanceActorIRI(actorIRI); ok {
					ctx = util.WithRateLimitIdentifier(ctx, "inbox-instance-actor:"+normalized)
				} else {
					ctx = util.WithRateLimitIdentifier(ctx, "inbox-lifecycle:"+recipient.Id)
				}
			} else {
				if u, perr := url.Parse(actorIRI); perr == nil {
					noteContentPrefetch(e.App, u.Host)
				}
				ctx = util.WithRateLimitIdentifier(ctx, "inbox-actor:"+actorIRI)
			}
			actor, err = GetActorByIRI(e.App, ctx, actorIRI, false)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Verify HTTP signature against the sender actor's stored public key.
	verified, err := util.VerifySignature(e.App, e.Request, body, actor.GetString("public_key"))
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
	case pub.FollowType:
		if err := ProcessFollowActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Follow activity", err)
		}
	case pub.AcceptType:
		if err := ProcessAcceptActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Accept activity", err)
		}
	case pub.RejectType:
		if err := ProcessRejectActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Reject activity", err)
		}
	case pub.UndoType:
		if err := ProcessUndoActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Undo activity", err)
		}
	case pub.CreateType:
		fallthrough
	case pub.UpdateType:
		if err := ProcessCreateOrUpdateActivity(e.App, e.Request.Context(), actor, recipient, activity); err != nil {
			return e.BadRequestError("Failed to process Create/Update activity", err)
		}
	case pub.DeleteType:
		if err := ProcessDeleteActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Delete activity", err)
		}
	case pub.LikeType:
		if err := ProcessLikeActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Like activity", err)
		}
	default:
		return e.BadRequestError("Unsupported activity type", nil)
	}

	return e.JSON(http.StatusOK, map[string]bool{"success": true})
}
