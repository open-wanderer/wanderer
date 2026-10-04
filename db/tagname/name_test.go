package tagname

import (
	"regexp"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	controls := make([]rune, 0, 33)
	for r := rune(0); r < 0x20; r++ {
		controls = append(controls, r)
	}
	controls = append(controls, 0x7f)
	for _, tt := range []struct {
		name, input, want string
	}{
		{"empty", "", ""},
		{"all controls", string(controls), ""},
		{"embedded controls", "Grü\nne\tzi\r\x00\x7f", "Grünezi"},
		{"plain text", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; Zürich  ", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; Zürich  "},
		{"other Unicode", "\u0085\u00a0\u2028\u200d", "\u0085\u00a0\u2028\u200d"},
		{"no truncation", strings.Repeat("🥾", MaxLength+1), strings.Repeat("🥾", MaxLength+1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if Normalize(got) != got {
				t.Fatal("normalization must be idempotent")
			}
			if !regexp.MustCompile(Pattern).MatchString(got) {
				t.Fatal("normalized name must satisfy the control-character pattern")
			}
		})
	}
}
