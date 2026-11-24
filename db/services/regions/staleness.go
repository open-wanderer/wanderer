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

// protomapsBuildBaseURL is the host of the Protomaps daily builds. Tests point
// it at a local server.
var protomapsBuildBaseURL = "https://build.protomaps.com"

// protomapsLookbackDays is how many daily builds the lookup probes, today's
// first. Protomaps publishes with about two days of delay (#1531), so a week
// leaves room for a skipped build. Each probe is bounded by
// protomapsHTTPClient's timeout, so a hung host holds the sync slot for at
// most 7 x 15s.
const protomapsLookbackDays = 7

// protomapsHTTPClient bounds the daily-build lookup. It runs while the sync
// slot is held, so a request without a deadline could stall every later sync.
var protomapsHTTPClient = &http.Client{Timeout: 15 * time.Second}

// SetProtomapsBuildsForTesting points the Protomaps build lookup at another
// host and HTTP client and returns a func that restores the previous ones. It
// is a seam for tests outside this package that run a build pass; the regions
// route tests are the first users.
func SetProtomapsBuildsForTesting(baseURL string, client *http.Client) (restore func()) {
	prevURL, prevClient := protomapsBuildBaseURL, protomapsHTTPClient
	protomapsBuildBaseURL, protomapsHTTPClient = baseURL, client
	return func() {
		protomapsBuildBaseURL, protomapsHTTPClient = prevURL, prevClient
	}
}

// ValidProtomapsDateAndURL returns the newest Protomaps daily build that
// answers 200 within protomapsLookbackDays, and ok=false when none does (an
// outage, or no build in a week).
func ValidProtomapsDateAndURL() (date string, url string, ok bool) {
	return newestProtomapsBuild(time.Now().UTC())
}

// newestProtomapsBuild probes the daily builds newest first, starting with
// now's date (UTC) and stepping one day back, and returns the first one whose
// URL answers 200.
func newestProtomapsBuild(now time.Time) (date, url string, ok bool) {
	for days := 0; days < protomapsLookbackDays; days++ {
		date = now.UTC().AddDate(0, 0, -days).Format(protomapsDateFormat)
		url = fmt.Sprintf("%s/%s.pmtiles", protomapsBuildBaseURL, date)
		if urlExists(url) {
			return date, url, true
		}
	}
	return "", "", false
}

// urlExists sends a HEAD request through protomapsHTTPClient and reports true
// only when the answer is 200.
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
