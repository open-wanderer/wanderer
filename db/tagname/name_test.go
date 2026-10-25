package tagname

import (
	"regexp"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	controls := make([]rune, 0, 33)
	removedControls := make([]rune, 0, 28)
	for r := rune(0); r < 0x20; r++ {
		controls = append(controls, r)
		if r < 9 || r > 13 {
			removedControls = append(removedControls, r)
		}
	}
	controls = append(controls, 0x7f)
	removedControls = append(removedControls, 0x7f)
	for _, tt := range []struct {
		name, input, want string
	}{
		{"empty", "", ""},
		{"all controls", string(controls), "     "},
		{"remaining controls removed", string(removedControls), ""},
		{"TAB word boundary", "Berg\tTour", "Berg Tour"},
		{"LF word boundary", "Berg\nTour", "Berg Tour"},
		{"VT word boundary", "Berg\vTour", "Berg Tour"},
		{"FF word boundary", "Berg\fTour", "Berg Tour"},
		{"CR word boundary", "Berg\rTour", "Berg Tour"},
		{"CRLF preserves both separators", "Berg\r\nTour", "Berg  Tour"},
		{"embedded controls", "Grü\nne\tzi\r\x00\x7f", "Grü ne zi "},
		{"no trim or collapse", "\t Berg\t \nTour\r", "  Berg   Tour "},
		{"plain text", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; Zürich  ", "  🇨🇭 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; Zürich  "},
		{"other Unicode", "\u0085\u00a0\u2028\u200d", "\u0085\u00a0\u2028\u200d"},
		{"limit-length whitespace mapping", strings.Repeat("Berg\t", 1000), strings.Repeat("Berg ", 1000)},
		{"no truncation", strings.Repeat("🥾", MaxLength+1), strings.Repeat("🥾", MaxLength+1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Fatalf("Normalize(%.80q) = %.80q, want %.80q", tt.input, got, tt.want)
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
