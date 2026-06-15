---
phase: quick-260615-k0w
plan: 01
subsystem: app/navigation
tags: [flutter, riverpod, navigation, valhalla, geospatial, tdd]
requires:
  - app/lib/models/navigate_response.dart (NavigateResponse, NavigateManeuver, shapeAsLatLng)
provides:
  - "Along-track projection maneuver advancement in Navigation notifier"
  - "Cumulative along-track distance tables precomputed per session"
  - "Multi-skip / midpoint-no-advance / forward-only test coverage"
affects:
  - app/lib/provider/navigation_provider.dart
tech-stack:
  added: []
  patterns:
    - "Equirectangular nearest-segment projection (cross-track + along-track) using dart:math"
    - "Precompute-once cumulative distance table in Riverpod notifier build()"
key-files:
  created: []
  modified:
    - app/lib/provider/navigation_provider.dart
    - app/test/provider/navigation_provider_test.dart
decisions:
  - "Replaced single-next-waypoint proximity check with along-track projection (research Option B) to resolve shortcut Failure mode A and switchback Failure mode B"
  - "Reframed the 30 m threshold as an along-track lookahead buffer rather than point proximity, preserving the constant"
  - "Corrected the _farFromManeuver1 test fixture: the old (47.010, 9.000) was collinear with and 445 m beyond the route end, which legitimately advances under projection; moved it off-route at the start so the no-advance intent holds"
metrics:
  duration_min: 8
  tasks: 2
  files: 2
  completed: "2026-06-15"
---

# Phase quick-260615-k0w Plan 01: Along-Track Projection for Waypoint Skip Summary

Maneuver advancement now projects the GPS fix onto the route polyline and jumps `currentManeuverIndex` past every maneuver a hiker shortcut past in a single fix, replacing the brittle single-next-waypoint 30 m proximity check.

## What Was Built

### Task 1 — Cumulative distance table + nearest-segment projection (commit c3982821)
- Added `_shapeCumulativeMeters` and `_maneuverCumulativeMeters` `late final` tables, populated once in `build()` from `shapeAsLatLng` using `Distance().as(LengthUnit.Meter, ...)`. Empty-shape case yields empty tables. Maneuver begin-shape indices are clamped to `0..shape.length-1` exactly like the prior code.
- Added `_projectAlongTrack(LatLng pos, List<LatLng> shape, int fromShapeIndex)`: forward-only nearest-segment projection. For each segment from `fromShapeIndex` onward it computes the scalar projection parameter `t` (clamped to `[0,1]`) in a local equirectangular frame (lat/lon deltas to meters via `111320` and `cos(midLat)`), measures cross-track distance to the projected foot, and returns `_shapeCumulativeMeters[bestSegmentStart] + t * geodesicSegmentLength` for the minimum-cross-track segment.
- Added `import 'dart:math' as math;`. No pub dependency added.

### Task 2 — onPosition rewrite + extended tests (commits d528118c RED, dc007157 GREEN)
- Rewrote `Navigation.onPosition` to: append breadcrumb (unchanged), guard empty shape/tables, start the projection scan at the current maneuver's begin-shape index (forward-only), project to an along-track distance, then advance to the highest maneuver `i` where `_maneuverCumulativeMeters[i] <= atd + 30.0`, breaking on the first not-yet-reached maneuver. Assignment is forward-only (`newIndex > current`) and capped at the last maneuver.
- Preserved `_kManeuverAdvanceThresholdMeters = 30.0`, `NavigationState`, `build` return shape, family key, and breadcrumb logic.
- Added three tests: shortcut skip (near maneuver 2 from index 0 to index 2), midpoint-no-advance (~178 m along-track stays at 0), exact-vertex projection (exactly on shape[6] to index 2).

## Verification

- `dart analyze lib/provider/navigation_provider.dart` — No issues found.
- `dart analyze` on provider + test together — No issues found.
- `flutter test test/provider/navigation_provider_test.dart` — All 9 tests pass (6 original + 3 new).
- pubspec.yaml / pubspec.lock unchanged (no new dependency).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected ambiguous `_farFromManeuver1` test fixture**
- **Found during:** Task 2 GREEN (existing test "far from maneuver 1 target: index stays at 0" failed).
- **Issue:** The fixture `LatLng(47.010, 9.000)` is collinear with the route (lon 9.000) and lies ~445 m *beyond* the route end. Under along-track projection it projects onto the route end (~667 m along-track), which correctly advances to the last maneuver — contradicting the test's "stays at 0" assertion. The point represented "far from a maneuver *point*" under the old algorithm, not "has not progressed along the route."
- **Fix:** Moved the fixture to `LatLng(47.000, 9.010)` — same start latitude, ~760 m east (large cross-track), projecting onto `shape[0]` (~0 m along-track). The no-advance intent now holds under projection. Documented the rationale in the fixture comment.
- **Files modified:** app/test/provider/navigation_provider_test.dart
- **Commit:** dc007157

## TDD Gate Compliance

- RED gate present: `test(quick-260615-k0w): add failing tests...` (d528118c) — confirmed failing (skip and vertex tests returned index 0).
- GREEN gate present: `feat(quick-260615-k0w): advance maneuvers via along-track projection` (dc007157) — all tests green.
- No REFACTOR commit needed.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: app/lib/provider/navigation_provider.dart (modified, contains `_projectAlongTrack`, `_maneuverCumulativeMeters`, `_shapeCumulativeMeters`)
- FOUND: app/test/provider/navigation_provider_test.dart (contains "skip" tests)
- FOUND commit c3982821
- FOUND commit d528118c
- FOUND commit dc007157
