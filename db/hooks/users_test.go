package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/auth"
)

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
		isNew      bool
		createData map[string]any
		raw        string
		expected   string
	}{
		{"display name is sanitised", "username", true, nil, "Karl Dörfinger", "Karl_Doerfinger"},
		{"taken name gets a suffix regardless of case", "username", true, nil, "Jane Doe", "Jane_Doe_2"},
		{"valid but taken name gets a suffix", "username", true, nil, "JANE_DOE", "JANE_DOE_2"},
		{"disabled mapping is respected", "", true, nil, "Karl Dörfinger", "Karl Dörfinger"},
		{"mapping to another field is respected", "name", true, nil, "Karl Dörfinger", "Karl Dörfinger"},
		{"submitted username takes precedence", "username", true, map[string]any{"username": "karl"}, "Karl Dörfinger", "Karl Dörfinger"},
		{"existing accounts are left alone", "username", false, nil, "Karl Dörfinger", "Karl Dörfinger"},
		{"unusable names are left to PocketBase", "username", true, nil, "Иван", "Иван"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			collection.OAuth2.MappedFields.Username = s.mapped

			e := &core.RecordAuthWithOAuth2RequestEvent{
				RequestEvent: &core.RequestEvent{App: app},
				OAuth2User:   &auth.AuthUser{Username: s.raw},
				CreateData:   s.createData,
				IsNewRecord:  s.isNew,
			}
			e.Collection = collection

			if err := OAuth2UsernameHandler()(e); err != nil {
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
