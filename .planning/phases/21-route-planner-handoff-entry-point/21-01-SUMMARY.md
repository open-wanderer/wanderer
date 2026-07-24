---
phase: 21-route-planner-handoff-entry-point
plan: 01
subsystem: mobile-route-planner
tags: [flutter, riverpod, gpx, dart, trail-handoff]

# Dependency graph
requires:
  - phase: 20-route-planner-views-waypoint-list-elevation-location-search
    provides: plannedGpxProvider (pre-elevation Gpx synthesis), ElevationTab's fetch/merge reference implementation
  - phase: 19-route-planner-core-waypoint-editing-routing-engine
    provides: route_anchor_provider.dart / RouteAnchors state, travelProfile fixed-for-session family design
provides:
  - "categoryForTravelProfile: reverse lookup from travel profile ('bicycle'/'pedestrian') to category id"
  - "route_planner_handoff_util.dart: finishPlanning orchestration, buildDraftTrail, mergeHeightsIntoGpx pure helpers"
affects: [21-02, 21-03, 21-04, route-planner-app-bar-finish-action]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reverse-lookup helper co-located with and documented as the inverse of its forward counterpart (costingForCategory / categoryForTravelProfile)"
    - "Pure orchestration function (finishPlanning) that composes existing tested mechanisms (pendingImportedTrail, ElevationTab fetch/merge, Trail.empty().copyWith) rather than building new infrastructure"

key-files:
  created:
    - app/lib/util/route_planner_handoff_util.dart
    - app/test/util/route_planner_handoff_util_test.dart
  modified:
    - app/lib/util/gpx_util.dart
    - app/test/util/gpx_util_test.dart

key-decisions:
  - "categoryForTravelProfile lives in gpx_util.dart next to costingForCategory (its inverse), matching RESEARCH's discoverability recommendation"
  - "finishPlanning has no pure/synchronous seam (network fetch + WidgetRef + BuildContext), so only its pure sub-helpers (buildDraftTrail, mergeHeightsIntoGpx) are unit-tested; the orchestration itself is exercised by Plan 04's app-bar Finish action wiring"

patterns-established:
  - "Draft Trail construction: Trail.empty().copyWith(category:, expand: TrailExpand(gpxData: <xml>, gpx: <parsed>, waypointsViaTrail: const []))"

requirements-completed: [HANDOFF-01]

# Metrics
duration: 15min
completed: 2026-07-17
---

# Phase 21 Plan 01: Route Planner Handoff Logic Summary

**`finishPlanning` orchestration util that turns the in-progress planner route into a draft Trail (GPX track only, no Waypoint records) with a silent one-time elevation merge and hike/bike category pre-fill, handed off via the existing `pendingImportedTrail` mechanism.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-07-17T14:44:35Z
- **Completed:** 2026-07-17T14:49:22Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- `categoryForTravelProfile` added to `gpx_util.dart` as the tested inverse of `costingForCategory` (D-08), returning a matching category id or `null` on no match
- `route_planner_handoff_util.dart` created with `finishPlanning` (orchestration entry point for Plan 04's app-bar Finish action), plus pure helpers `buildDraftTrail` and `mergeHeightsIntoGpx`
- `finishPlanning` composes four already-tested codebase mechanisms unchanged: `pendingImportedTrail` handoff, `ElevationTab`'s `/valhalla/height` fetch/merge shape, `Trail.empty().copyWith(...)`, and `plannedGpxProvider`'s synthesized route
- D-07 (route anchors never become Waypoint records) enforced structurally: `waypointsViaTrail: const []`, zero `Waypoint(` constructor calls anywhere in the new file
- D-06 (silent best-effort elevation) enforced structurally: the `/valhalla/height` fetch is wrapped in try/catch with an empty catch body — zero `toastProvider` references

## Task Commits

Each task was committed atomically:

1. **Task 1: Add categoryForTravelProfile reverse-lookup to gpx_util.dart (D-08)** - `d0b02295` (feat)
2. **Task 2: Create route_planner_handoff_util.dart — finishPlanning + pure builders + tests (HANDOFF-01, D-06/D-07/D-08)** - `8ef36021` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/util/gpx_util.dart` - Added `categoryForTravelProfile(travelProfile, categories) → String?`, the inverse of `costingForCategory`
- `app/test/util/gpx_util_test.dart` - Added `group('categoryForTravelProfile', ...)` with 5 cases (bike match, hike match, no match, empty list, forward/reverse symmetry)
- `app/lib/util/route_planner_handoff_util.dart` - New: `mergeHeightsIntoGpx`, `buildDraftTrail`, `finishPlanning`
- `app/test/util/route_planner_handoff_util_test.dart` - New: unit tests for the two pure helpers (8 cases)

## Decisions Made
- `categoryForTravelProfile`'s bike-branch check order (`'bike'` → `'cycling'` → `'bicycle'`) mirrors `costingForCategory`'s exact token order for symmetry, per RESEARCH Pitfall 5
- `finishPlanning`'s orchestration (network I/O + `WidgetRef`/`BuildContext`) is intentionally left without a direct unit test — no pure/synchronous seam exists without a widget-test harness; its pure sub-helpers (`buildDraftTrail`, `mergeHeightsIntoGpx`) carry the test coverage instead, and the full flow will be exercised end-to-end once Plan 04 wires the app-bar Finish action

## Deviations from Plan

None - plan executed exactly as written. One test-fixture correction made during Task 1 verification (not a plan deviation, a self-caught test bug): the initial bike-category test fixture used the string "Mountain Biking", which does not contain the literal substring `'bike'` (`"biking"` ≠ `"...bike..."`), causing the test to fail against the correctly-implemented function. Fixed the fixture to "Mountain Bike Trails" before committing; the implementation was correct throughout.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `finishPlanning` is ready for Plan 04 to wire as the app-bar Finish action's `onPressed` handler (`finishPlanning(ref: ref, navContext: context, travelProfile: widget.travelProfile)`)
- `categoryForTravelProfile` is ready for any future call site needing the hike/bike → category reverse mapping
- No blockers for Plans 02-04

---
*Phase: 21-route-planner-handoff-entry-point*
*Completed: 2026-07-17*

## Self-Check: PASSED

- FOUND: app/lib/util/route_planner_handoff_util.dart
- FOUND: app/test/util/route_planner_handoff_util_test.dart
- FOUND: .planning/phases/21-route-planner-handoff-entry-point/21-01-SUMMARY.md
- FOUND commit: d0b02295
- FOUND commit: 8ef36021
