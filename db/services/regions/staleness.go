package regions

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// protomapsDateFormat is the date stamp in Protomaps daily-build URLs and in
// region_archives.vector_built_date.
const protomapsDateFormat = "20060102"

// defaultRebuildDays is how old a vector archive may get before the nightly
// pass re-extracts it.
const defaultRebuildDays = 7

// protomapsHTTPClient bounds the daily-build lookup. It runs while the sync
// slot is held, so a request without a deadline could stall every later sync.
var protomapsHTTPClient = &http.Client{Timeout: 15 * time.Second}

// ValidProtomapsDateAndURL resolves today's valid Protomaps daily-build date
// and URL, falling back to yesterday's if today's build isn't live yet.
//
// This is a self-contained duplicate of the per-cell tile generator's
// equivalent daily-build-date resolution logic — the region-archive build
// path is parallel and independent, so this deliberately does
// NOT import or reference that sibling package. A ~12-line local copy is
// the safer choice over reopening its source file.
//
// The yesterday fallback is unconditional — it does not re-check urlExists —
// matching generator.go's existing behavior exactly. A subsequently-failing
// extract against a stale/missing daily build is handled by the caller's
// setVectorError path, not here.
func ValidProtomapsDateAndURL() (date string, url string) {
	now := time.Now().UTC()

	today := now.Format(protomapsDateFormat)
	todayURL := fmt.Sprintf("https://build.protomaps.com/%s.pmtiles", today)
	if urlExists(todayURL) {
		return today, todayURL
	}

	yesterday := now.AddDate(0, 0, -1).Format(protomapsDateFormat)
	return yesterday, fmt.Sprintf("https://build.protomaps.com/%s.pmtiles", yesterday)
}

// urlExists mirrors generator.go's urlExists (HEAD request, 200 check).
func urlExists(url string) bool {
	resp, err := protomapsHTTPClient.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// rebuildDays reads REGION_ARCHIVE_REBUILD_DAYS, the age in days at which a
// vector archive is re-extracted from a newer Protomaps build. Unset, invalid
// or values below 1 fall back to defaultRebuildDays.
func rebuildDays() int {
	if v := os.Getenv("REGION_ARCHIVE_REBUILD_DAYS"); v != "" {
		if days, err := strconv.Atoi(v); err == nil && days >= 1 {
			return days
		}
	}
	return defaultRebuildDays
}

// needsVectorRebuild reports whether a region's vector archive should be
// rebuilt: it has never been built, its stored date is unreadable, or it is at
// least maxAgeDays older than the current Protomaps build. Protomaps publishes
// a new build every day, so comparing for equality alone would re-extract
// every enabled region every night.
func needsVectorRebuild(storedDate, currentDate string, maxAgeDays int) bool {
	stored, err := time.Parse(protomapsDateFormat, storedDate)
	if err != nil {
		return true
	}
	current, err := time.Parse(protomapsDateFormat, currentDate)
	if err != nil {
		return false
	}
	return current.Sub(stored) >= time.Duration(maxAgeDays)*24*time.Hour
}

// bboxChanged reports whether the region's config bbox differs from the
// bbox stored on its region_archives record, used to gate DEM rebuilds
// (DEM builds once, rebuilding only when the config bbox changes).
//
// The comparison source is now the bbox resolved from region_geometry
// (via ResolveGeometry) rather than the seeded catalog record, since bbox
// no longer lives on the regions record — semantics
// and the exact-float64-equality rationale below are unchanged.
//
// Exact float64 equality is used (no epsilon) — both sides originate from
// the same JSON-parsed float64 values (config file -> Region.Bbox -> stored
// record fields), so no floating-point drift is introduced between them.
func bboxChanged(configBbox [4]float64, storedMinLon, storedMinLat, storedMaxLon, storedMaxLat float64) bool {
	return configBbox[0] != storedMinLon ||
		configBbox[1] != storedMinLat ||
		configBbox[2] != storedMaxLon ||
		configBbox[3] != storedMaxLat
}
