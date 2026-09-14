package util

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// the default pattern of the users.username field
var usernameField = regexp.MustCompile(`^[\w][\w.\-]*$`)

func TestSanitizeUsername(t *testing.T) {
	scenarios := []struct {
		name     string
		raw      string
		min      int
		max      int
		expected string
	}{
		{"already valid", "jane_doe", 3, 150, "jane_doe"},
		{"dots and dashes are kept", "jane.doe-1", 3, 150, "jane.doe-1"},
		{"spaces become underscores", "Jane Doe", 3, 150, "Jane_Doe"},
		{"umlauts are transliterated", "Jörg Müller", 3, 150, "Joerg_Mueller"},
		{"eszett is transliterated", "Straßer", 3, 150, "Strasser"},
		{"accents are stripped", "José Ángel", 3, 150, "Jose_Angel"},
		{"capital umlauts keep their case", "Örjan", 3, 150, "Oerjan"},
		{"unsupported scripts fall back", "Иван", 3, 150, ""},
		{"runs collapse into one underscore", "Jane   Doe", 3, 150, "Jane_Doe"},
		{"surrounding whitespace is ignored", "  Jane Doe  ", 3, 150, "Jane_Doe"},
		{"leading dot is dropped", ".jane", 3, 150, "jane"},
		{"leading dash is dropped", "-jane", 3, 150, "jane"},
		{"short names are padded", "ab", 3, 150, "ab_"},
		{"single character is padded", "a", 3, 150, "a__"},
		{"padding follows the field minimum", "ab", 5, 150, "ab___"},
		{"empty stays empty", "", 3, 150, ""},
		{"punctuation only falls back", "!!!", 3, 150, ""},
		{"only dots and dashes yields empty", ".-.", 3, 150, ""},
		{"long names are truncated", strings.Repeat("a", 200), 3, 150, strings.Repeat("a", 150)},
		{"truncation follows the field maximum", "Jane Doe", 3, 4, "Jane"},
		{"no maximum keeps the full name", strings.Repeat("a", 200), 3, 0, strings.Repeat("a", 200)},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			got := SanitizeUsername(s.raw, s.min, s.max)

			if got != s.expected {
				t.Fatalf("expected %q, got %q", s.expected, got)
			}

			// whatever comes out must be accepted by the field, or be empty
			if got == "" {
				return
			}

			if !usernameField.MatchString(got) {
				t.Fatalf("%q does not match the username field pattern", got)
			}

			if len(got) < s.min || (s.max > 0 && len(got) > s.max) {
				t.Fatalf("%q has length %d, outside [%d, %d]", got, len(got), s.min, s.max)
			}
		})
	}
}

func TestUniqueUsername(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	unique := createUsernameTestCollection(t, app, "username_unique", true)
	plain := createUsernameTestCollection(t, app, "username_plain", false)

	for _, username := range []string{"Alice", "bob", "bob_2", "abcdefghijkl"} {
		createUsernameTestUser(t, app, unique, username)
	}
	createUsernameTestUser(t, app, plain, "bob")

	scenarios := []struct {
		name       string
		collection *core.Collection
		username   string
		expected   string
	}{
		{"free name is kept", unique, "carol", "carol"},
		{"taken name gets a suffix", unique, "Alice", "Alice_2"},
		{"taken check ignores case like the index", unique, "alice", "alice_2"},
		{"suffixes skip taken variants", unique, "bob", "bob_3"},
		{"suffix fits the field maximum", unique, "ABCDEFGHIJKL", "ABCDEFGHIJ_2"},
		{"name the field rejects falls back", unique, ".jane", ""},
		{"empty falls back", unique, "", ""},
		{"no unique index keeps the name", plain, "bob", "bob"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			field := s.collection.Fields.GetByName("username").(*core.TextField)

			got := UniqueUsername(app, s.collection, field, s.username)
			if got != s.expected {
				t.Fatalf("expected %q, got %q", s.expected, got)
			}
		})
	}
}

func createUsernameTestCollection(t *testing.T, app core.App, name string, unique bool) *core.Collection {
	t.Helper()

	collection := core.NewAuthCollection(name)
	collection.Fields.Add(&core.TextField{
		Name:    "username",
		Min:     3,
		Max:     12,
		Pattern: `^[\w][\w\.\-]*$`,
	})
	if unique {
		collection.AddIndex("idx_"+name+"_username", true, "username COLLATE NOCASE", "")
	}

	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}

	return collection
}

func createUsernameTestUser(t *testing.T, app core.App, collection *core.Collection, username string) {
	t.Helper()

	record := core.NewRecord(collection)
	record.SetEmail(strings.ToLower(username) + "@example.com")
	record.SetPassword("1234567890")
	record.Set("username", username)

	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
}
