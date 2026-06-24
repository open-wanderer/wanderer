package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// generateInstanceKeyPair generates a 2048-bit RSA keypair for the instance actor.
// Copied from db/util/activitypub.go (unexported there; duplicated here to keep instance.go self-contained).
func generateInstanceKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	pub := &priv.PublicKey
	return priv, pub, nil
}

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

	// Parse ORIGIN to derive the domain name (strip www. prefix)
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("parsing ORIGIN: %w", err)
	}
	domain := strings.TrimPrefix(parsedOrigin.Hostname(), "www.")

	// Generate RSA keypair
	priv, pub, err := generateInstanceKeyPair()
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}

	// Encode and encrypt private key
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privEncrypted, err := security.Encrypt(privBytes, encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypting private key: %w", err)
	}

	// Encode public key as PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

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

// buildInstanceActorJSON constructs the ActivityPub actor document for the instance
// actor as a plain map so that manuallyApprovesFollowers (absent from the go-ap
// Actor struct) can be included. This helper is separated from the HTTP handler so
// that it can be exercised directly in tests.
func buildInstanceActorJSON(record *core.Record, iri string) map[string]any {
	return map[string]any{
		"@context": []any{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		"id":                        iri,
		"type":                      "Application",
		"preferredUsername":         record.GetString("preferred_username"),
		"name":                      record.GetString("username"),
		"inbox":                     record.GetString("inbox"),
		"outbox":                    record.GetString("outbox"),
		"manuallyApprovesFollowers": true,
		"publicKey": map[string]any{
			"id":           iri + "#main-key",
			"owner":        iri,
			"publicKeyPem": record.GetString("public_key"),
		},
	}
}

// InstanceActorGet handles GET /activitypub/instance and returns the instance
// actor as a valid ActivityPub JSON document.
//
// This handler looks up the actor record directly by IRI rather than going
// through actor assembly helpers, because those helpers assume every is_local
// actor has a non-empty user relation which is not true for the instance actor.
func InstanceActorGet(e *core.RequestEvent) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	iri := origin + "/api/v1/activitypub/instance"

	record, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", iri)
	if err != nil {
		return e.NotFoundError("Instance actor not found", err)
	}

	return e.JSON(http.StatusOK, buildInstanceActorJSON(record, iri))
}
