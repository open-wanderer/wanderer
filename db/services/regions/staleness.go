package regions

import (
	"fmt"
	"net/http"
	"time"
)

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
// setError path, not here.
func ValidProtomapsDateAndURL() (date string, url string) {
	now := time.Now().UTC()

	today := now.Format("20060102")
	todayURL := fmt.Sprintf("https://build.protomaps.com/%s.pmtiles", today)
	if urlExists(todayURL) {
		return today, todayURL
	}

	yesterday := now.AddDate(0, 0, -1).Format("20060102")
	return yesterday, fmt.Sprintf("https://build.protomaps.com/%s.pmtiles", yesterday)
}

// urlExists mirrors generator.go's urlExists (HEAD request, 200 check).
func urlExists(url string) bool {
	resp, err := http.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// needsVectorRebuild reports whether a region's vector archive should be
// rebuilt because the Protomaps daily-build date advanced since the last
// successful build. storedDate == "" (never built) naturally returns
// true since it always differs from a non-empty currentDate.
func needsVectorRebuild(storedDate, currentDate string) bool {
	return storedDate != currentDate
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
