package util

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
)

var (
	// ErrProxySecretNotConfigured is returned when POCKETBASE_PROXY_SECRET is unset.
	ErrProxySecretNotConfigured = errors.New("POCKETBASE_PROXY_SECRET not configured")
	// ErrInvalidInternalSecret is returned when the X-Internal-Secret header is
	// missing or does not match POCKETBASE_PROXY_SECRET.
	ErrInvalidInternalSecret = errors.New("invalid internal secret")
)

// RequireInternalProxy checks that the request carries the internal proxy
// secret set by the SvelteKit frontend. It fails when the secret is not
// configured and compares in constant time.
func RequireInternalProxy(r *http.Request) error {
	secret := os.Getenv("POCKETBASE_PROXY_SECRET")
	if secret == "" {
		return ErrProxySecretNotConfigured
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Secret")), []byte(secret)) != 1 {
		return ErrInvalidInternalSecret
	}
	return nil
}
