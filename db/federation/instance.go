package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
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
	return priv, &priv.PublicKey, nil
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

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("parsing ORIGIN: %w", err)
	}
	domain := strings.TrimPrefix(parsedOrigin.Hostname(), "www.")

	priv, pub, err := generateInstanceKeyPair()
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privEncrypted, err := security.Encrypt(privBytes, encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypting private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
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
