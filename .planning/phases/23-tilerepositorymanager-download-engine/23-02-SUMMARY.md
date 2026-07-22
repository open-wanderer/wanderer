---
phase: 23-tilerepositorymanager-download-engine
plan: 02
subsystem: mobile-offline-tiles
tags: [dart, flutter, objectbox, enum-persistence, path-traversal, region-tile-repository]

# Dependency graph
requires:
  - phase: 22-region-package-data-model
    provides: PackageStatus/RegionStatus enums, RegionEntity/DownloadedTilePackageEntity with append-only .code shadow pattern, CatalogStatus
provides:
  - PackageStatus.paused(3)/error(4) and RegionStatus.paused(4)/error(5) appended enum members
  - RegionEntity.status getter extended to map paused/error vector package states
  - region_file_path.dart: regionIdPattern allow-list + assertValidRegionId/isValidRegionId + regionVectorPath/regionDemPath/regionStorageDir
affects: [23-03, 23-04, 23-05, 23-06, download engine plans consuming region file paths and paused/error states]

# Tech tracking
tech-stack:
  added: []
  patterns: ["append-only explicit-.code enum extension (never renumber/insert)", "region-id allow-list validated before any p.join, mirroring backend RegionIdSchema"]

key-files:
  created:
    - app/lib/util/region_file_path.dart
    - app/test/util/region_file_path_test.dart
  modified:
    - app/lib/models/region_status.dart
    - app/lib/entities/region_entity.dart
    - app/test/models/region_status_test.dart
    - app/test/entities/region_entity_test.dart

key-decisions:
  - "PackageStatus.paused/error appended at codes 3/4; RegionStatus.paused/error appended at codes 4/5 — matches Phase 22's append-only .code contract exactly, no renumbering"
  - "region_file_path.dart's regionIdPattern is byte-for-byte identical to the backend's RegionIdSchema regex (^[a-z0-9][a-z0-9_-]*$), so an id the server accepts is the same id the app accepts"

patterns-established:
  - "Path-safety utility: validate-then-join — assertValidRegionId always runs before p.join, never string-concatenated"

requirements-completed: [TILE-01, TILE-04]

# Metrics
duration: 6min
completed: 2026-07-22
---

# Phase 23 Plan 02: Enum extension + region path-safety utility Summary

**Appended paused/error to PackageStatus and RegionStatus with stable new codes, wired RegionEntity.status to map them, and shipped region_file_path.dart validating catalog region ids against the backend's exact allow-list before building vector/dem archive paths.**

## Performance

- **Duration:** ~6 min
- **Tasks:** 2 completed
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments
- `PackageStatus.paused(3)`/`PackageStatus.error(4)` and `RegionStatus.paused(4)`/`RegionStatus.error(5)` appended after the existing members, preserving all prior codes (0/1/2/3) for on-device rows written before this change
- `RegionEntity.status`'s switch now exhaustively handles `PackageStatus.paused`/`PackageStatus.error`, mapping each to the matching `RegionStatus`
- New `app/lib/util/region_file_path.dart` provides `regionIdPattern`, `isValidRegionId`, `assertValidRegionId`, `regionStorageDir`, `regionVectorPath`, `regionDemPath` — every builder routes the id through `assertValidRegionId` before any `p.join`, never string-concatenating it

## Task Commits

Each task was committed atomically:

1. **Task 1: Append paused/error to PackageStatus + RegionStatus and handle them in RegionEntity.status** - `1653214c` (feat)
2. **Task 2: region_file_path.dart — region-id allow-list + vector/dem path builders** - `3ec812ee` (feat)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `app/lib/models/region_status.dart` - Appended `paused`/`error` members to `PackageStatus` (codes 3/4) and `RegionStatus` (codes 4/5)
- `app/lib/entities/region_entity.dart` - `status` getter switch extended with `PackageStatus.paused`/`PackageStatus.error` cases
- `app/test/models/region_status_test.dart` - Asserts new `.code` values, decode-by-code, and append-only ordering for both enums
- `app/test/entities/region_entity_test.dart` - Asserts `dbStatus` round-trips 3/4, and `RegionEntity.status` maps paused/error vector packages correctly
- `app/lib/util/region_file_path.dart` - New region-id allow-list + `regionVectorPath`/`regionDemPath`/`regionStorageDir` path builders
- `app/test/util/region_file_path_test.dart` - Valid/invalid id coverage, traversal-rejection, and `p.isWithin` happy-path assertions for both vector and dem paths

## Decisions Made
- Followed the plan's append-only contract exactly: `paused`/`error` placed at the end of each enum with brand-new codes, never inserted or renumbered
- `region_file_path.dart`'s allow-list regex is copied verbatim from the backend `RegionIdSchema` (`web/src/routes/api/v1/regions/[id]/download/+server.ts`) so the client never diverges from the server's validation

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `PackageStatus`/`RegionStatus` and `region_file_path.dart` are ready for consumption by the download-engine plans that persist paused/error states and read/write region archive files.
- No blockers identified.

---
*Phase: 23-tilerepositorymanager-download-engine*
*Completed: 2026-07-22*

## Self-Check: PASSED

- FOUND: app/lib/util/region_file_path.dart
- FOUND: app/test/util/region_file_path_test.dart
- FOUND: commit 1653214c
- FOUND: commit 3ec812ee
