package util

import (
	"regexp"
	"strings"
	"testing"
)

// the pattern and bounds of the users.username field
var usernameField = regexp.MustCompile(`^[\w][\w.\-]*$`)

func TestSanitizeUsername(t *testing.T) {
	scenarios := []struct {
		name     string
		raw      string
		expected string
	}{
		{"already valid", "jane_doe", "jane_doe"},
		{"dots and dashes are kept", "jane.doe-1", "jane.doe-1"},
		{"spaces become underscores", "Jane Doe", "Jane_Doe"},
		{"umlauts are transliterated", "Jörg Müller", "Joerg_Mueller"},
		{"eszett is transliterated", "Straßer", "Strasser"},
		{"accents are stripped", "José Ángel", "Jose_Angel"},
		{"capital umlauts keep their case", "Örjan", "Oerjan"},
		{"unsupported scripts fall back", "Иван", ""},
		{"runs collapse into one underscore", "Jane   Doe", "Jane_Doe"},
		{"surrounding whitespace is ignored", "  Jane Doe  ", "Jane_Doe"},
		{"leading dot is dropped", ".jane", "jane"},
		{"leading dash is dropped", "-jane", "jane"},
		{"short names are padded", "ab", "ab_"},
		{"single character is padded", "a", "a__"},
		{"empty stays empty", "", ""},
		{"punctuation only falls back", "!!!", ""},
		{"only dots and dashes yields empty", ".-.", ""},
		{"long names are truncated", strings.Repeat("a", 200), strings.Repeat("a", 150)},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			got := SanitizeUsername(s.raw)

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

			if len(got) < usernameMinLength || len(got) > usernameMaxLength {
				t.Fatalf("%q has length %d, outside [%d, %d]", got, len(got), usernameMinLength, usernameMaxLength)
			}
		})
	}
}
