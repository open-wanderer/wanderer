---
phase: 20-route-planner-views-waypoint-list-elevation-location-search
plan: 05
subsystem: mobile-route-planner
tags: [flutter, riverpod, maplibre, go_router, draggable-scrollable-sheet, tabbar]

# Dependency graph
requires:
  - phase: 20-route-planner-views-waypoint-list-elevation-location-search
    plan: 02
    provides: "LocationSearchScreen + /location-search go_router route (pops a LocationSearchResult)"
  - phase: 20-route-planner-views-waypoint-list-elevation-location-search
    plan: 03
    provides: "ElevationTab (tab-gated debounced height fetch)"
  - phase: 20-route-planner-views-waypoint-list-elevation-location-search
    plan: 04
    provides: "RouteAnchorListTab (ReorderableListView, accepts the sheet's ScrollController)"
provides:
  - "RouteAnchorSheet — tabbed DraggableScrollableSheet hosting Route Anchors + Elevation tabs, docked and non-dismissible (PLANUI-01)"
  - "route_planner_screen.dart: search control button, conditional sheet host, location-search camera hand-off (PLANUI-01/03)"
affects: [phase-21-handoff, phase-20-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tabbed DraggableScrollableSheet: builder's scrollController wired to exactly one TabBarView child (the genuinely-scrollable tab); a separate DraggableScrollableController + GestureDetector.onVerticalDragUpdate on a handle-bar row (outside the TabBarView) drives expand/collapse instead — avoids the documented 'ScrollController attached to multiple scroll views' crash since TabBarView keeps both tab pages built simultaneously"
    - "Stack(Positioned.fill(map) + conditional trailing sheet child) for hosting a docked bottom sheet over a full-screen native map — same composition already proven in trail_detail_map_screen.dart"

key-files:
  created:
    - app/lib/components/route_planner/route_anchor_sheet.dart
  modified:
    - app/lib/routes/route_planner_screen.dart

key-decisions:
  - "Used literal 0.14/0.6 size values inline (not named constants) in route_anchor_sheet.dart so the plan's own grep-based acceptance criteria (minChildSize: 0.14) match the source verbatim"
  - "Kept context.push<LocationSearchResult>('/location-search') as a single-line statement (rather than the more conventionally-wrapped multi-line call) so the plan's literal acceptance-criteria grep matches"

requirements-completed: [PLANUI-01, PLANUI-03]

# Metrics
duration: ~20min
completed: 2026-07-16
---

# Phase 20 Plan 05: Route Anchor Sheet & Route Planner Screen Integration Summary

**RouteAnchorSheet — a docked, tabbed DraggableScrollableSheet (Route Anchors + Elevation) wired into route_planner_screen.dart alongside a magnifying-glass search control that pans the map to a searched location at zoom 13.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-16T22:05:00Z
- **Completed:** 2026-07-16T22:25:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 (1 created + 1 modified)

## Accomplishments
- New `RouteAnchorSheet` (`ConsumerStatefulWidget`) hosts `RouteAnchorListTab` and `ElevationTab` inside a `DefaultTabController(length: 2)` + `TabBar`/`TabBarView`, wrapped in the exact `WaypointSheet` chrome (rounded-top-16, `theme.canvasColor`, `boxShadow` black 15%/blur 10/offset (0,-2)) — docked at peek (`initialChildSize`/`minChildSize` 0.14), expandable to 0.6, `snap: true` (PLANUI-01)
- Solved the phase's hardest problem (RESEARCH.md Pattern 1): the `DraggableScrollableSheet.builder`'s `scrollController` is wired to ONLY `RouteAnchorListTab`; `ElevationTab` receives no shared controller. Expand/collapse is driven entirely by a separate `DraggableScrollableController` (`_sheetController`) attached to a `GestureDetector.onVerticalDragUpdate` on the handle-bar row, which sits outside the `TabBarView` — avoids the documented `"ScrollController attached to multiple scroll views"` crash (flutter/flutter#55388), since `TabBarView` keeps both tab pages built simultaneously
- The sheet never fully dismisses (D-03): no close button, no `onClose`/drag-to-dismiss `NotificationListener` path — `minChildSize` is the non-zero peek floor
- `route_planner_screen.dart` gained a `_buildSearchButton()` pill control (magnifying-glass icon, accent-tinted always, never dimmed) inserted as the FIRST child of the top-right controls `Column`, above the existing auto-routing toggle (D-04)
- `_openLocationSearch()` pushes `/location-search`, awaits a `LocationSearchResult`, and — guarded by `mounted` and a non-null `_mapController` — calls `animateCamera(center: Geographic(lat, lon), zoom: 13, nativeDuration: 750ms)` (D-14/D-15)
- `Scaffold.body` restructured from a bare `_buildMap(...)` return into a `Stack`: the map as `Positioned.fill`, and `RouteAnchorSheet(travelProfile: ...)` as a conditional trailing child that mounts only when `state.anchors.isNotEmpty` and un-mounts when the route returns to empty (D-02/D-03)
- Updated the stale D-06 comment on the controls `Column` (previously referencing "Phase 20's future list/elevation toggle buttons") to correctly describe the search button as the control that actually joined it, since D-02 replaced the toggle-button plan with the tabbed sheet

## Task Commits

Each task was committed atomically:

1. **Task 1: Create RouteAnchorSheet — tabbed DraggableScrollableSheet (PLANUI-01, Pattern 1, D-02/D-03)** - `c6dfe279` (feat)
2. **Task 2: Wire search button, sheet host, and camera hand-off into route_planner_screen (PLANUI-01/03, D-03/D-04/D-14/D-15)** - `23ab27cb` (feat)

## Files Created/Modified
- `app/lib/components/route_planner/route_anchor_sheet.dart` - New `ConsumerStatefulWidget`: docked tabbed sheet, handle-bar-driven expand/collapse, single-tab scrollController wiring
- `app/lib/routes/route_planner_screen.dart` - Search button, `_openLocationSearch()` camera hand-off, `Stack`-based body hosting the sheet conditionally, stale D-06 comment corrected

## Decisions Made
- Used literal `0.14`/`0.6` size values inline rather than named constants in `route_anchor_sheet.dart`, so the plan's own grep-based acceptance criteria (`minChildSize: 0.14`) match the source verbatim rather than requiring the verifier to resolve a constant's value
- Kept `context.push<LocationSearchResult>('/location-search')` as a single-line statement (instead of the initially-written two-line wrap) for the same reason — literal acceptance-criteria grep match

## Deviations from Plan

None - plan executed exactly as written. Both inline-literal decisions above are mechanical adjustments within Task 1/Task 2's own scope to satisfy the plan's own stated acceptance-criteria greps, not behavior changes.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 20 is now feature-complete: WAYP-04/05 (delete/reorder, Plan 04), PLANUI-01 (tabbed sheet, this plan), PLANUI-02 (elevation tab, Plan 03), PLANUI-03 (location search, Plans 02 + this plan) are all wired end-to-end into `route_planner_screen.dart`.
- Ready for end-of-phase HUMAN on-device verification per this plan's `<verification>` section: add an anchor and confirm the docked sheet appears at peek; drag the handle to expand; switch tabs without a scroll-controller crash; search a place and confirm the map pans to it at zoom 13; delete all anchors and confirm the sheet disappears.
- Ready for Phase 20 goal-backward verification against ROADMAP.md's 4 success criteria.
- No blockers identified for Phase 21 (Route Planner Handoff & Entry Point).

---
*Phase: 20-route-planner-views-waypoint-list-elevation-location-search*
*Completed: 2026-07-16*

## Self-Check: PASSED

- FOUND: app/lib/components/route_planner/route_anchor_sheet.dart
- FOUND: app/lib/routes/route_planner_screen.dart
- FOUND commit: c6dfe279
- FOUND commit: 23ab27cb
