package util

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Inbox signatures must cover date or (created), the signed time must be within
// the window Mastodon accepts (up to 1h in the future, created + 5m plus 1h
// skew in the past), and a Digest header, if present, must match the body.
// Requests without a Digest header are accepted.
const (
	signatureClockSkew = time.Hour
	signatureValidity  = 5 * time.Minute
)

// signatureParams returns the parameters of the Signature header, or of an
// Authorization header using the "Signature" scheme, parsed like go-fed/httpsig.
func signatureParams(h http.Header) (map[string]string, error) {
	raw := h.Get("Signature")
	if raw == "" {
		auth := h.Get("Authorization")
		if !strings.HasPrefix(auth, "Signature ") {
			return nil, fmt.Errorf("no http signature found")
		}
		raw = strings.TrimPrefix(auth, "Signature ")
	}

	params := map[string]string{}
	for _, p := range strings.Split(raw, ",") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("malformed http signature parameter")
		}
		params[kv[0]] = strings.Trim(kv[1], `"`)
	}
	return params, nil
}

// signedHeaderNames returns the lower-cased names covered by the signature. A
// signature without a headers parameter covers date only, like go-fed.
func signedHeaderNames(params map[string]string) []string {
	headers := strings.Fields(strings.ToLower(params["headers"]))
	if len(headers) == 0 {
		return []string{"date"}
	}
	return headers
}

func containsHeader(signed []string, name string) bool {
	for _, s := range signed {
		if s == name {
			return true
		}
	}
	return false
}

func checkSignedTime(signedAt, now time.Time) error {
	if signedAt.After(now.Add(signatureClockSkew)) {
		return fmt.Errorf("signature time is too far in the future")
	}
	if signedAt.Before(now.Add(-(signatureClockSkew + signatureValidity))) {
		return fmt.Errorf("signature time is too old")
	}
	return nil
}

// checkSignatureTime refuses signatures that do not cover a time or whose signed
// time is outside the accepted window.
func checkSignatureTime(h http.Header, params map[string]string, signed []string, now time.Time) error {
	switch {
	case containsHeader(signed, "date"):
		raw := h.Get("Date")
		if raw == "" {
			return fmt.Errorf("date is signed but the Date header is missing")
		}
		date, err := http.ParseTime(raw)
		if err != nil {
			return fmt.Errorf("invalid Date header: %w", err)
		}
		return checkSignedTime(date, now)
	case containsHeader(signed, "(created)"):
		created, err := strconv.ParseInt(params["created"], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid created parameter: %w", err)
		}
		return checkSignedTime(time.Unix(created, 0), now)
	default:
		return fmt.Errorf("signature must cover date or (created)")
	}
}

// verifyBodyDigest checks a present Digest header against the body. A signed
// digest without a Digest header is refused; no Digest header at all passes.
func verifyBodyDigest(h http.Header, body []byte, signed []string) error {
	values := h.Values("Digest")
	if len(values) == 0 {
		if containsHeader(signed, "digest") {
			return fmt.Errorf("digest is signed but the Digest header is missing")
		}
		return nil
	}

	supported := false
	for _, value := range values {
		for _, entry := range strings.Split(value, ",") {
			kv := strings.SplitN(strings.TrimSpace(entry), "=", 2)
			if len(kv) != 2 {
				continue
			}
			var sum []byte
			switch strings.ToLower(kv[0]) {
			case "sha-256":
				s := sha256.Sum256(body)
				sum = s[:]
			case "sha-512":
				s := sha512.Sum512(body)
				sum = s[:]
			default:
				continue
			}
			supported = true
			got, err := base64.StdEncoding.DecodeString(kv[1])
			if err != nil || subtle.ConstantTimeCompare(got, sum) != 1 {
				return fmt.Errorf("digest does not match the request body")
			}
		}
	}
	if !supported {
		return fmt.Errorf("no supported digest algorithm")
	}
	return nil
}

// PrecheckSignature runs the checks of VerifySignature that need no key: a
// parseable signature whose keyId is on the claimed actor's host, the signed
// time window and the Digest. It is used before fetching an unknown actor and
// proves nothing about the key itself.
func PrecheckSignature(h http.Header, body []byte, actorIRI string, now time.Time) error {
	params, err := signatureParams(h)
	if err != nil {
		return err
	}

	keyID := params["keyId"]
	if keyID == "" {
		return fmt.Errorf("signature has no keyId")
	}
	keyURL, err := url.Parse(keyID)
	if err != nil || keyURL.Host == "" {
		return fmt.Errorf("signature keyId is not a valid IRI")
	}
	actorURL, err := url.Parse(actorIRI)
	if err != nil || actorURL.Host == "" {
		return fmt.Errorf("actor is not a valid IRI")
	}
	if !strings.EqualFold(keyURL.Host, actorURL.Host) {
		return fmt.Errorf("signature keyId is not on the actor's host")
	}

	signed := signedHeaderNames(params)
	if err := checkSignatureTime(h, params, signed, now); err != nil {
		return err
	}
	return verifyBodyDigest(h, body, signed)
}
