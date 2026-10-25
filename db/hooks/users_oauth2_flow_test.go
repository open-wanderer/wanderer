package hooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The unit tests of the OAuth2 handlers build the hook event by hand, so they
// never reach PocketBase's own OAuth2 handler. The review of #1346 found that
// the interesting behaviour lives in the interaction with it: an account that
// is created unverified is treated as possibly pre-created on every later
// login, which invalidates the sessions of other devices, and confirming its
// email unlinks the provider. These tests drive the real auth-with-oauth2 route
// against a fake identity provider.

const oidcFlowCollection = "oidc_flow_users"

var generatedUsername = regexp.MustCompile(`^users[0-9]{6}$`)

type oidcFlow struct {
	t          *testing.T
	app        *tests.TestApp
	mux        http.Handler
	collection *core.Collection

	// trust is read on every login so a test can switch the opt-in on later,
	// as an administrator would after accounts already exist
	trust atomic.Bool

	mu     sync.Mutex
	claims map[string]any
}

type oidcLogin struct {
	status int
	token  string
	record *core.Record
	isNew  bool
}

// newOIDCFlow sets up an auth collection with the oidc, oidc2 and google
// providers pointed at a fake identity provider. Only the oidc slot is opted in
// to trusting its claims, and only while the trust switch is on.
func newOIDCFlow(t *testing.T, trusted bool) *oidcFlow {
	t.Helper()

	f := &oidcFlow{t: t}
	f.trust.Store(trusted)

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "expires_in": 3600})
		case "/userinfo":
			f.mu.Lock()
			defer f.mu.Unlock()
			json.NewEncoder(w).Encode(f.claims)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(idp.Close)

	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	f.app = app

	collection := core.NewAuthCollection(oidcFlowCollection)
	// as in the users collection: accounts may have no email, the username is
	// generated when the provider has none that fits
	collection.Fields.GetByName(core.FieldNameEmail).(*core.EmailField).Required = false
	collection.Fields.Add(&core.TextField{
		Name:                "username",
		Required:            true,
		Min:                 3,
		Max:                 20,
		Pattern:             `^[\w][\w\.\-]*$`,
		AutogeneratePattern: "users[0-9]{6}",
	})
	collection.AddIndex("idx_oidc_flow_users_username", true, "username COLLATE NOCASE", "")
	// sign-up through OAuth2 goes through the create rule
	collection.CreateRule = types.Pointer("")
	collection.OAuth2.Enabled = true
	collection.OAuth2.MappedFields.Username = "username"
	for _, name := range []string{"oidc", "oidc2", "google"} {
		collection.OAuth2.Providers = append(collection.OAuth2.Providers, core.OAuth2ProviderConfig{
			Name:         name,
			ClientId:     "client",
			ClientSecret: "secret",
			AuthURL:      idp.URL + "/authorize",
			TokenURL:     idp.URL + "/token",
			UserInfoURL:  idp.URL + "/userinfo",
		})
	}
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	f.collection = collection

	trustedProviders := func(provider string) bool {
		return provider == "oidc" && f.trust.Load()
	}
	app.OnRecordAuthWithOAuth2Request().BindFunc(OAuth2UsernameHandler(trustedProviders))
	app.OnRecordAuthWithOAuth2Request().BindFunc(OAuth2EmailHandler(trustedProviders))

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	err = app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		f.mux, err = e.Router.BuildMux()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	return f
}

func (f *oidcFlow) request(method, path, token string, body any) (int, map[string]any) {
	f.t.Helper()

	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			f.t.Fatal(err)
		}
	}

	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	response := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &response)

	return rec.Code, response
}

// login plays a login without an existing session, as the web app does
// (authWithOAuth2Code without create data), against the given claims.
func (f *oidcFlow) login(provider string, claims map[string]any) oidcLogin {
	f.t.Helper()

	f.mu.Lock()
	f.claims = claims
	f.mu.Unlock()

	status, response := f.request(http.MethodPost, "/api/collections/"+oidcFlowCollection+"/auth-with-oauth2", "", map[string]any{
		"provider":     provider,
		"code":         "code",
		"codeVerifier": "verifier",
		"redirectURL":  "http://localhost/login/redirect",
	})

	result := oidcLogin{status: status}
	if status != http.StatusOK {
		f.t.Logf("login via %s failed with %d: %v", provider, status, response)
		return result
	}

	result.token, _ = response["token"].(string)
	meta, _ := response["meta"].(map[string]any)
	result.isNew, _ = meta["isNew"].(bool)

	record, _ := response["record"].(map[string]any)
	id, _ := record["id"].(string)
	result.record = f.record(id)

	return result
}

// valid reports whether a token still passes the auth-refresh call that
// hooks.server.ts makes on every request.
func (f *oidcFlow) valid(token string) bool {
	f.t.Helper()
	status, _ := f.request(http.MethodPost, "/api/collections/"+oidcFlowCollection+"/auth-refresh", token, nil)
	return status == http.StatusOK
}

func (f *oidcFlow) record(id string) *core.Record {
	f.t.Helper()
	record, err := f.app.FindRecordById(f.collection, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return record
}

func (f *oidcFlow) externalAuths(record *core.Record) int {
	f.t.Helper()
	auths, err := f.app.FindAllExternalAuthsByRecord(record)
	if err != nil {
		f.t.Fatal(err)
	}
	return len(auths)
}

func (f *oidcFlow) createAccount(email string, verified bool) *core.Record {
	f.t.Helper()
	record := core.NewRecord(f.collection)
	record.SetEmail(email)
	record.SetPassword("1234567890")
	record.SetVerified(verified)
	if err := f.app.Save(record); err != nil {
		f.t.Fatal(err)
	}
	return record
}

func (f *oidcFlow) link(record *core.Record, provider, providerId string) {
	f.t.Helper()
	auth := core.NewExternalAuth(f.app)
	auth.SetCollectionRef(f.collection.Id)
	auth.SetRecordRef(record.Id)
	auth.SetProvider(provider)
	auth.SetProviderId(providerId)
	if err := f.app.Save(auth); err != nil {
		f.t.Fatal(err)
	}
}

func requireLogin(t *testing.T, l oidcLogin) oidcLogin {
	t.Helper()
	if l.status != http.StatusOK || l.record == nil {
		t.Fatalf("expected a successful login, got status %d", l.status)
	}
	return l
}

// cloudflareClaims are the claims reported in #1342: no email_verified and no
// preferred_username.
func cloudflareClaims(sub, email, name string) map[string]any {
	return map[string]any{"sub": sub, "email": email, "name": name}
}

func TestOAuth2OIDCLoginFlow(t *testing.T) {
	t.Run("trusted slot: repeated logins keep every session valid and confirming does not unlink", func(t *testing.T) {
		f := newOIDCFlow(t, true)
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")

		first := requireLogin(t, f.login("oidc", claims))
		if !first.isNew {
			t.Fatal("expected the first login to create an account")
		}
		if got := first.record.Email(); got != "david@example.com" {
			t.Fatalf("expected the claimed email, got %q", got)
		}
		if !first.record.Verified() {
			t.Fatal("expected the account to be created verified")
		}
		if got := first.record.GetString("username"); got != "David_Schwartz" {
			t.Fatalf("expected the display name as username, got %q", got)
		}

		second := requireLogin(t, f.login("oidc", claims))
		third := requireLogin(t, f.login("oidc", claims))
		for _, l := range []oidcLogin{second, third} {
			if l.isNew || l.record.Id != first.record.Id {
				t.Fatalf("expected the same account, got %s (new=%v)", l.record.Id, l.isNew)
			}
		}
		for i, token := range []string{first.token, second.token, third.token} {
			if !f.valid(token) {
				t.Fatalf("token of login %d was invalidated by a later login", i+1)
			}
		}

		// confirming the email, or an administrator ticking verified
		confirmed := f.record(first.record.Id)
		confirmed.SetVerified(true)
		if err := f.app.Save(confirmed); err != nil {
			t.Fatal(err)
		}
		if got := f.externalAuths(confirmed); got != 1 {
			t.Fatalf("expected the provider link to survive, got %d external auths", got)
		}

		fourth := requireLogin(t, f.login("oidc", claims))
		if fourth.isNew || fourth.record.Id != first.record.Id {
			t.Fatalf("expected the same account after confirming, got %s (new=%v)", fourth.record.Id, fourth.isNew)
		}
		if got := fourth.record.Email(); got != "david@example.com" {
			t.Fatalf("expected the email to be kept, got %q", got)
		}
	})

	t.Run("without the opt-in nothing changes", func(t *testing.T) {
		f := newOIDCFlow(t, false)
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")

		first := requireLogin(t, f.login("oidc", claims))
		if first.record.Email() != "" || !first.record.Verified() {
			t.Fatalf("expected a verified account without email, got %q verified=%v", first.record.Email(), first.record.Verified())
		}
		if got := first.record.GetString("username"); !generatedUsername.MatchString(got) {
			t.Fatalf("expected a generated username, got %q", got)
		}

		second := requireLogin(t, f.login("oidc", claims))
		if second.record.Id != first.record.Id || !f.valid(first.token) {
			t.Fatal("expected repeated logins to keep the account and its sessions")
		}
	})

	t.Run("an explicit email_verified claim is respected", func(t *testing.T) {
		for name, verified := range map[string]any{"false": false, "null": nil, "false string": "false", "malformed": "maybe"} {
			t.Run(name, func(t *testing.T) {
				f := newOIDCFlow(t, true)
				claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")
				claims["email_verified"] = verified

				first := requireLogin(t, f.login("oidc", claims))
				if first.record.Email() != "" || !first.record.Verified() {
					t.Fatalf("expected a verified account without email, got %q verified=%v", first.record.Email(), first.record.Verified())
				}

				second := requireLogin(t, f.login("oidc", claims))
				if second.record.Id != first.record.Id || !f.valid(first.token) {
					t.Fatal("expected repeated logins to keep the account and its sessions")
				}
			})
		}
	})

	t.Run("a verified email_verified claim is handled by PocketBase", func(t *testing.T) {
		f := newOIDCFlow(t, true)
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")
		claims["email_verified"] = true

		first := requireLogin(t, f.login("oidc", claims))
		if first.record.Email() != "david@example.com" || !first.record.Verified() {
			t.Fatalf("expected a verified account with email, got %q verified=%v", first.record.Email(), first.record.Verified())
		}
	})

	t.Run("an existing account without email gets it on the next login", func(t *testing.T) {
		f := newOIDCFlow(t, false)
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")

		// created before the opt-in: verified, no email, generated username
		first := requireLogin(t, f.login("oidc", claims))
		username := first.record.GetString("username")

		f.trust.Store(true)
		second := requireLogin(t, f.login("oidc", claims))
		if second.isNew || second.record.Id != first.record.Id {
			t.Fatalf("expected the same account, got %s (new=%v)", second.record.Id, second.isNew)
		}
		if got := second.record.Email(); got != "david@example.com" {
			t.Fatalf("expected the email to be filled in, got %q", got)
		}
		if !second.record.Verified() {
			t.Fatal("expected the account to stay verified")
		}
		if got := second.record.GetString("username"); got != username {
			t.Fatalf("expected the username to stay %q, got %q", username, got)
		}
		if !f.valid(second.token) {
			t.Fatal("expected the new session to be valid")
		}

		third := requireLogin(t, f.login("oidc", claims))
		if !f.valid(second.token) || !f.valid(third.token) {
			t.Fatal("expected later logins to keep the sessions valid")
		}
	})

	t.Run("an email used by another account is not taken", func(t *testing.T) {
		f := newOIDCFlow(t, true)
		owner := f.createAccount("david@example.com", true)
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")

		// a new identity is not linked to the account that owns the address
		first := requireLogin(t, f.login("oidc", claims))
		if !first.isNew || first.record.Id == owner.Id {
			t.Fatalf("expected a new account, got %s (new=%v)", first.record.Id, first.isNew)
		}
		if first.record.Email() != "" {
			t.Fatalf("expected no email, got %q", first.record.Email())
		}
		if got := f.externalAuths(f.record(owner.Id)); got != 0 {
			t.Fatalf("expected the owner to stay unlinked, got %d external auths", got)
		}

		// an existing identity keeps working instead of failing the uniqueness check
		second := requireLogin(t, f.login("oidc", claims))
		if second.record.Id != first.record.Id || second.record.Email() != "" {
			t.Fatalf("expected the same account without email, got %s %q", second.record.Id, second.record.Email())
		}
		if !f.valid(first.token) {
			t.Fatal("expected repeated logins to keep the sessions valid")
		}
	})

	t.Run("an account left unverified by the first fix is healed", func(t *testing.T) {
		f := newOIDCFlow(t, true)
		legacy := f.createAccount("david@example.com", false)
		f.link(legacy, "oidc", "cf-1")
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")

		first := requireLogin(t, f.login("oidc", claims))
		if first.record.Id != legacy.Id || !first.record.Verified() {
			t.Fatalf("expected the legacy account to be verified, got %s verified=%v", first.record.Id, first.record.Verified())
		}
		if got := f.externalAuths(first.record); got != 1 {
			t.Fatalf("expected the provider link to be kept, got %d external auths", got)
		}

		second := requireLogin(t, f.login("oidc", claims))
		if second.record.Id != legacy.Id || !f.valid(first.token) || !f.valid(second.token) {
			t.Fatal("expected later logins to keep the sessions valid")
		}
	})

	t.Run("other slots and built-in providers keep their behaviour", func(t *testing.T) {
		f := newOIDCFlow(t, true)

		for _, provider := range []string{"oidc2", "google"} {
			claims := cloudflareClaims("sub-"+provider, provider+"@example.com", "Real Name "+provider)

			first := requireLogin(t, f.login(provider, claims))
			if first.record.Email() != "" || !first.record.Verified() {
				t.Fatalf("%s: expected a verified account without email, got %q verified=%v", provider, first.record.Email(), first.record.Verified())
			}
			if got := first.record.GetString("username"); !generatedUsername.MatchString(got) {
				t.Fatalf("%s: expected a generated username, got %q", provider, got)
			}

			second := requireLogin(t, f.login(provider, claims))
			if second.record.Id != first.record.Id || !f.valid(first.token) {
				t.Fatalf("%s: expected repeated logins to keep the account and its sessions", provider)
			}
		}
	})

	t.Run("a username sent by the provider wins over the display name", func(t *testing.T) {
		f := newOIDCFlow(t, true)
		claims := cloudflareClaims("cf-1", "david@example.com", "David Schwartz")
		claims["preferred_username"] = "dschwartz"

		first := requireLogin(t, f.login("oidc", claims))
		if got := first.record.GetString("username"); got != "dschwartz" {
			t.Fatalf("expected the provider username, got %q", got)
		}
	})
}
