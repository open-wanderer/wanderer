// Package regions implements the region catalog config loader and the
// filesystem path builders for pre-built region archives (vector + DEM
// PMTiles). It is a parallel, independent build path alongside
// pocketbase/services/tiles' existing per-cell cache — see
// .planning/phases/21.5-region-catalog-archive-pre-build-backend/21.5-CONTEXT.md
// (D-01/D-02/D-03) for the design rationale.
package regions

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
// `path` value from the `regions` table (per EXTRACT-02), not only an
// admin-typed slug — the trailing character class allows `.` (the
// materialized-path separator, e.g. "algeria.algeria_central") and `'`
// (e.g. "people's_republic_of_china"). The leading character stays
// [a-z0-9] so a leading dot/hyphen/apostrophe is still rejected. Defense in
// depth against a path-traversal id (e.g. "../etc") ever reaching
// filepath.Join is additionally enforced by the explicit ".." substring
// guard in IsValidRegionID below.
var regionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.'-]*$`)

// Region is one admin-configured entry from the region catalog config file.
//
// Bbox element order is [minLon, minLat, maxLon, maxLat] — this exact order
// matches generator.go's "%f,%f,%f,%f" (MinLon,MinLat,MaxLon,MaxLat) and the
// `pmtiles extract --bbox=` argument order. Phase 22's client manifest
// depends on this exact element order; do not reorder.
type Region struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Bbox [4]float64 `json:"bbox"`
}

// LoadRegionCatalog reads the admin-supplied JSON config file at the path
// named by REGION_CATALOG_CONFIG_PATH and returns the valid region entries.
//
// Per D-05 / BACK-01, a missing REGION_CATALOG_CONFIG_PATH env var, a
// missing file, or an empty file are all tolerated and return an empty
// catalog with no error — only a real read error (permissions, etc.) or
// malformed JSON is returned as an error. Individual entries that fail
// ValidateRegion are skipped (logged, not fatal) rather than failing the
// whole load.
func LoadRegionCatalog() ([]Region, error) {
	path := os.Getenv("REGION_CATALOG_CONFIG_PATH")
	if path == "" {
		return []Region{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Region{}, nil
		}
		return []Region{}, err
	}

	if len(data) == 0 {
		return []Region{}, nil
	}

	var regions []Region
	if err := json.Unmarshal(data, &regions); err != nil {
		return nil, fmt.Errorf("parse region catalog config: %w", err)
	}

	valid := make([]Region, 0, len(regions))
	for _, r := range regions {
		if err := ValidateRegion(r); err != nil {
			log.Printf("[regions] skipping invalid region catalog entry %q: %v", r.ID, err)
			continue
		}
		valid = append(valid, r)
	}

	return valid, nil
}

// ValidateRegion checks a config entry's id against the allow-list and its
// bbox against valid lat/lon ranges and ordering. Reuses the exact
// comparison idiom from db/routes/waypoint_cluster.go.
func ValidateRegion(r Region) error {
	if !regionIDPattern.MatchString(r.ID) {
		return fmt.Errorf("id must match %s", regionIDPattern.String())
	}

	minLon, minLat, maxLon, maxLat := r.Bbox[0], r.Bbox[1], r.Bbox[2], r.Bbox[3]

	if minLat < -90 || minLat > 90 {
		return fmt.Errorf("bbox min_lat out of range: %f", minLat)
	}
	if maxLat < -90 || maxLat > 90 {
		return fmt.Errorf("bbox max_lat out of range: %f", maxLat)
	}
	if minLon < -180 || minLon > 180 {
		return fmt.Errorf("bbox min_lon out of range: %f", minLon)
	}
	if maxLon < -180 || maxLon > 180 {
		return fmt.Errorf("bbox max_lon out of range: %f", maxLon)
	}
	if minLon >= maxLon {
		return fmt.Errorf("bbox min_lon must be less than max_lon")
	}
	if minLat >= maxLat {
		return fmt.Errorf("bbox min_lat must be less than max_lat")
	}

	return nil
}

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
