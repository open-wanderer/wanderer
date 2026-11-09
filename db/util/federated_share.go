package util

import (
	"context"
	"errors"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"net/url"
	"strings"
)

const FederatedShareTokenTag = "wanderer:share-token"

type federatedShareKey struct{}
type federatedShare struct{ origin, token string }

func ValidFederatedShareToken(token string) bool {
	return len(token) == 32 && strings.Trim(token, "abcdefghijklmnopqrstuvwxyz0123456789") == ""
}

// WithFederatedShareToken keeps a directed share's fetch credential at its origin.
func WithFederatedShareToken(ctx context.Context, origin, token string) context.Context {
	if !ValidFederatedShareToken(token) {
		return context.WithValue(ctx, federatedShareKey{}, federatedShare{})
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return ctx
	}
	return context.WithValue(ctx, federatedShareKey{}, federatedShare{strings.ToLower(parsed.Scheme + "://" + parsed.Host), token})
}

func FederatedShareURL(ctx context.Context, raw string) string {
	if ctx == nil {
		return raw
	}
	share, ok := ctx.Value(federatedShareKey{}).(federatedShare)
	if !ok {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || strings.ToLower(parsed.Scheme+"://"+parsed.Host) != share.origin {
		return raw
	}
	q := parsed.Query()
	q.Set("share", share.token)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func WithoutFederatedShareToken(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := parsed.Query()
	if !q.Has("share") {
		return raw
	}
	q.Del("share")
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

// EnsureFederatedTrailShareToken grants reads for exactly this directed share.
func EnsureFederatedTrailShareToken(app core.App, share *core.Record) (string, error) {
	if share.Collection().Fields.GetByName("federation_token") == nil {
		return "", nil
	}
	if token := share.GetString("federation_token"); ValidFederatedShareToken(token) {
		return token, nil
	}
	token := security.RandomStringWithAlphabet(32, "abcdefghijklmnopqrstuvwxyz0123456789")
	share.Set("federation_token", token)
	if err := app.Save(share); err != nil {
		return "", err
	}
	return token, nil
}

func RemoteTrailShareTokens(app core.App, trailID string) []string {
	shares, err := app.FindRecordsByFilter("trail_share", "trail={:trail}", "id", -1, 0, dbx.Params{"trail": trailID})
	if err != nil {
		return nil
	}
	tokens := []string{}
	seen := map[string]bool{}
	for _, share := range shares {
		if token := share.GetString("remote_token"); ValidFederatedShareToken(token) && !seen[token] {
			tokens = append(tokens, token)
			seen[token] = true
		}
	}
	return tokens
}

type federatedShareError struct{ error }

func (e federatedShareError) Error() string {
	message := e.error.Error()
	// URL query values are encoded before requests; preserve useful context
	// without copying the share credential into response or worker logs.
	for start := strings.Index(message, "share="); start >= 0; {
		offset := start + len("share=")
		end := offset
		for end < len(message) && strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789", rune(message[end])) {
			end++
		}
		if end == offset {
			break
		}
		message = message[:offset] + "[redacted]" + message[end:]
		next := strings.Index(message[offset+len("[redacted]"):], "share=")
		if next < 0 {
			break
		}
		start = offset + len("[redacted]") + next
	}
	return message
}
func (e federatedShareError) Unwrap() error { return e.error }
func RedactFederatedShareError(err error) error {
	if err == nil {
		return nil
	}
	var redacted federatedShareError
	if errors.As(err, &redacted) {
		return err
	}
	return federatedShareError{err}
}
