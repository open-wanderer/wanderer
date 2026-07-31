# Phase 33: Conversion Correctness - Context

**Gathered:** 2026-07-31
**Status:** Ready for planning
**Source:** Orchestrator-captured during /gsd-plan-phase (no discuss-phase pass; decisions raised by RESEARCH.md findings that contradicted the ROADMAP)

<domain>
## Phase Boundary

Fix four real defects in Wanderer's shared TypeScript GPX computation so that every GPX
converted anywhere — web upload or server-side conversion — reports correct distance,
elevation, and duration, before the Dart port is pinned against it.

**In scope (web-only, per ROADMAP scope note):**
- `web/src/lib/models/gpx/gpx.ts`
- `web/src/lib/models/gpx/gpx-metrics-computation.ts`
- `web/src/lib/util/gpx_util.ts`
- Vitest fixture suite for the above (none exists today)
- The minimum change to `web/src/routes/trail/edit/[id]/+page.svelte` required by D-01/D-02
  below — this is a deliberate, bounded extension of the 3-file scope note, admitted only
  because CONV-05 cannot be delivered without it.

**Out of scope:**
- CONV-06 (moving time) — moved to Phase 34; depends on app-side `pausedAccum`.
- Any Dart or Go change.
- Recomputing / migrating already-stored trail distance and elevation values
  (CONV-F01, already deferred in REQUIREMENTS.md). Flag the risk; do not act on it.

</domain>

<decisions>
## Implementation Decisions

### CONV-05 — `cumulativeDistance` disposition

- **D-01 [locked]:** Do **not** delete `cumulativeDistance`. Repair it in place: rebuild it
  as an **index-aligned, raw (unsmoothed) per-point array**, explicitly decoupled from the
  reported `distance` total (which becomes `metrics.totalDistanceSmoothed`).

  **Rationale:** ROADMAP criterion 3 calls the array "dead", but RESEARCH.md verified it is a
  live, wired consumer — the trail-edit route-crop slider
  (`web/src/routes/trail/edit/[id]/+page.svelte`, `updateCropMarkers()` /
  `getCoordinateAtDistance()`, driven by `RouteEditor.svelte`). The array is genuinely
  *misaligned* (the `i = 1` loop bug), which is the real defect the ROADMAP was describing.
  Deleting it outright would break a shipped feature. The ROADMAP's "is gone" language is
  satisfied by the misaligned array being gone, replaced by a correct one.

- **D-02 [locked]:** The crop slider's percentage math must **rescale to the raw cumulative
  array's own total**, not to the reported (now-smoothed) route `distance`.

  **Rationale:** once `distance` is sourced from the smoothed accumulator, the two totals
  diverge by the smoothing delta. `getCoordinateAtDistance()` interpolates between adjacent
  raw points, so its percentage basis must be the raw array's total or pin placement drifts —
  the drift grows with GPS jitter. Keeps "what we display" and "where we interpolate"
  cleanly separated.

### Sequencing

- **D-03 [locked]:** Fix the `gpx.ts` loop bound (`i = 1` → `i = 0`, with the paired
  centroid divisor) **first**. It is the single root cause behind CONV-01, CONV-02, and the
  route planner's "cuts the corner at each anchor" symptom (each planner leg is its own
  `TrackSegment`, so the anchor-hop between legs is dropped). The `cumulativeDistance`
  rebuild (D-01) depends on the loop bound already being correct, so it lands **last**.

### Testing

- **D-04 [locked]:** Ship a Vitest fixture suite covering, at minimum, one fixture per
  success criterion: a 2-point segment, a partial-elevation track, a steep switchback /
  low-horizontal-movement stretch, a jittery track, and a multi-anchor planned route.
  No `gpx*.test.ts` exists in the repo today; Vitest is installed and ready.

- **D-05 [locked]:** Fixtures are **inline, not disk files**. This follows the repo's only
  Vitest precedent (`web/src/lib/models/trail.test.ts`), which builds fixtures as inline
  object graphs through the real class constructors — no mocks, no disk reads. Concretely:
  - Tests that exercise the **XML parse path** (notably CONV-03, where a missing `<ele>`
    tag must stay `undefined` rather than becoming `0`) use **inline GPX XML strings**.
    A missing `<ele>` cannot be simulated by an object graph, because `Waypoint` only
    yields the genuine `undefined` when the tag is absent from the parsed XML
    (`waypoint.ts:53-55` — note `lat`/`lon` fall back to `-1` but `ele` has no fallback).
  - Tests that exercise **computation only** use inline `new GPX({...})` object graphs.

  Test files co-locate beside the code under test in `web/src/lib/models/gpx/`, which the
  Vitest `include: ['src/**/*.{test,spec}.{js,ts}']` glob in `web/vite.config.ts` already
  picks up. No new test-runner config is needed.

### Claude's Discretion

- Fixture helper/builder naming and internal structure (within the D-05 constraint).
- Whether the empty-segment / zero-point edge case (pre-existing, not one of the four
  named defects) gets its own regression-guard fixture — include it if it costs little.
- Internal function decomposition within the three in-scope files.
- How elevation "no data" is represented internally (skip vs. carry-forward vs. explicit
  `undefined` guard), provided a missing `<ele>` never contributes a phantom drop to 0m.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase research
- `.planning/phases/33-conversion-correctness/33-RESEARCH.md` — root-cause analysis with
  file:line references for all four defects, the full consumer/blast-radius map, and the
  `cumulativeDistance` correction that produced D-01/D-02.

### Project scope
- `.planning/ROADMAP.md` — Phase 33 section, including the web-only scope note.
- `.planning/REQUIREMENTS.md` — CONV-01 … CONV-05; CONV-F01 (deferred migration).

</canonical_refs>

<specifics>
## Specific Ideas

- `gpx.ts:125` — `for (let i = 1; ...)` is the off-by-one root cause (CONV-01, CONV-02, and
  the planner corner-cutting symptom).
- `gpx.ts:145` — centroid divides by `allPoints.length` while the loop summed fewer points.
- `gpx-metrics-computation.ts:28-29,52` — `point.ele ?? 0` conflates "no `<ele>` tag" with
  "sea level" (CONV-03). `Waypoint.ele` is genuinely `undefined` when the tag is absent.
- `gpx-metrics-computation.ts:63-82` — the early `return` on
  `smoothedDistance < thresholdXY_m` gates *all* smoothed elevation logic behind horizontal
  movement, so switchbacks never register (CONV-04).
- `gpx.ts:142` — `metrics.totalDistance` should read `metrics.totalDistanceSmoothed` (CONV-05).
- `ElevationProfileControl` (`elevationprofile.ts:965`) computes its **own** cumulative
  distance from raw GeoJSON and is confirmed **not** a consumer of `GPXFeature.cumulativeDistance`
  — out of blast radius.

</specifics>

<deferred>
## Deferred Ideas

- CONV-06 (moving time / pause handling) — Phase 34.
- CONV-F01: recompute or migrate distance/elevation on already-stored trails computed with
  the buggy code. Explicitly deferred in REQUIREMENTS.md. The planner must surface this as a
  known risk in the plan's notes and must NOT add a migration task.

</deferred>

---

*Phase: 33-conversion-correctness*
*Context captured: 2026-07-31 via /gsd-plan-phase (research-driven decisions)*
