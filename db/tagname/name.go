// Package tagname defines the plain-text tag-name policy shared by schema
// migrations and incoming federation imports.
package tagname

import "strings"

const (
	// MaxLength counts Unicode code points, matching PocketBase's text limit.
	MaxLength = 5000
	Pattern   = `^[^\x00-\x1f\x7f]*$`
)

// Normalize maps TAB, LF, VT, FF and CR to one ASCII space per character and
// removes the remaining C0 and DEL controls. All other text is preserved;
// names are never trimmed, collapsed or truncated.
func Normalize(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 9 && r <= 13 {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
}
