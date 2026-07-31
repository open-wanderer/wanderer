---
phase: 33-conversion-correctness
plan: 02
subsystem: gpx-conversion
tags: [typescript, vitest, gpx, elevation, gpx-metrics]

# Dependency graph
requires:
  - phase: 33-conversion-correctness (plan 01)
    provides: "GPX.getTotals() loop starting at i = 0, so every segment's first point reaches metrics.addAndFilter()"
provides:
  - "parseElevation(): the single coercion point for <ele> element text (undefined-aware, string-vs-number-safe)"
  - "Undefined-aware elevation accumulation in GpxMetricsComputation.addAndFilter() (CONV-03)"
  - "Smoothed elevation sampling decoupled from the horizontal distance threshold (CONV-04)"
  - "Vitest fixture suite for gpx-metrics-computation.ts (9 tests, XML-parse-path coverage for the missing/empty/zero-elevation distinction)"
affects: [33-03, 34]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Inline GPX XML string fixtures (trkptXml/gpxXml helpers) for tests that must exercise the real GPX.parse() path, distinct from 33-01's object-graph builders — used whenever a genuinely-absent <ele> tag (vs. a present-but-empty one) matters"

key-files:
  created: [web/src/lib/models/gpx/gpx-metrics-computation.test.ts]
  modified: [web/src/lib/models/gpx/gpx-metrics-computation.ts]

key-decisions:
  - "Tasks 1 and 2 (both tagged tdd=\"true\" in the plan) were executed as direct, verified code fixes rather than a literal per-task RED/GREEN test-file cycle: neither task's <files> scope includes a test file, and Task 3 is the plan's own dedicated task for the fixture suite covering both defects at once. This mirrors 33-01's established Task 1 (fix) / Task 2 (test) split. Each fix was verified via grep acceptance gates, the existing full unit suite, svelte-check, and an ad-hoc in-repo Vitest spot-check (created, run, then deleted) proving the exact numeric fixtures in the plan's <behavior>/<acceptance_criteria> before committing."
  - "parseElevation applies Number(...) uniformly to both string and number input (no separate typeof branch) — a real number input passes through Number() unchanged, keeping the function's control flow minimal while still satisfying every accepted/rejected case in the plan's <behavior> spec"
  - "The smoothed elevation-diff block and the raw elevation-diff block are now both evaluated unconditionally (gated only on elevation !== undefined and their own per-axis threshold), ahead of the horizontal-threshold gate, which was rewritten from an early return into a positive smoothedDistance >= thresholdXY_m block containing only the totalDistanceSmoothed accumulation and the lastFilteredPointXY advance"

requirements-completed: [CONV-03, CONV-04]

# Metrics
duration: ~10min
completed: 2026-07-31
---

# Phase 33 Plan 02: Undefined-Aware Elevation and Threshold-Independent Smoothing Summary

**Fixed two elevation defects in `GpxMetricsComputation.addAndFilter()` — missing `<ele>` no longer fabricates a plunge to sea level (CONV-03), and smoothed elevation gain/loss is no longer gated behind horizontal movement (CONV-04) — backed by a 9-test Vitest fixture suite.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-07-31T10:15:00Z
- **Completed:** 2026-07-31T10:24:47Z
- **Tasks:** 3
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments

- Added `parseElevation()`, the single coercion point for `<ele>` element text: `undefined`/`null`/empty-or-whitespace strings/non-numeric strings/`NaN`/`Infinity` all map to `undefined`, while a genuine `0` (numeric or `"0"`) is preserved as real sea-level data.
- Replaced all three `point.ele ?? 0` sites in `gpx-metrics-computation.ts`. A partially-elevation-tagged 5-point track that previously reported ~1015 m of gain and ~1005 m of loss now correctly reports 15 m gain and 0 m loss — the gap in `<ele>` tagging is bridged (carry-forward) rather than treated as a drop to and climb back from sea level.
- Removed both `@ts-ignore` comments — the null-anchor branches are now explicit instead of asserted away.
- Restructured `addAndFilter()` so the raw and smoothed elevation-diff blocks run for every elevation-bearing point, independent of horizontal movement. The horizontal threshold now gates only `totalDistanceSmoothed` and the `lastFilteredPointXY` anchor advance via a positive `smoothedDistance >= thresholdXY_m` block (the old early `return` is gone). An 88 m climb spread over ~4.4 m of horizontal movement now reports 88 m of gain instead of 0, while `totalDistanceSmoothed` for that same stretch correctly stays 0.
- Verified via a 16-point GPS-jitter fixture that distance smoothing itself was not weakened: `totalDistanceSmoothed` still suppresses to ~100.075 m against a raw ~110.083 m.
- Created `web/src/lib/models/gpx/gpx-metrics-computation.test.ts`: 9 tests across 5 describe blocks — `parseElevation` unit coverage, CONV-03 missing/empty elevation (driven through the real `GPX.parse()` XML path per D-05), CONV-03 genuine sea level, CONV-04 steep/low-horizontal-movement stretch (both the elevation and the distance-smoothing anti-regression assertion), and the unchanged jitter-track distance-smoothing guard.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add parseElevation and make elevation accumulation undefined-aware (CONV-03)** - `0ea325a7` (fix)
2. **Task 2: Decouple smoothed elevation sampling from the horizontal distance threshold (CONV-04)** - `d7023fda` (fix)
3. **Task 3: Create the GpxMetricsComputation Vitest fixture suite for CONV-03 and CONV-04** - `79ff370c` (test)

**Plan metadata:** pending (docs: complete plan)

## Files Created/Modified

- `web/src/lib/models/gpx/gpx-metrics-computation.ts` - New exported `parseElevation()` helper; `addAndFilter()` rewritten to be undefined-aware for elevation and to decouple smoothed elevation sampling from the horizontal-threshold gate; both `@ts-ignore` comments removed
- `web/src/lib/models/gpx/gpx-metrics-computation.test.ts` - New Vitest suite: `trkptXml`/`gpxXml` inline XML-fixture helpers, 5 describe blocks, 9 tests, all passing

## Decisions Made

- Followed the plan's task split literally: Tasks 1 and 2 are code-correctness fixes verified via grep acceptance gates, the pre-existing full unit suite (must stay green), `svelte-check`, and disposable ad-hoc Vitest spot-checks proving each plan-specified numeric fixture before committing — the dedicated, permanent fixture suite is Task 3's explicit deliverable, matching 33-01's precedent exactly (fix task, then test task).
- Kept the horizontal-threshold restructure minimal and structurally defensive per the threat model (T-33-07): the new `if (smoothedDistance >= this.thresholdXY_m) { ... }` block contains *only* the two statements the old code executed after its early return (`totalDistanceSmoothed +=` and `lastFilteredPointXY =`), so nothing downstream of the old `return` can accidentally be skipped again and no sub-threshold distance can leak into `totalDistanceSmoothed`.
- `gpxXml`/`trkptXml` test helpers construct GPX XML via plain template strings (not object graphs) specifically for the CONV-03 cases, since only the real XML parse path can produce the genuine `undefined` (tag omitted) vs. `""` (tag present but empty) distinction that `Waypoint`'s object-graph constructor cannot simulate — this follows 33-CONTEXT.md's D-05 exactly.

## Deviations from Plan

None - plan executed exactly as written. All three tasks' actions, acceptance criteria, and verification commands were followed as specified; every grep gate and behavioral fixture in the plan passed on first execution.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `gpx-metrics-computation.ts` now exports `parseElevation` and has a fully undefined-aware, threshold-decoupled elevation pipeline — the precondition 33-03 needs before touching `cumulativeDistance` and the `totalDistance` → `totalDistanceSmoothed` source swap in `gpx.ts`.
- The horizontal threshold (`thresholdXY_m`) still exclusively gates `totalDistanceSmoothed` and `lastFilteredPointXY`, confirmed unweakened by the jitter-track regression guard — 33-03 can safely make `totalDistanceSmoothed` the reported trail distance without inheriting new GPS-jitter inflation.
- No blockers. `gpx.ts` was confirmed untouched by this plan (`git diff --name-only f6476a5a..HEAD` lists only the two `gpx-metrics-computation.*` files), keeping 33-03's `gpx.ts`/`+page.svelte` scope a clean starting point.

---
*Phase: 33-conversion-correctness*
*Completed: 2026-07-31*

## Self-Check: PASSED

- FOUND: web/src/lib/models/gpx/gpx-metrics-computation.ts
- FOUND: web/src/lib/models/gpx/gpx-metrics-computation.test.ts
- FOUND: commit 0ea325a7
- FOUND: commit d7023fda
- FOUND: commit 79ff370c
