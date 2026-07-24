---
phase: 19-route-planner-core-waypoint-editing-routing-engine
plan: 01
subsystem: mobile-app
tags: [flutter, freezed, polyline, valhalla, route-planner, dart]

# Dependency graph
requires:
  - phase: 14-coordinate-type-migration
    provides: "Geographic-typed polyline_util.dart, established freezed 3.x model convention"
provides:
  - "Precision-parameterized PolylineUtil.decode/encode (default 5, Valhalla callers pass 6)"
  - "RouteAnchor/RouteSegment/SegmentState/RouteAnchorsSnapshot in-memory route data model (D-01)"
affects: [19-02-state-provider, 19-03-map-overlays, 19-04-screen]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Precision-aware polyline codec: named `{int precision = 5}` parameter + `math.pow(10, precision)` factor, default preserves existing callers"
    - "In-memory-only freezed models omit `part 'x.g.dart'` (no JSON codegen needed) when the model never crosses a serialization boundary"
    - "Explicit before/after-id segment identity instead of array-position identity, for stability across list mutation"

key-files:
  created:
    - app/lib/models/route_anchor.dart
    - app/lib/models/route_anchor.freezed.dart
    - app/test/util/polyline_util_test.dart
  modified:
    - app/lib/util/polyline_util.dart

key-decisions:
  - "Divergence test asserts on longitude, not latitude, because Geographic clamps latitude to [-90, 90] which masked the intended 10x-scale assertion at the chosen fixture coordinates"
  - "route_anchor.dart's D-01 no-'waypoint'-string constraint applies to comments too, not just identifiers — a first-draft doc comment mentioning the sibling Trail model by name was reworded to satisfy the plan's own grep gate"

patterns-established:
  - "Precision-parameterized codec pattern for any future Google-encoded-polyline consumer in this app"
  - "In-memory freezed model with no .g.dart part for state that never serializes to/from JSON"

requirements-completed: [WAYP-01, ROUTE-01, ROUTE-05]

# Metrics
duration: 12min
completed: 2026-07-16
---

# Phase 19 Plan 01: Polyline Precision Fix + Route Anchor Model Summary

**Precision-parameterized Google-encoded-polyline codec (default 5, Valhalla decodes at 6) plus a new freezed `RouteAnchor`/`RouteSegment`/`SegmentState`/`RouteAnchorsSnapshot` in-memory route model that never reuses the persisted `Waypoint` type**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-07-16T13:18:27Z
- **Completed:** 2026-07-16T13:25:14Z
- **Tasks:** 2 completed
- **Files modified:** 4 (1 modified, 3 created)

## Accomplishments
- Fixed the verified polyline-precision bug: `PolylineUtil.decode`/`encode` now accept a `precision` parameter (default 5, unchanged for existing trail-GPX callers); Phase 19's Valhalla `/route` consumer (19-02) will pass `precision: 6`
- Added a 4-test regression suite proving default-5 round-trip, precision-6 round-trip, precision-5-vs-6 divergence, and defensive empty-string handling
- Defined the new in-memory route-planner data model (`RouteAnchor`, `RouteSegment`, `SegmentState`, `RouteAnchorsSnapshot`) per D-01 — a deliberately distinct type set from the persisted route-point model, with explicit `beforeAnchorId`/`afterAnchorId` segment identity for stability across list mutation

## Task Commits

Each task was committed atomically (Task 1 followed TDD RED→GREEN):

1. **Task 1: Add precision parameter to PolylineUtil.decode/encode**
   - `4b34b914` (test) - failing precision round-trip tests, confirmed RED against unmodified `polyline_util.dart`
   - `11944d70` (feat) - precision parameter + factor implementation, all 4 tests pass (GREEN)
2. **Task 2: Create RouteAnchor / RouteSegment / SegmentState / RouteAnchorsSnapshot freezed models (D-01)** - `df89a2dc` (feat)

**Plan metadata:** commit skipped — `commit_docs: false` in `.planning/config.json` (see State Updates below)

## Files Created/Modified
- `app/lib/util/polyline_util.dart` - `decode`/`encode` gained a named `precision` parameter (default 5); hardcoded `1E5` literals replaced with `math.pow(10, precision)`
- `app/test/util/polyline_util_test.dart` - 4-test regression suite for the precision codec
- `app/lib/models/route_anchor.dart` - `RouteAnchor` (id/lat/lon + `point` getter), `SegmentState` enum, `RouteSegment` (explicit before/after anchor ids), `RouteAnchorsSnapshot` (undo/redo unit)
- `app/lib/models/route_anchor.freezed.dart` - generated via `dart run build_runner build --delete-conflicting-outputs`

## Decisions Made
- Divergence-test fixture originally asserted on latitude; `Geographic` clamps latitude to [-90, 90], which silently capped the intended 10x-scaled value and made the assertion pass for the wrong reason during a debug run. Switched the assertion to longitude (unclamped in this range), verified the ratio is genuinely ~10x before fixing forward.
- `route_anchor.dart`'s doc comment initially referenced the sibling persisted-route-point model by its literal type name to explain the D-01 distinction; the plan's own acceptance criterion greps the whole file case-insensitively for that word, so the comment was reworded to describe the distinction without naming the type.

## Deviations from Plan

None - plan executed exactly as written. Both fixture/comment fixes above were self-contained corrections made while satisfying the plan's own explicit acceptance criteria (test correctness and the D-01 grep gate), not scope changes.

## Issues Encountered

None beyond the two self-corrections noted in Decisions Made above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `PolylineUtil` is ready for 19-02's Valhalla `/route` consumer (`decode(shape, precision: 6)`).
- `RouteAnchor`/`RouteSegment`/`SegmentState`/`RouteAnchorsSnapshot` are ready for 19-02's `route_anchor_provider.dart` state notifier to consume as its core types.
- No blockers identified for 19-02 (Route Planner state provider / routing engine).

## Self-Check: PASSED

All created/modified files verified present on disk; all 3 task commit hashes (`4b34b914`, `11944d70`, `df89a2dc`) verified present in git log.

---
*Phase: 19-route-planner-core-waypoint-editing-routing-engine*
*Completed: 2026-07-16*
