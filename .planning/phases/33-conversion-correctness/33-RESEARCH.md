# Phase 33: Conversion Correctness - Research

**Researched:** 2026-07-31
**Domain:** Shared TypeScript GPX-to-trail metrics computation (defect fix, not greenfield)
**Confidence:** HIGH

## Summary

This phase fixes four real, confirmed defects in three existing TypeScript files:
`web/src/lib/models/gpx/gpx.ts`, `web/src/lib/models/gpx/gpx-metrics-computation.ts`, and
`web/src/lib/util/gpx_util.ts`. All four defects trace to two root causes, both read and
verified line-by-line in this research pass:

1. **An off-by-one accumulation loop** in `GPX.getTotals()` (`gpx.ts:125`) that starts at
   `i = 1` instead of `i = 0`, silently dropping every track segment's first point from
   distance, bounding box, and centroid — while the centroid's divisor (`allPoints.length`,
   `gpx.ts:145`) still counts that dropped point. This single bug is the direct cause of
   CONV-01, half of CONV-02, and the route planner's "cuts the corner at every anchor"
   symptom (each planner leg is its own track segment — see Architecture Patterns).
2. **Elevation and distance sourced from the wrong accumulator** in
   `GpxMetricsComputation` (`gpx-metrics-computation.ts`) — missing `<ele>` values coerced
   to `0` (phantom sea-level drop, CONV-03), smoothed elevation gated behind a horizontal
   distance threshold that a steep/low-movement stretch never clears (CONV-04), and the
   reported `distance` sourced from the raw unfiltered haversine sum instead of the
   already-computed smoothed accumulator (CONV-05).

**One material correction to the ROADMAP's framing:** `cumulativeDistance` is **not dead
code**. It is a live, wired-up dependency of the trail-edit screen's route-crop slider
(`web/src/routes/trail/edit/[id]/+page.svelte`, `updateCropMarkers()` /
`getCoordinateAtDistance()`, driven by `RouteEditor.svelte`'s crop UI). It is genuinely
*misaligned* (per the ROADMAP's own description) because of the `i = 1` loop bug, and it
will need a compatible replacement, not a bare deletion — see Common Pitfalls and Open
Questions for the exact interaction with CONV-05.

**Primary recommendation:** Fix the loop bound first (unblocks CONV-01/02/D in one change),
then fix elevation semantics (CONV-03/04) independently, then swap the reported distance
source and rebuild `cumulativeDistance` as an index-aligned, raw (unsmoothed) per-point
array reserved for position-interpolation only — explicitly decoupled from the reported
`distance` total (CONV-05). Add a Vitest fixture suite; none exists for this domain today.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CONV-01 | First track point included in distance/bbox/centroid; planned route doesn't cut corners | Root-caused to `gpx.ts:125`'s `i = 1` loop start — see Common Pitfalls "Defect A" |
| CONV-02 | Centroid divides by the same point count it summed | Same root cause as CONV-01 — `gpx.ts:145` divides by `allPoints.length` (n) while the loop sums n−k points |
| CONV-03 | Points with no elevation excluded from gain/loss, not treated as 0m | `gpx-metrics-computation.ts:28-29,52` — `point.ele ?? 0` conflates "no data" with "sea level"; `Waypoint.ele` is genuinely `undefined` (not defaulted) when `<ele>` is absent |
| CONV-04 | Elevation gain/loss sampled independently of horizontal threshold | `gpx-metrics-computation.ts:63-82` — early `return` on `smoothedDistance < thresholdXY_m` gates all smoothed elevation logic behind horizontal movement |
| CONV-05 | Distance from smoothed accumulator; dead `cumulativeDistance` removed | `gpx.ts:142` uses `metrics.totalDistance` (raw) instead of `metrics.totalDistanceSmoothed`; `cumulativeDistance` is **not dead** — see Summary correction and Common Pitfalls |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| GPX parsing (XML → object model) | Frontend Server (SSR) / Browser (isomorphic) | — | `gpx.ts` runs both server-side (`+server.ts` upload/convert routes) and client-side (`trail/edit` import, route planner); `isomorphic-xml2js` makes this dual-context by design |
| Distance/elevation/centroid/bbox computation | Frontend Server (SSR) / Browser (isomorphic) | — | `GpxMetricsComputation` and `GPX.getTotals()` are pure functions called from both SvelteKit API routes (`+server.ts`) and browser-side stores (`valhalla_store.svelte.ts`) — same code path, no server/client divergence today |
| Trail persistence + Meilisearch indexing | API / Backend (SvelteKit route → PocketBase) | — | `trails_create()` (called from `/api/v1/trail/upload`) persists the already-computed `trail.distance`/`elevation_gain`/`elevation_loss` fields; no independent recomputation happens at the persistence layer |
| Route-crop UI (percentage-along-route → coordinate) | Browser (Client) | — | `updateCropMarkers()`/`getCoordinateAtDistance()` in `trail/edit/[id]/+page.svelte` run entirely client-side against `valhallaStore.route.features.cumulativeDistance` |
| Elevation profile chart rendering | Browser (Client) | — | `ElevationProfileControl` (vendored maplibre plugin) computes its **own** independent cumulative-distance array from raw GeoJSON coordinates (`elevationprofile.ts:965`) — confirmed NOT a consumer of the buggy `GPXFeature.cumulativeDistance`; out of this phase's blast radius |

## Package Legitimacy Audit

**Not applicable.** This phase modifies existing TypeScript logic in three already-installed,
already-imported files. No new npm packages are introduced. `isomorphic-xml2js`, `ngeohash`,
and other imports used by these files are pre-existing dependencies, unchanged by this phase.

## Standard Stack

No new dependencies. This phase is a pure logic fix within the existing stack:

| Tool | Version (installed) | Role in this phase |
|------|---------|--------------------|
| TypeScript | `^6.0.3` [VERIFIED: web/package.json] | Language for all three in-scope files |
| Vitest | `^4.1.9` [VERIFIED: web/package.json, `npx vitest --version` → `vitest/4.1.9`] | Test runner — no GPX-domain tests exist yet; this phase should add them |
| SvelteKit | `^2.68.0` [VERIFIED: web/package.json] | Hosts the `/api/v1/trail/upload` and `/api/v1/trail/convert` server routes that call the fixed code |

Note: CLAUDE.md's Technology Stack table lists TypeScript 5.9.3 / SvelteKit 2.60.1 — the
installed `web/package.json` versions above are newer (`^6.0.3` / `^2.68.0`). Treat the
installed versions as authoritative; CLAUDE.md's snapshot appears stale relative to current
`package.json`. [ASSUMED: not independently reconciled with a full dependency audit — flagged
for the planner, not blocking, since no new install is needed either way.]

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────────┐
                    │              Entry points (3)                │
                    ├─────────────────────────────────────────────┤
                    │ 1. PUT /api/v1/trail/upload  (web upload)     │
                    │ 2. POST /api/v1/trail/convert (no-persist)    │
                    │ 3. trail/edit/[id] import + route planner     │
                    │    (client-side, valhalla_store.svelte.ts)    │
                    └───────────────────┬───────────────────────────┘
                                         │  gpxString
                                         ▼
                              ┌────────────────────┐
                              │  GPX.parse()        │  gpx.ts
                              │  (xml2js → model)    │
                              └──────────┬───────────┘
                                         │  new GPX(...)
                                         ▼
                              ┌────────────────────────────┐
                              │  GPX constructor            │
                              │  → this.features =           │
                              │      this.getTotals()        │  gpx.ts:92
                              └──────────┬───────────────────┘
                                         │  per track → per segment → per point
                                         ▼
                    ┌───────────────────────────────────────────────┐
                    │  getTotals() loop  (gpx.ts:109-157)             │
                    │  ┌─────────────────────────────────────────┐  │
                    │  │ for (const track of this.trk)             │  │
                    │  │   for (const segment of track.trkseg)      │  │
                    │  │     allPoints.push(...points)   ← ALL pts  │  │
                    │  │     for (i = 1; i < pointLength; i++)      │  │
                    │  │       metrics.addAndFilter(points[i])      │  │◄─ DEFECT A: skips index 0
                    │  │       totalLat/Lon += points[i]            │  │   (CONV-01, CONV-02, and
                    │  │       min/maxLat/Lon from points[i]         │  │   route-planner corner-cut)
                    │  └─────────────────────────────────────────┘  │
                    └──────────────────────┬──────────────────────────┘
                                            │  point-by-point
                                            ▼
                    ┌───────────────────────────────────────────────┐
                    │  GpxMetricsComputation.addAndFilter()           │  gpx-metrics-computation.ts
                    │  ┌─────────────────────────────────────────┐  │
                    │  │ elevation = point.ele ?? 0        ◄──────┼──┼─ DEFECT B: missing ele → 0m
                    │  │ raw distance → totalDistance             │  │  (CONV-03)
                    │  │ smoothed distance (if >= 5m) →            │  │
                    │  │   totalDistanceSmoothed                   │  │
                    │  │ if (smoothedDistance < 5m) return  ◄──────┼──┼─ DEFECT B2: elevation gated
                    │  │   [elevation-diff code unreachable here]  │  │  behind horizontal threshold
                    │  └─────────────────────────────────────────┘  │  (CONV-04)
                    └──────────────────────┬──────────────────────────┘
                                            │  metrics.totalDistance (raw)
                                            │  metrics.totalDistanceSmoothed (filtered)
                                            ▼
                    ┌───────────────────────────────────────────────┐
                    │  gpx.ts:140-155 assembles GPXFeature             │
                    │  totalDistance = metrics.totalDistance  ◄────────┼─ DEFECT C: should read
                    │  (raw, jitter-inflated)                          │  .totalDistanceSmoothed
                    │  cumulativeDistance = metrics.cumulativeDistance │  (CONV-05)
                    │  (misaligned — see Defect A propagation)         │
                    └──────────────────────┬──────────────────────────┘
                                            │  trail.distance / .elevation_gain / .elevation_loss
                                            ▼
                    ┌───────────────────────────────────────────────┐
                    │  gpx2trail() (gpx_util.ts)                       │
                    │  → Trail object → trails_create() → PocketBase   │
                    │    + Meilisearch index (via /trail/upload)       │
                    └───────────────────────────────────────────────┘

  Separately, client-side only:
    valhallaStore.route (GPX instance, one trkseg PER planner leg)
       → getTotals() (same buggy loop) → features.cumulativeDistance
       → trail/edit/[id]/+page.svelte: updateCropMarkers() binary-searches
         cumulativeDistance to place the crop-range slider's start/end pins
         (LIVE consumer — not dead code, see Common Pitfalls)
```

### Recommended Approach (no new files needed)

This is a targeted logic fix inside three existing files — no new project structure is
required. The planner should sequence tasks by root cause, not by requirement ID, since
CONV-01/02 share one fix and CONV-03/04 are adjacent but independent fixes in the same
function:

1. **Fix the loop bound** (`gpx.ts:125`, `i = 1` → `i = 0`) — resolves CONV-01, CONV-02, and
   the route-planner corner-cut symptom in one change. Verify with a 2-point-segment fixture
   and a multi-anchor planner fixture (see Code Examples).
2. **Fix elevation "no data" handling** (`gpx-metrics-computation.ts:28-29,52`) — distinguish
   `point.ele === undefined` from `point.ele === 0`. Resolves CONV-03.
3. **Decouple elevation sampling from the horizontal threshold**
   (`gpx-metrics-computation.ts:63-82`) — resolves CONV-04. Do this as a distinct change from
   step 2 even though both touch elevation logic; they are independently testable and the
   planner's verification steps should treat them as separate fixture cases (partial-elevation
   track vs. steep-switchback track).
4. **Swap the reported distance source** (`gpx.ts:142`, `metrics.totalDistance` →
   `metrics.totalDistanceSmoothed`) and **rebuild `cumulativeDistance`** as an index-aligned,
   raw per-point array used only by the crop-slider's position interpolation — see Common
   Pitfalls for the exact compatibility requirement. Do this step LAST, after step 1's loop
   fix, since the array's alignment depends on the loop bound being correct first.

### Anti-Patterns to Avoid

- **Deleting `cumulativeDistance` outright:** Breaks the trail-edit crop slider
  (`updateCropMarkers`/`getCoordinateAtDistance`). The ROADMAP's "dead ... array is gone"
  language describes the *current broken/misaligned* array, not an unused one — confirmed by
  a real, wired UI consumer. Any deletion must ship a replacement the crop feature can use.
- **Rescaling `cumulativeDistance` to match the smoothed total instead of keeping it raw:**
  The crop slider's `getCoordinateAtDistance()` interpolates between adjacent points in
  `flatten()`'s point-index order. `totalDistanceSmoothed` skips points below the 5m
  threshold, so a smoothed cumulative array would NOT have one entry per point and could not
  be index-aligned with `flatten()`. Keep the position-interpolation array raw; keep the
  *reported* `distance` total smoothed. These are different needs.
- **Fixing the horizontal-threshold gate by removing the threshold from distance too:** The
  threshold correctly suppresses GPS-jitter-inflated distance (CONV-05's whole point is to use
  the smoothed, thresholded distance). Only the elevation-diff evaluation should stop being
  gated by it (CONV-04) — the distance-smoothing behavior it also gates should be left alone.

## Don't Hand-Roll

Not applicable to this phase — there is no library-replacement decision here. The fixes are
corrections to existing bespoke arithmetic (haversine distance, threshold-based smoothing),
which is intentionally hand-rolled already and out of scope to replace with a library in this
defect-fix phase (that would be a much larger, unrequested change).

## Common Pitfalls

### Pitfall 1: The `cumulativeDistance` array is not dead code (CONV-05 blast radius)

**What goes wrong:** Removing or reshaping `cumulativeDistance` without checking consumers
silently breaks the route-crop feature (`trail/edit/[id]/+page.svelte`'s `RouteEditor` crop
slider) — the crop-range pins will land on the wrong coordinates, or `getCoordinateAtDistance`
will throw/return `NaN` if the array becomes empty or wrongly shaped.

**Why it happens:** The ROADMAP's phrasing ("dead, misaligned array") is half right — the
array genuinely IS misaligned (confirmed: it is built from calls skipping index 0 of every
segment, per Defect A, while `flatten()` — the array `getCoordinateAtDistance` indexes
against — includes every point). But "misaligned" was conflated with "unused." It is used.

**How to avoid:** Before removing/reshaping the field, grep `cumulativeDistance` across
`web/src` (confirmed exhaustively during this research: `gpx.ts`, `gpx-metrics-computation.ts`,
and `trail/edit/[id]/+page.svelte` are the only real consumers — `list_panel.svelte`'s
identically-named local variable at lines 30/160 is an unrelated concept, the sum of whole
trails' `.distance` for a list, not this field; do not confuse the two). Provide a
correctly-aligned raw per-point cumulative array for `getCoordinateAtDistance` to keep
consuming, separate from the smoothed `distance` total used for display (see Open Questions
for the exact scaling interaction).

**Warning signs:** If `getCoordinateAtDistance`'s binary search in
`trail/edit/[id]/+page.svelte:1508-1533` starts returning coordinates that don't lie on the
route polyline, or the crop-range slider visibly snaps to the wrong point, the array's
alignment or scale has regressed.

### Pitfall 2: Fixing the loop bound changes `metrics`'s "first call" semantics across segment boundaries

**What goes wrong:** `GpxMetricsComputation.addAndFilter()` (gpx-metrics-computation.ts:23-31)
treats its very first-ever call specially (initializes anchors, computes no distance). Today,
because `gpx.ts`'s loop starts at `i = 1`, the first REAL call for any given segment is
`points[1]`, not `points[0]`. After fixing the loop to `i = 0`, the first call for the very
first segment of the very first track becomes `points[0]` (correct — this is the genuine
start of the whole route and should have no "distance from previous point" to report). But
for segment 2+, `points[0]` is now a normal (non-initializing) call, because `metrics`'s
internal `lastPointXY` still holds the previous segment's last point (the `metrics` instance
is shared across all segments in one `getTotals()` call — declared once at `gpx.ts:105`,
before the segment loop). This is the CORRECT behavior needed to fix the route-planner
corner-cut (Defect A / D) — flagging it here because it is easy to "fix" defensively by
resetting `lastPointXY` at each segment boundary, which would reintroduce the corner-cutting
bug in a new form (each segment would again start disconnected from the previous one).

**How to avoid:** Do NOT reset `metrics`'s internal anchors at segment boundaries. The fix is
purely `gpx.ts:125`'s loop bound; `gpx-metrics-computation.ts`'s cross-segment continuity is
already correct and must be preserved.

### Pitfall 3: `Waypoint.$.lat`/`$.lon` default to `-1`, not `undefined`, on missing input

**What goes wrong:** `web/src/lib/models/gpx/waypoint.ts:53-54` defaults missing/falsy
`lat`/`lon` to `-1` (`object.$.lat || object.lat || -1`), NOT `undefined`. This is different
from `ele`, which has no default and is genuinely `undefined` when absent (`waypoint.ts:55`,
`this.ele = object.ele;` — no `??` or `||` fallback). Any elevation-fix code that copies the
`lat`/`lon` "falsy-coalesce" pattern for elevation would reintroduce a phantom-value bug
identical to the one CONV-03 is fixing, just for coordinates instead of elevation. Keep the
CONV-03 fix scoped to `ele` only.

**How to avoid:** When writing the CONV-03 fix, explicitly test `point.ele === undefined`
(not `!point.ele`, which would also treat `ele === 0` as missing — undoing the exact
distinction CONV-03 requires).

### Pitfall 4: `getTotals()` divides by zero / produces `Infinity` bounds on a track with zero points

**What goes wrong:** Not one of the four named defects, but adjacent: if `allPoints.length`
is `0` (e.g., a GPX with a `<trk>` containing no `<trkseg>`, or empty segments), `centroid`
becomes `{ lat: NaN, lon: NaN }` and `boundingBox` stays at its `Infinity`/`-Infinity` sentinel
values (`gpx.ts:107`). This pre-exists the four defects and is not required by CONV-01..05,
but the planner should confirm the fix doesn't change this edge case's behavior unexpectedly
(e.g., if the loop-bound fix causes a previously-skipped empty-segment edge case to behave
differently). Recommend a fixture test asserting current behavior is preserved (not silently
changed) for a zero-point track, even if "fixing" it is out of this phase's stated scope.

## Code Examples

### Recommended fixture shapes (no existing GPX fixtures in the repo — confirmed via repo-wide
search; `find . -iname "*.gpx"` returns nothing under `web/`)

```typescript
// Source: derived from gpx.ts / gpx-metrics-computation.ts read during this research pass.
// File: web/src/lib/models/gpx/gpx.test.ts (new — no existing GPX test file)
import { describe, expect, it } from "vitest";
import GPX from "./gpx";

describe("GPX.getTotals — CONV-01/02 (off-by-one loop)", () => {
  it("reports non-zero distance for a 2-point segment", () => {
    const gpx = new GPX({
      trk: [{
        trkseg: [{
          trkpt: [
            { $: { lat: 47.0, lon: 11.0 } },
            { $: { lat: 47.001, lon: 11.001 } }, // ~130m away
          ],
        }],
      }],
    });
    expect(gpx.features.distance).toBeGreaterThan(0);
  });

  it("centroid and bbox include the segment's first point", () => {
    const gpx = new GPX({
      trk: [{
        trkseg: [{
          trkpt: [
            { $: { lat: 40.0, lon: 10.0 } }, // extreme point — must not be dropped
            { $: { lat: 47.0, lon: 11.0 } },
            { $: { lat: 48.0, lon: 12.0 } },
          ],
        }],
      }],
    });
    expect(gpx.features.boundingBox.minLat).toBe(40.0);
    // centroid must average all 3 points, not 2
    expect(gpx.features.centroid.lat).toBeCloseTo((40.0 + 47.0 + 48.0) / 3, 5);
  });
});

describe("GpxMetricsComputation — CONV-03 (missing elevation)", () => {
  it("does not report a phantom drop to sea level for untagged points", () => {
    // points alternate real elevation / no <ele> tag / real elevation again
    // assert totalElevationLoss stays 0 (or near-0) across the gap, not ~1800m
  });
});

describe("GpxMetricsComputation — CONV-04 (steep, low-horizontal-movement stretch)", () => {
  it("measures elevation gain across a switchback with <5m horizontal steps", () => {
    // consecutive points each move <5m horizontally but climb steadily;
    // assert totalElevationGainSmoothed reflects the real climb, not 0
  });
});

describe("GPX.getTotals — CONV-05 (smoothed vs raw distance)", () => {
  it("uses the smoothed accumulator, not the jittery raw sum", () => {
    // a track with GPS jitter (small back-and-forth sub-threshold movements)
    // interleaved with real forward movement; assert reported distance is
    // close to the real forward distance, not inflated by the jitter
  });
});
```

```typescript
// Route planner corner-cutting fixture (Defect D), mirrors valhalla_store.svelte.ts's
// insertIntoRoute() pattern of one TrackSegment per anchor-to-anchor leg.
// Source: web/src/lib/stores/valhalla_store.svelte.ts:112-128 (read during this research pass)
import Track from "$lib/models/gpx/track";
import TrackSegment from "$lib/models/gpx/track-segment";
import GPX from "$lib/models/gpx/gpx";

const route = new GPX({
  trk: [new Track({
    trkseg: [
      new TrackSegment({ trkpt: [/* leg 1: anchor A -> via points -> anchor B */] }),
      new TrackSegment({ trkpt: [/* leg 2: anchor B -> via points -> anchor C */] }),
    ],
  })],
});
// Before the fix: distance silently skips the anchor-B hop between legs.
// After the fix: features.distance follows the full routed polyline through anchor B.
```

## State of the Art

Not applicable — this is a targeted bugfix in bespoke code, not a library/framework
currency question. No "old approach vs. current approach" axis exists here.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Installed TypeScript (`^6.0.3`) / SvelteKit (`^2.68.0`) versions in `web/package.json` are authoritative over CLAUDE.md's listed 5.9.3 / 2.60.1 | Standard Stack | Low — no version-specific behavior is relied on by the fix; flagged only for planner awareness, not load-bearing |

**All other claims in this research were verified by direct file reads (`gpx.ts`,
`gpx-metrics-computation.ts`, `gpx_util.ts`, `waypoint.ts`, `utils.ts`,
`valhalla_store.svelte.ts`, `trail/edit/[id]/+page.svelte`, `trail/upload/+server.ts`,
`trail/convert/+server.ts`, `map_with_elevation_maplibre.svelte`,
`elevationprofile.ts`) or by repo-wide grep during this session — no external/training-data
claims about this codebase's own logic were needed.**

## Open Questions

1. **What should `cumulativeDistance` (or its replacement) contain after the CONV-05 fix?**
   - What we know: it must stay index-aligned with `flatten()` (one entry per point, in
     order) for `getCoordinateAtDistance`'s binary search + linear interpolation to keep
     working; the reported `distance` total will become the smoothed accumulator, which is
     NOT per-point-aligned (it skips sub-threshold points).
   - What's unclear: whether the crop slider's percentage math (`features.distance * (start /
     100)`) should be rescaled to use the raw array's own total (`cumulative[cumulative.length
     - 1]`) as its 100% reference, decoupling it from the reported smoothed total, or whether
     some other reconciliation is preferred.
   - Recommendation: rescale the crop slider to use the raw cumulative array's own last entry
     as its reference total (Anti-Patterns option 1 in Architecture Patterns above) — smallest
     change, keeps the two concerns (reported distance vs. position interpolation) cleanly
     separated. Surface this explicitly as a plan-time decision rather than assuming.

2. **Should the zero-point / empty-segment edge case (Pitfall 4) be explicitly asserted as
   unchanged, or is it fully out of scope?**
   - What we know: it's not one of CONV-01..05, and the ROADMAP scope note doesn't mention it.
   - What's unclear: whether the loop-bound fix (Defect A) could inadvertently change this
     edge case's output (e.g., `NaN` centroid) as a side effect.
   - Recommendation: add one fixture asserting current (unchanged) empty-track behavior as a
     regression guard, without attempting to "fix" the `NaN`/`Infinity` sentinel behavior
     itself — that's a separate, unrequested defect.

3. **Backward compatibility: trails already saved with the buggy metrics.**
   - What we know: `CONV-F01` ("Backfill corrected metrics onto trails saved before v1.8") is
     explicitly a **Future Requirement**, deferred with the stated reason "no migration path
     exists today, and there is no meaningful install base to migrate" (REQUIREMENTS.md).
     `PUT /trail/form` stores client-submitted values and never recomputes server-side
     (confirmed via REQUIREMENTS.md's Out of Scope table: "Migrating already-saved trails to
     corrected metrics" — reason: "app is pre-production").
   - What's unclear: nothing — this is a resolved, explicit decision already recorded in
     REQUIREMENTS.md and ROADMAP.md, not an open question for this phase's planner. Included
     here only so the planner doesn't accidentally scope-creep into a migration task.
   - Recommendation: no migration/backfill task in this phase's plan. Existing stored trails
     keep their pre-fix values until re-saved or re-imported; this is accepted risk per the
     already-recorded product decision.

## Runtime State Inventory

Not applicable — this phase is a code-only defect fix in existing TypeScript. It is not a
rename/refactor/migration phase (no strings, IDs, or identifiers change), so the Runtime
State Inventory trigger does not apply. No stored data, live service config, OS-registered
state, secrets, or build artifacts reference anything renamed by this phase.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Running Vitest / SvelteKit dev | ✓ [VERIFIED: `npx vitest` ran successfully] | v22.12.0 | — |
| Vitest | New unit tests for the four defects | ✓ [VERIFIED: `npx vitest --version` → 4.1.9] | 4.1.9 | — |
| npm | Dependency resolution (no new deps needed) | ✓ (implied by successful `npx` invocation) | — | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None — no new external dependency is introduced by
this phase.

## Security Domain

`security_enforcement` is `true` in `.planning/config.json` with `security_asvs_level: 1`.
This phase touches pure arithmetic/data-transformation code with no new network boundary,
no new authentication/authorization surface, and no new persistence write path — the existing
`/api/v1/trail/upload` and `/api/v1/trail/convert` routes already parse untrusted user-supplied
GPX/XML content via `isomorphic-xml2js`; this phase does not change that parsing step, only
the arithmetic performed on already-parsed numeric fields.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Unchanged — routes already gate on `event.locals.user`/PocketBase auth where relevant, not touched by this phase |
| V3 Session Management | No | Unaffected |
| V4 Access Control | No | Unaffected |
| V5 Input Validation | Partial | Existing `Waypoint`/`GPX` parsing already coerces malformed numeric input (`Number.isInteger`/`parseFloat` in `GPX.parse`, `gpx.ts:227-232`); this phase must not weaken that — new elevation "undefined vs 0" handling must not introduce `NaN` propagation into `distance`/`elevation_gain`/`elevation_loss` (guard with `Number.isFinite` checks where the fix touches numeric branches, matching the existing `Number.isFinite(bbox.minLat)` pattern already used at `gpx_util.ts:82`) |
| V6 Cryptography | No | Unaffected |

### Known Threat Patterns for this change

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed/adversarial GPX causing `NaN`/`Infinity` to reach `trail.distance` etc., corrupting duplicate-detection thresholds in `findDuplicate()` (`trail/upload/+server.ts:119-140`) | Tampering (data integrity) | Preserve/extend existing `Number.isFinite` guarding when reworking the elevation and distance accumulation branches; do not let an untagged-elevation or zero-point edge case produce non-finite values that silently pass PocketBase's numeric field validation |

## Sources

### Primary (HIGH confidence — direct file reads during this research session)
- `web/src/lib/models/gpx/gpx.ts` (full file, 304 lines) — `getTotals()`, `GPXFeature` type, `correctElevation()`
- `web/src/lib/models/gpx/gpx-metrics-computation.ts` (full file, 87 lines) — `addAndFilter()`
- `web/src/lib/util/gpx_util.ts` (full file, 420 lines) — `gpx2trail()`, `fromFile()`, format converters
- `web/src/lib/models/gpx/waypoint.ts` — `Waypoint.$`/`.ele` field defaulting behavior
- `web/src/lib/models/gpx/utils.ts` — `haversineDistance()` implementation
- `web/src/lib/stores/valhalla_store.svelte.ts` (full file, 301 lines) — route planner's per-leg `TrackSegment` construction (`insertIntoRoute`)
- `web/src/routes/trail/edit/[id]/+page.svelte` (targeted reads, lines 1420-1560, 2400-2440) — `updateCropMarkers`, `getCoordinateAtDistance`, `updateTotals`, `RouteEditor` wiring
- `web/src/routes/api/v1/trail/upload/+server.ts`, `web/src/routes/api/v1/trail/convert/+server.ts` — both entry points confirmed to call `gpx2trail()`
- `web/src/lib/components/trail/map_with_elevation_maplibre.svelte` and `web/src/lib/vendor/maplibre-elevation-profile/elevationprofile.ts` — confirmed elevation profile chart computes its own independent distance array, not a consumer of the buggy field
- `web/src/lib/components/list/list_panel.svelte` — confirmed its `cumulativeDistance` local variable is an unrelated concept (list-level trail-distance sum)
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` (Phase 32-34 sections), `.planning/STATE.md` — project scope, decisions, and traceability
- `web/package.json` — installed dependency versions
- Repo-wide grep confirming zero existing `*.gpx` fixture files and zero existing `gpx*.test.ts` files under `web/src`

### Secondary (MEDIUM confidence)
None used — all findings verified directly against source.

### Tertiary (LOW confidence)
None used.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies; versions read directly from `package.json`
- Architecture: HIGH - every code path traced by direct file read, including cross-file consumer discovery (crop slider, elevation profile chart)
- Pitfalls: HIGH - the `cumulativeDistance` "not actually dead" finding is a direct, verified correction to the ROADMAP's own framing, based on reading the actual consumer code

**Research date:** 2026-07-31
**Valid until:** No expiry pressure — this is a static defect-fix domain, not a fast-moving external dependency. Re-verify only if the three in-scope files change before planning executes.
