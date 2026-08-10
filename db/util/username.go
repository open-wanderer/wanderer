package util

import (
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/text/unicode/norm"
)

const (
	usernameMinLength = 3
	usernameMaxLength = 150
	// how many suffixed variants to try before giving up on a name
	usernameMaxAttempts = 50
)

var (
	// characters the users.username field does not accept
	usernameDisallowed = regexp.MustCompile(`[^\w.\-]+`)
	// the field additionally requires the first character to be a word character
	usernameLeading = regexp.MustCompile(`^[.\-]+`)
	// characters that expand to more than one letter when transliterated
	usernameExpansions = strings.NewReplacer(
		"ä", "ae", "Ä", "Ae",
		"ö", "oe", "Ö", "Oe",
		"ü", "ue", "Ü", "Ue",
		"ß", "ss",
		"æ", "ae", "Æ", "Ae",
		"ø", "oe", "Ø", "Oe",
	)
)

// transliterate replaces accented Latin characters with their ASCII
// equivalents, so that "Karl Dörfinger" becomes "Karl Doerfinger" rather than
// losing the umlaut to an underscore.
//
// Characters that conventionally expand to two letters are mapped explicitly;
// the rest have their diacritics stripped ("José" -> "Jose"). Anything outside
// the Latin script is left alone and handled by SanitizeUsername.
func transliterate(raw string) string {
	expanded := usernameExpansions.Replace(raw)

	var b strings.Builder
	for _, r := range norm.NFD.String(expanded) {
		if unicode.Is(unicode.Mn, r) {
			continue // combining mark left over from decomposition
		}
		b.WriteRune(r)
	}

	return b.String()
}

// SanitizeUsername converts a username coming from an OAuth2 provider into a
// value the users.username field accepts: word characters, dots and dashes,
// starting with a word character, between 3 and 150 characters long.
//
// Accented Latin characters are transliterated first, so "Karl Dörfinger"
// becomes "Karl_Doerfinger". Remaining disallowed characters are replaced with
// underscores, which keeps names such as "Jane Doe" recognisable as "Jane_Doe".
// Names that are too short are padded with underscores.
//
// Returns an empty string when nothing usable remains. Callers should then
// leave the username unset and let PocketBase generate one, rather than
// submitting a value the field will reject.
func SanitizeUsername(raw string) string {
	username := usernameDisallowed.ReplaceAllString(transliterate(strings.TrimSpace(raw)), "_")
	username = usernameLeading.ReplaceAllString(username, "")

	// a name that transliterated to nothing recognisable - a script the field
	// cannot represent, or punctuation only - is better left to PocketBase
	// than turned into a meaningless "___"
	if !strings.ContainsFunc(username, func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}) {
		return ""
	}

	if len(username) > usernameMaxLength {
		username = username[:usernameMaxLength]
	}

	for len(username) < usernameMinLength {
		username += "_"
	}

	return username
}

// UniqueUsername returns username, or the first free variant of it suffixed
// with "_2", "_3" and so on.
//
// Returns an empty string when no free variant was found within
// usernameMaxAttempts, so that callers can fall back to a generated username
// instead of submitting a value that would collide with the unique index.
func UniqueUsername(app core.App, username string) string {
	for attempt := 1; attempt <= usernameMaxAttempts; attempt++ {
		candidate := username

		if attempt > 1 {
			suffix := "_" + strconv.Itoa(attempt)
			if len(candidate)+len(suffix) > usernameMaxLength {
				candidate = candidate[:usernameMaxLength-len(suffix)]
			}
			candidate += suffix
		}

		_, err := app.FindFirstRecordByData("users", "username", candidate)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			return candidate
		case err != nil:
			// treat a failed lookup as taken rather than risking a collision
			continue
		}
	}

	return ""
}
