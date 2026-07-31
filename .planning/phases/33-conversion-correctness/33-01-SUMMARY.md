---
phase: 33-conversion-correctness
plan: 01
subsystem: gpx-conversion
tags: [typescript, vitest, gpx, gpx-metrics, distance-calculation]

# Dependency graph
requires: []
provides:
  - "Corrected GPX.getTotals() accumulation loop starting at index 0 (CONV-01)"
  - "Centroid divisor structurally tied to summedPointCount, not allPoints.length (CONV-02)"
  - "First Vitest fixture suite for web/src/lib/models/gpx/, establishing the inline-fixture pattern for 33-02/33-03"
affects: [33-02, 33-03, 34]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Inline GPX fixture construction via real GPX/Track/TrackSegment/Waypoint constructors (no disk .gpx files, no mocks) — established for gpx-metrics-computation.test.ts (33-02) to follow"

key-files:
  created: [web/src/lib/models/gpx/gpx.test.ts]
  modified: [web/src/lib/models/gpx/gpx.ts]

key-decisions:
  - "Loop-bound fix (i = 1 -> i = 0) landed strictly before any distance-source change, per D-03, so 33-03's cumulativeDistance rebuild is index-aligned"
  - "No metrics-anchor reset introduced at segment boundaries — cross-segment continuity (shared GpxMetricsComputation instance) is what makes the multi-leg planner route measure through its anchors instead of restarting each leg"
  - "Zero-point sentinel behavior (NaN centroid, Infinity/-Infinity bbox, 0 distance) deliberately left unchanged and pinned by a regression-guard test, not treated as a defect"

requirements-completed: [CONV-01, CONV-02]

# Metrics
duration: 5min
completed: 2026-07-31
---

# Phase 33 Plan 01: Fix getTotals() Off-By-One and Centroid Divisor Summary

**Fixed the root-cause off-by-one accumulation loop in `GPX.getTotals()` (index 0 instead of 1) and made the centroid divisor structurally match the summed point count, backed by the domain's first Vitest fixture suite.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-31T10:10:07Z
- **Completed:** 2026-07-31T10:14:59Z
- **Tasks:** 2
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments

- `GPX.getTotals()`'s point loop now starts at `i = 0`, so every track segment's first point contributes to distance, bounding box, and centroid — fixes CONV-01 and the route planner's dropped opening hop between legs.
- Centroid now divides by a new `summedPointCount` local, incremented in lockstep with `totalLat`/`totalLon` inside the same loop — the divisor can never disagree with the numerator by construction (CONV-02).
- Created `web/src/lib/models/gpx/gpx.test.ts`, the first Vitest suite for this domain: 6 tests across 4 describe blocks covering the 2-point segment, extreme-first-point centroid/bounding-box, the 2-leg planner route with a shared duplicated anchor, and a zero-point regression guard.
- Verified numerically: a 2-point ~134.6 m segment now reports 134.59 m (was 0); a 4-hop, 2-leg planner route reports 444.78 m through its shared anchor (was 333.585 m, silently dropping one hop).

## Task Commits

1. **Task 1: Start the getTotals() accumulation loop at index 0 and divide the centroid by the summed point count** - `330fe7df` (fix)
2. **Task 2: Create the GPX.getTotals() Vitest fixture suite for CONV-01 and CONV-02** - `853c28c8` (test)

**Plan metadata:** pending (docs: complete plan)

## Files Created/Modified

- `web/src/lib/models/gpx/gpx.ts` - `getTotals()` loop now starts at `i = 0`; new `summedPointCount` local drives the centroid divisor instead of `allPoints.length`
- `web/src/lib/models/gpx/gpx.test.ts` - New Vitest suite: `waypointAt`/`gpxFromSegments`/`hopMetres` inline-fixture helpers, 4 describe blocks, 6 tests, all passing

## Decisions Made

- Followed 33-CONTEXT.md D-03 exactly: loop-bound fix first, `cumulativeDistance` rebuild deferred to 33-03. No assertions on `cumulativeDistance` or elevation were added in this file (verified via grep gates), keeping those defects scoped to their owning plans (33-02, 33-03).
- Used `hopMetres` (wrapping `haversineDistance`) to express the 2-point and multi-leg expected distances as sums of real hop computations rather than hardcoded magic numbers, per the plan's `read_first` guidance — while still asserting the literal pre-fix/post-fix numeric values (134.592, 444.78) as a second, human-readable check.
- Chose not to perform a literal `git stash`/revert cycle to prove test-catches-bug (Task 2's acceptance criteria offer this as an alternative to "reasoning against the recorded pre-fix values"); the pre-fix values (exactly 0, and 333.585) were independently derived by hand-tracing `GpxMetricsComputation.addAndFilter()`'s shared-instance first-call semantics against the old `i = 1` loop, confirming the tests are load-bearing without touching already-committed history.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `GpxMetricsComputation.addAndFilter()` is now called once per point starting at index 0 for every segment, which is the precondition 33-03's `cumulativeDistance` rebuild needs to be index-aligned.
- The inline-fixture Vitest pattern (`waypointAt`/`gpxFromSegments`-style builders, no disk reads, no mocks) is established and ready for 33-02's `gpx-metrics-computation.test.ts` to follow.
- No blockers. `gpx-metrics-computation.ts` was confirmed untouched by this plan (required for 33-02 to start from a clean baseline).

---
*Phase: 33-conversion-correctness*
*Completed: 2026-07-31*

## Self-Check: PASSED

- FOUND: web/src/lib/models/gpx/gpx.ts
- FOUND: web/src/lib/models/gpx/gpx.test.ts
- FOUND: .planning/phases/33-conversion-correctness/33-01-SUMMARY.md
- FOUND: commit 330fe7df
- FOUND: commit 853c28c8
