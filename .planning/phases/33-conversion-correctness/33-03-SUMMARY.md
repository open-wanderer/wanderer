---
phase: 33-conversion-correctness
plan: 03
subsystem: gpx-conversion
tags: [typescript, vitest, svelte, gpx, gpx-metrics, distance-calculation, trail-edit]

# Dependency graph
requires:
  - phase: 33-conversion-correctness (plan 01)
    provides: "GPX.getTotals() loop starting at i = 0, so cumulativeDistance's rebuild is index-aligned"
  - phase: 33-conversion-correctness (plan 02)
    provides: "Undefined-aware, threshold-decoupled elevation pipeline in GpxMetricsComputation, confirming totalDistanceSmoothed stays unweakened before becoming the reported distance"
provides:
  - "Reported trail distance sourced from metrics.totalDistanceSmoothed instead of the raw haversine sum (CONV-05)"
  - "cumulativeDistance repaired in place as a raw, index-aligned per-point array with a leading 0 entry (D-01)"
  - "Trail-edit crop slider rescaled to the raw cumulative array's own total, with an empty/short-route guard against NaN coordinates (D-02)"
  - "CONV-05/D-01 fixture coverage appended to gpx.test.ts (12 tests total in that file)"
affects: [34]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "cumulativeDistance.push() happens exactly once per addAndFilter() call, including the first-call branch, keeping array length structurally equal to call count regardless of input validity"

key-files:
  created: []
  modified:
    - web/src/lib/models/gpx/gpx-metrics-computation.ts
    - web/src/lib/models/gpx/gpx.ts
    - web/src/routes/trail/edit/[id]/+page.svelte
    - web/src/lib/models/gpx/gpx.test.ts

key-decisions:
  - "cumulativeDistance stays raw (accumulates `distance`, never `smoothedDistance`) even though the reported distance is now smoothed — the two are deliberately decoupled per D-01, and an executable invariant (distance < cumulativeDistance[last]) pins this for the jitter fixture"
  - "totalDistance accumulation in gpx-metrics-computation.ts is now gated on Number.isFinite(distance) while the cumulativeDistance push remains unconditional, so a hostile/non-finite coordinate can neither poison the running total nor desync the array's length from the point count (T-33-10/T-33-11)"
  - "updateCropMarkers() gets an early return when cumulativeRoute.length < 2 or rawRouteTotal is non-finite, closing a pre-existing path where an empty/short route reached getCoordinateAtDistance() and produced NaN coordinates for MapLibre's setLngLat (T-33-12)"

requirements-completed: [CONV-05]

# Metrics
duration: ~8min
completed: 2026-07-31
---

# Phase 33 Plan 03: Source Reported Distance from Smoothed Total, Repair cumulativeDistance Summary

**A converted trail's reported `distance` now comes from `metrics.totalDistanceSmoothed` instead of the raw GPS-jitter-inflated haversine sum (CONV-05), with `cumulativeDistance` rebuilt in place as a raw, index-aligned array and the trail-edit crop slider rescaled to that array's own total so its pins stay on the route polyline.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-07-31T10:24:00Z
- **Completed:** 2026-07-31T10:32:27Z
- **Tasks:** 3
- **Files modified:** 4 (3 modified in-place, 1 test file extended)

## Accomplishments

- `GpxMetricsComputation.addAndFilter()` now pushes to `cumulativeDistance` exactly once per call — including the first-call branch, which previously returned before ever pushing — so the array holds one raw entry per point with a leading `0`, index-aligned with `flatten()`.
- Guarded `totalDistance` accumulation with `Number.isFinite(distance)` while keeping the `cumulativeDistance` push unconditional, so a hostile/non-finite coordinate can neither poison the running total nor desync the array's length from the point count.
- `GPX.getTotals()` now reads `metrics.totalDistanceSmoothed` for the reported `distance` (CONV-05). Verified numerically: the 16-point jitter fixture reports `~100.075 m` (real forward travel) instead of `~110.083 m` (raw haversine sum), while `cumulativeDistance` stays raw and ends at `~110.083 m`, now length 16 (was 15).
- The two-leg planner route is unaffected: every hop clears the 5 m smoothing threshold, so raw and smoothed totals coincide and `distance` stays `~444.78 m`.
- `updateCropMarkers()` in `trail/edit/[id]/+page.svelte` now rescales both crop-target percentages to `rawRouteTotal` (the raw cumulative array's own last entry) instead of the now-smoothed `features.distance`, and passes the same `cumulativeRoute` local to both `getCoordinateAtDistance()` calls so the interpolation basis and percentage basis are provably the same array. Added an early return when `cumulativeRoute.length < 2 || !Number.isFinite(rawRouteTotal)`, closing a pre-existing path where an empty/short route produced NaN coordinates for MapLibre's `setLngLat`.
- Extended `gpx.test.ts` with three new `describe` blocks (6 new tests, 12 total in the file): CONV-05 smoothed-distance assertion plus an executable `distance < cumulativeDistance[last]` decoupling invariant, D-01 index-alignment coverage across a 2-point segment / two-leg planner route (including a non-decreasing check) / empty segment, and a CONV-05 non-regression check on the planner route. Whole unit suite: 31 tests across 5 files, all green.

## Task Commits

1. **Task 1: Rebuild cumulativeDistance as an index-aligned raw array and report the smoothed distance** - `b7631bef` (fix)
2. **Task 2: Rescale the trail-edit crop slider to the raw cumulative total (D-02)** - `023de25a` (fix)
3. **Task 3: Add CONV-05 fixtures to gpx.test.ts and prove the whole suite green** - `6397edef` (test)

**Plan metadata:** pending (docs: complete plan)

## Files Created/Modified

- `web/src/lib/models/gpx/gpx-metrics-computation.ts` - `cumulativeDistance.push()` now runs once per call including the first-call branch; `totalDistance` accumulation gated on `Number.isFinite(distance)`
- `web/src/lib/models/gpx/gpx.ts` - `getTotals()` reads `metrics.totalDistanceSmoothed` for the reported distance; added a comment recording the D-01 raw/index-aligned contract at the `cumulativeDistance` return site
- `web/src/routes/trail/edit/[id]/+page.svelte` - `updateCropMarkers()` rescales to `rawRouteTotal`, shares one `cumulativeRoute` local across both `getCoordinateAtDistance()` calls, early-returns on a too-short/non-finite route
- `web/src/lib/models/gpx/gpx.test.ts` - 3 new describe blocks / 6 new tests covering CONV-05 and D-01, reusing the existing `waypointAt`/`gpxFromSegments`/`hopMetres` helpers

## Decisions Made

- Followed the plan's task split exactly: Task 1 is the code-correctness fix (verified via grep acceptance gates, the pre-existing full unit suite, `svelte-check`, `npm run build`, and a disposable ad-hoc Vitest spot-check proving the plan's exact numeric fixtures before committing — matching 33-01/33-02's precedent), Task 2 is the bounded `+page.svelte` edit, Task 3 is the dedicated fixture-suite task.
- Kept `cumulativeDistance` accumulating `distance` (raw), never `smoothedDistance` — verified no `cumulativeDistance.push(this.totalDistanceSmoothed` occurrence exists, per D-01's explicit decoupling requirement.
- No 33-01/33-02 test assertions needed changing — the two-leg planner fixture's every hop clears the 5 m threshold, so its `features.distance` assertion was unaffected by the CONV-05 source swap, exactly as the plan predicted.

## Deviations from Plan

None - plan executed exactly as written. All three tasks' actions, acceptance criteria, and verification commands were followed as specified; every grep gate and behavioral fixture in the plan passed on first execution.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CONV-01 through CONV-05 are now all fixed and fixture-covered; Phase 34 (Dart Conversion Port) can port this corrected algorithm without pinning any of the four original defects.
- `cumulativeDistance`'s raw/index-aligned contract (`length === flatten().length`, first entry `0`, raw not smoothed) is now the load-bearing invariant behind the trail-edit crop slider — any future change to `GpxMetricsComputation.addAndFilter()`'s call cadence must preserve exactly-one-push-per-call.
- **Known risk carried forward (no task, deliberate, per CONV-F01):** trails already persisted with the buggy metrics keep their wrong `distance`/`elevation_gain`/`elevation_loss` until re-saved or re-imported. A re-saved trail's numbers will now visibly differ from its stored numbers. No migration task exists by explicit product decision.
- **Known risk carried forward (out of this phase's locked scope):** `web/src/lib/components/trail/trail_anchor_list.svelte` carries an independent copy of the `i = 1` loop bug and reads `metrics.totalDistance` (raw) for its per-anchor readout, so its per-segment distances will not sum to the route's reported smoothed distance. Flagged for follow-up; not edited.
- **Deferred human verification (per `workflow.human_verify_mode = end-of-phase`):** Task 2's `<human-check>` — dragging the trail-edit crop range slider with 3+ anchors to confirm the crop pins land exactly on the route polyline at every position, including 0%/100%, with no NaN totals during drag — is deferred to the end-of-phase verification pass rather than gated here.

---
*Phase: 33-conversion-correctness*
*Completed: 2026-07-31*

## Self-Check: PASSED

- FOUND: web/src/lib/models/gpx/gpx-metrics-computation.ts
- FOUND: web/src/lib/models/gpx/gpx.ts
- FOUND: web/src/routes/trail/edit/[id]/+page.svelte
- FOUND: web/src/lib/models/gpx/gpx.test.ts
- FOUND: .planning/phases/33-conversion-correctness/33-03-SUMMARY.md
- FOUND: commit b7631bef
- FOUND: commit 023de25a
- FOUND: commit 6397edef
- FOUND: commit 00032d82
