package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"pocketbase/federation"
	"pocketbase/routes"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type localHandleFixture struct {
	app      *core.BaseApp
	actor    *core.Record
	settings *core.Record
	handler  http.Handler
	token    string
	decoys   []*core.Record
}

func newLocalHandleFixture(t *testing.T, origin string, decoys bool) *localHandleFixture {
	t.Helper()
	app := newRealSchemaApp(t)
	t.Setenv("ORIGIN", origin)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.Set("username", "Alice")
	user.Set("email", "alice@example.test")
	user.SetPassword("local-handle-test-password")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	settingsCollection, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewRecord(settingsCollection)
	settings.Set("user", user.Id)
	settings.Set("bio", "Local profile biography")
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	f := &localHandleFixture{app: app, settings: settings}
	if decoys {
		actors, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"alice", "other-user"} {
			for _, domain := range []string{"local.example", "www.local.example"} {
				// The canonical local name/domain pair is unique in the real schema.
				if name == "alice" && domain == "local.example" {
					continue
				}
				actor := core.NewRecord(actors)
				actor.Set("username", name)
				actor.Set("preferred_username", name)
				actor.Set("domain", domain)
				actor.Set("iri", "https://remote.example/"+domain+"/"+name)
				actor.Set("inbox", actor.GetString("iri")+"/inbox")
				actor.Set("public_key", "remote-profile-key")
				if err := app.Save(actor); err != nil {
					t.Fatal(err)
				}
				actor, err = app.FindRecordById("activitypub_actors", actor.Id)
				if err != nil {
					t.Fatal(err)
				}
				f.decoys = append(f.decoys, actor)
			}
		}
	}
	f.actor, err = util.ActorFromUser(app, user)
	if err != nil {
		t.Fatal(err)
	}
	f.actor, err = app.FindRecordById("activitypub_actors", f.actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	f.token, err = user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/activitypub/actor", routes.ActivitypubActor)
	f.handler, err = router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *localHandleFixture) request(t *testing.T, handle string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/activitypub/actor?resource="+url.QueryEscape("acct:"+handle)+"&follows=true", nil)
	// Local profile resolution needs no network. A cancelled request context
	// makes an accidental WebFinger or actor fetch fail without contacting a host.
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(ctx)
	if authenticated {
		request.Header.Set("Authorization", f.token)
	}
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	return response
}

func (f *localHandleFixture) assertActor(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("profile response status=%d, want 200: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Actor map[string]any `json:"actor"`
		Error any            `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Actor["id"] != f.actor.Id || payload.Actor["iri"] != f.actor.GetString("iri") || payload.Actor["is_local"] != true || payload.Error != nil {
		t.Fatalf("profile did not resolve the stored local actor: %s", response.Body.String())
	}
	stored, err := f.app.FindRecordById("activitypub_actors", f.actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	f.assertLocalIdentity(t, stored)
	if payload.Actor["summary"] != f.settings.GetString("bio") {
		t.Errorf("local profile biography=%v, want %q", payload.Actor["summary"], f.settings.GetString("bio"))
	}
}

func (f *localHandleFixture) assertLocalIdentity(t *testing.T, actor *core.Record) {
	t.Helper()
	if actor == nil || actor.Id != f.actor.Id || !actor.GetBool("is_local") {
		t.Fatalf("lookup did not return the stored local actor: %v", actor)
	}
	for _, field := range []string{"iri", "is_local", "user", "username", "preferred_username", "domain", "public_key", "private_key", "inbox", "outbox", "followers", "following", "published"} {
		if !reflect.DeepEqual(actor.Get(field), f.actor.Get(field)) {
			t.Errorf("local identity field %q changed after lookup", field)
		}
	}
}

func TestLocalActorHandlesUseCurrentOriginAndStoredDomain(t *testing.T) {
	for _, scenario := range []struct {
		name, origin string
		handles      []string
	}{
		{
			name: "www-origin", origin: "https://www.local.example",
			handles: []string{
				"alice", "ALICE", "Alice", "alice@local.example", "alice@www.local.example", "ALICE@www.local.example", "Alice@LOCAL.EXAMPLE", "alice@WWW.LOCAL.EXAMPLE",
				"@alice@www.local.example", "@Alice@www.local.example", "@ALICE@WWW.LOCAL.EXAMPLE",
			},
		},
		{
			name: "plain-origin", origin: "https://local.example",
			handles: []string{
				"alice", "ALICE", "Alice", "alice@local.example", "ALICE@local.example", "Alice@LOCAL.EXAMPLE",
				"@alice@local.example", "@Alice@LOCAL.EXAMPLE",
			},
		},
		{
			name: "origin-port", origin: "https://www.local.example:8443",
			handles: []string{
				"alice", "ALICE", "Alice", "alice@local.example", "alice@www.local.example", "ALICE@WWW.LOCAL.EXAMPLE",
				"alice@www.local.example:8443", "ALICE@WWW.LOCAL.EXAMPLE:8443",
				"@alice@www.local.example", "@Alice@WWW.LOCAL.EXAMPLE", "@ALICE@WWW.LOCAL.EXAMPLE:8443",
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newLocalHandleFixture(t, scenario.origin, false)
			if f.actor.GetString("domain") != "local.example" {
				t.Fatalf("ActorFromUser domain=%q, want canonical domain without www or port", f.actor.GetString("domain"))
			}
			for _, handle := range scenario.handles {
				t.Run(handle, func(t *testing.T) {
					f.assertActor(t, f.request(t, handle, false))
				})
			}
			for _, lookup := range []struct{ name, origin string }{
				{"exact-iri-current-origin", scenario.origin},
				{"exact-iri-after-origin-change", "https://next.local.example"},
			} {
				t.Run(lookup.name, func(t *testing.T) {
					t.Setenv("ORIGIN", lookup.origin)
					ctx, cancel := context.WithCancel(t.Context())
					cancel()
					actor, err := federation.GetActorByIRI(f.app, ctx, f.actor.GetString("iri"), true)
					if err != nil {
						t.Fatalf("exact local IRI lookup failed: %v", err)
					}
					f.assertLocalIdentity(t, actor)
				})
			}
		})
	}
}

func TestLocalActorHandleAliasesKeepProfilePrivacy(t *testing.T) {
	f := newLocalHandleFixture(t, "https://www.local.example", false)
	f.settings.Set("privacy", map[string]any{"account": "private"})
	if err := f.app.Save(f.settings); err != nil {
		t.Fatal(err)
	}
	for _, handle := range []string{"ALICE", "ALICE@www.local.example", "Alice@local.example", "@alice@www.local.example", "@Alice@WWW.LOCAL.EXAMPLE"} {
		t.Run(handle, func(t *testing.T) {
			response := f.request(t, handle, false)
			if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "profile is private") {
				t.Fatalf("anonymous private profile status=%d body=%s", response.Code, response.Body.String())
			}
			f.assertActor(t, f.request(t, handle, true))
		})
	}
	for _, lookup := range []struct{ name, origin string }{
		{"exact-iri-current-origin", "https://www.local.example"},
		{"exact-iri-after-origin-change", "https://next.local.example"},
	} {
		t.Run(lookup.name, func(t *testing.T) {
			t.Setenv("ORIGIN", lookup.origin)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			actor, err := federation.GetActorByIRI(f.app, ctx, f.actor.GetString("iri"), true)
			if !errors.Is(err, federation.ErrProfilePrivate) {
				t.Fatalf("anonymous exact private IRI lookup error=%v, want private profile", err)
			}
			f.assertLocalIdentity(t, actor)
			ownerCtx := context.WithValue(ctx, "actor", "actor:"+f.actor.Id)
			actor, err = federation.GetActorByIRI(f.app, ownerCtx, f.actor.GetString("iri"), true)
			if err != nil {
				t.Fatalf("owner exact private IRI lookup failed: %v", err)
			}
			f.assertLocalIdentity(t, actor)
		})
	}
}

func TestLocalActorHandlesIgnoreRemoteRecordsWithMatchingNames(t *testing.T) {
	f := newLocalHandleFixture(t, "https://www.local.example", true)
	for _, domain := range []string{"local.example", "www.local.example"} {
		t.Run(domain, func(t *testing.T) {
			f.assertActor(t, f.request(t, "alice@"+domain, false))
			response := f.request(t, "other-user@"+domain, false)
			if response.Code != http.StatusNotFound {
				t.Fatalf("missing local profile status=%d, want 404: %s", response.Code, response.Body.String())
			}
		})
	}
	for _, remote := range f.decoys {
		stored, err := f.app.FindRecordById("activitypub_actors", remote.Id)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored.FieldsData(), remote.FieldsData()) {
			t.Errorf("handle lookup changed remote record %s", remote.Id)
		}
	}
}
