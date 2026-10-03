package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
)

const testInboxPath = "/api/v1/activitypub/user/alice/inbox"

// signedInboxRequest builds a signed POST as the SvelteKit proxy forwards it
// to the backend.
func signedInboxRequest(t *testing.T, priv *rsa.PrivateKey, headers []string, signedBody, sentBody []byte, date time.Time, mutate func(*http.Request)) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "https://local.example.com"+testInboxPath, bytes.NewReader(sentBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Date", date.UTC().Format(http.TimeFormat))
	req.Header.Set("Host", "local.example.com")

	signer, _, err := httpsig.NewSigner(
		[]httpsig.Algorithm{httpsig.RSA_SHA256},
		httpsig.DigestSha256,
		headers,
		httpsig.Signature, 60)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	if err := signer.SignRequest(priv, "https://remote.example.com/users/bob#main-key", req, signedBody); err != nil {
		t.Fatalf("sign request: %v", err)
	}
	if mutate != nil {
		mutate(req)
	}
	req.Header.Set("X-Forwarded-Path", testInboxPath)
	return req
}

func TestVerifySignature(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example.com")

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	pemFor := func(k *rsa.PrivateKey) string {
		der, err := x509.MarshalPKIXPublicKey(&k.PublicKey)
		if err != nil {
			t.Fatalf("marshal public key: %v", err)
		}
		return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	}

	wanderer := []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}
	mastodon := []string{"(request-target)", "host", "date", "digest", "content-type"}
	noDigest := []string{"(request-target)", "host", "date"}
	noDate := []string{"(request-target)", "host", "digest"}
	createdHeaders := []string{"(request-target)", "(created)", "host", "digest"}

	body := []byte(`{"type":"Create","actor":"https://remote.example.com/users/bob"}`)
	swapped := []byte(`{"type":"Delete","actor":"https://remote.example.com/users/bob"}`)
	otherSum := sha256.Sum256(swapped)
	otherDigest := "SHA-256=" + base64.StdEncoding.EncodeToString(otherSum[:])

	now := time.Now()
	tests := []struct {
		name       string
		headers    []string
		sentBody   []byte
		date       time.Time
		mutate     func(*http.Request)
		key        *rsa.PrivateKey
		wantVerify bool
	}{
		{name: "wanderer PostActivity style", headers: wanderer, sentBody: body, date: now, wantVerify: true},
		{name: "mastodon style lowercase headers", headers: mastodon, sentBody: body, date: now, wantVerify: true},
		{name: "tampered body with valid signed headers", headers: mastodon, sentBody: swapped, date: now, wantVerify: false},
		{name: "date two hours old", headers: mastodon, sentBody: body, date: now.Add(-2 * time.Hour), wantVerify: false},
		{name: "date thirty minutes old", headers: mastodon, sentBody: body, date: now.Add(-30 * time.Minute), wantVerify: true},
		{name: "date two hours in the future", headers: mastodon, sentBody: body, date: now.Add(2 * time.Hour), wantVerify: false},
		{name: "date not covered by the signature", headers: noDate, sentBody: body, date: now, wantVerify: false},
		{
			name: "created covered and no Date header", headers: createdHeaders, sentBody: body, date: now,
			mutate: func(r *http.Request) { r.Header.Del("Date") }, wantVerify: true,
		},
		{
			name: "no Digest header at all", headers: noDigest, sentBody: body, date: now,
			mutate: func(r *http.Request) { r.Header.Del("Digest") }, wantVerify: true,
		},
		{
			name: "unsigned Digest header with unsupported algorithm only", headers: noDigest, sentBody: body, date: now,
			mutate: func(r *http.Request) { r.Header.Set("Digest", "MD5=1B2M2Y8AsgTpgAmY7PhCfg==") }, wantVerify: false,
		},
		{
			name: "unsigned Digest header that does not match", headers: noDigest, sentBody: body, date: now,
			mutate: func(r *http.Request) { r.Header.Set("Digest", otherDigest) }, wantVerify: false,
		},
		{
			name: "digest signed but header missing", headers: mastodon, sentBody: body, date: now,
			mutate: func(r *http.Request) { r.Header.Del("Digest") }, wantVerify: false,
		},
		{name: "wrong key", headers: mastodon, sentBody: body, date: now, key: other, wantVerify: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := signedInboxRequest(t, priv, tc.headers, body, tc.sentBody, tc.date, tc.mutate)
			pubPem := pemFor(priv)
			if tc.key != nil {
				pubPem = pemFor(tc.key)
			}
			ok, err := VerifySignature(nil, req, tc.sentBody, pubPem)
			if tc.wantVerify {
				if err != nil || !ok {
					t.Fatalf("expected the signature to verify, got ok=%v err=%v", ok, err)
				}
				return
			}
			if err == nil && ok {
				t.Fatal("expected the signature to be refused")
			}
		})
	}
}

func TestCheckSignatureTimeCreated(t *testing.T) {
	now := time.Now()
	h := http.Header{}
	signed := []string{"(request-target)", "(created)", "host"}
	at := func(d time.Duration) map[string]string {
		return map[string]string{"created": strconv.FormatInt(now.Add(d).Unix(), 10)}
	}
	if err := checkSignatureTime(h, at(-time.Minute), signed, now); err != nil {
		t.Fatalf("fresh created refused: %v", err)
	}
	if err := checkSignatureTime(h, at(-2*time.Hour), signed, now); err == nil {
		t.Fatal("stale created accepted")
	}
	if err := checkSignatureTime(h, at(2*time.Hour), signed, now); err == nil {
		t.Fatal("future created accepted")
	}
	if err := checkSignatureTime(h, map[string]string{}, signed, now); err == nil {
		t.Fatal("missing created accepted")
	}
}

// TestPrecheckSignature covers the checks that need no key: a parseable
// signature with a keyId on the actor's host, a recent signed time, and a
// matching Digest.
func TestPrecheckSignature(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example.com")

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	wanderer := []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}
	mastodon := []string{"(request-target)", "host", "date", "digest", "content-type"}
	noDate := []string{"(request-target)", "host", "digest"}

	body := []byte(`{"type":"Create","actor":"https://remote.example.com/users/bob"}`)
	swapped := []byte(`{"type":"Delete","actor":"https://remote.example.com/users/bob"}`)

	const sameHostActor = "https://remote.example.com/users/bob"
	now := time.Now()
	tests := []struct {
		name     string
		headers  []string
		sentBody []byte
		date     time.Time
		actor    string
		mutate   func(*http.Request)
		wantErr  bool
	}{
		{name: "wanderer PostActivity style", headers: wanderer, sentBody: body, date: now, actor: sameHostActor},
		{name: "mastodon style", headers: mastodon, sentBody: body, date: now, actor: sameHostActor},
		{name: "keyId host compared case-insensitively", headers: mastodon, sentBody: body, date: now, actor: "https://REMOTE.example.com/users/bob"},
		{
			name: "authorization signature scheme", headers: mastodon, sentBody: body, date: now, actor: sameHostActor,
			mutate: func(r *http.Request) {
				r.Header.Set("Authorization", "Signature "+r.Header.Get("Signature"))
				r.Header.Del("Signature")
			},
		},
		{
			name: "missing signature header", headers: mastodon, sentBody: body, date: now, actor: sameHostActor,
			mutate: func(r *http.Request) { r.Header.Del("Signature") }, wantErr: true,
		},
		{
			name: "malformed signature header", headers: mastodon, sentBody: body, date: now, actor: sameHostActor,
			mutate: func(r *http.Request) { r.Header.Set("Signature", "garbage") }, wantErr: true,
		},
		{
			name: "missing keyId", headers: mastodon, sentBody: body, date: now, actor: sameHostActor,
			mutate:  func(r *http.Request) { r.Header.Set("Signature", `headers="date",signature="abc"`) },
			wantErr: true,
		},
		{name: "keyId on another host", headers: mastodon, sentBody: body, date: now, actor: "https://evil.example.com/users/bob", wantErr: true},
		{name: "keyId on the same host but another port", headers: mastodon, sentBody: body, date: now, actor: "https://remote.example.com:8443/users/bob", wantErr: true},
		{name: "date not covered by the signature", headers: noDate, sentBody: body, date: now, actor: sameHostActor, wantErr: true},
		{name: "stale date", headers: mastodon, sentBody: body, date: now.Add(-2 * time.Hour), actor: sameHostActor, wantErr: true},
		{name: "digest does not match the body", headers: mastodon, sentBody: swapped, date: now, actor: sameHostActor, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := signedInboxRequest(t, priv, tc.headers, body, tc.sentBody, tc.date, tc.mutate)
			err := PrecheckSignature(req.Header, tc.sentBody, tc.actor, now)
			if tc.wantErr && err == nil {
				t.Fatal("expected the precheck to refuse the request")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected the precheck to pass, got %v", err)
			}
		})
	}
}
