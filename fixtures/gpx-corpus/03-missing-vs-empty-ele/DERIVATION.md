# Derivation: 03-missing-vs-empty-ele

## What this fixture is

Five track points at lat `47.000`..`47.004` (0.001 steps), lon `11.0`, with elevations in order
`1000`, `1005`, an **empty `<ele></ele>` element pair with no body**, `1010`, `1015`.

## Defect pinned

A missing or empty `<ele>` must be treated as "no data" (`undefined`), never coerced to
`0`. This fixture's third point is the parser landmine — `<ele></ele>` is present in the XML
(not omitted), so any parser that unconditionally calls `double.parse`/`Number()` on the element
text without an empty-string guard will either throw (Dart's `GpxReader`, confirmed in
verified against the real installed `gpx: ^2.3.0` package —
`FormatException: Invalid double`) or silently coerce to `0` (a naive `Number("")` call, which
JavaScript evaluates to `0`, not `NaN`). `parseElevation()` (`gpx-metrics-computation.ts:15-24`)
explicitly guards `typeof raw === 'string' && raw.trim() === ''` before the `Number(raw)` call to
prevent exactly this coercion. This is the canary this whole fixture exists to be: it is
mandatory that `<ele></ele>` (not just an omitted tag) appears literally in `input.gpx`, so an
unsanitised Dart `GpxReader().fromString()` crashes loudly on load rather than disagreeing
silently on the elevation value.

## Pre-fix value

`elevationGain = 1015`, `elevationLoss = 1005` (`gpx-metrics-computation.test.ts:36`) — the
missing/empty tag coerced to `0`, fabricating a plunge to sea level and back: `1000 -> 1005 ->
0 (loss 1005) -> 1010 (gain 1010) -> 1015 (gain 5)`, netting `elevationGain = 5 + 1010 = 1015`
(approximately; the exact pre-fix bookkeeping differs by code path, but the documented pre-fix
figure for this exact fixture shape is `1015`/`1005` per the existing TS regression suite).

## Derivation of expected values

### elevationGain / elevationLoss

Reasoning over the algorithm's stated rules (threshold 5 m, defer-then-publish, discard only on
a horizontal-stillness return — `gpx-metrics-computation.ts:97-235`), with the third point's
elevation treated as `undefined` (no data, carries the elevation anchor forward unchanged):

| i | lat | ele | parsed | effect |
|---|-----|-----|--------|--------|
| 0 | 47.000 | 1000 | 1000 | establishes `lastFilteredZ = 1000` (first point with elevation) |
| 1 | 47.001 | 1005 | 1005 | diff = +5; `Math.abs(5) < 5` is false (5 is not `< 5`) — wait: threshold check is `Math.abs(diff) < thresholdZ_m`; `5 < 5` is false, so this is NOT below the noise floor — it is at-or-above threshold, so it becomes the pending excursion: `pendingDelta = +5`, `pendingAnchorZ = 1000`, `lastFilteredZ = 1005` |
| 2 | 47.002 | (undefined) | — | no elevation data; the `if (elevation !== undefined)` block is skipped entirely — `pendingDelta` and `lastFilteredZ` are untouched |
| 3 | 47.003 | 1010 | 1010 | diff from `lastFilteredZ` (1005) = +5, at-or-above threshold; sign of new diff (+) is not opposite pendingDelta's sign (+), so `cancelsPending` is false — this confirms and publishes the pending +5 (`totalElevationGainSmoothed += 5` -> 5), then becomes the new pending: `pendingAnchorZ = 1005`, `pendingDelta = +5`, `lastFilteredZ = 1010` |
| 4 | 47.004 | 1015 | 1015 | diff from `lastFilteredZ` (1010) = +5, at-or-above threshold; same-sign as pending, so this publishes the pending +5 (`totalElevationGainSmoothed += 5` -> 10), then becomes the new pending: `pendingDelta = +5`, `lastFilteredZ = 1015` |

At the end of the track, `totalElevationGainSmoothed = 10`, and `pendingDelta = +5` is still
outstanding (never confirmed by a 6th point). `finalElevationGain = totalElevationGainSmoothed +
max(pendingDelta, 0) = 10 + 5 = 15`. `totalElevationLossSmoothed = 0` throughout (every diff was
positive), so `finalElevationLoss = 0 + max(-5, 0) = 0`.

`elevationGain = 15`, `elevationLoss = 0` — matching the existing TS regression suite's
documented expectation for this exact fixture shape (`gpx-metrics-computation.test.ts:38-39`,
cited per the plan's instruction to cite that file's reasoning rather than re-run it).

### distance

Four hops of 0.001 deg latitude each, haversine evaluated independently:

```
$ node -e 'function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
let lats=[47.000,47.001,47.002,47.003,47.004], total=0;
for(let i=1;i<lats.length;i++) total += haversine(lats[i-1],11.0,lats[i],11.0);
console.log(total);'
444.77970657798846
```

Each ~111.19 m hop clears the 5 m threshold, so the smoothed total equals the raw sum:
`distance = 444.77970657798846 m`.

### boundingBox / centroid

- `minLat = 47.000`, `maxLat = 47.004`, `minLon = maxLon = 11.0` (all points share the same
  longitude)
- `centroid.lat = (47.000+47.001+47.002+47.003+47.004)/5 = 235.01/5 = 47.002`
- `centroid.lon = 11.0`

### durationMs / pointCount

No `<time>` elements: `durationMs = 0`. Five `<trkpt>` elements: `pointCount = 5`.

## No implementation-under-test was executed to obtain these values

Elevation gain/loss was derived by hand-tracing the documented state machine rules against this
fixture's own point sequence (citing `gpx-metrics-computation.test.ts`'s already-documented
reasoning for the identical fixture shape, per the plan's explicit instruction), not by running
`GpxMetricsComputation`. Distance came from an independently-transcribed haversine formula.
Bounding box and centroid came from plain arithmetic.
