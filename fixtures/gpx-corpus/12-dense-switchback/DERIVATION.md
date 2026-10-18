# Derivation: 12-dense-switchback

## What this fixture is

Forty-one track points, no elevation, generated from an integer counter `i = 0..40`:

- `lat = 47 + 144*i/1e7` — a steady meridian climb, one hop per point.
- `lon = 11` when `i` is even, `11.0000462` when `i` is odd — the switchback zig-zag.

No `<ele>` and no `<time>` on any point (elevation and duration are deliberately out of scope
here; fixture `04-switchback-scramble` already pins threshold-independent elevation).

## Why this fixture exists — the corpus's blind spot

Every other fixture in this corpus samples far more sparsely than a real GPS watch.
`10-realistic-track` has a mean hop of 44.8 m and a minimum hop of 39.39 m — **zero** hops
anywhere near the 5 m distance-smoothing threshold. A real watch recording a switchback at a
4-second sample interval produces hops around 4.27 m — an order of magnitude denser. No existing
fixture could see the defect this fixes (the 5 m gate chord-shortcutting real
switchback geometry), because none of them sample densely enough for the gate to ever fire.

This fixture closes that gap: every one of its 40 consecutive hops measures ~3.852 m, under the
5 m gate, at the density class a real recording actually produces.

## Defect pinned (distance smoothing, since retired)

The reported distance is now the raw accumulator (`totalDistance`), not the 5 m-gated smoothed
accumulator (`totalDistanceSmoothed`) the corpus originally required. This fixture is the
regression guard: if a future change silently re-introduces the 5 m gate on the reporting path,
this fixture's expected distance (~154 m) would fail by ~77 m — impossible to miss.

## Derivation of expected values

### distance — raw sum and the gated counterfactual

Both figures come from an independently-transcribed haversine formula in a scratch `node -e` run
(never from calling `GPX.getTotals()`, `gpx2trail()`, `GpxMetricsComputation`, or
`computeTrailMetrics`):

```
$ node -e '
function haversine(lat1,lon1,lat2,lon2){
  const R = 6371000;
  const toRad = d => d * Math.PI / 180;
  const dLat = toRad(lat2-lat1);
  const dLon = toRad(lon2-lon1);
  const a = Math.sin(dLat/2)**2 + Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
  const c = 2*Math.atan2(Math.sqrt(a), Math.sqrt(1-a));
  return R*c;
}
const points = [];
for (let i = 0; i <= 40; i++) {
  const lat = 47 + 144*i/1e7;
  const lon = 11 + (i % 2 === 1 ? 462/1e7 : 0);
  points.push([lat, lon]);
}
let raw = 0;
const hops = [];
for (let i = 1; i < points.length; i++) {
  const d = haversine(points[i-1][0], points[i-1][1], points[i][0], points[i][1]);
  hops.push(d);
  raw += d;
}
console.log("raw sum:", raw);
console.log("min hop:", Math.min(...hops), "max hop:", Math.max(...hops), "mean hop:", raw/hops.length);

// gated counterfactual: threshold 5 m, anchor advances only when the hop from it clears 5 m
let lastFiltered = points[0];
let gated = 0;
for (let i = 1; i < points.length; i++) {
  const d = haversine(lastFiltered[0], lastFiltered[1], points[i][0], points[i][1]);
  if (d >= 5) {
    gated += d;
    lastFiltered = points[i];
  }
}
console.log("gated counterfactual:", gated);
'
raw sum: 154.08415712669284
min hop: 3.8520871812209956 max hop: 3.8521206751887584 mean hop: 3.8521039281673213
gated counterfactual: 77.29220749531189
```

**Raw sum ≈ 154.08415712669284 m** (agrees with the cross-check magnitude 154.0841571 m to well
within 1e-6). Every hop is ~3.852 m, comfortably under the 5 m threshold — at real-watch sampling
density (a real recording samples around ~4.27 m per hop at a 4 s interval, hiking pace).

**Gated counterfactual ≈ 77.29220749531189 m** (agrees with 77.2922075 m to well within 1e-6):
because every hop is under 5 m, the superseded gate's anchor (`lastFilteredPointXY`) advances
only once every third point — one hop out of every three clears 5 m cumulatively from the
still-unmoved anchor, while the other two are absorbed and lost. The gate loses just over half
the track's real length (77.29 m reported vs. 154.08 m actual) — this is the concrete cost of the
defect the 5 m gate had, at the sampling density a real device actually produces.

**The corpus asserts 154.08415712669284 (the raw sum), not 77.292 (the gate's counterfactual).**

### elevationGain / elevationLoss / durationMs

No `<ele>` or `<time>` elements on any point: `elevationGain = 0`, `elevationLoss = 0`,
`durationMs = 0`.

### boundingBox

Plain arithmetic over the generated coordinates:

- `minLat = 47` (i = 0), `maxLat = 47 + 144*40/1e7 = 47.000576` (i = 40)
- `minLon = 11` (every even `i`), `maxLon = 11.0000462` (every odd `i`)

### centroid

Exact arithmetic, not a floating-point running sum:

- `centroid.lat`: the mean of `47 + 144*i/1e7` for `i = 0..40`. The mean of `i = 0..40` is
  `(0+1+...+40)/41 = 820/41 = 20`, so `centroid.lat = 47 + 20*0.0000144 = 47.000288`.
- `centroid.lon`: 20 of the 41 points (the odd indices `i = 1, 3, ..., 39`) carry
  `lon = 11.0000462`; the other 21 carry `lon = 11`. `centroid.lon = 11 + (20*0.0000462)/41`.
  Computed exactly: `(20*462/1e7)/41 = 0.00002253658536585366`, so
  `centroid.lon = 11.000022536585366`. (A naive floating-point sum-then-divide over the 41-point
  loop above lands at `11.000022536585362` — a ~4e-15 difference from summation rounding, far
  inside the corpus's 1e-9 degree tolerance; the exact-arithmetic value above is what
  `expected.json` records.)

### pointCount

Forty-one `<trkpt>` elements: `pointCount = 41`.

## No implementation-under-test was executed to obtain these values

The raw sum and the gated counterfactual both come from an independently-transcribed haversine
formula and a fresh hand-transcription of the 5 m gate rule, run in a scratch `node -e` snippet —
never by calling `GPX.getTotals()`, `gpx2trail()`, `GpxMetricsComputation`, or
`computeTrailMetrics`. Bounding box and centroid come from plain/exact arithmetic over the
fixture's own generating formula.
