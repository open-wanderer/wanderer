package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
)

func TestVerifySignatureBindsKeyIDToActivityActorHost(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	actor := "https://remote.example/api/v1/activitypub/user/alice"
	body := []byte(`{"type":"Create"}`)
	for _, tc := range []struct {
		keyID    string
		signer   *rsa.PrivateKey
		verified bool
	}{
		{keyID: actor + "#main-key", signer: key, verified: true},
		{keyID: actor + "/main-key", signer: key, verified: true},
		{keyID: actor + "/#main-key", signer: key, verified: true},
		{keyID: actor, signer: key, verified: true},
		{keyID: "https://REMOTE.example/api/v1/activitypub/user/alice#main-key", signer: key, verified: true},
		{keyID: "https://other.example/api/v1/activitypub/user/alice#main-key", signer: key},
		{keyID: "https://remote.example.other.example/key#main-key", signer: key},
		{keyID: "https://remote.example:8443/api/v1/activitypub/user/alice#main-key", signer: key},
		{keyID: "https://alice@remote.example/api/v1/activitypub/user/alice#main-key", signer: key},
		{keyID: "acct:alice@remote.example", signer: key},
		{keyID: "ftp://remote.example/api/v1/activitypub/user/alice#main-key", signer: key},
		{keyID: "/api/v1/activitypub/user/alice#main-key", signer: key},
		// The key id is only a hint: the signature is checked against the
		// actor's stored key, so naming that key does not help another key.
		{keyID: actor + "#main-key", signer: otherKey},
	} {
		req, err := http.NewRequest("POST", "https://local.example/inbox", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Forwarded-Path", "/inbox")
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "Date", "Digest"}, httpsig.Signature, 60)
		if err != nil {
			t.Fatal(err)
		}
		if err := signer.SignRequest(tc.signer, tc.keyID, req, body); err != nil {
			t.Fatal(err)
		}
		verified, err := VerifySignature(nil, req, publicKey, body, actor)
		if tc.verified && (!verified || err != nil) {
			t.Fatalf("valid signature with key id %q rejected: %v", tc.keyID, err)
		}
		if !tc.verified && (verified || err == nil) {
			t.Fatalf("signature with key id %q accepted (own key: %v)", tc.keyID, tc.signer == key)
		}
	}
}
