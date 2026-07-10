---
phase: quick-260710-lem
plan: 01
subsystem: ui
tags: [flutter, maplibre, wanderer_map, list_detail_map_screen, race-condition, compass]

# Dependency graph
requires:
  - phase: quick-260710-kpd
    provides: The 6 UI-gap fixes this quick task's manual verification uncovered 2 remaining issues in (compass placement misapplied; camera-fit root cause misdiagnosed)
provides:
  - "WandererMap's onMapCreated/onStyleLoaded now share the same _pendingStyle race buffer already used in search_map.dart and navigation_screen.dart"
  - "_fitInitialCamera reads the record's populated trail.bounds (min/max_lat/lon) again, with the gpx.getBounds() detour removed"
  - "list_detail_map_screen's MapCompass is SafeArea-wrapped and vertically aligned with the AppBar back button"
affects: [wanderer_map, list_detail_map_screen, trail_detail_map_screen, trail_detail_screen]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Third call site (after search_map.dart, navigation_screen.dart) for the _pendingStyle onMapCreated/onStyleLoaded race buffer pattern for maplibre 0.3.5's unreliable callback ordering"

key-files:
  created: []
  modified:
    - app/lib/components/base/wanderer_map.dart
    - app/lib/routes/list_detail_map_screen.dart

key-decisions:
  - "Confirmed (per user correction) that trail.bounds (min/max_lat/lon) IS populated on GET /trail/:id; the prior kpd fix's switch to gpx.getBounds() was treating a symptom, not the cause — reverted to trail.bounds"
  - "list_detail_map_screen's compass uses an explicit padding override (top: 6, right: 8) inside SafeArea rather than a Positioned/Column wrapper (as navigation_screen.dart does), since this screen only needs one control, not a stacked column of controls"

requirements-completed: [LEM-01, LEM-02]

duration: 6min
completed: 2026-07-10
---

# Quick Task 260710-lem: Fix 2 Issues Found During Manual Verification Summary

**Added the onMapCreated/onStyleLoaded race buffer to WandererMap (the last unpatched map host) and moved the list-map compass out of the status-bar zone into SafeArea, aligned with the back button**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-10T13:32:00Z
- **Completed:** 2026-07-10T13:33:52Z
- **Tasks:** 2 completed
- **Files modified:** 2

## Accomplishments
- `WandererMap` (the single-trail map host used by `trail_detail_map_screen` and `trail_detail_screen`) now buffers a style-loaded event that arrives before `onMapCreated`, replaying it once the controller is set — the same fix already applied to `search_map.dart` (Phase 16-03) and `navigation_screen.dart` (Phase 17-01). This was the actual root cause of the "camera stuck on the trailhead" bug; the kpd quick task's switch to `gpx.getBounds()` never addressed it.
- `_fitInitialCamera` reads `widget.trail.bounds` directly again; the `gpx.getBounds()` fallback and the now-unused `gpx_util.dart` import are removed.
- `list_detail_map_screen`'s `ml.MapCompass` is wrapped in a `SafeArea` with a small top/right padding override, so it renders at the AppBar back button's height instead of jammed against the status bar.

## Task Commits

Each task was committed atomically:

1. **Task 1 (LEM-02): Fix trail-detail initial camera fit — race buffer + bounds-source revert** - `d155bba4` (fix)
2. **Task 2 (LEM-01): Reposition the list-map compass to align with the AppBar back button** - `e2ea2f0f` (fix)

_Note: no plan-metadata commit is included here — per this quick task's execution constraints, docs artifacts (SUMMARY.md, STATE.md) are committed separately by the orchestrator._

## Files Created/Modified
- `app/lib/components/base/wanderer_map.dart` - Added `_pendingStyle` buffer field, extracted `_onStyleLoaded(style)` from the inline `onStyleLoaded` callback, rewired `onMapCreated`/`onStyleLoaded` to buffer/replay, reverted `_fitInitialCamera` to read `widget.trail.bounds`, removed the unused `gpx_util.dart` import
- `app/lib/routes/list_detail_map_screen.dart` - Wrapped the `ml.MapCompass` child in a `const SafeArea(...)` with `padding: EdgeInsets.only(top: 6, right: 8)`

## Decisions Made
- Verified with the user that `trail.bounds` (`min/max_lat/lon`) is populated on every trail record including `GET /trail/:id` — the kpd-era comment claiming otherwise was incorrect and has been replaced.
- Kept the compass fix minimal (padding override inside `SafeArea`) rather than restructuring into a `Positioned`+`Column` control stack like `navigation_screen.dart`, since this screen has only the one floating control.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Both fixes are visual/behavioral and still need on-device confirmation per the plan's verification section: (1) `trail_detail_map_screen` fits the whole trail on first open, repeatably across several opens/trails; (2) `list_detail_map_screen`'s compass appears at back-button height when the map is rotated off-north.
- `flutter analyze` reports zero new errors (36 pre-existing info/warning issues unchanged, carried since Phase 18).

---
*Phase: quick-260710-lem*
*Completed: 2026-07-10*

## Self-Check: PASSED
