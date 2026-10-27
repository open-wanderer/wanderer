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
	"pocketbase/util"
	"strings"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestInboxRefreshBudgetRejectsWipedActorThenAcceptsRetry(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_PROXY_SECRET", "retry-secret")
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "inbox"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.BoolField{Name: "is_local"}, &core.TextField{Name: "preferred_username"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(actors)
	actor.Set("iri", "https://remote.example/users/alice")
	actor.Set("domain", "remote.example")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	recipient := core.NewRecord(actors)
	recipient.Set("is_local", true)
	recipient.Set("preferred_username", "bob")
	recipient.Set("iri", "https://local.example/api/v1/activitypub/user/bob")
	recipient.Set("inbox", recipient.GetString("iri")+"/inbox")
	if err := app.Save(recipient); err != nil {
		t.Fatal(err)
	}
	freshKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&freshKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	previous := getInboxActorByIRI
	t.Cleanup(func() { getInboxActorByIRI = previous })
	// Keep the production 30-request policy, shortening only the window so
	// a real expiration/retry can be tested without a minute-long wait.
	limiter := util.NewRateLimiter(30, 500*time.Millisecond)
	identifier := "actor:" + recipient.Id
	for i := 0; i < 30; i++ {
		if err := limiter.CheckRateLimit(identifier, "remote.example"); err != nil {
			t.Fatalf("request %d refused before budget: %v", i+1, err)
		}
	}
	getInboxActorByIRI = func(app core.App, ctx context.Context, iri string, includeFollows bool) (*core.Record, error) {
		if ctx.Value("actor") != identifier || iri != actor.GetString("iri") || includeFollows {
			t.Fatal("refresh changed caller or identity")
		}
		if err := limiter.CheckRateLimit(identifier, "remote.example"); err != nil {
			return actor, err
		}
		refreshed, err := app.FindRecordById(actors, actor.Id)
		if err != nil {
			return nil, err
		}
		refreshed.Set("public_key", publicKey)
		if err := app.Save(refreshed); err != nil {
			t.Errorf("could not persist the origin refresh: %v", err)
			return nil, err
		}
		return refreshed, nil
	}
	process := func() error {
		path := "/api/v1/activitypub/user/bob/inbox"
		body := fmt.Sprintf(`{"type":"Like","id":%q,"actor":%q,"object":"https://local.example/api/v1/trail/missing"}`, actor.GetString("iri")+"/like/retry", actor.GetString("iri"))
		req := httptest.NewRequest(http.MethodPost, "https://local.example"+path, strings.NewReader(body))
		req.Header.Set("X-Internal-Secret", "retry-secret")
		req.Header.Set("X-Forwarded-Path", path)
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
		if err != nil {
			t.Fatal(err)
		}
		if err := signer.SignRequest(freshKey, actor.GetString("iri")+"#main-key", req, []byte(body)); err != nil {
			t.Fatal(err)
		}
		e := &core.RequestEvent{App: app}
		e.Request = req
		e.Response = httptest.NewRecorder()
		return ActivitypubActivityProcess(e)
	}
	var apiErr *router.ApiError
	if err := process(); !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("exhausted refresh budget was accepted: %v", err)
	}
	stored, err := app.FindRecordById(actors, actor.Id)
	if err != nil || stored.GetString("public_key") != "" {
		t.Fatal("rejected refresh restored an unverified cached key")
	}
	<-time.After(525 * time.Millisecond)
	if err := process(); err != nil {
		t.Fatalf("signed activity retry after budget expiration refused: %v", err)
	}
	stored, err = app.FindRecordById(actors, actor.Id)
	if err != nil || stored.GetString("public_key") != publicKey {
		t.Fatal("retry did not persist the fresh verified key")
	}
}
