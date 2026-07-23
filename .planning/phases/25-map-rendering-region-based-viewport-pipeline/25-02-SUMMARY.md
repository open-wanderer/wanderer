---
phase: 25-map-rendering-region-based-viewport-pipeline
plan: 02
subsystem: mobile-offline-maps
tags: [flutter, dart, objectbox, tile-repository, maplibre, tdd]

# Dependency graph
requires:
  - phase: 25-01
    provides: RENDER-03 incremental composition strategy decision (addSource/removeSource/addLayer/removeLayer), which Wave 2's map-screen wiring will pair with this plan's split return type
provides:
  - "TileRepositoryManager.localTilePathsForBounds returning a typed ({List<String> vectorPaths, List<String> demPaths}) record instead of one merged List<String>"
  - "@visibleForTesting splitRegionTilePaths pure helper, unit-tested without a live ObjectBox store"
  - "Unit-proven anti-conflation guarantee: a DEM archive path can never land in vectorPaths"
affects: [25-03, 25-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure per-region helper takes Iterable<RegionEntity> instead of _store, mirroring the existing bboxOverlaps @visibleForTesting precedent, so store-dependent logic gains an in-memory-testable seam"
    - "Typed Dart record ({List<String> vectorPaths, List<String> demPaths}) as a public method return type to make a two-list contract structurally explicit at call sites"

key-files:
  created: []
  modified:
    - app/lib/services/tile_repository_manager.dart
    - app/test/services/tile_repository_manager_test.dart
    - app/test/services/tile_repository_manager_harness.dart

key-decisions:
  - "No architectural deviations — plan executed exactly as written"

patterns-established:
  - "Any future TileRepositoryManager query method that must be unit-tested without ObjectBox should extract its per-region logic into a @visibleForTesting top-level pure function taking Iterable<RegionEntity>, following bboxOverlaps/splitRegionTilePaths"

requirements-completed: [RENDER-01]

# Metrics
duration: ~10min
completed: 2026-07-23
---

# Phase 25 Plan 02: Split localTilePathsForBounds into a typed vector/DEM record Summary

**`TileRepositoryManager.localTilePathsForBounds` now returns `({List<String> vectorPaths, List<String> demPaths})` via a unit-tested `@visibleForTesting splitRegionTilePaths` pure helper, closing the RENDER-01 data-shape gap before any Wave 2 rendering code can conflate a DEM archive with a vector cell.**

## Performance

- **Duration:** ~10 min
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments
- Split `localTilePathsForBounds`'s single merged `List<String>` return into a typed two-list record so vector and DEM paths are structurally impossible to conflate
- Extracted the per-region split logic into a new `@visibleForTesting splitRegionTilePaths` top-level function (sibling to the existing `bboxOverlaps`), unit-testable with plain in-memory `RegionEntity`/`DownloadedTilePackageEntity` objects — no live ObjectBox `Store` required
- Added 6 new unit tests proving every case in the plan's `<behavior>` block, including the DEM-only anti-conflation guarantee (a DEM path lands in `demPaths` and never in `vectorPaths`)
- Updated the sole existing call site — the 23-06 on-device harness — to destructure and print the new record shape

## Task Commits

Each task was committed atomically (Task 1 followed the plan's TDD flow):

1. **Task 1 — RED: add failing splitRegionTilePaths tests** — `4b3368bf` (test)
2. **Task 1 — GREEN: implement splitRegionTilePaths + split localTilePathsForBounds** — `5d639a5c` (feat)
3. **Task 2: Update the 23-06 harness call site to the split return shape** — `92bd26af` (fix)

_No REFACTOR commit needed — the GREEN implementation needed no cleanup pass._

## Files Created/Modified
- `app/lib/services/tile_repository_manager.dart` - Added `@visibleForTesting splitRegionTilePaths` pure helper; `localTilePathsForBounds` now delegates to it and returns the typed record
- `app/test/services/tile_repository_manager_test.dart` - New `splitRegionTilePaths` test group (6 cases) mirroring the existing `bboxOverlaps` group's shape
- `app/test/services/tile_repository_manager_harness.dart` - `_queryBounds` destructures `result.vectorPaths`/`result.demPaths` and prints both, instead of treating the return value as a bare `List<String>`

## Decisions Made
None - plan executed exactly as written.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The RENDER-01 blocker identified in `25-RESEARCH.md` is closed: `localTilePathsForBounds` now hands Wave 2's map-screen wiring (25-03/25-04) a typed, test-proven `({vectorPaths, demPaths})` contract that maps 1:1 onto `rewriteStyleForOffline`'s `cellPaths`/`demCellPaths` params, making the DEM-into-vector-cell conflation bug class structurally unreachable at this layer.

Per STATE.md, Wave 2 (25-03/25-04) still needs plan revision before execution to account for two on-device findings from 25-01 (repaint-on-remove gap, hillshade z-order gap) — unrelated to this plan's scope, tracked separately.

## Self-Check: PASSED

- FOUND: app/lib/services/tile_repository_manager.dart
- FOUND: app/test/services/tile_repository_manager_test.dart
- FOUND: app/test/services/tile_repository_manager_harness.dart
- FOUND commit: 4b3368bf
- FOUND commit: 5d639a5c
- FOUND commit: 92bd26af

---
*Phase: 25-map-rendering-region-based-viewport-pipeline*
*Completed: 2026-07-23*
