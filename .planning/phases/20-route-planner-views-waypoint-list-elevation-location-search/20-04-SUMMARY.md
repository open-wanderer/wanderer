---
phase: 20-route-planner-views-waypoint-list-elevation-location-search
plan: 04
subsystem: mobile-route-planner
tags: [flutter, riverpod, route-planner, reorderable-list, dart]

# Dependency graph
requires:
  - phase: 20-route-planner-views-waypoint-list-elevation-location-search
    plan: 01
    provides: "RouteAnchors.deleteAnchor(String)/reorderAnchors(List<String>) mutators on routeAnchorsProvider"
provides:
  - "RouteAnchorListTab — ReorderableListView of route anchors with numbered badge, coordinate subtitle, immediate delete, and long-press-drag reorder (WAYP-04/05)"
affects: [route-planner-sheet-shell, phase-20-plan-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Optimistic reorder working-copy (_orderedIds/_reordering) mirrored from settings_categories_screen.dart, but with no revert-on-error branch since reorderAnchors is a synchronous in-memory mutation that cannot throw"

key-files:
  created:
    - app/lib/components/route_planner/route_anchor_list_tab.dart
  modified: []

key-decisions:
  - "No try/catch or snapshot-revert around reorderAnchors — it's a synchronous, in-memory-only mutation (no Dio call, no persistence), so fabricating an async failure branch would be dead code contradicting the plan's own guidance"
  - "ReorderableListView.builder's parameter is scrollController (not controller); the plan's action text referenced the (deprecated in newer Flutter releases) generic name — matched to the actual widget API"

requirements-completed: [WAYP-04, WAYP-05]

# Metrics
duration: 8min
completed: 2026-07-16
---

# Phase 20 Plan 04: Route Anchor List Tab Summary

**RouteAnchorListTab — a ReorderableListView.builder of route anchors with a numbered accent badge, coordinate subtitle, immediate no-confirmation delete, and long-press-drag reorder wired directly to the sheet's ScrollController.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-16T21:59:58Z
- **Completed:** 2026-07-16T22:02:30Z
- **Tasks:** 1 completed
- **Files modified:** 1 (created)

## Accomplishments
- `RouteAnchorListTab` renders every in-progress route anchor in order via `ReorderableListView.builder`, each row a single-line `ListTile` with a 24px accent-fill numbered badge, `'Anchor {n}'` title, `'{lat}, {lon}'` (5-decimal) grey subtitle, and a trailing `Colors.red.shade400` delete icon (WAYP-04)
- Delete calls `deleteAnchor(id)` immediately on tap — no confirmation dialog, no snackbar-undo (D-06); the button is never disabled, so deleting to zero anchors is valid (D-07)
- Long-press-drag reorder follows `settings_categories_screen.dart`'s exact optimistic working-copy pattern: local `_orderedIds` seeded from the provider whenever `!_reordering`, canonical `if (newIndex > oldIndex) newIndex -= 1` index-shift, calls `reorderAnchors(_orderedIds)` (WAYP-05)
- Accepts the sheet's `ScrollController` directly via constructor (`RouteAnchorListTab({required travelProfile, required scrollController})`) so Pattern 1 wires the sheet's builder-supplied controller to this tab specifically, ahead of Plan 05's sheet-shell consumption

## Task Commits

Each task was committed atomically:

1. **Task 1: Create RouteAnchorListTab — numbered rows, delete, reorder (WAYP-04/05, D-05..D-08)** - `b9e44c8d` (feat)

## Files Created/Modified
- `app/lib/components/route_planner/route_anchor_list_tab.dart` - New `ConsumerStatefulWidget`: `RouteAnchorListTab`, holding `_orderedIds`/`_reordering` state, rendering the reorderable/deletable anchor list

## Decisions Made
- Omitted a try/catch/revert-on-error branch around `reorderAnchors` — unlike the `settings_categories_screen.dart` analog (which persists to a server and can fail), `reorderAnchors` is purely synchronous in-memory state, so an error path would be unreachable dead code
- Used `ReorderableListView.builder`'s actual `scrollController` named parameter (not `controller`, which doesn't exist on that widget) to satisfy the plan's Pattern 1 wiring requirement

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `ReorderableListView.builder` has no `controller` parameter**
- **Found during:** Task 1 (initial write, caught by IDE diagnostics before commit)
- **Issue:** The plan's action text says `controller: widget.scrollController`, but the widget's actual named parameter for this purpose is `scrollController`.
- **Fix:** Changed to `scrollController: widget.scrollController`.
- **Files modified:** `app/lib/components/route_planner/route_anchor_list_tab.dart`
- **Verification:** `flutter analyze` clean.
- **Committed in:** `b9e44c8d` (Task 1 commit)

**2. [Rule 1 - Bug] `Colors.red.shade400` not const-evaluable; stray `.let()` extension typo**
- **Found during:** Task 1 (initial write, caught by IDE diagnostics before commit)
- **Issue:** `MaterialColor.shade400` is a runtime property lookup, not usable inside a `const FaIcon(...)` expression; an earlier draft also left a stray non-existent `.let((icon) => icon)` call on the `FaIcon`.
- **Fix:** Removed `const` from the `FaIcon` constructor call and the `.let(...)` no-op.
- **Files modified:** `app/lib/components/route_planner/route_anchor_list_tab.dart`
- **Verification:** `flutter analyze` clean.
- **Committed in:** `b9e44c8d` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 bugs caught by IDE diagnostics before the commit — no behavior change from the plan's intent)
**Impact on plan:** Both fixes necessary for the file to compile; no scope creep.

## Issues Encountered
`ReorderableListView`'s deprecated-`onReorder`-in-favor-of-`onReorderItem` lint is an information-severity diagnostic, not an error — kept `onReorder` with an `// ignore: deprecated_member_use` comment (matching the `settings_categories_screen.dart` analog verbatim) since the plan's acceptance criteria explicitly grep for the canonical `if (newIndex > oldIndex) newIndex -= 1` shift inside a manual `onReorder` handler.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `RouteAnchorListTab` is ready for Plan 05's sheet shell to mount as the first `TabBarView` child, passing it the `DraggableScrollableSheet.builder`'s `scrollController` per Pattern 1.
- No blockers identified for subsequent Phase 20 plans.

---
*Phase: 20-route-planner-views-waypoint-list-elevation-location-search*
*Completed: 2026-07-16*

## Self-Check: PASSED

Created file `app/lib/components/route_planner/route_anchor_list_tab.dart` verified present on disk; task commit `b9e44c8d` verified present in git log.
