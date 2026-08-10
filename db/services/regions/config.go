// Package regions implements the seeded `regions` table's id-validation and
// the filesystem path builders for pre-built region archives (vector + DEM
// PMTiles). It is a parallel, independent build path alongside
// pocketbase/services/tiles' existing per-cell cache. The old
// admin-supplied region_config.json / REGION_CATALOG_CONFIG_PATH loader has
// been retired — the seeded `regions` PocketBase collection is now the sole
// source of truth for the region catalog.
package regions

import (
	"path/filepath"
	"regexp"
	"strings"
)

// RegionCacheDir is the on-disk root under which every region's pre-built
// archives are stored, one subdirectory per region id:
// RegionCacheDir/{id}/vector.pmtiles and RegionCacheDir/{id}/dem.pmtiles.
const RegionCacheDir = "./pb_data/region_archives"

// regionIDPattern is the allow-list a region id must satisfy before it is
// ever used to build a filesystem path. It now guards a seeded materialized
// `path` value from the `regions` table, not merely an
// admin-typed slug — the trailing character class allows `.` (the
// materialized-path separator, e.g. "algeria.algeria_central") and `'`
// (e.g. "people's_republic_of_china"). The leading character stays
// [a-z0-9] so a leading dot/hyphen/apostrophe is still rejected. Defense in
// depth against a path-traversal id (e.g. "../etc") ever reaching
// filepath.Join is additionally enforced by the explicit ".." substring
// guard in IsValidRegionID below.
var regionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.'-]*$`)

// IsValidRegionID reports whether id satisfies the allow-list regex. It is
// exported so callers outside this package (e.g. the download route added
// in a later plan) can re-validate an untrusted URL path param before using
// it to build a filesystem path.
func IsValidRegionID(id string) bool {
	return regionIDPattern.MatchString(id) && !strings.Contains(id, "..")
}

// RegionArchivePath returns the on-disk path to a region's pre-built vector
// PMTiles archive.
func RegionArchivePath(id string) string {
	return filepath.Join(RegionCacheDir, id, "vector.pmtiles")
}

// RegionDemPath returns the on-disk path to a region's pre-built DEM
// PMTiles archive.
func RegionDemPath(id string) string {
	return filepath.Join(RegionCacheDir, id, "dem.pmtiles")
}
