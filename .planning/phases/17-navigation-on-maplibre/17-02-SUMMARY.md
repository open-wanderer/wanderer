---
phase: 17-navigation-on-maplibre
plan: 02
subsystem: mobile-map
tags: [flutter, maplibre, cleanup, compass]

# Dependency graph
requires:
  - phase: 17-navigation-on-maplibre (plan 01)
    provides: "navigation_screen.dart fully migrated off flutter_map — map_compass.dart and pm_tile_provider.dart left with zero real importers"
provides:
  - "map_compass.dart and pm_tile_provider.dart deleted from the repo (CORE-05, OFFL-06)"
  - "trail_layer.dart free of flutter_map imports; legacy TrailLayer/_TrailLayerState widget removed, native addTrailTrackLayers/TrailMarkerLayer retained"
  - "trail_detail_map_screen.dart shows a native ml.MapCompass again, closing the Phase-15 CORE-05 gap recorded in STATE.md"
  - "Repo-wide grep confirms zero AnimatedMapController/CurrentLocationLayer references remain in lib/ (ROADMAP Phase 17 success criterion 4)"
affects: [17-03-navigation-on-maplibre, 18-retire-flutter-map-and-flomp-forks]

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - app/lib/components/map/trail_layer.dart
    - app/lib/routes/trail_detail_map_screen.dart
    - app/lib/util/tracelet_position_source.dart
    - app/lib/routes/map_screen.dart

key-decisions:
  - "Reworded two stray doc-comment references to the deleted CurrentLocationLayer name (tracelet_position_source.dart, map_screen.dart) — not in the plan's files_modified list, but required to make the plan's own repo-wide grep gate (AnimatedMapController/CurrentLocationLayer) genuinely pass; the plan's Task 2 action text explicitly anticipated fixing any stray reference surfaced."
  - "Left lib/vendor/vector_map_tiles/ as an empty directory after deleting pm_tile_provider.dart — no other files remain there; removing the directory itself is not required by any acceptance criterion and git does not track empty directories."

patterns-established: []

requirements-completed: [CORE-05, CORE-06, CORE-07, OFFL-06]

# Metrics
duration: 8min
completed: 2026-07-10
---

# Phase 17 Plan 02: Retire flutter_map Holdout Files Summary

**Deleted the app-local `map_compass.dart` (CORE-05) and deferred `pm_tile_provider.dart` (OFFL-06), stripped the legacy flutter_map `TrailLayer` widget from `trail_layer.dart`, and restored a native `ml.MapCompass` to `trail_detail_map_screen.dart` — closing the Phase-15 CORE-05 gap and passing a repo-wide grep gate proving zero `AnimatedMapController`/`CurrentLocationLayer` references remain in `lib/`.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-07-10T10:00:00Z (approx, session start)
- **Completed:** 2026-07-10T10:08:21Z
- **Tasks:** 2/2 completed
- **Files modified:** 4 (2 deleted, 4 modified — `trail_layer.dart` counted once)

## Accomplishments

- Deleted `app/lib/components/map/map_compass.dart` (the app-local, flutter_map-only compass widget, CORE-05) — its only real importer, `navigation_screen.dart`, migrated off it in 17-01.
- Deleted `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart` (`MultiPmTilesVectorTileProvider`, OFFL-06) — navigation was its last caller.
- Removed the legacy `flutter_map`-based `TrailLayer`/`_TrailLayerState` widget from `trail_layer.dart`, along with its now-unused `package:flutter_map/flutter_map.dart` and `map_coordinate_adapter.dart` imports. `addTrailTrackLayers`, `_colorToHex`, `TrailMarkerLayer`, and `_buildCircularMarker` are untouched and still exported for `WandererMap`/`navigation_screen.dart`.
- Restored `const ml.MapCompass(hideIfRotatedNorth: true)` to `trail_detail_map_screen.dart`'s `WandererMap.controls` list, replacing the stale Phase-15 explanatory comment — matches `map_screen.dart`'s established treatment exactly.
- Reworded two stray doc-comment mentions of the deleted `CurrentLocationLayer` class name (in `tracelet_position_source.dart` and `map_screen.dart`) so the repo-wide grep gate is genuinely clean, not just code-clean.
- Ran the full repo-wide verification: zero `AnimatedMapController`/`CurrentLocationLayer` references, zero `map_compass.dart`/`pm_tile_provider.dart`/`MultiPmTilesVectorTileProvider` references, and `flutter analyze` over the whole `lib/` tree reports zero errors (36 pre-existing warning/info issues remain, all in unrelated files — logged to `deferred-items.md`, not fixed, per scope boundary).

## Task Commits

1. **Task 1: Delete map_compass.dart + pm_tile_provider.dart and strip the legacy TrailLayer widget from trail_layer.dart** - `48f62a2b` (feat)
2. **Task 2: Restore native ml.MapCompass to trail_detail_map_screen and grep-verify no flutter_map controller/location references remain in lib/** - `70014e0d` (feat)

**Plan metadata:** commit skipped — `commit_docs: false` in `.planning/config.json` (see Final Commit note below)

## Files Created/Modified

- `app/lib/components/map/map_compass.dart` - Deleted (CORE-05).
- `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart` - Deleted (OFFL-06).
- `app/lib/components/map/trail_layer.dart` - Legacy `TrailLayer`/`_TrailLayerState` widget and its `flutter_map`/`map_coordinate_adapter` imports removed; native `addTrailTrackLayers`/`TrailMarkerLayer`/`_buildCircularMarker` retained.
- `app/lib/routes/trail_detail_map_screen.dart` - Native `ml.MapCompass(hideIfRotatedNorth: true)` added to `WandererMap.controls`, replacing the stale explanatory comment.
- `app/lib/util/tracelet_position_source.dart` - Doc comment reworded to drop the stale `CurrentLocationLayer` name reference.
- `app/lib/routes/map_screen.dart` - Doc comment reworded to drop the stale `CurrentLocationLayer` name reference.

## Decisions Made

- Fixed the two stray doc-comment references to `CurrentLocationLayer` even though `tracelet_position_source.dart`/`map_screen.dart` are outside this plan's `files_modified` frontmatter — Task 2's own action text explicitly instructs "Fix any stray reference surfaced," and leaving them would make the plan's own acceptance-criteria grep fail.
- Left `lib/vendor/vector_map_tiles/` as an empty directory rather than removing it — not required by any acceptance criterion.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Reworded two stray CurrentLocationLayer doc-comment references outside files_modified scope**
- **Found during:** Task 2 (repo-wide grep gate)
- **Issue:** `tracelet_position_source.dart:8` and `map_screen.dart:738` still contained the literal string `CurrentLocationLayer` in doc comments (referring to the old, already-deleted flutter_map widget by name), which would make the plan's own mandatory grep gate (`! grep -rn "AnimatedMapController\|CurrentLocationLayer" lib/`) fail.
- **Fix:** Reworded both comments to describe the replacement (native location puck) without using the retired class name, preserving their documentation intent.
- **Files modified:** `app/lib/util/tracelet_position_source.dart`, `app/lib/routes/map_screen.dart`
- **Verification:** `grep -rn "AnimatedMapController\|CurrentLocationLayer" lib/` returns no matches; `flutter analyze` unaffected (no new issues in either file).
- **Committed in:** `70014e0d` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking, doc-comment only — no behavior change)
**Impact on plan:** Necessary to satisfy the plan's own verification gate; no scope creep — comment text only, same commit as the task that required it.

## Issues Encountered

None — `flutter analyze` reported zero errors after each task (36 pre-existing warning/info issues remain in unrelated files: `trail_dropdown.dart`, `settings_categories_screen.dart`, `settings_subcategories_screen.dart`, `icon_util.dart`, `test/models/feed_item_test.dart` — logged to `.planning/phases/17-navigation-on-maplibre/deferred-items.md`, out of scope per the Scope Boundary rule since none touch files this plan modified).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CORE-05, CORE-06, CORE-07, and OFFL-06 are formally closed: `map_compass.dart` and `pm_tile_provider.dart` are deleted, `trail_layer.dart` is flutter_map-free, and a repo-wide grep confirms zero `AnimatedMapController`/`CurrentLocationLayer` references remain in `lib/`.
- `trail_detail_map_screen.dart`'s Phase-15 compass gap (the STATE.md-documented blocker) is closed.
- `flutter_map`/plugin removal from `pubspec.yaml` itself remains Phase 18 (CLEAN-01/02/03) scope — out of scope here, unaffected.
- Ready for 17-03 (on-device navigation checkpoint) per the phase's wave sequencing.

## Self-Check: PASSED

- MISSING: `app/lib/components/map/map_compass.dart` (expected — deleted by design, CORE-05)
- MISSING: `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart` (expected — deleted by design, OFFL-06)
- FOUND: `app/lib/components/map/trail_layer.dart`
- FOUND: `app/lib/routes/trail_detail_map_screen.dart`
- FOUND: `48f62a2b` (Task 1 commit)
- FOUND: `70014e0d` (Task 2 commit)

---
*Phase: 17-navigation-on-maplibre*
*Completed: 2026-07-10*
