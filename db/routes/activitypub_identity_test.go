package routes

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestInboxAuthenticatesCachedActorsOnlyOnTheirOwnHost(t *testing.T) {
	for _, tc := range []struct {
		name, iri, domain, keyID string
		accepted                 bool
	}{
		{name: "mastodon", iri: "https://masto.example/users/alice", domain: "masto.example", keyID: "https://masto.example/users/alice#main-key", accepted: true},
		{name: "gotosocial", iri: "https://gts.example/users/alice", domain: "gts.example", keyID: "https://gts.example/users/alice/main-key", accepted: true},
		{name: "bookwyrm", iri: "https://bw.example/user/alice", domain: "bw.example", keyID: "https://bw.example/user/alice/#main-key", accepted: true},
		// A cached record whose stored domain differs from its IRI.
		{name: "inconsistent-cache", iri: "https://origin.example/users/carol", domain: "peer.example", keyID: "https://origin.example/users/carol#main-key"},
		{name: "foreign-key-host", iri: "https://masto.example/users/bob", domain: "masto.example", keyID: "https://other.example/users/bob#main-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ORIGIN", "https://local.example")
			t.Setenv("POCKETBASE_PROXY_SECRET", "inbox-test-secret")
			app, err := pbtests.NewTestApp(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(app.Cleanup)
			drainTestAppStartupLogs(t, app)
			actors := core.NewBaseCollection("activitypub_actors")
			actors.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "inbox"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"})
			if err := app.Save(actors); err != nil {
				t.Fatal(err)
			}
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				t.Fatal(err)
			}
			remote := core.NewRecord(actors)
			remote.Set("iri", tc.iri)
			remote.Set("domain", tc.domain)
			remote.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
			if err := app.Save(remote); err != nil {
				t.Fatal(err)
			}
			recipient := core.NewRecord(actors)
			recipient.Set("inbox", "https://local.example/api/v1/activitypub/user/bob/inbox")
			if err := app.Save(recipient); err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(`{"@context":"https://www.w3.org/ns/activitystreams","type":"Like","id":"%s/likes/1","actor":%q,"object":"https://local.example/api/v1/trail/missing"}`, tc.iri, tc.iri)
			path := "/api/v1/activitypub/user/bob/inbox"
			req := httptest.NewRequest(http.MethodPost, "https://local.example"+path, strings.NewReader(body))
			req.Header.Set("X-Internal-Secret", "inbox-test-secret")
			req.Header.Set("X-Forwarded-Path", path)
			req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
			signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
			if err != nil {
				t.Fatal(err)
			}
			if err := signer.SignRequest(key, tc.keyID, req, []byte(body)); err != nil {
				t.Fatal(err)
			}
			event := &core.RequestEvent{App: app}
			event.Request = req
			event.Response = httptest.NewRecorder()
			err = ActivitypubActivityProcess(event)
			var apiErr *router.ApiError
			refused := errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
			if tc.accepted && err != nil {
				t.Fatalf("signed activity refused: %v", err)
			}
			if !tc.accepted && !refused {
				t.Fatalf("signed activity was not refused as unauthorized: %v", err)
			}
		})
	}
}
