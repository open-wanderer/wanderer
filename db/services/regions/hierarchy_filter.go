package regions

import "strings"

// AncestorGroupPaths returns the set of proper-prefix (ancestor) paths for the
// given dotted materialized leaf paths. A leaf path "canada.alberta.south"
// yields "canada" and "canada.alberta" — every path segment prefix except the
// full path itself.
//
// It is the core of the `?enabled=true` server-side filter (RegionsList): given
// the paths of the enabled leaves, the result is exactly the group rows that
// must be retained to reach them. A group whose descendants are all disabled is
// never a prefix of any enabled leaf, so it is naturally dropped — the same
// "prune childless groups" outcome the Flutter client used to compute locally,
// but derived here from the materialized path with no tree walk.
//
// Pure and DB-free (mirrors this package's other testable helpers) so the
// filter's tricky part is unit-covered without a test database.
func AncestorGroupPaths(leafPaths []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, p := range leafPaths {
		segs := strings.Split(p, ".")
		// Every prefix segs[:i] for i in 1..len-1 is an ancestor; segs[:len]
		// is the leaf's own path, which is not a group and never included.
		for i := 1; i < len(segs); i++ {
			out[strings.Join(segs[:i], ".")] = struct{}{}
		}
	}
	return out
}
