package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/auth"
)

// trustOnly returns a trusted-provider lookup that accepts the given slots.
func trustOnly(names ...string) func(string) bool {
	return func(provider string) bool {
		for _, name := range names {
			if provider == name {
				return true
			}
		}
		return false
	}
}

func TestOAuth2UsernameHandler(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	collection := core.NewAuthCollection("oauth2_username_users")
	collection.Fields.Add(
		&core.TextField{
			Name:    "username",
			Min:     3,
			Max:     20,
			Pattern: `^[\w][\w\.\-]*$`,
		},
		// a plain text field without a pattern, as an alternative mapping target
		&core.TextField{Name: "name"},
	)
	collection.AddIndex("idx_oauth2_username_users_username", true, "username COLLATE NOCASE", "")
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}

	existing := core.NewRecord(collection)
	existing.SetEmail("jane@example.com")
	existing.SetPassword("1234567890")
	existing.Set("username", "jane_doe")
	if err := app.Save(existing); err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		name       string
		mapped     string
		provider   string // defaults to "oidc"
		trusted    func(string) bool
		isNew      bool
		createData map[string]any
		raw        string
		id         string
		oauthName  string
		expected   string
	}{
		{name: "display name is sanitised", mapped: "username", isNew: true, raw: "Karl Dörfinger", expected: "Karl_Doerfinger"},
		{name: "taken name gets a suffix regardless of case", mapped: "username", isNew: true, raw: "Jane Doe", expected: "Jane_Doe_2"},
		{name: "valid but taken name gets a suffix", mapped: "username", isNew: true, raw: "JANE_DOE", expected: "JANE_DOE_2"},
		{name: "disabled mapping is respected", mapped: "", isNew: true, raw: "Karl Dörfinger", expected: "Karl Dörfinger"},
		{name: "mapping to another field is respected", mapped: "name", isNew: true, raw: "Karl Dörfinger", expected: "Karl Dörfinger"},
		{name: "submitted username takes precedence", mapped: "username", isNew: true, createData: map[string]any{"username": "karl"}, raw: "Karl Dörfinger", expected: "Karl Dörfinger"},
		{name: "existing accounts are left alone", mapped: "username", isNew: false, raw: "Karl Dörfinger", expected: "Karl Dörfinger"},
		{name: "unusable names are left to PocketBase", mapped: "username", isNew: true, raw: "Иван", expected: "Иван"},

		// the display name fallback is limited to the slots the administrator opted in
		{name: "missing username falls back to the name", mapped: "username", trusted: trustOnly("oidc"), isNew: true, id: "beaf7823", oauthName: "David Schwartz", expected: "David_Schwartz"},
		{name: "subject id as username falls back to the name", mapped: "username", trusted: trustOnly("oidc"), isNew: true, raw: "beaf7823", id: "beaf7823", oauthName: "David Schwartz", expected: "David_Schwartz"},
		{name: "missing username and name are left to PocketBase", mapped: "username", trusted: trustOnly("oidc"), isNew: true, id: "beaf7823", expected: ""},
		{name: "second and third slot fall back to the name when opted in", mapped: "username", provider: "oidc3", trusted: trustOnly("oidc3"), isNew: true, id: "beaf7823", oauthName: "David Schwartz", expected: "David_Schwartz"},
		{name: "a provider username is used before the name", mapped: "username", trusted: trustOnly("oidc"), isNew: true, raw: "dschwartz", id: "beaf7823", oauthName: "David Schwartz", expected: "dschwartz"},
		{name: "no opt-in keeps the name out of the username", mapped: "username", isNew: true, id: "beaf7823", oauthName: "David Schwartz", expected: ""},
		{name: "subject id as username without opt-in is not replaced by the name", mapped: "username", isNew: true, raw: "beaf7823", id: "beaf7823", oauthName: "David Schwartz", expected: "beaf7823"},
		{name: "opt-in of another slot does not apply", mapped: "username", provider: "oidc2", trusted: trustOnly("oidc"), isNew: true, id: "beaf7823", oauthName: "David Schwartz", expected: ""},
		{name: "built-in providers never fall back to the name", mapped: "username", provider: "google", trusted: trustOnly("oidc", "oidc2", "oidc3"), isNew: true, id: "1234", oauthName: "Gerda Google", expected: ""},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			collection.OAuth2.MappedFields.Username = s.mapped

			provider := s.provider
			if provider == "" {
				provider = "oidc"
			}

			e := &core.RecordAuthWithOAuth2RequestEvent{
				RequestEvent: &core.RequestEvent{App: app},
				OAuth2User:   &auth.AuthUser{Id: s.id, Name: s.oauthName, Username: s.raw},
				CreateData:   s.createData,
				IsNewRecord:  s.isNew,
				ProviderName: provider,
			}
			e.Collection = collection

			if err := OAuth2UsernameHandler(s.trusted)(e); err != nil {
				t.Fatal(err)
			}

			if e.OAuth2User.Username != s.expected {
				t.Fatalf("expected %q, got %q", s.expected, e.OAuth2User.Username)
			}

			if _, ok := s.createData["username"]; !ok && e.CreateData["username"] != nil {
				t.Fatalf("expected createData to stay untouched, got %v", e.CreateData)
			}
		})
	}
}

func TestOAuth2EmailHandler(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	collection := core.NewAuthCollection("oauth2_email_users")
	// the users collection allows accounts without an email
	collection.Fields.GetByName(core.FieldNameEmail).(*core.EmailField).Required = false
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}

	save := func(email string) *core.Record {
		r := core.NewRecord(collection)
		r.SetEmail(email)
		r.SetPassword("1234567890")
		r.SetVerified(true)
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
		return r
	}

	other := save("jane@example.com")
	self := save("david@example.com")
	noEmail := save("")
	changed := save("changed@example.com")

	// "absent" leaves the claim out of the userinfo document, nil sends a JSON null
	absent := struct{}{}

	scenarios := []struct {
		name       string
		provider   string // defaults to "oidc"
		trusted    func(string) bool
		record     *core.Record // nil = new account
		email      string       // the email PocketBase already took over
		verified   any          // the email_verified claim, absent = not sent
		rawEmail   any
		createData map[string]any
		expected   string
	}{
		// new accounts
		{name: "email of a trusted slot is kept", trusted: trustOnly("oidc"), verified: absent, rawEmail: "new@example.com", expected: "new@example.com"},
		{name: "surrounding whitespace is trimmed", trusted: trustOnly("oidc"), verified: absent, rawEmail: "  new@example.com ", expected: "new@example.com"},
		{name: "second and third slot work when opted in", provider: "oidc2", trusted: trustOnly("oidc2"), verified: absent, rawEmail: "new@example.com", expected: "new@example.com"},
		{name: "no opt-in leaves the email alone", verified: absent, rawEmail: "new@example.com"},
		{name: "opt-in of another slot does not apply", provider: "oidc2", trusted: trustOnly("oidc"), verified: absent, rawEmail: "new@example.com"},
		{name: "built-in providers are never trusted", provider: "google", trusted: trustOnly("oidc", "oidc2", "oidc3"), verified: absent, rawEmail: "new@example.com"},
		{name: "email PocketBase already took over is left alone", trusted: trustOnly("oidc"), email: "verified@example.com", verified: true, rawEmail: "verified@example.com"},

		// only an absent email_verified claim is overridden
		{name: "explicit false is respected", trusted: trustOnly("oidc"), verified: false, rawEmail: "new@example.com"},
		{name: "explicit true string is respected", trusted: trustOnly("oidc"), verified: "true", rawEmail: "new@example.com"},
		{name: "explicit false string is respected", trusted: trustOnly("oidc"), verified: "false", rawEmail: "new@example.com"},
		{name: "null is respected", trusted: trustOnly("oidc"), verified: nil, rawEmail: "new@example.com"},
		{name: "malformed value is respected", trusted: trustOnly("oidc"), verified: "maybe", rawEmail: "new@example.com"},

		// unusable claims
		{name: "email of another account is not taken", trusted: trustOnly("oidc"), verified: absent, rawEmail: "jane@example.com"},
		{name: "invalid email is ignored", trusted: trustOnly("oidc"), verified: absent, rawEmail: "not an email"},
		{name: "missing email claim is ignored", trusted: trustOnly("oidc"), verified: absent},
		{name: "non-string email claim is ignored", trusted: trustOnly("oidc"), verified: absent, rawEmail: 42},
		{name: "submitted email takes precedence for new accounts", trusted: trustOnly("oidc"), verified: absent, rawEmail: "new@example.com", createData: map[string]any{"email": "other@example.com"}},

		// existing accounts
		{name: "existing account without email gets it", trusted: trustOnly("oidc"), record: noEmail, verified: absent, rawEmail: "new@example.com", expected: "new@example.com"},
		{name: "existing account that already has the address keeps it", trusted: trustOnly("oidc"), record: self, verified: absent, rawEmail: "david@example.com", expected: "david@example.com"},
		{name: "existing account with another address is left alone", trusted: trustOnly("oidc"), record: changed, verified: absent, rawEmail: "new@example.com"},
		{name: "existing account with the address in another case is left alone", trusted: trustOnly("oidc"), record: self, verified: absent, rawEmail: "David@Example.com"},
		{name: "existing account is not given the address of another account", trusted: trustOnly("oidc"), record: noEmail, verified: absent, rawEmail: other.Email()},
		{name: "existing account without opt-in is left alone", record: noEmail, verified: absent, rawEmail: "new@example.com"},
		{name: "existing account with explicit false is left alone", trusted: trustOnly("oidc"), record: noEmail, verified: false, rawEmail: "new@example.com"},
		{name: "submitted email is ignored for existing accounts", trusted: trustOnly("oidc"), record: noEmail, verified: absent, rawEmail: "new@example.com", createData: map[string]any{"email": "other@example.com"}, expected: "new@example.com"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			raw := map[string]any{}
			if s.rawEmail != nil {
				raw["email"] = s.rawEmail
			}
			if s.verified != absent {
				raw["email_verified"] = s.verified
			}

			provider := s.provider
			if provider == "" {
				provider = "oidc"
			}

			e := &core.RecordAuthWithOAuth2RequestEvent{
				RequestEvent: &core.RequestEvent{App: app},
				OAuth2User:   &auth.AuthUser{Email: s.email, RawUser: raw},
				CreateData:   s.createData,
				IsNewRecord:  s.record == nil,
				Record:       s.record,
				ProviderName: provider,
			}
			e.Collection = collection

			if err := OAuth2EmailHandler(s.trusted)(e); err != nil {
				t.Fatal(err)
			}

			expected := s.expected
			if s.email != "" {
				expected = s.email
			}
			if e.OAuth2User.Email != expected {
				t.Fatalf("expected the OAuth2 user email %q, got %q", expected, e.OAuth2User.Email)
			}

			// the email must reach PocketBase as the verified OAuth2 user email: as
			// create data it would produce an unverified account
			if got := e.CreateData["email"]; got != s.createData["email"] {
				t.Fatalf("expected createData to stay untouched, got %v", e.CreateData)
			}
		})
	}
}
