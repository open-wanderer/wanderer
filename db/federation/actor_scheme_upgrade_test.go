package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"pocketbase/util"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
)

func TestActorHTTPSUpgradeUsesFetchedKeyAndKeepsExistingIdentity(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
	app := setupActorDeleteTestApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(&core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.DateField{Name: "last_fetched"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	requested := "http://remote.example:8443/api/v1/activitypub/user/alice"
	document := strings.Replace(requested, "http://", "https://", 1)
	old, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&fresh.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	actor := newTestActor(t, app, requested, false)
	actor.Set("domain", "remote.example:8443")
	// As after an upgrade, neither the cached key nor its timestamp can be
	// used to authenticate. The replacement key must come from the origin.
	actor.Set("public_key", "")
	actor.Set("last_fetched", time.Time{})
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	previous := newActorFetchHTTPClient
	t.Cleanup(func() { newActorFetchHTTPClient = previous })
	var fetches atomic.Int32
	newActorFetchHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(req *http.Request) (*http.Response, error) {
			fetches.Add(1)
			if req.URL.String() != requested {
				t.Errorf("unexpected actor fetch %s", req.URL)
			}
			body := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":%q}}`, document, document+"/inbox", document+"/outbox", document+"#main-key", document, publicKey)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: req, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
	}
	fetched, err := GetActorByIRI(app, context.Background(), requested, false)
	if err != nil || fetched == nil {
		t.Fatalf("same-host upgrade failed: %v", err)
	}
	if fetched.Id != actor.Id || fetched.GetString("iri") != requested || fetched.GetString("public_key") != publicKey {
		t.Fatalf("identity or fetched key changed: %v", fetched)
	}
	if !CachedActorIdentityMatchesOrigin(fetched) {
		t.Fatal("refreshed actor failed cache gate")
	}
	cached, err := GetActorByIRI(app, t.Context(), requested, false)
	if err != nil || cached.Id != actor.Id || fetches.Load() != 1 {
		t.Fatalf("actor split or repeated refresh: actor=%v err=%v fetches=%d", cached, err, fetches.Load())
	}
	for _, tc := range []struct {
		name  string
		key   *rsa.PrivateKey
		valid bool
	}{{"fresh", fresh, true}, {"former", old, false}} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "https://local.example/inbox", nil)
			req.Header.Set("X-Forwarded-Path", "/inbox")
			req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
			signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
			if err != nil {
				t.Fatal(err)
			}
			if err := signer.SignRequest(tc.key, document+"#main-key", req, []byte{}); err != nil {
				t.Fatal(err)
			}
			valid, err := util.VerifySignature(nil, req, fetched.GetString("public_key"), []byte{}, requested)
			if tc.valid && (!valid || err != nil) {
				t.Fatalf("new origin key rejected: %v", err)
			}
			if !tc.valid && valid {
				t.Fatal("former key authenticated after refresh")
			}
		})
	}
}

func TestActorFetchHTTPSUpgradeRejectsDifferentIdentity(t *testing.T) {
	requested := "http://remote.example/api/v1/activitypub/user/alice"
	for _, tc := range []struct{ name, requested, document, owner string }{
		{"path", requested, "https://remote.example/api/v1/activitypub/user/bob", "https://remote.example/api/v1/activitypub/user/bob"},
		{"host", requested, "https://elsewhere.example/api/v1/activitypub/user/alice", "https://elsewhere.example/api/v1/activitypub/user/alice"},
		{"port", requested, "https://remote.example:443/api/v1/activitypub/user/alice", "https://remote.example:443/api/v1/activitypub/user/alice"},
		{"query", requested, "https://remote.example/api/v1/activitypub/user/alice?alias=1", "https://remote.example/api/v1/activitypub/user/alice?alias=1"},
		{"downgrade", strings.Replace(requested, "http://", "https://", 1), requested, requested},
		{"owner", requested, strings.Replace(requested, "http://", "https://", 1), requested},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
			app := setupActorDeleteTestApp(t)
			previous := newActorFetchHTTPClient
			t.Cleanup(func() { newActorFetchHTTPClient = previous })
			newActorFetchHTTPClient = func() *http.Client {
				return &http.Client{Transport: identityLookupTransport(func(req *http.Request) (*http.Response, error) {
					body := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"key"}}`, tc.document, tc.document+"/inbox", tc.document+"/outbox", tc.document+"#main-key", tc.owner)
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: req, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}
			}
			if actor, err := GetActorByIRI(app, t.Context(), tc.requested, false); err == nil || actor != nil {
				t.Fatalf("different identity accepted: actor=%v err=%v", actor, err)
			}
			if count, err := app.CountRecords("activitypub_actors"); err != nil || count != 0 {
				t.Fatalf("rejected actor persisted: count=%d err=%v", count, err)
			}
		})
	}
}
