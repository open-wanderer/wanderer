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

// InitInstanceActor idempotently creates the Application-type instance actor in
// the activitypub_actors collection. It must be called synchronously at startup
// (before the HTTP server begins accepting requests) so the actor is available
// immediately for any incoming ActivityPub request.
//
// If the actor already exists (checked by IRI), this function returns nil without
// touching the stored keypair. The keypair is NEVER regenerated after first creation.
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

	// Idempotency check: if the actor record already exists, return immediately.
	// This is the critical guard that prevents keypair regeneration on restart.
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

// InstanceInboxHandler is the PocketBase route handler for
// POST /activitypub/instance/inbox.
//
// It mirrors ActivitypubActivityProcess in db/routes/activitypub.go but is
// scoped to the instance actor's inbox. It dispatches Follow/Accept/Undo for
// the follow lifecycle and Create/Update/Delete for content synchronization
// (SYNC-01).
//
// HTTP signature verification is performed via util.VerifySignature before any
// activity processing (T-02-03: mitigate spoofing).
func InstanceInboxHandler(e *core.RequestEvent) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}

	var activity pub.Activity
	if err = activity.UnmarshalJSON(body); err != nil {
		return err
	}

	// Reconstruct the full inbox IRI from the X-Forwarded-Path header set by the
	// SvelteKit proxy (T-02-04: header is set by the trusted SvelteKit layer, not
	// accepted from the remote client).
	inbox := fmt.Sprintf("%s%s", origin, e.Request.Header.Get("X-Forwarded-Path"))

	recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)
	if err != nil {
		return err
	}

	// Look up the sender actor locally; fetch remotely on cache miss.
	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", activity.Actor.GetID().String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx, ctxErr := util.GetSafeActorContext(e.Request, recipient)
			if ctxErr != nil {
				return ctxErr
			}
			actor, err = GetActorByIRI(e.App, ctx, activity.Actor.GetID().String(), false)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Verify HTTP signature against the sender actor's stored public key (T-02-03).
	verified, err := util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))
	if err != nil || !verified {
		e.App.Logger().Error("instance inbox: invalid http signature", "err", err)
		return e.UnauthorizedError("Invalid http signature", err)
	}

	// Dispatch to the appropriate federation processor.
	// Only Follow/Accept/Undo are handled here (D-03).
	// CR-05: each case propagates errors as 400 responses so the remote caller
	// can detect failures; unknown activity types are rejected with 400 rather
	// than silently accepted with 200 null.
	switch activity.Type {
	case pub.FollowType:
		if err := ProcessFollowActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Follow activity", err)
		}
	case pub.AcceptType:
		if err := ProcessAcceptActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Accept activity", err)
		}
	case pub.UndoType:
		if err := ProcessUndoActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Undo activity", err)
		}
	case pub.CreateType:
		fallthrough
	case pub.UpdateType:
		if err := ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity); err != nil {
			return e.BadRequestError("Failed to process Create/Update activity", err)
		}
	case pub.DeleteType:
		if err := ProcessDeleteActivity(e.App, actor, activity); err != nil {
			return e.BadRequestError("Failed to process Delete activity", err)
		}
	default:
		return e.BadRequestError("Unsupported activity type", nil)
	}

	return e.JSON(http.StatusOK, map[string]bool{"success": true})
}
