package routes

import (
	"context"
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

func TestInboxRefreshesInconsistentActorBeforeCheckingSignature(t *testing.T) {
	for _, scenario := range []string{"fresh-key", "cached-key", "refresh-failed", "still-inconsistent"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv("ORIGIN", "https://local.example")
			t.Setenv("POCKETBASE_PROXY_SECRET", "refresh-test-secret")
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
			cachedKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			freshKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatal(err)
			}
			publicKey := func(key *rsa.PrivateKey) string {
				der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
				if err != nil {
					t.Fatal(err)
				}
				return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
			}
			iri := "https://remote.example/users/alice"
			actor := core.NewRecord(actors)
			actor.Set("iri", iri)
			actor.Set("domain", "other.example")
			actor.Set("public_key", publicKey(cachedKey))
			if err := app.Save(actor); err != nil {
				t.Fatal(err)
			}
			recipient := core.NewRecord(actors)
			recipient.Set("inbox", "https://local.example/api/v1/activitypub/user/bob/inbox")
			if err := app.Save(recipient); err != nil {
				t.Fatal(err)
			}
			original := getInboxActorByIRI
			calls := 0
			getInboxActorByIRI = func(gotApp core.App, ctx context.Context, gotIRI string, includeFollows bool) (*core.Record, error) {
				calls++
				if gotApp != app || gotIRI != iri || includeFollows || ctx.Value("actor") != "actor:"+recipient.Id {
					t.Fatalf("incorrect refresh context: iri=%q follows=%v actor=%v", gotIRI, includeFollows, ctx.Value("actor"))
				}
				if scenario == "refresh-failed" {
					return actor, errors.New("origin unavailable")
				}
				refreshed := actor.Clone()
				refreshed.Set("public_key", publicKey(freshKey))
				if scenario != "still-inconsistent" {
					refreshed.Set("domain", "remote.example")
				}
				return refreshed, nil
			}
			t.Cleanup(func() { getInboxActorByIRI = original })
			path := "/api/v1/activitypub/user/bob/inbox"
			body := fmt.Sprintf(`{"type":"Like","id":%q,"actor":%q,"object":"https://local.example/api/v1/trail/missing"}`, iri+"/like/1", iri)
			req := httptest.NewRequest(http.MethodPost, "https://local.example"+path, strings.NewReader(body))
			req.Header.Set("X-Internal-Secret", "refresh-test-secret")
			req.Header.Set("X-Forwarded-Path", path)
			req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
			signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
			if err != nil {
				t.Fatal(err)
			}
			signingKey := freshKey
			if scenario == "cached-key" {
				signingKey = cachedKey
			}
			if err := signer.SignRequest(signingKey, iri+"/main-key", req, []byte(body)); err != nil {
				t.Fatal(err)
			}
			e := &core.RequestEvent{App: app}
			e.Request = req
			e.Response = httptest.NewRecorder()
			err = ActivitypubActivityProcess(e)
			if calls != 1 {
				t.Fatalf("refresh count=%d", calls)
			}
			if scenario == "fresh-key" {
				if err != nil {
					t.Fatalf("refreshed signature refused: %v", err)
				}
			} else {
				var apiErr *router.ApiError
				if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
					t.Fatalf("expected authorization refusal: %v", err)
				}
			}
		})
	}
}
