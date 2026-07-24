---
phase: 27-legacy-cleanup
plan: 01
subsystem: mobile-app
tags: [flutter, riverpod, tile-download, cleanup, dead-code-removal]

# Dependency graph
requires:
  - phase: 26-trail-download-guard
    provides: DownloadingTrailIds.download() with vectorLatched/demLatched aggregate progress, coverage guard, trailSucceeded-gated success notification
provides:
  - Trail download guard's downloadTrail() call site with only onProgress (no onGeneratingChanged)
  - DownloadNotificationService without showGenerating()
  - TrailDownloadService.downloadTrail() reduced to photo + waypoint-photo + nav-cache download
  - map_cell.dart (+ generated siblings) deleted app-wide
affects: [27-02-legacy-cleanup]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - app/lib/provider/trail/trail_download_state_provider.dart
    - app/lib/services/download_notification_service.dart
    - app/lib/services/trail_download_service.dart
    - app/lib/models/map_cell.dart (deleted)
    - app/lib/models/map_cell.freezed.dart (deleted)
    - app/lib/models/map_cell.g.dart (deleted)

key-decisions:
  - "totalPoints is now computed up front from photoTotal * _pointsPerUnit instead of inside the deleted tile future's onCellTotal callback, so onProgress reporting keeps working for photo/waypoint-photo downloads"
  - "package:maplibre import removed from trail_download_service.dart since LngLatBounds was only used by the deleted _downloadMapTiles method"

patterns-established: []

requirements-completed: [CLEAN-01]

# Metrics
duration: 12min
completed: 2026-07-24
---

# Phase 27 Plan 01: Strip Trail-Scoped Tile-Download Machinery Summary

**Deleted the three trail-scoped tile-download methods, all generating-state wiring, the showGenerating() spinner, and the orphaned map_cell.dart model from the Flutter app, leaving downloadTrail() to handle only photos/waypoint-photos/nav-cache with a fixed up-front progress total.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-24T17:25:00Z
- **Completed:** 2026-07-24T17:37:10Z
- **Tasks:** 2 completed
- **Files modified:** 6 (3 modified, 3 deleted)

## Accomplishments
- Removed the `onGeneratingChanged`/`showGenerating` wiring from the Phase 26 guard's `downloadTrail()` call site (D-03), and deleted `DownloadNotificationService.showGenerating()` after confirming zero remaining callers (D-07)
- Deleted `TrailDownloadService._downloadMapTiles`, `_fetchCellList`, `_pollUntilReady` and every piece of tile/generating-state wiring around them: `isGenerating`, `handleGeneratingChanged`, `onCellPointsDelta`, `tileResult`, both `entity.pmTiles`/`entity.demPmTiles` writes, `_pollInterval`/`_pollTimeout` (D-01, D-02)
- Fixed a would-be functional regression: since the deleted tile future's `onCellTotal` was the only place `totalPoints` got assigned, `totalPoints` is now computed up front (`photoTotal * _pointsPerUnit`) so `onProgress` continues to fire correctly for the remaining photo + waypoint-photo download work (D-04)
- Deleted the now-orphaned `map_cell.dart`, `map_cell.freezed.dart`, `map_cell.g.dart` (`MapCellInfoList`/`MapCellInfo`/`MapCellStatusResponse`/`MapCellStatus`) — zero remaining references app-wide
- 26-04/26-05 invariants (vectorLatched/demLatched latch, single `remove(trail.id)`, single `invalidate(regionListNotifierProvider)`, `trailSucceeded`-gated deferred `showSuccess`) verified untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: Strip onGeneratingChanged/showGenerating wiring from the Phase 26 guard and delete the dead showGenerating() spinner** - `e6dc8d50` (refactor)
2. **Task 2: Delete the three tile-download methods and all tile/generating wiring from TrailDownloadService; delete map_cell.dart** - `62c700a7` (refactor)

## Files Created/Modified
- `app/lib/provider/trail/trail_download_state_provider.dart` - `downloadTrail()` call site now passes only `onProgress`; `onGeneratingChanged` block deleted
- `app/lib/services/download_notification_service.dart` - `showGenerating()` method and its "Generating map tiles..." string deleted
- `app/lib/services/trail_download_service.dart` - `_downloadMapTiles`/`_fetchCellList`/`_pollUntilReady` and all tile/generating-state wiring deleted; `downloadTrail()`'s `onGeneratingChanged` param removed; `totalPoints` now initialized up front; unused `package:maplibre` import removed
- `app/lib/models/map_cell.dart` - deleted
- `app/lib/models/map_cell.freezed.dart` - deleted
- `app/lib/models/map_cell.g.dart` - deleted

## Decisions Made
- `totalPoints` changed from a nullable, callback-assigned value to a `final` computed immediately after `photoTotal`, since the only assignment site (the deleted tile future's `onCellTotal`) no longer exists — this also removed dead-code/unnecessary-null-check warnings the analyzer flagged
- Removed the now-unused `package:maplibre/maplibre.dart` import from `trail_download_service.dart` (its only use, `LngLatBounds bounds = trail.bounds`, lived inside the deleted `_downloadMapTiles`)

## Deviations from Plan

None - plan executed exactly as written. The two functional fixes called out above (`totalPoints` up-front initialization, unused-import removal) were explicitly anticipated by the plan's own action text ("CRITICAL functional fix" / Rule 1 auto-fix for the resulting dead code and unused import), not unplanned deviations.

## Issues Encountered

`flutter analyze` and `flutter test` both surfaced pre-existing, out-of-scope failures unrelated to this plan's files:
- 37 analyzer issues in `map_screen.dart`, `icon_util.dart`, `navigation_stats_provider.dart`, `settings_categories_screen.dart`, `settings_subcategories_screen.dart` (info/warning level, no errors) — confirmed identical before and after this plan's edits
- 4 failing tests in `test/components/route_planner/settings_tab_test.dart` — confirmed via a targeted revert-and-rerun of this plan's two edited files that these failures exist on the pre-plan baseline (unrelated to trail download or notification code)

Both were confirmed unrelated to this plan's `files_modified` scope and logged here rather than fixed, per the executor's scope-boundary rule.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CLEAN-01's runtime portion is complete: trail-scoped tile-download methods, generating-state wiring, `showGenerating()`, and `map_cell.dart` are gone with zero remaining references
- `TrailEntity.pmTiles`/`Trail.pmTiles`/`demPmTiles` fields intentionally still exist (unwritten) — this is the expected intermediate state; plan 27-02 removes them and regenerates ObjectBox/freezed code with a single `build_runner` pass
- No blockers for 27-02

---
*Phase: 27-legacy-cleanup*
*Completed: 2026-07-24*

## Self-Check: PASSED

- FOUND: app/lib/provider/trail/trail_download_state_provider.dart
- FOUND: app/lib/services/download_notification_service.dart
- FOUND: app/lib/services/trail_download_service.dart
- CONFIRMED DELETED: app/lib/models/map_cell.dart
- CONFIRMED DELETED: app/lib/models/map_cell.freezed.dart
- CONFIRMED DELETED: app/lib/models/map_cell.g.dart
- FOUND commit: e6dc8d50
- FOUND commit: 62c700a7
