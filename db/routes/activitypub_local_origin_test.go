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
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestInboxUsesLocalStoredKeysForOriginsWithPortsAndWWW(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	for _, origin := range []string{"http://local.example", "http://localhost:3000", "https://local.example:8443", "https://local.example:443", "https://www.local.example"} {
		t.Run(origin, func(t *testing.T) {
			t.Setenv("ORIGIN", origin)
			t.Setenv("POCKETBASE_PROXY_SECRET", "local-key-test-secret")
			app, err := pbtests.NewTestApp(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			app.Settings().Logs.MaxDays = 0
			t.Cleanup(app.Cleanup)
			drainTestAppStartupLogs(t, app)
			add := func(name string, fields ...core.Field) *core.Collection {
				collection := core.NewBaseCollection(name)
				collection.Fields.Add(fields...)
				if err := app.Save(collection); err != nil {
					t.Fatal(err)
				}
				return collection
			}
			save := func(collection *core.Collection, values map[string]any) *core.Record {
				record := core.NewRecord(collection)
				record.Load(values)
				if err := app.Save(record); err != nil {
					t.Fatal(err)
				}
				return record
			}
			actors := add("activitypub_actors", &core.TextField{Name: "iri"}, &core.TextField{Name: "inbox"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.BoolField{Name: "is_local"}, &core.TextField{Name: "user"}, &core.TextField{Name: "preferred_username"})
			parsed, _ := url.Parse(origin)
			localDomain := strings.TrimPrefix(parsed.Hostname(), "www.")
			iri := origin + "/api/v1/activitypub/user/alice"
			actor := save(actors, map[string]any{"iri": iri, "domain": localDomain, "public_key": publicKey, "is_local": true, "preferred_username": "alice"})
			path := "/api/v1/activitypub/user/bob/inbox"
			recipient := save(actors, map[string]any{"iri": origin + "/api/v1/activitypub/user/bob", "inbox": origin + path, "domain": localDomain, "is_local": true, "user": "bob", "preferred_username": "bob"})
			settings := add("settings", &core.TextField{Name: "user"}, &core.JSONField{Name: "notifications"})
			save(settings, map[string]any{"user": "bob", "notifications": map[string]any{"trail_like": map[string]any{"web": true, "email": false}}})
			notifications := add("notifications", &core.TextField{Name: "type"}, &core.JSONField{Name: "metadata"}, &core.TextField{Name: "recipient"}, &core.TextField{Name: "author"}, &core.BoolField{Name: "seen"})
			trails := add("trails", &core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.TextField{Name: "author"})
			trail := save(trails, map[string]any{"iri": origin + "/api/v1/trail/present", "name": "Present", "author": recipient.Id})
			original := getInboxActorByIRI
			refreshes := 0
			getInboxActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
				refreshes++
				return nil, errors.New("local actor must not be remotely refreshed")
			}
			t.Cleanup(func() { getInboxActorByIRI = original })
			for _, valid := range []bool{true, false} {
				body := fmt.Sprintf(`{"type":"Like","id":%q,"actor":%q,"object":%q}`, iri+"/like/1", iri, trail.GetString("iri"))
				req := httptest.NewRequest(http.MethodPost, origin+path, strings.NewReader(body))
				req.Header.Set("X-Internal-Secret", "local-key-test-secret")
				req.Header.Set("X-Forwarded-Path", path)
				req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
				signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
				if err != nil {
					t.Fatal(err)
				}
				signingKey := key
				if !valid {
					signingKey = wrongKey
				}
				if err := signer.SignRequest(signingKey, iri+"#main-key", req, []byte(body)); err != nil {
					t.Fatal(err)
				}
				response := httptest.NewRecorder()
				event := &core.RequestEvent{App: app}
				event.Request = req
				event.Response = response
				err = ActivitypubActivityProcess(event)
				if valid {
					if err != nil || response.Code != http.StatusOK {
						t.Fatalf("local signed activity rejected: %v status=%d", err, response.Code)
					}
					notification, err := app.FindFirstRecordByData(notifications, "author", actor.Id)
					if err != nil || notification.GetString("recipient") != recipient.Id {
						t.Fatalf("local notification not delivered: %v", err)
					}
				} else {
					var apiErr *router.ApiError
					if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
						t.Fatalf("wrong local key accepted: %v", err)
					}
				}
			}
			if refreshes != 0 {
				t.Fatalf("local signer caused %d remote refreshes", refreshes)
			}
			count, err := app.CountRecords(notifications)
			if err != nil || count != 1 {
				t.Fatalf("notification count=%d error=%v", count, err)
			}
		})
	}
}
