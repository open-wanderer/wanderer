---
phase: 24-settings-offline-maps-regions-ui
plan: 04
subsystem: ui
tags: [flutter, riverpod, objectbox, region-download]

# Dependency graph
requires:
  - phase: 24-settings-offline-maps-regions-ui
    provides: "Plan 02's _buildActiveRow row rendering and Plan 03's disk-space fallback fix, both prerequisites for this staleness bug becoming observable"
provides:
  - "resolveRowStatus pure resolver preferring live ephemeral download state over the ObjectBox ToOne-stale region.status, except for DEM-only downloads"
  - "_buildActiveRow now renders the live downloading/paused state during an in-flight vector download instead of a frozen not-downloaded/error status"
affects: [25-map-rendering-region-based-viewport-pipeline]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure resolver util pattern (mirrors disk_space_util.dart's hasEnoughSpace, map_cache_path.dart): isolate a stale-vs-live state decision into a dependency-free, unit-tested function rather than inlining it in a widget build method"

key-files:
  created:
    - app/lib/util/region_row_status_util.dart
    - app/test/util/region_row_status_util_test.dart
  modified:
    - app/lib/routes/settings_offline_regions_screen.dart

key-decisions:
  - "resolveRowStatus falls back to persisted region.status for DEM-only downloads (demProgress != null, vectorProgress == null) so region.status keeps tracking only the vector package lifecycle, per RegionEntity.status's own doc comment and to avoid regressing UAT test 2 (SETUI-04)"

patterns-established:
  - "Pure resolver util pattern for stale-ObjectBox-ToOne vs. live-ephemeral-provider render decisions"

requirements-completed: [SETUI-03, SETUI-04]

# Metrics
duration: 10min
completed: 2026-07-23
---

# Phase 24 Plan 04: Region Row Status Resolver Summary

**Fixed the stale ObjectBox ToOne-cached `region.status` render bug by adding a pure `resolveRowStatus` resolver that prefers the live ephemeral download state during an in-flight vector download, restoring the progress bar/pause-button UI and unblocking the paused mid-transfer disk-usage check.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-07-23T07:50:00Z (approx)
- **Completed:** 2026-07-23T07:51:18Z
- **Tasks:** 2 completed
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- Added `resolveRowStatus(RegionStatus persisted, RegionDownloadState? live)`, a pure, dependency-free resolver documenting and fixing the root cause diagnosed in `.planning/debug/region-download-stale-toone.md` (ObjectBox `ToOne.target` caches permanently per Dart-object instance)
- Wired `_buildActiveRow` to compute `status` via `resolveRowStatus(region.status, downloadState)` instead of reading the ToOne-stale `region.status` directly
- Closes UAT test 1's gap (never-downloaded region now visibly transitions through `downloading` with a progress bar and pause button) and unblocks UAT test 3 (paused mid-transfer state is now reachable, so disk-usage partial `.part` bytes can be observed)
- Preserves UAT test 2 (DEM-only downloads keep the row on its persisted vector lifecycle state — checkmark/delete for a downloaded region, download button for a not-downloaded one — while the inline DEM spinner still shows)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create the pure resolveRowStatus resolver + unit test** - `697accb0` (feat)
2. **Task 2: Wire _buildActiveRow to render from resolveRowStatus** - `7ebde9db` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/util/region_row_status_util.dart` - Pure `resolveRowStatus` resolver: `live == null` → `persisted`; DEM-only `live` → `persisted`; otherwise → `live.status`
- `app/test/util/region_row_status_util_test.dart` - Unit tests: idle, terminal-idle (error), vector-download-in-flight, resume-in-flight, DEM-only (downloaded persisted), DEM-only (notDownloaded persisted)
- `app/lib/routes/settings_offline_regions_screen.dart` - Added `region_row_status_util.dart` import; `_buildActiveRow`'s `status` now computed via `resolveRowStatus(region.status, downloadState)` instead of the bare `region.status`

## Decisions Made
- Kept the minimal-blast-radius fix identified in the debug session: only the one render-source expression changed. No changes to `TileRepositoryManager`, `_save`'s terminal-invalidate contract, the DEM toggle/spinner logic, `_combinedProgress`, or `_buildTrailingActions` — all of those already correctly switch on whatever `status` value they're handed, so the fix is entirely in choosing the right `status` value upstream.

## Deviations from Plan

None - plan executed exactly as written. `dart format` reformatted the resolver's function signature across three lines (cosmetic only, functionally identical to the plan's action text).

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both diagnosed UAT re-verification gaps (test 1 and test 3) are closed at the code level. On-device re-verification is deferred to the phase's end-of-phase human-verify UAT re-run (per `human_verify_mode: end-of-phase` in config), not gated in this plan.
- `flutter test test/util/region_row_status_util_test.dart test/models/region_download_state_test.dart test/entities/region_entity_test.dart` all pass (29 total), confirming no collateral breakage in the surrounding region model layer.
- `flutter analyze lib/routes/settings_offline_regions_screen.dart` reports no issues.

---
*Phase: 24-settings-offline-maps-regions-ui*
*Completed: 2026-07-23*

## Self-Check: PASSED

- FOUND: app/lib/util/region_row_status_util.dart
- FOUND: app/test/util/region_row_status_util_test.dart
- FOUND: 697accb0 (Task 1 commit)
- FOUND: 7ebde9db (Task 2 commit)
