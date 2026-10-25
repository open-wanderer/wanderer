package routes

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

type activityProcessingLoggerApp struct {
	core.App
	logger      *slog.Logger
	mailFailure bool
}

func (app activityProcessingLoggerApp) Logger() *slog.Logger { return app.logger }

type failedNotificationMailer struct{}

func (failedNotificationMailer) Send(*mailer.Message) error {
	return errors.New("sendmail is unavailable")
}

func (app activityProcessingLoggerApp) NewMailClient() mailer.Mailer {
	if app.mailFailure {
		return failedNotificationMailer{}
	}
	return app.App.NewMailClient()
}

func TestVerifiedRejectedInboxActivityLogsWarningAndKeepsAcknowledgement(t *testing.T) {
	testInboxProcessingWarning(t, false)
}

func TestVerifiedAppliedInboxActivityLogsEmailFailureSeparately(t *testing.T) {
	// Production resolves its notification template from the database cwd.
	t.Chdir("..")
	testInboxProcessingWarning(t, true)
}

func testInboxProcessingWarning(t *testing.T, mailFailure bool) {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_PROXY_SECRET", "inbox-test-secret")
	base, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(base.Cleanup)
	drainTestAppStartupLogs(t, base)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "inbox"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.BoolField{Name: "is_local"}, &core.TextField{Name: "user"}, &core.TextField{Name: "username"}, &core.TextField{Name: "preferred_username"})
	if err := base.Save(actors); err != nil {
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
	remoteIRI := "https://remote.example/api/v1/activitypub/user/alice"
	remote := core.NewRecord(actors)
	remote.Set("iri", remoteIRI)
	remote.Set("domain", "remote.example")
	remote.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
	if err := base.Save(remote); err != nil {
		t.Fatal(err)
	}
	recipient := core.NewRecord(actors)
	recipient.Set("inbox", "https://local.example/api/v1/activitypub/user/bob/inbox")
	recipient.Set("is_local", true)
	recipient.Set("preferred_username", "bob")
	if err := base.Save(recipient); err != nil {
		t.Fatal(err)
	}
	objectIRI := "https://local.example/api/v1/trails/missing"
	if mailFailure {
		add := func(name string, fields ...core.Field) *core.Collection {
			collection := core.NewBaseCollection(name)
			collection.Fields.Add(fields...)
			if err := base.Save(collection); err != nil {
				t.Fatal(err)
			}
			return collection
		}
		save := func(collection *core.Collection, fields map[string]any) *core.Record {
			record := core.NewRecord(collection)
			record.Load(fields)
			if err := base.Save(record); err != nil {
				t.Fatal(err)
			}
			return record
		}
		users, err := base.FindCollectionByNameOrId("users")
		if err != nil {
			users = core.NewAuthCollection("users")
			if err := base.Save(users); err != nil {
				t.Fatal(err)
			}
		}
		user := save(users, map[string]any{"email": "inbox-email-recipient@example.com", "username": "inbox-email-recipient", "password": "long-test-password"})
		recipient.Set("user", user.Id)
		recipient.Set("is_local", true)
		recipient.Set("username", "Bob")
		if err := base.Save(recipient); err != nil {
			t.Fatal(err)
		}
		settings := add("settings", &core.TextField{Name: "user"}, &core.JSONField{Name: "notifications"})
		save(settings, map[string]any{"user": user.Id, "notifications": map[string]any{}})
		add("notifications", &core.TextField{Name: "type"}, &core.JSONField{Name: "metadata"}, &core.BoolField{Name: "seen"}, &core.TextField{Name: "recipient"}, &core.TextField{Name: "author"})
		trails := add("trails", &core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.TextField{Name: "author"})
		add("trail_like", &core.TextField{Name: "trail"}, &core.TextField{Name: "actor"})
		objectIRI = "https://local.example/api/v1/trail/present"
		save(trails, map[string]any{"iri": objectIRI, "name": "Present", "author": recipient.Id})
	}
	var logs bytes.Buffer
	app := activityProcessingLoggerApp{App: base, logger: slog.New(slog.NewJSONHandler(&logs, nil)), mailFailure: mailFailure}
	body := fmt.Sprintf(`{"@context":"https://www.w3.org/ns/activitystreams","type":"Like","id":"https://remote.example/activities/like-1","actor":%q,"object":%q}`, remoteIRI, objectIRI)
	path := "/api/v1/activitypub/user/bob/inbox"
	req := httptest.NewRequest(http.MethodPost, "https://local.example"+path, strings.NewReader(body))
	req.Header.Set("X-Internal-Secret", "inbox-test-secret")
	req.Header.Set("X-Forwarded-Path", path)
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
	if err != nil {
		t.Fatal(err)
	}
	if err := signer.SignRequest(key, remoteIRI+"#main-key", req, []byte(body)); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: app}
	event.Request = req
	event.Response = response
	if err := ActivitypubActivityProcess(event); err != nil {
		t.Fatalf("verified processing path failed before acknowledgement: %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("inbox HTTP contract changed: %d", response.Code)
	}
	message := "ActivityPub activity could not be processed"
	if mailFailure {
		message = "ActivityPub notification email could not be delivered"
		for _, collection := range []string{"trail_like", "notifications"} {
			count, err := base.CountRecords(collection)
			if err != nil || count != 1 {
				t.Fatalf("mail failure lost committed %s: count=%d error=%v", collection, count, err)
			}
		}
		if strings.Contains(logs.String(), "ActivityPub activity could not be processed") {
			t.Fatalf("applied activity was reported as rejected: %s", logs.String())
		}
	}
	if strings.Count(logs.String(), message) != 1 || !strings.Contains(logs.String(), `"type":"Like"`) || !strings.Contains(logs.String(), remote.Id) {
		t.Fatalf("rejected signed activity stayed silent: %s", logs.String())
	}
}
