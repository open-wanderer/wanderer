---
phase: 20-route-planner-views-waypoint-list-elevation-location-search
plan: 02
subsystem: ui
tags: [flutter, go_router, riverpod, search, maplibre]

# Dependency graph
requires:
  - phase: 19-route-planner-core-waypoint-editing-routing-engine
    provides: RoutePlannerScreen shell that will push this screen (integration wired in a later plan of this phase)
provides:
  - LocationSearchScreen — locations-only search shell reusing globalSearchProvider
  - /location-search go_router route, pushable with a LocationSearchResult return value
affects: [20-05 (RoutePlannerScreen integration), 21 (handoff)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Locations-only search screen filters the shared globalSearchProvider via setCategory(...locations) in initState (post-frame) and resets to .all on dispose, avoiding a second provider or duplicated debounce logic"
    - "Result-returning push: context.pop(result) from a pushed screen, resolved by the caller's context.push<T>(...) Future — no context.go side-navigation"

key-files:
  created:
    - app/lib/routes/location_search_screen.dart
  modified:
    - app/lib/provider/router_provider.dart

key-decisions:
  - "Captured the GlobalSearchNotifier as a field in initState rather than calling ref.read(...) inside dispose() — avoids the avoid_ref_inside_state_dispose lint (ref may already be torn down by the time State.dispose runs)"

patterns-established:
  - "Pattern: dedicated single-category search screens reuse the existing debounced multi-category provider by toggling its category filter around the screen's lifecycle, rather than adding a new provider or query path"

requirements-completed: [PLANUI-03]

# Metrics
duration: 8min
completed: 2026-07-16
---

# Phase 20 Plan 02: Location Search Screen Summary

**LocationSearchScreen — a locations-only mirror of GlobalSearchScreen that reuses the existing debounced globalSearchProvider and pops its result back to the caller via `/location-search`, a new pushable go_router route.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-16T21:38:00Z
- **Completed:** 2026-07-16T21:46:36Z
- **Tasks:** 2 completed
- **Files modified:** 2

## Accomplishments
- Built `LocationSearchScreen`, a `ConsumerStatefulWidget` mirroring `GlobalSearchScreen`'s app-bar `TextField` shell, empty-query state, and no-results SVG state, but with the category-chip row and trail/list/actor result branches removed entirely.
- Filtered results to locations only by toggling `globalSearchProvider`'s existing `setCategory(GlobalSearchCategory.locations)` on entry and restoring `.all` on exit — no new provider, no un-debounced fetch path (preserves the 500ms `Timer` debounce per T-20-02-01).
- Result rows reuse `_LocationTile`'s exact visual structure but call `context.pop(location)` instead of `context.go('/map', ...)`, handing the selected `LocationSearchResult` back to whichever screen pushed this one.
- Registered `/location-search` as a sibling `GoRoute` of the existing `/search` route in `router_provider.dart`, resolving to `LocationSearchScreen`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Create LocationSearchScreen (locations-only, pops result)** - `77fcb20f` (feat)
2. **Task 2: Register the /location-search go_router route** - `ae9dd224` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/routes/location_search_screen.dart` - New `ConsumerStatefulWidget` — locations-only search shell, pops `LocationSearchResult` on tap
- `app/lib/provider/router_provider.dart` - Added `/location-search` `GoRoute` + import, sibling to the existing `/search` route

## Decisions Made
- Captured `ref.read(globalSearchProvider.notifier)` as a `late final` field in `initState` rather than reading it inside `dispose()`, since the IDE flagged `avoid_ref_inside_state_dispose` (ref may already be disposed by the time `State.dispose` runs). This is a code-quality fix within Task 1's own scope, not a plan deviation — the plan's action text described *what* dispose should do (reset category/query), not *how* to safely access the notifier reference.

## Deviations from Plan

None - plan executed exactly as written. The `ref`-in-`dispose` lint fix above is a mechanical safety adjustment to the same task's code, not a scope change.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `LocationSearchScreen` and `/location-search` are ready for Plan 05 (or whichever remaining plan in this phase wires the map control button) to push and await a `LocationSearchResult`, then call `_mapController.animateCamera(...)` at zoom 13 per D-15.
- On-device verification of search behavior (debounce, result rendering, camera hand-back) is deferred to end-of-phase human verification alongside the integration plan, per this plan's own `<verification>` section.
- No blockers.

---
*Phase: 20-route-planner-views-waypoint-list-elevation-location-search*
*Completed: 2026-07-16*

## Self-Check: PASSED

- FOUND: app/lib/routes/location_search_screen.dart
- FOUND: app/lib/provider/router_provider.dart
- FOUND commit: 77fcb20f
- FOUND commit: ae9dd224
