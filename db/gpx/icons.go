package gpx

import (
	_ "embed"
	"strings"
)

// iconList is the closed set of waypoint icons, one name per line. It is
// generated from the `icons` array in web/src/lib/util/icon_util.ts, which is
// what the web importer checks a <sym> against, and icons_test.go fails when
// the two drift apart.
//
//go:embed icons.txt
var iconList string

var iconSet = buildIconSet(iconList)

// buildIconSet turns the embedded list into a lookup set, skipping blank lines.
func buildIconSet(list string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, name := range strings.Split(list, "\n") {
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

// iconFor returns sym when it names a known icon and "" otherwise. It is a
// closed-set lookup, so a hostile <sym> can never become stored content.
func iconFor(sym string) string {
	if _, ok := iconSet[sym]; ok {
		return sym
	}
	return ""
}
