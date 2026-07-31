---
phase: 33-conversion-correctness
reviewed: 2026-07-31T12:45:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - web/src/lib/models/gpx/gpx.ts
  - web/src/lib/models/gpx/gpx.test.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.ts
  - web/src/lib/models/gpx/gpx-metrics-computation.test.ts
  - web/src/routes/trail/edit/[id]/+page.svelte
findings:
  critical: 3
  warning: 9
  info: 6
  total: 18
status: issues_found
---

# Phase 33: Code Review Report

**Reviewed:** 2026-07-31T12:45:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Phase 33 fixes five real defects (CONV-01..CONV-05) and the fixes it claims to make are
present and test-covered. The 21 shipped tests pass. That is where the good news ends.

Every finding below was **executed**, not inferred: I built a scratch vitest probe against the
shipped code, captured real output, and deleted it (working tree is clean). Three defects are
blockers:

1. The crop slider produces `NaN` marker coordinates on routes whose first two trackpoints are
   coincident (a very common GPS artefact, and unconditionally true for a zero-length route).
   The new `cumulativeRoute.length < 2 || !Number.isFinite(rawRouteTotal)` guard does **not**
   catch it — it guards the wrong two conditions. MapLibre's `LngLat` constructor throws on
   `NaN`, so the crop UI dies with an uncaught exception. This is precisely the case the phase
   deferred to end-of-phase human verification ("including 0%/100%, with no NaN totals");
   it fails.
2. CONV-04 removed the horizontal-displacement gate from elevation smoothing **entirely**
   rather than replacing it with something noise-aware. A stationary 60-sample track with
   ±7 m altitude jitter — a lunch break, a paused recording, a phone on a table — now reports
   **210 m of gain and 203 m of loss over 0 m of travel**. Before this phase it reported 0/0.
   That number is written straight into `trail.elevation_gain`.
3. The new early `return` in `updateCropMarkers()` leaves `croppedGPX` holding the previous
   route. `confirmCrop()` does not re-validate, so confirming a crop can silently resurrect a
   route the user already replaced.

Beyond the blockers, the CONV-05 raw→smoothed swap is systematically biased low (chord-cutting
on dense/curvy geometry, plus an always-dropped trailing residual) and it has fractured the
editor into three surfaces that now report three different distances for the same route.
CONV-03's `parseElevation` is documented as "the single coercion point" but is not applied at
the GeoJSON boundary, so the elevation *profile chart* still draws the sea-level plunge that
CONV-03 removed from the *number* — the two disagree on screen.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01 (BLOCKER): Crop markers become `NaN` when the route starts with coincident points — uncaught MapLibre throw

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1478-1485`, `web/src/routes/trail/edit/[id]/+page.svelte:1537-1540`

**Issue:**
D-01 made `cumulativeDistance[0] === 0` by design. `getCoordinateAtDistance()` then computes
`ratio = (target - prevDist) / (nextDist - prevDist)`. When `target === 0` (the slider fires
`onupdate` with `[0, 100]` on mount — `double_slider.svelte:42`) the binary search lands on
`i = Math.max(1, 0) = 1`, so `prevDist = cumulative[0]` and `nextDist = cumulative[1]`. If the
route's first two points are coincident, `cumulative[1]` is also `0` and the ratio is `0 / 0`
= `NaN`. Both returned coordinates are `NaN`.

The guard added by this phase does not cover it. Measured against the shipped code:

```
P1  cumulative: [0, 0, 111.19, 222.39]   total 222.39   guard passes: true
    coord at 0%: [NaN, NaN, 1]
P2  cumulative: [0, 0, 0]                total 0        guard passes: true
    coord at 0%: [NaN, NaN, 1]
```

`cropStartMarker.setLngLat([NaN, NaN])` reaches `LngLat.convert`, which throws
`Invalid LngLat object: (NaN, NaN)` (`node_modules/maplibre-gl/dist/maplibre-gl.js`). Nothing
catches it, so `updateCropMarkers` aborts mid-flight — `croppedGPX` is never assigned and
`updateTotals` never runs — on every slider movement.

Coincident leading points are not exotic: GPS loggers emit repeated identical fixes while the
device is stationary at the trailhead, and a fully degenerate route (P2) hits it for *any*
target. Note also that `!Number.isFinite(rawRouteTotal)` is effectively dead code —
`totalDistance` only ever accumulates values that already passed
`Number.isFinite(distance)` (`gpx-metrics-computation.ts:80-82`), so the last entry is always
finite. The guard checks a condition that cannot occur while missing the one that does.

Pre-phase this produced a *wrongly placed* marker (negative ratio extrapolating backwards), not
a crash — so this is a regression in failure mode introduced by the D-01 leading-zero entry.

**Fix:** guard on a usable interpolation basis, and make the interpolation itself degenerate-safe.

```ts
// +page.svelte, replacing the length/isFinite guard
const cumulativeRoute = valhallaStore.route.features.cumulativeDistance;
const rawRouteTotal = cumulativeRoute[cumulativeRoute.length - 1];

if (cumulativeRoute.length < 2 || !(rawRouteTotal > 0)) {
    // No positive interpolation basis: empty, single-point, or zero-length route.
    croppedGPX = null;               // see CR-03
    toggleCropMarkers(false);        // don't strand pins at [0, 0] — see WR-05
    return;
}
```

```ts
// getCoordinateAtDistance(), guarding the zero-length span
const span = nextDist - prevDist;
const ratio = span > 0 ? (target - prevDist) / span : 0;
```

Add regression tests for both fixtures above (leading duplicate point; all-identical points).

---

### CR-02 (BLOCKER): CONV-04 fabricates hundreds of metres of elevation gain on stationary GPS noise

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:107-128`

**Issue:**
The pre-phase code evaluated smoothed elevation *only* for points that had already cleared the
5 m horizontal threshold (`if (smoothedDistance < this.thresholdXY_m) return;` guarded the
elevation block). CONV-04 correctly identified that this loses genuine steep/low-horizontal
climbs — but the fix removes the horizontal criterion outright instead of replacing it with a
noise-tolerant one. Elevation is now diffed against `lastFilteredZ` on **every** sample, and
the anchor re-arms on **every** ±5 m excursion.

Consumer GPS/barometric altitude noise is routinely ±5–15 m. Measured against the shipped code
with a completely stationary track (identical lat/lon for all 60 samples, altitude alternating
1000 / 1007 m — well inside normal noise):

```
P3  gain: 210   loss: 203   distance: 0
```

210 m of climb over zero metres of travel. Pre-phase this fixture returned 0/0. `getTotals()`
publishes these as `features.elevationGain` / `elevationLoss` (`gpx.ts:142-143, 158-159`),
which `updateTotals()` writes to `$formData.elevation_gain` / `elevation_loss`
(`+page.svelte:1569-1570`) and `gpx_util.ts:74-75` writes to the persisted trail. Because
CONV-F01 (migration) is deferred, these wrong values are permanent once saved.

The shipped test suite cannot catch this: the only CONV-04 fixture
(`gpx-metrics-computation.test.ts:88-116`) is a monotonic climb, which is exactly the case where
removing the gate is safe. There is no oscillating/stationary fixture anywhere in the phase.

**Fix:** keep the CONV-04 win (elevation must not require horizontal travel) but restore
noise rejection with hysteresis on the *direction* of travel, so an out-and-back excursion
inside the noise band cannot ratchet:

```ts
// Track the running extremum since the last committed anchor; only commit a
// gain/loss once the elevation has moved thresholdZ_m *and* reversed by less
// than thresholdZ_m from the extremum (classic ZigZag / hysteresis filter).
if (elevation !== undefined) {
  if (this.lastFilteredZ === null) {
    this.lastFilteredZ = elevation;
    this.pendingExtremum = elevation;
    this.pendingDirection = 0;
  } else {
    const diff = elevation - this.lastFilteredZ;
    const direction = Math.sign(diff);
    if (Math.abs(diff) >= this.thresholdZ_m) {
      if (direction === this.pendingDirection || this.pendingDirection === 0) {
        // sustained move in one direction — commit it
        if (diff > 0) this.totalElevationGainSmoothed += diff;
        else this.totalElevationLossSmoothed -= diff;
        this.lastFilteredZ = elevation;
        this.pendingDirection = direction;
      } else {
        // reversal: re-anchor without crediting the reversal itself
        this.lastFilteredZ = elevation;
        this.pendingDirection = direction;
      }
    }
  }
}
```

Whatever filter is chosen, the phase must add a stationary-noise regression fixture (60
samples, fixed lat/lon, ±7 m alternation) asserting `elevationGain === 0`, alongside the
existing monotonic-climb fixture asserting `88`.

---

### CR-03 (BLOCKER): Stale `croppedGPX` after the new early return can silently restore a discarded route

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1480-1485`, `web/src/routes/trail/edit/[id]/+page.svelte:1513-1521`

**Issue:**
`croppedGPX` is assigned only at `:1504` and initialised to `null` only at its declaration
(`:166`); nothing ever resets it — not `resetRoute()`, not `replaceRoute()`, not
`toggleCropMarkers(false)`. The `return` introduced at `:1484` therefore leaves the previous
route's crop in `croppedGPX`, and `confirmCrop()` (`:1513`) only checks `if (!croppedGPX)`
before calling `setRoute(croppedGPX, true)`.

Reachable sequence, all through the normal UI (the crop button in `route_editor.svelte:405-417`
is not gated on `routeHasTrackPoints()`):

1. Open the crop panel on route A, drag the slider → `croppedGPX` = crop of A.
2. Close crop, `replaceRoute()` / draw a new short or empty route B.
3. Open the crop panel again → `DoubleSlider` fires `onupdate([0, 100])` on mount →
   `updateCropMarkers` hits the new guard and returns → `croppedGPX` is *still* the crop of A.
4. Click "crop" → `confirmCrop()` → `setRoute(crop-of-A)` → route B is destroyed and replaced
   by a stale crop of a route the user already discarded.

This is a data-loss path created by this phase's change; before the early return existed,
`croppedGPX` was always recomputed from the current route on every slider event.

**Fix:** clear the derived state on every non-productive exit, and re-derive rather than trust
the cache on confirm.

```ts
if (cumulativeRoute.length < 2 || !(rawRouteTotal > 0)) {
    croppedGPX = null;
    return;
}
```

Additionally reset `croppedGPX = null` in `resetRoute()`, `replaceRoute()`, and
`toggleCropMarkers(false)` so the cache cannot outlive the route it was derived from.

---

## Warnings

### WR-01: Reported distance permanently drops the trailing sub-threshold residual; short routes report exactly 0

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:125-128`, `web/src/lib/models/gpx/gpx.ts:144`

**Issue:** `totalDistanceSmoothed` only accumulates when a hop clears `thresholdXY_m`; the
final partial span from `lastFilteredPointXY` to the last point is never added. Since CONV-05
made this the *reported* distance, every trail now under-reports by up to 5 m, and any route
shorter than the threshold reports zero. Measured:

```
P5  raw: [0, 2.224]                  reported: 0        <- a 2.2 m route reports 0 m
P7  raw: [0, 111.195, 113.419]       reported: 111.195  <- trailing 2.22 m silently dropped
```

A user who draws two anchors 3 m apart sees "0 km" and cannot tell the editor is working.

**Fix:** flush the residual when reading the total.

```ts
distanceTotal(): number {
  if (!this.lastFilteredPointXY || !this.lastPointXY) return this.totalDistanceSmoothed;
  const residual = haversineDistance(
    this.lastFilteredPointXY.$.lat, this.lastFilteredPointXY.$.lon,
    this.lastPointXY.$.lat, this.lastPointXY.$.lon,
  );
  return this.totalDistanceSmoothed + (Number.isFinite(residual) ? residual : 0);
}
```

and use it at `gpx.ts:144`. Add a test asserting a 2-point 2.2 m route reports ~2.2 m, not 0.

---

### WR-02: CONV-05 chord-cuts curves — reported distance is systematically short on dense/switchback geometry

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:73-78`, `web/src/lib/models/gpx/gpx.ts:144`

**Issue:** `smoothedDistance` is the straight-line distance from the last *anchor*, not the sum
of the intermediate hops, so any curve sampled finer than 5 m is replaced by its chord. Valhalla
polylines and 1 Hz recordings both sample switchbacks at 1–3 m. Measured on a 30 m-radius circle
sampled every ~2 m:

```
P4  true circumference: 188.496   raw: 188.253   reported (smoothed): 186.125
```

~1.3 % short on a gentle circle; the error grows as turn radius shrinks relative to the
threshold (a tight hairpin can lose a third of its arc). Both CONV-05 tests
(`gpx.test.ts:117-138`, `:207-226`) use collinear north-south points, where chord == arc, so the
bias is structurally invisible to the suite.

**Fix:** accumulate the *path* length between anchors rather than the chord, i.e. keep a
running sum of raw hops and commit that sum when the anchor displacement clears the threshold:

```ts
this.pendingPathLength += distance;                 // raw hop, already computed above
if (smoothedDistance >= this.thresholdXY_m) {
  this.totalDistanceSmoothed += this.pendingPathLength;
  this.pendingPathLength = 0;
  this.lastFilteredPointXY = point;
}
```

This preserves the jitter rejection the existing tests assert (jitter never clears the anchor
threshold, so the out-and-back cancels only if you also reset `pendingPathLength` on
non-committing spans — pick one semantic and test both fixtures) while ending the chord bias.
At minimum, add a curved-geometry fixture so the trade-off is pinned by a test.

---

### WR-03: A single `GpxMetricsComputation` spans `trk` boundaries — disjoint tracks accrue a phantom connecting leg

**File:** `web/src/lib/models/gpx/gpx.ts:106`, `web/src/lib/models/gpx/gpx.ts:111-140`

**Issue:** `metrics` is constructed once outside both loops, so the last point of track *n* is
the anchor for the first point of track *n+1*. Multiple `<trk>` elements in one file mean
genuinely separate activities (that is what the container is for), not one continuous polyline.
Measured with two 111 m tracks ~134 km apart:

```
P6  reported distance: 134319.75 m   cumulative: [0, 111.19, 134208.55, 134319.75]
```

134 km of travel that never happened, plus a `cumulativeDistance` array whose interpolation
basis is dominated by the gap (so the crop slider's 50 % lands in the middle of the void).

CONV-01 did not introduce this — the `i = 1` bound spanned the gap too — but the phase edited
exactly this loop, added tests that lock in cross-*segment* concatenation as intended
(`gpx.test.ts:58-103`), and never distinguished the segment case (planner legs, correctly
continuous) from the track case (not continuous).

**Fix:** keep one accumulator per `trk` and sum the per-track results; segments stay
concatenated within a track. Concatenate the per-track `cumulativeDistance` arrays with each
track re-based on the running total so index alignment with `flatten()` is preserved. Add a
two-disjoint-track fixture asserting `distance ≈ 222 m`, not 134 km.

---

### WR-04: Crop start is off by one point — the marker and the crop disagree at 0 %

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1537-1549`, `web/src/routes/trail/edit/[id]/+page.svelte:1504-1508`

**Issue:** `getCoordinateAtDistance` returns `i` — the index of the point *after* the target —
while interpolating the coordinate between `points[i-1]` and `points[i]`. At `target = 0` the
returned coordinate is exactly `points[0]` but the returned index is `1`, and `cropGPX()`
(`lib/util/gpx_util.ts:389-396`) starts collecting at the identity-matched `start` point. So a
crop at 0 % draws the pin on the first trackpoint but silently discards it, and the cropped
route begins one point late. The same asymmetry means the reported crop totals never match what
the pins show.

**Fix:** return the index of the point the interpolated coordinate is closest to (or the floor
index, and let `cropGPX` include it):

```ts
const startIndex = ratio > 0 ? i : i - 1;   // for the start pin
```

Return `[lon, lat, i - 1, i, ratio]` and let the caller pick floor (start) vs. ceil (end).

---

### WR-05: Degenerate routes strand two crop pins at Null Island

**File:** `web/src/routes/trail/edit/[id]/+page.svelte:1439-1467`, `web/src/routes/trail/edit/[id]/+page.svelte:1480-1485`

**Issue:** The markers are created and `setLngLat([0, 0]).addTo(map!)` runs *before* the new
guard. On an empty/degenerate route the guard returns immediately, leaving two visible pins at
0°N 0°E (`toggleCropMarkers(true)` has already set opacity to `1`), plus a `map!` non-null
assertion that will throw if the crop panel is somehow opened before the map initialises.

**Fix:** move marker creation after the guard, and call `toggleCropMarkers(false)` (or skip
`addTo`) on the degenerate path. Replace `map!` with an explicit `if (!map) return;`.

---

### WR-06: `parseElevation` is not applied at the GeoJSON boundary — the profile chart still draws the sea-level plunge CONV-03 removed

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:3-14` (docstring claim), `web/src/lib/models/gpx/track-segment.ts:22-26`, `web/src/lib/models/gpx/waypoint.ts:87`, `web/src/lib/models/gpx/route.ts:48`

**Issue:** The docstring calls `parseElevation` "the single coercion point", but it is only
called inside `GpxMetricsComputation`. The `toGeoJSON` paths still emit `pt.ele ?? 0` verbatim,
and `ele` is a **string** at rest for every XML-parsed GPX (the phase's own test documents this
at `gpx-metrics-computation.test.ts:69`). Measured:

```
P8  GeoJSON coordinates: [[11,47,"1000"],[11,47.001,""]]
```

That is not valid GeoJSON (altitude must be a number), and it feeds
`smoothElevations()` (`lib/vendor/maplibre-elevation-profile/tools.ts:42`), whose
`sum + elevation * weights[idx]` coerces `""` to **0**. So the elevation *profile chart* still
renders the fabricated plunge to sea level for a missing `<ele>`, while the *number* next to it
now correctly reports 15 m of gain. The two disagree on screen, and CONV-03 is only half done.

**Fix:** coerce at the model boundary so every consumer sees numbers or `undefined`:

```ts
// track-segment.ts / waypoint.ts / route.ts
import { parseElevation } from './gpx-metrics-computation';   // or move it to utils.ts
const coordinates = (this.trkpt || []).map(pt => {
  const ele = parseElevation(pt.ele);
  return ele === undefined ? [pt.$.lon ?? 0, pt.$.lat ?? 0] : [pt.$.lon ?? 0, pt.$.lat ?? 0, ele];
});
```

Better still: coerce once in the `Waypoint` constructor (`this.ele = parseElevation(object.ele)`)
so the declared type `ele?: number` stops being a lie. Note `correctElevation` already assigns
numbers, so `ele` is a `string | number | undefined` union in practice — exactly the ambiguity
`parseElevation` was introduced to eliminate.

---

### WR-07: The editor now shows three different distances for the same route

**File:** `web/src/lib/models/gpx/gpx.ts:144`, `web/src/lib/components/trail/trail_anchor_list.svelte:221`, `web/src/lib/components/trail/trail_anchor_list.svelte:242-244`, `web/src/lib/vendor/maplibre-elevation-profile/elevationprofile.ts:915-917`

**Issue:** CONV-05 changed the reported total to smoothed, but two sibling surfaces in the same
screen were not aligned:

- `trail_anchor_list.svelte:243` still carries a verbatim copy of the CONV-01 bug
  (`for (let i = 1; i < points.length; i++)`) and reads `metrics.totalDistance` (raw) at `:221`.
  Its per-anchor distances therefore drop each segment's first hop *and* use a different
  smoothing basis, so they cannot sum to the form's total.
- The elevation profile computes its own `haversineCumulatedDistanceWgs84` (raw) for the x-axis,
  so the chart's end-of-route distance exceeds the form's by the smoothing loss (WR-01 + WR-02).

`33-03-SUMMARY.md:105` self-flags the anchor-list copy as a carried risk, which is honest — but
it is shipping a user-visible arithmetic contradiction in the same view, and the "sum of the
legs ≠ the whole" symptom is exactly the bug report class this phase exists to eliminate.

**Fix:** fix the `i = 1` bound and switch to `totalDistanceSmoothed` in
`trail_anchor_list.svelte` in this phase (it is a two-line change against code the phase already
proved correct), or explicitly label the anchor-list column as raw. Extract the shared
"accumulate a GPX's metrics" loop into one function so a third copy cannot appear.

---

### WR-08: `correctElevation` throws a `TypeError` when Valhalla omits `height`, and silently blanks trailing points on a short response

**File:** `web/src/lib/models/gpx/gpx.ts:203-222`

**Issue:** (Pre-existing; unchanged by this phase but inside a reviewed file, and directly
upstream of the metrics this phase rewrote.)

- `heights` defensively defaults to `[]` at `:204`, but the write loop at `:218` dereferences
  `heightResponse.height[heightIndex]` — the *undefaulted* field. If the endpoint returns
  `{}`, `heights.length === 0` skips the `:207` guard and `:218` throws
  `Cannot read properties of undefined (reading '0')`, unhandled, mid-mutation.
- There is no length check between `heightResponse.height` and the point count. A short array
  assigns `undefined` to every trailing point, wiping existing elevation data for the tail of
  the track with no error surfaced. A long array is silently ignored.

**Fix:**

```ts
const heights = heightResponse.height ?? [];
const pointCount = coordinates.length;
if (heights.length !== pointCount) {
  throw new APIError(502, "elevation-response-length-mismatch",
    `expected ${pointCount} heights, got ${heights.length}`);
}
if (heights.every((h) => !Number.isFinite(h))) return;
// ...then assign from `heights`, never from `heightResponse.height`
```

---

### WR-09: Malformed / non-GPX uploads surface as `Cannot read properties of undefined (reading 'gpx')`

**File:** `web/src/lib/models/gpx/gpx.ts:229-254`

**Issue:** (Pre-existing; inside a reviewed file.) The `parseString` callback assigns
`error = err` and then *immediately* constructs `new GPX({ $: xml.gpx.$, ... })` without
checking `err` or the existence of `xml.gpx`. The `if (error) throw error` at `:250` is
unreachable for any input that actually fails to parse. Verified against the shipped code:

```
GPX.parse("<gpx><trk></gpx>")  ->  TypeError: Cannot read properties of undefined (reading 'gpx')
```

User-supplied files are the input here, so this is missing input validation on an untrusted
path: the real parse diagnostic is discarded and the user gets an internal `TypeError`.

**Fix:**

```ts
xml2js.parseString(sanitizedGPX, opts, (err, xml) => {
  if (err) { error = err; return; }
  if (!xml?.gpx) { error = new Error("not-a-gpx-document"); return; }
  data = new GPX({ $: xml.gpx.$, metadata: xml.gpx.metadata, wpt: xml.gpx.wpt, rte: xml.gpx.rte, trk: xml.gpx.trk });
});
if (error) throw error;
if (!data) throw new Error("gpx-parse-produced-no-data");
```

---

## Info

### IN-01: Dead raw accumulators and an unreachable anchor branch

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:33-34`, `web/src/lib/models/gpx/gpx-metrics-computation.ts:90-105`

`totalElevationGain` / `totalElevationLoss` (raw) are computed on every point but read nowhere
in production — `getTotals()` uses only the smoothed pair (`gpx.ts:142-143`) and
`trail_anchor_list.svelte:222-223` likewise. CONV-03 added a `lastZ === null` anchor branch for
this dead path, so that logic ships untested and unused. Either delete the raw elevation
accumulators or add an assertion that consumes them; carrying two parallel implementations of
the same rule is how the smoothed one drifts.

### IN-02: `summedPointCount` is provably equal to `allPoints.length`, and nothing reads `centroid`

**File:** `web/src/lib/models/gpx/gpx.ts:104`, `web/src/lib/models/gpx/gpx.ts:114`, `web/src/lib/models/gpx/gpx.ts:132`, `web/src/lib/models/gpx/gpx.ts:147`

After CONV-01 fixed the loop bound, `summedPointCount++` runs exactly once per
`allPoints.push(...points)` entry, so the two counters cannot diverge — CONV-02's separate
counter is now redundant bookkeeping. More to the point, `features.centroid` has **no**
production consumer (grep finds references only in `gpx.test.ts`), so CONV-02 fixed a value
nobody reads. Consider deleting `centroid` outright, or at least drop the duplicate counter.

### IN-03: Duplicated `if (elevation !== undefined)` blocks

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:91-105`, `web/src/lib/models/gpx/gpx-metrics-computation.ts:107-123`

Two consecutive guards on the same condition with near-identical anchor-init logic. Merge into
one block with two sub-branches; it removes ~8 lines and one opportunity for the two paths to
drift apart.

### IN-04: `any` typing defeats the type discipline CONV-03 was meant to establish

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:29-30`, `web/src/lib/models/gpx/gpx-metrics-computation.ts:46`

`addAndFilter(point: any)`, `lastPointXY: any`, `lastFilteredPointXY: any` in a `strict: true`
project. `parseElevation(raw: unknown)` is correctly typed, but its caller hands it a value the
compiler knows nothing about, so no future refactor of `Waypoint` will be caught here. Type
these as `Waypoint` (and let `parseElevation` remain `unknown`-tolerant for the string case).

### IN-05: Test-suite smells

**File:** `web/src/lib/models/gpx/gpx.test.ts:228-230`, `web/src/lib/models/gpx/gpx.test.ts:105-115`, `web/src/lib/models/gpx/gpx-metrics-computation.test.ts:88-116`

- `waypointAt(lat, lon, ele?)` declares an `ele` parameter that no call site passes — dead
  parameter in a brand-new helper.
- `gpx.test.ts:109` asserts `Number.isNaN(centroid.lat)` for an empty track, promoting an
  accidental `0/0` sentinel into a contract. If `centroid` survives IN-02, it should return
  `null`, not `NaN`.
- The 12-point CONV-04 fixture and the 5×jitter fixture are each rebuilt inline in two
  different tests; hoist them into named builders so a fixture change cannot update one copy
  only.
- Magic expectations (`134.592`, `444.78`, `100.075`) are repeated across blocks with no shared
  constant.

### IN-06: A single non-finite coordinate silently truncates raw distance and flatlines `cumulativeDistance`

**File:** `web/src/lib/models/gpx/gpx-metrics-computation.ts:80-88`

`Number.isFinite(distance)` correctly skips the accumulation, but `lastPointXY = point` runs
unconditionally at `:88`, so the poisoned point becomes the anchor and the *next* hop is
non-finite too — two hops of real distance vanish with no diagnostic, and
`cumulativeDistance` records a flat plateau that the crop interpolator will treat as a
zero-length span (feeding CR-01). Currently hard to reach because `Waypoint` coerces missing
coordinates to `-1` (`waypoint.ts:53-54`, itself questionable), but the guard's intent is
defeated. Either skip the anchor update for non-finite hops or drop such points before they
reach the accumulator.

---

_Reviewed: 2026-07-31T12:45:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
