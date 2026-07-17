---
phase: 19-route-planner-core-waypoint-editing-routing-engine
plan: 02
subsystem: mobile-app
tags: [flutter, riverpod, dio, valhalla, route-planner, race-guard, dart]

# Dependency graph
requires:
  - phase: 19-01
    provides: "PolylineUtil.decode(shape, precision: 6), RouteAnchor/RouteSegment/SegmentState/RouteAnchorsSnapshot models (D-01)"
provides:
  - "RouteAnchors @riverpod family notifier: appendAnchor/dragAnchor/insertAnchorOnSegment/retrySegment/toggleAutoRouting/undo/redo"
  - "segmentKey (stable anchor-id-pair identity) and splitSegmentAt (geometric insert) in route_segment_util.dart"
affects: [19-03-map-overlays, 19-04-screen]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-segment CancelToken + generation-counter guard against out-of-order Valhalla responses, keyed by segmentKey (anchor-id pair), never array index"
    - "Immutable-snapshot undo/redo stack (List<RouteAnchorsSnapshot>), not a diff/patch stack"
    - "Fake-Dio-interceptor test harness (InterceptorsWrapper.onRequest -> handler.resolve/reject) with zero new test-mocking dependency"
    - "Seeded-provider test harness: a RouteAnchors subclass overriding build() to inject fixture state, decoupling segment-resolution tests from the anchor-mutation API"

key-files:
  created:
    - app/lib/provider/route_anchor_provider.dart
    - app/lib/provider/route_anchor_provider.g.dart
    - app/lib/util/route_segment_util.dart
    - app/test/provider/route_anchor_provider_test.dart
    - app/test/util/route_segment_util_test.dart

key-decisions:
  - "toggleAutoRouting() OFF leaves every EXISTING segment untouched (no straight-line rewrite, zero Dio calls); only segments created afterward become straight. Corrects the plan's own Task 1 <behavior>/<action> prose (which said OFF should rewrite every segment to straight) to match the plan's must_haves truth, per a mid-execution correction from the coordinator citing that truth as authoritative."
  - "splitSegmentAt projects the tap point onto every sub-edge of the segment's existing polyline (nearest-point-on-line-segment, not nearest-existing-vertex) — the plan's literal action text (snap to nearest existing vertex, skipping index 0) degenerates to one of the two endpoints for a straight 2-point segment (the common case for a freshly appended or dragged anchor pair), which fails the plan's own acceptance test requiring the split point to sit genuinely between the two endpoints."
  - "Empirically verified (see Deviations) that dio 5.9.2's cancellation races the ENTIRE interceptor step via Future.any, not just the final dispatch — so a request cancelled before its own response arrives ALWAYS settles via DioExceptionType.cancel, never via a late 'successful' response. The race-guard test therefore verifies the OBSERVABLE contract (no corruption, no uncaught throw) rather than asserting the generation-counter counter specifically wins over cancellation, which is not reproducible via black-box request timing against this Dio version."
  - "Added a _SeededRouteAnchors test-only subclass (overrides build() to inject fixture anchors/segments) so segment-resolution tests do not depend on appendAnchor -- keeps Task 1's test file runnable and analyze-clean without Task 2's mutation methods."

requirements-completed: [WAYP-01, WAYP-02, WAYP-03, ROUTE-01, ROUTE-02, ROUTE-04, ROUTE-05]

# Metrics
duration: ~40min
completed: 2026-07-16
---

# Phase 19 Plan 02: Route Planner State Provider — Routing Engine, Mutations, Undo/Redo Summary

**Class-based `RouteAnchors` `@riverpod` family notifier: per-segment Valhalla routing engine with a CancelToken + generation-counter race-guard, append/drag/insert anchor mutations, geometric segment-split for plain taps, and an immutable-snapshot undo/redo stack**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-07-16
- **Tasks:** 2 completed
- **Files modified:** 5 (all created)

## Accomplishments

- Built `RouteAnchorsState` (plain class, `NavigationState`-shaped `copyWith`) and the `RouteAnchors` class-based `@riverpod` family notifier keyed by `travelProfile` (fixed for the notifier's lifetime, D-07)
- Implemented `_resolveSegment`: validates lat/lon range before ever calling Valhalla (V5), decodes `trip.legs[0].shape` at `precision: 6`, and guards every apply with a per-segment `CancelToken` + monotonically increasing generation counter keyed by `segmentKey` (anchor-id pair, never array index — Pitfall 3)
- ROUTE-05: a failed segment is marked `blocked` with its prior polyline left untouched — never silently reverted to straight
- `retrySegment` (D-09) and `toggleAutoRouting` (ROUTE-01/02) both dispatch through `_resolveSegment`; turning auto-routing ON re-resolves every existing segment via `Future.wait` (parallel, not sequential)
- `appendAnchor` (WAYP-01), `dragAnchor` (WAYP-02, re-resolves only its ≤2 adjacent segments), `insertAnchorOnSegment` (WAYP-03, geometric split via `splitSegmentAt`, never calls Valhalla)
- `undo()`/`redo()` (ROUTE-04): immutable `RouteAnchorsSnapshot` stack; every mutation pushes onto `undoStack` and clears `redoStack` (D-11)
- `segmentKey`/`splitSegmentAt` in `route_segment_util.dart`; `splitSegmentAt` projects the tap point onto every sub-edge of the existing polyline (not just the nearest vertex) so straight 2-point segments split correctly
- 16 automated tests (6 resolution-engine, 6 mutation/undo-redo, 4 `route_segment_util`), all against a fake Dio interceptor harness with zero new test-mocking dependency

## Task Commits

1. **Task 1: Segment resolution engine — Valhalla call, race-guard, retry, auto-routing toggle** — `921a25f2` (feat)
2. **Task 2: Anchor mutations (append/drag/insert) + geometric split + undo/redo** — `1d567fd4` (feat)

**Plan metadata:** commit skipped — `commit_docs: false` in `.planning/config.json` (see State Updates below)

## Files Created/Modified

- `app/lib/provider/route_anchor_provider.dart` — `RouteAnchorsState` + `RouteAnchors` notifier: resolution engine, mutations, undo/redo
- `app/lib/provider/route_anchor_provider.g.dart` — generated via `dart run build_runner build --delete-conflicting-outputs`
- `app/lib/util/route_segment_util.dart` — `segmentKey`, `splitSegmentAt`
- `app/test/provider/route_anchor_provider_test.dart` — 12 behavioral tests + fake-Dio-interceptor harness + seeded-provider test harness
- `app/test/util/route_segment_util_test.dart` — 4 tests for `segmentKey`/`splitSegmentAt`

## Decisions Made

- **`toggleAutoRouting()` OFF behavior corrected mid-execution.** The plan's Task 1 `<behavior>`/`<action>` prose said turning auto-routing OFF should replace every existing segment's polyline with a straight 2-point line. The plan's own `must_haves.truths` (the authoritative spec, per this plan's frontmatter) says the opposite: "Toggling auto-routing off leaves existing segments untouched. New segments after that are added as a straight line without a network call." The coordinator flagged this discrepancy mid-task (citing a user edit to `19-02-PLAN.md`) and directed implementing the must_haves version. Implemented: OFF now flips the flag and returns immediately (no segment mutation, zero Dio calls); ON is unchanged (re-resolves every existing segment via Valhalla in parallel). Only NEW segments created while auto-routing is off (via `appendAnchor`/`dragAnchor`/`insertAnchorOnSegment`) become straight lines — this was already correct in the original implementation and needed no change.
- **`splitSegmentAt` uses nearest-point-on-sub-edge, not nearest-existing-vertex.** The plan's action text describes snapping to the nearest existing polyline vertex (correct for a multi-point routed polyline, where the split point should lie on the already-rendered path). Applied literally to a straight, 2-point (un-routed) segment — the common case for a freshly appended or dragged anchor pair, per WAYP-01/02 — this degenerates: the only candidate vertex checked (loop starts at index 1) is the segment's own end anchor, so the "split" would sit exactly on one of the two original endpoints rather than genuinely between them. The plan's own `route_segment_util_test.dart` behavior spec explicitly requires "the split point inserted between them" for a straight 2-point segment. Implemented a generalized nearest-point-on-line-segment projection (mirrors `navigation_provider.dart`'s existing cross-track math) that subsumes the vertex-snap behavior for genuinely multi-vertex routed polylines while correctly handling the 2-point case.
- **Race-guard test verifies the observable contract, not "generation counter specifically, not cancellation."** Direct experimentation against the installed `dio` 5.9.2 (see Deviations below) showed that `dio_mixin.dart`'s `listenCancelForAsyncTask` races the ENTIRE interceptor step (not just the final HTTP dispatch) via `Future.any([theStep, cancelToken.whenCancel.then((e) => throw e)])`. Because `_resolveSegment` always cancels the previous in-flight token before dispatching a new request for the same segment key, and because a still-pending step always loses that race to an already-fired `cancel()`, the earlier of two back-to-back `retrySegment` calls for the same key reliably settles via `DioExceptionType.cancel` — it can never "win" the race with a late, successful response in this harness. The generation-counter check remains present and correct as defense-in-depth (verified via source read, matches the plan's acceptance-criteria grep), but isolating it from cancellation via black-box request timing proved unreproducible against this Dio version. The test was adjusted to verify what's actually true and stable: a stale, superseded dispatch never corrupts state and never throws uncaught, regardless of which of the two guards catches it.
- **Added a seeded-provider test harness (`_SeededRouteAnchors`).** The segment-resolution engine's tests originally used `appendAnchor` to build 2-anchor/1-segment fixtures, which made Task 1's test file depend on Task 2's mutation API — incompatible with atomic, task-ordered commits. Added a `RouteAnchors` subclass overriding `build()` to inject fixture state directly, keeping Task 1's tests self-contained.
- **`routeAnchorsProvider` is `autoDispose` (no `keepAlive`).** Discovered mid-task: `container.read(...)` alone doesn't keep an `autoDispose` family provider instance alive across an `await` gap, causing "Cannot use the Ref ... after it has been disposed" errors and cross-test-leaked async errors once tests started `await`-ing real mutation methods. Fixed via `container.listen(routeAnchorsProvider(_profile), (_, _) {})` in the test harness, mirroring how a real screen's `ref.watch` would keep it alive in production — no production code change needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `toggleAutoRouting()` OFF behavior corrected to match the plan's must_haves truth**
- **Found during:** Task 1, flagged by the coordinator mid-task after a user edit to `19-02-PLAN.md`'s `must_haves.truths` (line 17)
- **Issue:** Implementation (and the plan's own Task 1 `<behavior>`/`<action>` prose) replaced every existing segment's polyline with a straight line on toggle-OFF; the plan's must_haves truth requires existing segments to be left untouched
- **Fix:** `toggleAutoRouting()`'s OFF branch now only flips the flag and returns; no segment mutation, zero Dio calls. Doc comment updated to cite the must_haves truth as the corrected source.
- **Files modified:** `app/lib/provider/route_anchor_provider.dart`, `app/test/provider/route_anchor_provider_test.dart`
- **Commit:** `921a25f2`

**2. [Rule 1 - Bug] `splitSegmentAt` fixed to handle straight (2-point) segments correctly**
- **Found during:** Task 2, while making the plan's own `route_segment_util_test.dart` behavior spec pass
- **Issue:** The plan's literal nearest-existing-vertex algorithm degenerates for a 2-point straight segment (splits at an endpoint, not a genuine midpoint), failing the plan's own acceptance test
- **Fix:** Generalized to project the tap point onto every sub-edge of the polyline (nearest-point-on-line-segment), which correctly subsumes both the straight-2-point and multi-vertex-routed cases
- **Files modified:** `app/lib/util/route_segment_util.dart`
- **Commit:** `1d567fd4`

**3. [Rule 3 - Blocking issue] `routeAnchorsProvider` autoDispose lifecycle in tests**
- **Found during:** Task 1, while writing async tests
- **Issue:** `container.read()` alone doesn't retain an `autoDispose` family provider across an `await` gap; in-flight fire-and-forget work threw "Ref used after disposed" and leaked uncaught errors into unrelated tests
- **Fix:** Test harness establishes a persistent `container.listen(...)` subscription, mirroring production's `ref.watch` usage pattern
- **Files modified:** `app/test/provider/route_anchor_provider_test.dart`
- **Commit:** `921a25f2`

None of the above are architectural changes (Rule 4) — all are bug fixes or test-infrastructure corrections within the plan's existing design.

## Issues Encountered

- Empirically investigated `dio` 5.9.2's cancellation semantics (via a throwaway experiment script, since removed) to determine why a race-guard test hung/mis-asserted. Root cause: `dio_mixin.dart`'s `Future(...)` constructor uses `Timer.run` (a macrotask), not a microtask, so a pure `await Future.value()` loop does not flush it — fixed test timing helpers to use a real (if brief) `Future.delayed`. Separately confirmed cancellation races the entire interceptor step, not just the final dispatch (see Decisions Made above).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `RouteAnchors` exposes the full public API 19-03 (map overlays) and 19-04 (screen) need: `appendAnchor`, `dragAnchor`, `insertAnchorOnSegment`, `retrySegment`, `toggleAutoRouting`, `undo`, `redo`.
- `segmentKey`/`splitSegmentAt` are ready for 19-03's GeoJSON feature-property keying and hit-test-driven insert flow.
- No blockers identified for 19-03 (map overlays: anchor markers, segment rendering, gesture routing) or 19-04 (screen shell, app-bar undo/redo buttons).

## Self-Check: PASSED
