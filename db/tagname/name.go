// Package tagname defines the plain-text tag-name policy shared by schema
// migrations and incoming federation imports.
package tagname

import "strings"

const (
	// MaxLength counts Unicode code points, matching PocketBase's text limit.
	MaxLength = 5000
	Pattern   = `^[^\x00-\x1f\x7f]*$`
)

// Normalize removes C0 and DEL control characters. All other text, including
// whitespace, Unicode, punctuation, markup and entities, is preserved.
func Normalize(name string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
}
