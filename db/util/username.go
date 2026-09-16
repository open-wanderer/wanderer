package util

import (
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/dbutils"
	"github.com/pocketbase/pocketbase/tools/inflector"
	"golang.org/x/text/unicode/norm"
)

// how many suffixed variants to try before giving up on a name
const usernameMaxAttempts = 50

var (
	// characters the default users.username pattern does not accept
	usernameDisallowed = regexp.MustCompile(`[^\w.\-]+`)
	// the default pattern additionally requires the first character to be a word character
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
// value the default users.username pattern accepts: word characters, dots and
// dashes, starting with a word character. The result is truncated to
// maxLength (when > 0) and padded with underscores up to minLength.
//
// Accented Latin characters are transliterated first, so "Karl Dörfinger"
// becomes "Karl_Doerfinger". Remaining disallowed characters are replaced with
// underscores, which keeps names such as "Jane Doe" recognisable as "Jane_Doe".
//
// Returns an empty string when nothing usable remains. Callers should then
// leave the username alone and let PocketBase generate one.
func SanitizeUsername(raw string, minLength, maxLength int) string {
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

	// the sanitised value is ASCII only, so byte and character lengths match
	if maxLength > 0 && len(username) > maxLength {
		username = username[:maxLength]
	}

	for len(username) < minLength {
		username += "_"
	}

	return username
}

// UniqueUsername returns username, or the first variant of it suffixed with
// "_2", "_3" and so on, that passes the field's validation and is not taken yet.
//
// Uniqueness is checked the same way PocketBase checks it for the OAuth2
// username mapping: against the field's single column unique index, honouring
// its collation, so "alice" counts as taken when "Alice" exists.
//
// Returns an empty string when no usable variant was found within
// usernameMaxAttempts, so that callers can fall back to PocketBase's own
// handling instead.
func UniqueUsername(app core.App, collection *core.Collection, field *core.TextField, username string) string {
	if username == "" {
		return ""
	}

	index, hasUnique := dbutils.FindSingleColumnUniqueIndex(collection.Indexes, field.GetName())
	column := inflector.Columnify(field.GetName())

	for attempt := 1; attempt <= usernameMaxAttempts; attempt++ {
		candidate := username

		if attempt > 1 {
			suffix := "_" + strconv.Itoa(attempt)
			if field.Max > len(suffix) && len(candidate)+len(suffix) > field.Max {
				candidate = candidate[:field.Max-len(suffix)]
			}
			candidate += suffix
		}

		if field.ValidatePlainValue(candidate) != nil {
			continue
		}

		if !hasUnique {
			return candidate
		}

		var expr dbx.Expression
		if strings.EqualFold(index.Columns[0].Collate, "nocase") {
			expr = dbx.NewExp("[["+column+"]] = {:username} COLLATE NOCASE", dbx.Params{"username": candidate})
		} else {
			expr = dbx.HashExp{column: candidate}
		}

		var exists int
		err := app.RecordQuery(collection).Select("(1)").AndWhere(expr).Limit(1).Row(&exists)

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
