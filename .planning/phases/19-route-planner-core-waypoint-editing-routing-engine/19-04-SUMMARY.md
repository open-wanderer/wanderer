---
phase: 19-route-planner-core-waypoint-editing-routing-engine
plan: 04
subsystem: mobile-app
tags: [flutter, riverpod, maplibre, route-planner, screen, dart]

# Dependency graph
requires:
  - phase: 19-02
    provides: "RouteAnchors @riverpod family notifier (appendAnchor/dragAnchor/insertAnchorOnSegment/retrySegment/toggleAutoRouting/undo/redo); segmentKey in route_segment_util.dart"
  - phase: 19-03
    provides: "RouteAnchorLayer (numbered draggable markers), RouteSegmentLayer (GeoJSON segment renderer with route-segments-hit hit-test layer)"
provides:
  - "RoutePlannerScreen — the screen that makes every Phase 19 requirement reachable by a user"
affects: [] # Phase 19's final plan; Phase 21 (HANDOFF-02/03) will register this screen in the router

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Screen-level map host reusing TrailMap's _pendingStyle race-buffer pattern verbatim (Pitfall 5)"
    - "ref.listen segment-list identity check (identical(prev?.segments, next.segments)) to skip redundant native GeoJSON pushes on anchors-only state emissions"
    - "Local notifier variable hoisted before the widget tree so onPressed ternaries stay on one line (dart format would otherwise wrap a long chained .read(...).notifier.undo() call across lines)"

key-files:
  created:
    - app/lib/routes/route_planner_screen.dart
  modified: []

key-decisions:
  - "static final _segmentLayer = RouteSegmentLayer() instead of static const — RouteSegmentLayer's constructor is non-const (19-03's own documented deviation: its mutable _added field forbids a const constructor). The plan's action text didn't specify const/non-const for this call site, but a plain const declaration fails to compile against the actual (correctly non-const) class."
  - "_retryAttempted field pre-declared in Task 1, not Task 2 as the plan's task split implied. Task 1's own onEvent action text uses _retryAttempted.add(...) in the blocked-segment tap branch, but the field's declaration only appears in Task 2's action text (\"Add two instance fields...\"). Declared it in Task 1 instead so Task 1 compiles standalone (flutter analyze passes) without waiting for Task 2; Task 2 only adds the second field (_blockedNotified), which is genuinely first-used there."
  - "Undo/redo onPressed hoists a local `notifier` variable (ref.read(...).notifier) before the Scaffold, then uses `notifier.undo`/`notifier.redo` directly as VoidCallback? tear-offs, instead of an inline `() => ref.read(...).undo()` closure. The inline form's full line exceeds 80 chars, and dart format wraps the ternary's `?`/`:` onto separate lines from `isNotEmpty` — which fails the plan's own literal grep acceptance criterion (`undoStack.isNotEmpty ? `). The hoisted-variable form keeps the ternary on one line and is also just cleaner."

requirements-completed: [WAYP-01, WAYP-02, WAYP-03, ROUTE-01, ROUTE-02, ROUTE-04, ROUTE-05]

# Metrics
duration: ~15min
completed: 2026-07-16
---

# Phase 19 Plan 04: Route Planner Screen — Map Host, Gesture Wiring, App Bar Summary

**`RoutePlannerScreen`: the screen that hosts the native map, disambiguates marker/segment/empty-map taps into the correct 19-02 mutation, exposes the auto-routing toggle, and puts undo/redo + blocked-segment/retry toast copy in the app bar — closing the goal-backward reachability chain for all of Phase 19's requirements**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-16T14:15:00Z (approx, first read)
- **Completed:** 2026-07-16T14:30:14Z
- **Tasks:** 2 completed
- **Files modified:** 1 (created)

## Accomplishments

- `RoutePlannerScreen` (`ConsumerStatefulWidget`, constructor `{required travelProfile, required initialCenter, title = 'Plan a Route'}`) hosts the native `MapLibreMap` with the standard `_pendingStyle` style-loaded race buffer, copied verbatim from `TrailMap`'s proven pattern (Pitfall 5).
- `onEvent`'s `MapEventClick` handler disambiguates every tap via `featuresAtPoint(event.screenPoint, layerIds: ['route-segments-hit'])`: a marker tap never reaches this handler (D-04, `RouteAnchorLayer`'s own `GestureDetector` consumes it first); a segment hit routes to `retrySegment` if `state == 'blocked'` (D-09) or `insertAnchorOnSegment` otherwise (WAYP-03); an empty-map tap always appends (D-03, WAYP-01).
- Top-right auto-routing toggle pill (D-06) matches `_buildMapControls`' existing styling exactly: `FontAwesomeIcons.route`, accent tint when ON, ~40%-opacity muted when OFF, state-reflecting tooltip. Lives in the same `Align`/`Column` slot Phase 20's future list/elevation buttons will join.
- A centered "Tap the map to start your route." hint (UI-SPEC's exact copy) shows until the first anchor is placed, then is dismissed permanently for the session via `_hintDismissed`.
- `ref.listen` keeps `RouteSegmentLayer`'s native GeoJSON in sync with every state mutation via `updateGeoJsonSource` (no remove/re-add flicker), skipping redundant pushes when only `anchors` changed (`identical(prev?.segments, next.segments)` guard).
- App bar (`Scaffold(extendBodyBehindAppBar: true, ...)`, matching `trail_create_screen.dart`'s shape): back button, then undo before redo (D-10), each `IconButton.onPressed` a plain `notifier.undo`/`notifier.redo` tear-off gated by `state.undoStack.isNotEmpty`/`state.redoStack.isNotEmpty` — `null` when empty, rendering Flutter's built-in disabled/~38%-opacity style automatically (D-11).
- Blocked-segment toast copy exactly matches UI-SPEC's Copywriting Contract: first-time failure shows the persistent error toast once (`_blockedNotified` dedup set, cleared when the segment leaves `blocked` so a *future* failure re-shows the first-time copy); a retry that fails again shows the distinct "Still couldn't find..." copy (`_retryAttempted` set, populated at the tap-to-retry call site, consumed/removed by the listener); every retry tap shows the transient "Retrying route…" info toast.
- No "waypoint"/"Waypoint" substring anywhere in the file (D-01, confirmed via grep — 0 occurrences).

## Task Commits

1. **Task 1: Map host, tap routing, auto-routing toggle, empty-state hint** — `372708fe` (feat)
2. **Task 2: App bar undo/redo + blocked-segment/retry toast copy** — `32b61456` (feat)

**Plan metadata:** commit skipped — `commit_docs: false` in `.planning/config.json` (see State Updates below)

## Files Created/Modified

- `app/lib/routes/route_planner_screen.dart` — `RoutePlannerScreen` (`ConsumerStatefulWidget`): native map host, gesture disambiguation, auto-routing toggle, app-bar undo/redo, blocked-segment/retry toast copy

## Decisions Made

- **`static final _segmentLayer = RouteSegmentLayer()`, not `static const`.** `RouteSegmentLayer`'s constructor is non-const per 19-03's own documented deviation (its mutable `_added` field forbids a const constructor — confirmed via `dart analyze`'s `const_constructor_with_non_final_field`). Declaring this call site with `const` (which the plan's action text left unspecified) fails to compile; `static final` is the correct, compilable form.
- **`_retryAttempted` declared in Task 1, not Task 2.** Task 1's own `onEvent` action text calls `_retryAttempted.add(...)` inside the blocked-segment tap branch, but the field's declaration only appears in Task 2's action text. Declaring it in Task 1 (rather than waiting for Task 2) keeps Task 1 independently compilable and `flutter analyze`-clean, matching this plan's own Task 1 acceptance criteria which run `flutter analyze` standalone. Task 2 adds only `_blockedNotified` (the field genuinely first-used there).
- **Undo/redo `onPressed` uses a hoisted `notifier` local + method tear-off (`notifier.undo`/`notifier.redo`), not an inline closure.** An inline `() => ref.read(routeAnchorsProvider(widget.travelProfile).notifier).undo()` exceeds 80 chars, and `dart format` wraps the ternary's `?`/`:` onto separate lines from `.isNotEmpty` — which fails the plan's own literal acceptance-criteria grep (`grep -n "undoStack.isNotEmpty ? \|redoStack.isNotEmpty ? "`, expecting the condition and `?` on the same line). The hoisted-variable/tear-off form keeps the ternary on one line, satisfies the grep, and is simpler code.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `_segmentLayer` field changed from `const` to `final`**
- **Found during:** Task 1, writing the class-field declaration exactly as scaffolded from the plan's `RouteSegmentLayer()` reference
- **Issue:** `RouteSegmentLayer`'s constructor is non-const (19-03's documented deviation); `static const _segmentLayer = RouteSegmentLayer();` fails to compile (`const_with_non_const`)
- **Fix:** Changed to `static final _segmentLayer = RouteSegmentLayer();`
- **Files modified:** `app/lib/routes/route_planner_screen.dart`
- **Commit:** `372708fe`

**2. [Rule 1 - Bug] `_retryAttempted` field moved from Task 2 to Task 1's declaration set**
- **Found during:** Task 1, implementing the `onEvent` handler exactly as the plan's action text specifies (which references `_retryAttempted.add(...)` before the field is ever declared, per Task 2's action text)
- **Issue:** Task 1's own action text uses a field that, per the plan's task split, is only declared in Task 2 — Task 1 alone would not compile
- **Fix:** Declared `final Set<String> _retryAttempted = {};` in Task 1 instead of waiting for Task 2; Task 2 only adds `_blockedNotified`
- **Files modified:** `app/lib/routes/route_planner_screen.dart`
- **Commit:** `372708fe`

**3. [Rule 1 - Bug] Undo/redo `onPressed` restructured to a hoisted-variable tear-off to satisfy the plan's own acceptance grep**
- **Found during:** Task 2, after `dart format` wrapped the initial inline-closure implementation's ternary across multiple lines, causing the plan's own `grep -n "undoStack.isNotEmpty ? \|redoStack.isNotEmpty ? "` acceptance criterion to return no matches
- **Issue:** The plan's literal action text (`onPressed: state.undoStack.isNotEmpty ? () => ref.read(...).undo() : null`) exceeds 80 characters once expanded with the real provider expression, and this codebase's `dart format` convention wraps long ternaries — breaking the plan's own single-line grep expectation
- **Fix:** Hoisted `final notifier = ref.read(routeAnchorsProvider(widget.travelProfile).notifier);` once, before the `Scaffold`, then used `notifier.undo`/`notifier.redo` directly as `VoidCallback?` tear-offs — short enough that `dart format` keeps the ternary on one line
- **Files modified:** `app/lib/routes/route_planner_screen.dart`
- **Commit:** `32b61456`

None of the above are architectural changes (Rule 4) — all are compile/format-driven corrections that keep the plan's own literal action text and acceptance criteria simultaneously satisfiable, matching the precedent already established in 19-02/19-03 for this same class of contradiction.

## Must-Haves Verification

Re-read `19-04-PLAN.md` fresh per this run's guidance (not from cached/summarized understanding). All 6 `must_haves.truths` entries verified satisfied, with no internal inconsistency found between them and the plan's Task 1/Task 2 prose (unlike 19-02's `toggleAutoRouting` conflict):

- Tap-on-empty-map adds an anchor connected to the previous one by a segment (WAYP-01) — `notifier.appendAnchor(event.point)` at the `onEvent` fallthrough; connecting-segment creation is 19-02's `appendAnchor` internals, unchanged here.
- Drag repositions and re-resolves connected segments (WAYP-02) — delegated to 19-03's `RouteAnchorLayer`, embedded unchanged as a `children` entry.
- Tap-on-segment inserts; tap-on-blocked-segment retries (WAYP-03/D-09) — source-asserted: `'blocked'` only appears in the branch calling `retrySegment`, never `insertAnchorOnSegment`.
- Auto-routing toggle re-resolves every segment (ROUTE-01/02) — toggle button calls `notifier.toggleAutoRouting()`, whose re-resolve-on-ON behavior is 19-02's existing, unchanged logic.
- Undo/redo app-bar icons, disabled when their stack is empty (ROUTE-04/D-11) — `state.undoStack.isNotEmpty ? notifier.undo : null` / same for redo.
- Blocked-segment toast copy: persistent first-time error, distinct second-consecutive-failure copy, transient "Retrying route…" (ROUTE-05, UI-SPEC) — all three exact strings present, gated by `_blockedNotified`/`_retryAttempted` bookkeeping verified via source, not just visual inspection.

## Issues Encountered

None beyond the three self-contained compile/format corrections noted above (all found and fixed while satisfying the plan's own `flutter analyze`/grep acceptance criteria, not scope changes).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `RoutePlannerScreen` is feature-complete for every Phase 19 requirement (WAYP-01/02/03, ROUTE-01/02/04/05) and ready for Phase 21's router registration (HANDOFF-02/03) — Phase 21 supplies the real `travelProfile`/`initialCenter` values and wires a route into the navigation stack.
- End-of-phase human verification (per `human_verify_mode=end-of-phase`) still needs to be run on a physical/emulated device: tap-to-add, drag-to-reposition, tap-to-insert, blocked-segment tap-to-retry, auto-routing toggle re-resolving all segments, undo/redo round-tripping, and toast copy. No widget-test precedent exists for a native-map-hosting screen in this codebase (`ml.MapLibreMap`'s native platform channel cannot be constructed in the widget-test harness), so this is a manual verification step, not an automated gap.
- No blockers identified for Phase 20 (Route Planner Views — waypoint list, elevation, location search) or Phase 21 (Handoff & Entry Point).

## Self-Check: PASSED

Verified `app/lib/routes/route_planner_screen.dart` exists on disk with both tasks' content; both commit hashes (`372708fe`, `32b61456`) verified present in `git log`.

---
*Phase: 19-route-planner-core-waypoint-editing-routing-engine*
*Completed: 2026-07-16*
