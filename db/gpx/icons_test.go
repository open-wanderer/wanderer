package gpx

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestIconSetMatchesWeb fails when the embedded icon set and the `icons` array
// in web/src/lib/util/icon_util.ts differ. The TS array is the source of truth;
// regenerate icons.txt from it when this fails.
func TestIconSetMatchesWeb(t *testing.T) {
	path := filepath.Join("..", "..", "web", "src", "lib", "util", "icon_util.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %q: %v (run go test with the working directory set to db/gpx)", path, err)
	}
	src := string(raw)
	const open = "export const icons = ["
	a := strings.Index(src, open)
	if a < 0 {
		t.Fatalf("%q has no %q", path, open)
	}
	b := strings.Index(src[a:], "] as const")
	if b < 0 {
		t.Fatalf("%q has no closing %q", path, "] as const")
	}
	web := make(map[string]struct{})
	for _, m := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(src[a+len(open):a+b], -1) {
		web[m[1]] = struct{}{}
	}
	if len(web) == 0 {
		t.Fatalf("parsed no icon names from %q", path)
	}

	missing := diff(web, iconSet)
	extra := diff(iconSet, web)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("embedded icon set drifted from icon_util.ts: %d missing from icons.txt (first %v), %d only in icons.txt (first %v)",
			len(missing), first(missing, 10), len(extra), first(extra, 10))
	}
}

// diff is the names in a that b lacks, in map order made stable by sorting-free
// first-N reporting only; the caller just needs a sample.
func diff(a, b map[string]struct{}) []string {
	var out []string
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

func first(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func TestIconFor(t *testing.T) {
	for _, tt := range []struct{ name, sym, want string }{
		{"known", "campground", "campground"},
		{"another known", "mountain", "mountain"},
		{"empty", "", ""},
		{"unknown", "airbnb", ""},
		{"case sensitive", "Campground", ""},
		{"untrimmed is not trimmed here", " campground", ""},
		{"markup", `"><script>`, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := iconFor(tt.sym); got != tt.want {
				t.Fatalf("iconFor(%.80q) = %q, want %q", tt.sym, got, tt.want)
			}
		})
	}
}
