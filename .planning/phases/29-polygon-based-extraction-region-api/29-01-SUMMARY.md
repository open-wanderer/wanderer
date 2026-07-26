---
phase: 29-polygon-based-extraction-region-api
plan: 01
subsystem: backend
tags: [go, pocketbase, pmtiles, cron, region-archives, regions-table]

# Dependency graph
requires:
  - phase: 28-region-catalog-data-model-seeding
    provides: Seeded `regions` PocketBase collection (1306 rows, hierarchical group/leaf, canonical `polygon` + `bbox` per leaf, materialized `path` unique key)
provides:
  - "regions.path-derived region ids that safely allow '.' and apostrophe while still rejecting path traversal"
  - "writePolygonTempFile helper for marshaling a leaf's GeoJSON polygon to a unique temp .geojson file"
  - "Table-driven, polygon-based cron build path: BuildAll queries `regions` (kind='leaf' && enabled=true) and both buildVector/buildDem extract via `pmtiles extract --region=<polygon file>`"
affects: [29-02-region-api-and-catalog-loader-retirement, 30-admin-region-picker-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "pmtiles extract --region=<temp .geojson file> replaces --bbox=<string> for polygon-accurate archive clipping"
    - "Cron build targets sourced by direct dbx-filtered FindAllRecords query against a PocketBase collection, replacing a config-file loader"

key-files:
  created:
    - db/services/regions/polygon_extract.go
    - db/services/regions/polygon_extract_test.go
  modified:
    - db/services/regions/config.go
    - db/services/regions/config_test.go
    - db/services/regions/builder.go

key-decisions:
  - "regionIDPattern relaxed to ^[a-z0-9][a-z0-9_.'-]*$ (allows the materialized-path separator '.' and apostrophes in real seeded names) with an explicit strings.Contains(id, \"..\") guard as defense-in-depth against traversal"
  - "region_archives.region_id now stores regions.path (the provably-unique seeded key) instead of an admin-typed slug, per A2"
  - "Polygon temp file is written lazily — only when a vector or DEM rebuild is actually needed — and always cleaned up via a single top-level defer os.Remove"
  - "The old bboxChanged forced-rebuild trigger is kept, re-keyed to the seeded catalog's bbox (A4), not deleted, to preserve the rare-catalog-refresh rebuild path"
  - "config.go's LoadRegionCatalog/Region/ValidateRegion catalog loader intentionally left in place — still referenced by db/routes/regions_get.go, retired in plan 29-02"

patterns-established:
  - "Any future PocketBase-record-driven cron/build path should follow buildRegion's shape: derive an id from a table field (not the record's own opaque .Id), read structured fields via UnmarshalJSONField, and gate expensive work (temp file writes, subprocess calls) behind an explicit needs-rebuild check computed up front"

requirements-completed: [EXTRACT-01, EXTRACT-02]

# Metrics
duration: 25min
completed: 2026-07-26
---

# Phase 29 Plan 1: Polygon-Based Extraction & Table-Driven Cron Build Summary

**Region-archive cron now reads build targets from the seeded `regions` table (`kind='leaf' && enabled=true`) and clips both vector and DEM PMTiles archives to each leaf's canonical GeoJSON polygon via `pmtiles extract --region=<temp file>`, retiring bbox-based extraction from the build path.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-07-26T19:35:00Z
- **Completed:** 2026-07-26T20:00:00Z
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments
- Relaxed the region-id allow-list (`regionIDPattern`) to accept real seeded `regions.path` values containing `.` and `'`, while adding an explicit `..`-substring traversal guard to `IsValidRegionID`
- Added `writePolygonTempFile`, a small helper that marshals a leaf's GeoJSON `polygon` field to a unique temp `.geojson` file for `pmtiles extract --region=`, rejecting nil/empty geometry
- Rewrote `builder.go`'s entire build path to be table-driven: `BuildAll` now queries `regions` directly instead of `LoadRegionCatalog()`, and both `buildVector`/`buildDem` clip via `--region=<polygon file>` instead of `--bbox=`

## Task Commits

Each task was committed atomically:

1. **Task 1: Relax region-id allow-list for materialized paths + traversal guard** - `9cd102a9` (feat)
2. **Task 2: writePolygonTempFile helper (polygon field -> temp .geojson)** - `7914f444` (feat)
3. **Task 3: Table-driven, polygon-based cron build path** - `0a63d41b` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `db/services/regions/config.go` - `regionIDPattern` relaxed to `^[a-z0-9][a-z0-9_.'-]*$`; `IsValidRegionID` adds an explicit `..` substring guard
- `db/services/regions/config_test.go` - Extended `TestIsValidRegionID` / `TestRegionIDCannotProduceTraversalPath` with accept (`.`/`'`-containing paths) and reject (`..`, `a\b`, leading `.`) cases
- `db/services/regions/polygon_extract.go` - New `writePolygonTempFile(polygon map[string]any) (string, error)` helper
- `db/services/regions/polygon_extract_test.go` - Covers valid Polygon/MultiPolygon, nil/empty rejection, and distinct-path uniqueness
- `db/services/regions/builder.go` - `BuildAll`/`buildRegionSafely`/`buildRegion`/`findOrCreateRegionRecord`/`buildVector`/`buildDem` rewritten to operate on `*core.Record` + a `path`-derived region id; both extraction calls now use `--region=`

## Decisions Made
- Kept `config.go`'s catalog loader (`LoadRegionCatalog`/`Region`/`ValidateRegion`) untouched in this plan since `db/routes/regions_get.go` still depends on it — full retirement is scoped to plan 29-02, keeping the `db` package compiling at this plan's boundary (matches the plan's own stated design).
- Polygon temp file write is lazy (only when `needsVector || needsDem`), avoiding an unnecessary filesystem write on every cron pass for regions that are already fully built and current.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `db` package builds and vets cleanly (`go build -C db ./...`, `go vet -C db ./services/regions/`); `go test -C db ./services/regions/` passes in full (config, staleness, polygon, builder-adjacent tests)
- Plan 29-02 can now retire `LoadRegionCatalog`/`Region`/`ValidateRegion` from `config.go` and migrate `db/routes/regions_get.go`'s `RegionsList` handler to read the `regions` table directly, adding the `parent`/`path`/`depth` hierarchy fields (EXTRACT-03)
- Known non-blocking note carried into future verification: the orphaned `munich` `region_archives` row/files (no matching `regions.path`) are inert dead data now that catalog parsing here is bypassed for the cron — flagged in the plan's own `<notes>`, no cleanup task needed
- End-to-end real polygon-clipping and cron table-driven selection are proven live in research (Pattern 1, hands-on `pmtiles extract --region=` run) and deferred to the 29-04 human-verify checkpoint, not re-proven by unit tests here (consistent with the plan's own `<verification>` section)

## Self-Check: PASSED

- `db/services/regions/polygon_extract.go` exists, contains `func writePolygonTempFile` ✓
- `db/services/regions/builder.go` contains `--region=` (2 occurrences) ✓
- `db/services/regions/config.go` contains the relaxed regex and traversal guard ✓
- `git log --oneline --grep="29-01"` returns 3 commits (task commits use `feat(29-01):` prefix) ✓
- All task-level acceptance criteria re-verified: `go test -C db ./services/regions/ -run 'TestIsValidRegionID|TestRegionIDCannotProduceTraversalPath'` PASS; `go test -C db ./services/regions/ -run TestWritePolygonTempFile` PASS; `go build -C db ./...` clean; `go vet -C db ./services/regions/` clean; `go test -C db ./services/regions/` full suite PASS
- Plan-level `<verification>` commands re-run: all green

---
*Phase: 29-polygon-based-extraction-region-api*
*Completed: 2026-07-26*
