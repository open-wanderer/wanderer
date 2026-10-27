package util

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RequestSignature contains the parameters used by the request policy and RSA
// verification, so both checks interpret the same signed headers and key id.
type RequestSignature struct {
	KeyID     string
	algorithm string
	headers   []string
	params    map[string]string
}

// CheckRequestSignature checks signed-header coverage, the exact body digest,
// and the signature's time window. It does not verify the public-key signature.
func CheckRequestSignature(req *http.Request, body []byte, now time.Time) (*RequestSignature, error) {
	raw := req.Header.Get("Signature")
	auth := req.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Signature ") {
		if raw != "" {
			return nil, fmt.Errorf("multiple request signatures")
		}
		raw = auth
	}
	params, err := parseRequestSignature(strings.TrimPrefix(raw, "Signature "))
	if err != nil {
		return nil, err
	}
	sig := &RequestSignature{KeyID: params["keyId"], algorithm: "hs2019", params: params}
	if sig.KeyID == "" || params["signature"] == "" {
		return nil, fmt.Errorf("missing signature key id or value")
	}
	if algorithm, ok := params["algorithm"]; ok {
		sig.algorithm = algorithm
	}
	if sig.algorithm != "hs2019" && sig.algorithm != "rsa-sha256" {
		return nil, fmt.Errorf("unsupported signature algorithm")
	}
	headers, ok := params["headers"]
	if !ok {
		headers = "(created)"
		if sig.algorithm == "rsa-sha256" {
			headers = "date"
		}
	}
	sig.headers = strings.Fields(strings.ToLower(headers))
	signed := make(map[string]bool, len(sig.headers))
	for _, header := range sig.headers {
		signed[header] = true
	}
	if !signed["date"] && !signed["(created)"] {
		return nil, fmt.Errorf("signature must cover date or (created)")
	}
	if !signed["(request-target)"] && !signed["digest"] {
		return nil, fmt.Errorf("signature must cover request target or digest")
	}
	if req.Method == http.MethodPost && !signed["digest"] {
		return nil, fmt.Errorf("POST signature must cover digest")
	}
	if req.Method == http.MethodGet && !signed["host"] {
		return nil, fmt.Errorf("GET signature must cover host")
	}
	for _, name := range []string{"created", "expires"} {
		if signed["("+name+")"] && (sig.algorithm != "hs2019" || params[name] == "") {
			return nil, fmt.Errorf("invalid signature pseudo-header (%s)", name)
		}
	}
	if signed["digest"] {
		if err := checkRequestDigest(req, body); err != nil {
			return nil, err
		}
	}

	var created int64
	if value, present := params["created"]; present {
		created, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid signature created time: %w", err)
		}
	}
	if _, present := params["created"]; !present || sig.algorithm != "hs2019" {
		date, err := http.ParseTime(req.Header.Get("Date"))
		if err != nil {
			return nil, fmt.Errorf("invalid signature date: %w", err)
		}
		created = date.Unix()
	}
	if created > signatureTimeAdd(now.Unix(), 3600) {
		return nil, fmt.Errorf("signature date is in the future")
	}
	expires := signatureTimeAdd(created, 300)
	if value, present := params["expires"]; present {
		expires, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid signature expiry time: %w", err)
		}
	}
	expires = min(expires, signatureTimeAdd(created, 12*3600))
	end := signatureTimeAdd(expires, 3600)
	if now.Unix() > end || (now.Unix() == end && now.Nanosecond() > 0) {
		return nil, fmt.Errorf("signature has expired")
	}
	return sig, nil
}

func signatureTimeAdd(seconds, delta int64) int64 {
	if seconds > math.MaxInt64-delta {
		return math.MaxInt64
	}
	return seconds + delta
}

func checkRequestDigest(req *http.Request, body []byte) error {
	for _, pair := range strings.Split(strings.Join(req.Header.Values("Digest"), ","), ",") {
		algorithm, value, ok := strings.Cut(pair, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(algorithm), "sha-256") {
			continue
		}
		value = strings.TrimSpace(value)
		decoded, err := base64.StdEncoding.Strict().DecodeString(value)
		if err != nil || len(decoded) != sha256.Size || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("invalid SHA-256 request digest")
		}
		expected := sha256.Sum256(body)
		if subtle.ConstantTimeCompare(decoded, expected[:]) != 1 {
			return fmt.Errorf("request body does not match digest")
		}
		return nil
	}
	return fmt.Errorf("missing SHA-256 request digest")
}

// verify uses RSA-SHA256 for both supported algorithm labels. The HTTP signature
// library's verifier imposes a separate ten-second clock margin, so it cannot
// be used with the time policy checked above.
func (sig *RequestSignature) verify(req *http.Request, key *rsa.PublicKey) error {
	lines := make([]string, 0, len(sig.headers))
	for _, header := range sig.headers {
		var value string
		switch header {
		case "(request-target)":
			// The forwarded public path is already in its signed wire form.
			value = strings.ToLower(req.Method) + " " + req.URL.Path
			if req.URL.RawQuery != "" {
				value += "?" + req.URL.RawQuery
			}
		case "(created)", "(expires)":
			value = sig.params[strings.Trim(header, "()")]
		case "host":
			value = req.Host
		default:
			values := append([]string(nil), req.Header.Values(header)...)
			for i := range values {
				values[i] = strings.TrimSpace(values[i])
			}
			value = strings.Join(values, ", ")
		}
		lines = append(lines, header+": "+value)
	}
	decoded, err := base64.StdEncoding.DecodeString(sig.params["signature"])
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], decoded)
}

func signatureToken(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))
}

// parseRequestSignature accepts comma-separated token or quoted parameters,
// including commas inside quoted values, and rejects duplicate parameter names.
func parseRequestSignature(raw string) (map[string]string, error) {
	params := make(map[string]string)
	for {
		raw = strings.TrimSpace(raw)
		end := 0
		for end < len(raw) && signatureToken(raw[end]) {
			end++
		}
		if end == 0 {
			return nil, fmt.Errorf("invalid request signature parameter")
		}
		name := raw[:end]
		raw = strings.TrimSpace(raw[end:])
		if len(raw) == 0 || raw[0] != '=' {
			return nil, fmt.Errorf("missing signature parameter value")
		}
		raw = strings.TrimSpace(raw[1:])
		var value string
		if len(raw) > 0 && raw[0] == '"' {
			end = 1
			for end < len(raw) && raw[end] != '"' {
				if raw[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(raw) {
				return nil, fmt.Errorf("unterminated signature parameter")
			}
			value, raw = raw[1:end], raw[end+1:]
		} else {
			end = 0
			for end < len(raw) && signatureToken(raw[end]) {
				end++
			}
			if end == 0 {
				return nil, fmt.Errorf("invalid signature parameter value")
			}
			value, raw = raw[:end], raw[end:]
		}
		if _, exists := params[name]; exists {
			return nil, fmt.Errorf("duplicate signature parameter")
		}
		params[name] = value
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return params, nil
		}
		if raw[0] != ',' {
			return nil, fmt.Errorf("invalid signature parameter separator")
		}
		raw = raw[1:]
	}
}
