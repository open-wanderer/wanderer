---
phase: 18-retire-flutter-map-and-the-flomp-forks
plan: 01
subsystem: mobile-map-cleanup
tags: [flutter, dart, dependency-cleanup, flutter_map, vector_map_tiles, flutter_map_location_marker, dead-code-removal]

# Dependency graph
requires:
  - phase: 17-navigation-on-maplibre
    provides: navigation_screen.dart fully migrated to native ml.MapLibreMap, map_compass.dart and pm_tile_provider.dart deleted, leaving map_style_provider.dart and flutter_map_location_marker's data classes as the only remaining source-level holdouts
provides:
  - effectiveBrightness(ThemeMode) relocated from map_style_provider.dart into map_style_json_provider.dart
  - File-local LocationMarkerPosition and ServiceDisabledException classes in foreground_position_stream_provider.dart, replacing the flutter_map_location_marker imports
  - Deletion of map_style_provider.dart, map_style_provider.g.dart, tool/extract_map_styles.dart, and lib/util/map_coordinate_adapter.dart
  - A codebase with zero source-level imports of any of the six map packages (flutter_map, flutter_map_animations, flutter_map_location_marker, flutter_map_marker_cluster, vector_map_tiles, vector_tile_renderer) while all six remain installed in pubspec.yaml
affects: [18-02-pubspec-removal, 18-03-on-device-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Code-before-manifest sequencing for dependency removal: sever every source import first (while the package stays installed, so flutter analyze diagnoses real code bugs, not missing-dependency errors), then remove the pubspec entry in a later plan"
    - "File-local plain-data-class replacement for a plugin's incidental data types: LocationMarkerPosition/ServiceDisabledException were consumed only as a Stream<T> payload shape, unrelated to the plugin's actual rendering widget, so a shape-identical local class is a drop-in replacement with zero call-site changes"

key-files:
  created: []
  modified:
    - app/lib/provider/map_style_json_provider.dart
    - app/lib/components/base/wanderer_map.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/provider/foreground_position_stream_provider.dart
    - app/lib/routes/map_screen.dart

key-decisions:
  - "Relocated effectiveBrightness() verbatim into map_style_json_provider.dart rather than a new shared file — both other importers already had an unrestricted import of that file, so this was a zero-new-import change"
  - "Left the 3 pre-existing flutter test failures (feed_item_test.dart x2, settings_screen_test.dart x1) undisturbed — confirmed via git stash bisect against the parent commit that they reproduce identically without any of this plan's changes, so they are out of scope per the executor's scope-boundary rule"

patterns-established:
  - "Sever-then-remove for Flutter package cleanup: any future package removal in this repo should first grep+delete all source-level importers with the package still in pubspec.yaml, run flutter analyze (whole package, not path-scoped) as the gate, and only then touch pubspec.yaml in a following plan/task"

requirements-completed: [CLEAN-01, CLEAN-02]

# Metrics
duration: 15min
completed: 2026-07-10
---

# Phase 18 Plan 01: Sever source-level map-package dependencies Summary

**Relocated `effectiveBrightness()` and replaced `LocationMarkerPosition`/`ServiceDisabledException` with file-local equivalents, then deleted four dead files — codebase now has zero source imports of the six map packages while all six remain installed, clearing the way for Plan 02's pubspec removal.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-10T12:08:00Z (approx.)
- **Completed:** 2026-07-10T12:23:30Z
- **Tasks:** 2/2 completed
- **Files modified:** 5 (3 deletions of dead files not counted separately here; see Files Created/Modified)

## Accomplishments
- `effectiveBrightness(ThemeMode)` now lives in `map_style_json_provider.dart`; its three former importers (`map_style_json_provider.dart` itself, `wanderer_map.dart`, `navigation_screen.dart`) reference nothing from the deleted `map_style_provider.dart`
- `LocationMarkerPosition` and `ServiceDisabledException` are now pure-Dart, file-local classes in `foreground_position_stream_provider.dart`, shape-identical to the originals — `wanderer_map.dart` and `map_screen.dart` resolve them through their existing unrestricted import of that file
- Deleted `map_style_provider.dart`, `map_style_provider.g.dart`, `tool/extract_map_styles.dart` (and the now-empty `app/tool/` directory), and `lib/util/map_coordinate_adapter.dart`
- Whole-package `flutter analyze` is clean (36 pre-existing, unrelated warnings/infos — zero errors) with all six map packages still present in `pubspec.yaml`
- Repo-wide grep confirms zero remaining source imports of `flutter_map`, `vector_map_tiles`, or `vector_tile_renderer` anywhere in `app/lib`, `app/test` (`app/tool` no longer exists)

## Task Commits

Each task was committed atomically:

1. **Task 1: Relocate effectiveBrightness and delete the four dead vector-map / adapter files** - `c06d9546` (feat)
2. **Task 2: Replace LocationMarkerPosition / ServiceDisabledException with file-local classes and repoint importers** - `1ffaa725` (feat)

**Plan metadata:** _pending final docs commit_

## Files Created/Modified
- `app/lib/provider/map_style_json_provider.dart` - Added the relocated `effectiveBrightness(ThemeMode)` top-level function; dropped its now-redundant import of `map_style_provider.dart`
- `app/lib/components/base/wanderer_map.dart` - Dropped the `map_style_provider.dart show effectiveBrightness` import (Task 1) and the `flutter_map_location_marker show LocationMarkerPosition` import (Task 2); both symbols now resolve via already-present unrestricted imports
- `app/lib/routes/navigation_screen.dart` - Dropped the `map_style_provider.dart show effectiveBrightness` import; resolves via the already-present unrestricted `map_style_json_provider.dart` import
- `app/lib/provider/foreground_position_stream_provider.dart` - Removed the `flutter_map_location_marker` import; added file-local `LocationMarkerPosition` (const, 3 required doubles) and `ServiceDisabledException implements Exception` (const, no-arg)
- `app/lib/routes/map_screen.dart` - Dropped the `flutter_map_location_marker show LocationMarkerPosition` import; resolves via the already-present unrestricted `foreground_position_stream_provider.dart` import
- `app/lib/provider/map_style_provider.dart` - DELETED (dead `mapStyleProvider` Riverpod provider; its only live export was relocated)
- `app/lib/provider/map_style_provider.g.dart` - DELETED (generated part file of the above)
- `app/tool/extract_map_styles.dart` - DELETED (one-off asset generator that imported the flomp `vector_tile_renderer` fork's internal theme source files; its committed output assets stay)
- `app/lib/util/map_coordinate_adapter.dart` - DELETED (only remaining importer of bare `package:flutter_map`; zero importers of itself anywhere in the tree)

## Decisions Made
- Relocated `effectiveBrightness()` verbatim per RESEARCH's Code Examples rather than restructuring it — no new import was needed since `Brightness`/`ThemeMode`/`WidgetsBinding` all resolve via the destination file's existing `package:flutter/material.dart` import
- Kept the local `LocationMarkerPosition`/`ServiceDisabledException` classes field-shape-identical to the originals so no call-site logic changes were required anywhere in `foreground_position_stream_provider.dart`, `wanderer_map.dart`, or `map_screen.dart`

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written for both tasks.

### Out-of-Scope Findings (logged, not fixed)

Three pre-existing `flutter test` failures were observed during Task 2's verification gate:
- `test/models/feed_item_test.dart`: `FeedItem.fromJson type "list" returns FeedItemList with ListSearchResult`
- `test/models/feed_item_test.dart`: `FeedItem.fromJson type "trail" returns FeedItemTrail with TrailSearchResult`
- `test/routes/settings_screen_test.dart`: `settings screen lists all 5 rows in D-06 order (SETNAV-01)`

Confirmed via `git stash` + re-run against the parent commit (`c06d9546`, before this plan's Task 2 changes) that all three reproduce identically with zero relation to maps, location, or this plan's files. Logged to `.planning/phases/18-retire-flutter-map-and-the-flomp-forks/deferred-items.md` per the executor's scope-boundary rule — not fixed here.

---

**Total deviations:** 0 auto-fixed; 3 pre-existing failures logged as out-of-scope (deferred-items.md)
**Impact on plan:** None — plan's own acceptance criteria (`flutter analyze` clean, zero remaining package imports, four files deleted) are all met. The 3 test failures do not touch this plan's `files_modified` list.

## Issues Encountered
None beyond the pre-existing test failures noted above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 02 (pubspec.yaml removal of the six packages + two `flomp/*` dependency_overrides entries + `maplibre` version pin) is now unblocked: zero source files import any of the six packages, so removing them from `pubspec.yaml` will not break `flutter analyze` or `flutter pub get`
- The 3 pre-existing test failures (feed_item_test.dart, settings_screen_test.dart) remain open in `deferred-items.md` — not a blocker for Plan 02 or 03, but should be picked up by a future quick task

---
*Phase: 18-retire-flutter-map-and-the-flomp-forks*
*Completed: 2026-07-10*
