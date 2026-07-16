---
phase: 20-route-planner-views-waypoint-list-elevation-location-search
plan: 03
subsystem: mobile-route-planner
tags: [flutter, riverpod, gpx, fl_chart, route-planner, dart]

# Dependency graph
requires:
  - phase: 20-route-planner-views-waypoint-list-elevation-location-search
    plan: 01
    provides: "plannedGpxProvider (pre-elevation Gpx derived from routeAnchorsProvider), buildGpxFromPoints, buildNavShape"
provides:
  - "ElevationProfile.trail: Trail? — reusable for a trail-less planned route, gpx-derived stats fallback via GpxStats.totalElevationGain/totalElevationloss"
  - "ElevationTab — tab-gated (TabController index==1, !indexIsChanging), 500ms-debounced /valhalla/height fetch with a local ele-merged Gpx, empty state under 2 anchors"
affects: [route-planner-tabbed-sheet, phase-21-handoff]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tab-visibility gating: didChangeDependencies captures DefaultTabController.maybeOf(context), adds a listener, tracks visible = index==1 && !indexIsChanging with an edge-triggered (visible && !_wasVisible) fetch schedule — mirrors trail_panel.dart's existing _TabContentState idiom"
    - "Ele-merge-by-index: build the elevation-merged Gpx from the exact buildNavShape() downsampled shape sent to /valhalla/height (not the full original point list), so response array index alignment always holds even when >500 points triggers downsampling"

key-files:
  created:
    - app/lib/components/route_planner/elevation_tab.dart
  modified:
    - app/lib/components/trail/elevation_profile.dart

key-decisions:
  - "ElevationTab never writes ele back into plannedGpxProvider (D-10) — holds its own local _eleMergedGpx state, rebuilt on each successful height fetch, passed to ElevationProfile(trail: null, gpx: ...)"
  - "Height fetch failures are silently swallowed (best-effort) — the tab keeps rendering the last successfully-merged Gpx (or the pre-elevation one) rather than surfacing an error state, matching D-11's framing of this as a live/best-effort profile"

requirements-completed: [PLANUI-02]

# Metrics
duration: 20min
completed: 2026-07-16
---

# Phase 20 Plan 03: Elevation Tab (Trail-less ElevationProfile + Tab-Gated Height Fetch) Summary

**ElevationProfile now accepts a null Trail with gpx.getTotals()-derived stats and no anchor icons; new ElevationTab fetches /valhalla/height only while visible, debounced 500ms, with index-aligned ele-merging and a <2-anchor empty state.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-16T21:50:16Z
- **Completed:** 2026-07-16T21:57:24Z
- **Tasks:** 2 completed
- **Files modified:** 2 (1 created + 1 modified)

## Accomplishments
- `ElevationProfile.trail` changed from `Trail` (required) to `Trail?` (optional) — the non-scrub stats header branches to `widget.gpx.getTotals()` (`GpxStats.totalElevationGain`/`totalElevationloss`, matching the existing lowercase-`l` typo) when `trail` is null, and the `_buildChart` waypoint overlay reads `widget.trail?.expand?.waypointsViaTrail ?? []` so no route-anchor icons render on a trail-less chart (D-12)
- New `ElevationTab` (`ConsumerStatefulWidget`) watches `plannedGpxProvider(travelProfile)`, gates the `/valhalla/height` POST on `TabController` visibility (`index == 1 && !indexIsChanging`, assuming Elevation is the sheet's second tab) rather than widget build/mount — fetches are edge-triggered on the visible-false→true transition and re-scheduled (debounced 500ms) on further route edits while visible (D-11, Pitfall 2)
- Height responses are merged into a **local** ele-merged `Gpx` copy built from the exact `buildNavShape()`-downsampled shape that was sent (preserving index alignment even past the 500-point downsample threshold), never written back into `plannedGpxProvider` (D-10)
- Under 2 anchors, `ElevationTab` renders a local `_ElevationEmptyState` (200px, `colorScheme.secondaryContainer`, 16px radius, `Icons.terrain` 48px + the D-13 copy "Add at least 2 anchors to see the elevation profile.") and skips the height fetch entirely

## Task Commits

Each task was committed atomically:

1. **Task 1: Make ElevationProfile.trail optional (Trail?) with a gpx-derived stats fallback (D-12)** - `9328d64b` (feat)
2. **Task 2: Create ElevationTab — tab-gated debounced height fetch + empty state (PLANUI-02, D-10/D-11/D-13)** - `7bc32b8a` (feat)

## Files Created/Modified
- `app/lib/components/trail/elevation_profile.dart` - `trail` field/constructor param made nullable; stats header and chart waypoint overlay guarded for a null trail
- `app/lib/components/route_planner/elevation_tab.dart` - New `ElevationTab` widget: tab-visibility-gated debounced height fetch, index-aligned ele-merge, empty state

## Decisions Made
- `ElevationTab` assumes it is mounted as tab index 1 of a 2-tab `DefaultTabController` (Route Anchors = 0, Elevation = 1) per the plan's explicit spec — this couples it to the sibling sheet plan's (20-04) tab ordering, which matches the UI-SPEC's "Tab labels: Route Anchors / Elevation" order
- Reused `trail_panel.dart`'s existing `DefaultTabController.of`/`didChangeDependencies` idiom (via `maybeOf` for null-safety) rather than introducing a new tab-tracking pattern, since it's already proven working in this codebase for the same "derive visibility from index + indexIsChanging" problem
- Height-fetch failures are swallowed silently (best-effort), leaving the last successfully-merged `Gpx` in place — no error state/retry UI was specified for this tab in D-11/D-13, and this keeps the tab from flashing an error banner on every transient network blip during route planning

## Deviations from Plan

None - plan executed exactly as written. Both tasks' acceptance-criteria greps (`Trail? trail`, no unguarded `widget.trail.` access, `getTotals()`, `class ElevationTab`, `plannedGpxProvider`, `valhalla/height`, `buildNavShape`, `indexIsChanging`, `Timer(const Duration(milliseconds: 500)`, `trail: null`) all pass, and `flutter analyze` on both files reports no issues.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `ElevationTab` is ready to be dropped into Plan 04's `TabBarView` (or the tabbed sheet host) as the second tab, alongside the Route Anchors list tab
- `ElevationProfile`'s nullable-`trail` change is backward compatible — all existing callers (`trail_panel.dart`, `navigation_screen.dart`, `trail_create_screen.dart`, `trail_detail_map_screen.dart`) still pass a non-null `trail` and are unaffected
- No blockers identified for Plan 04 or the rest of Phase 20

---
*Phase: 20-route-planner-views-waypoint-list-elevation-location-search*
*Completed: 2026-07-16*

## Self-Check: PASSED

Both created/modified files verified present on disk; both task commits (`9328d64b`, `7bc32b8a`) verified present in git log.
