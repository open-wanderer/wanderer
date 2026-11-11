# Derivation: 07-rolling-terrain

## What this fixture is

Six track points at `lat = 47 + i * 0.0009` for `i = 0..5` (each hop ~100 m apart, well over the
5 m horizontal threshold), `lon = 11.0` throughout, elevations `1000, 1008, 1000, 1008, 1000,
1008`.

## Defect pinned

The noise-rejection guard: the discard rule (fixtures 05/06) must never fire when the
track genuinely moves horizontally between elevation swings — real rolling terrain (up-down-up-
down with real forward travel) produces the identical elevation series as stationary
GPS/altimeter noise, and only the horizontal-stillness check distinguishes them.

## Derivation of expected values

### elevationGain / elevationLoss

Reasoning over the algorithm's stated rules: every step's diff is `+/-8`, `>= 5` (above the noise
floor). The discard condition additionally requires `returnDistance < thresholdXY_m (5 m)` —
since consecutive points here are ~100 m apart, `returnDistance` is always ~100 m, never under 5
m, so `cancelsPending` is false for every step. Every excursion is therefore confirmed (published)
by the next step, never discarded:

| i | ele | diff from lastFilteredZ | action |
|---|-----|--------------------------|--------|
| 0 | 1000 | — | establishes `lastFilteredZ = 1000` |
| 1 | 1008 | +8 | first excursion: `pendingDelta = 8`, anchor `1000` |
| 2 | 1000 | -8 | `returnDistance` (~100 m) is NOT `< 5`, so `cancelsPending` is false even though the sign is opposite and the elevation matches the anchor — publishes prior `+8` (`totalElevationGainSmoothed: 0 -> 8`); new pending `-8`, anchor `1008` |
| 3 | 1008 | +8 | publishes prior `-8` (`totalElevationLossSmoothed: 0 -> 8`); new pending `+8`, anchor `1000` |
| 4 | 1000 | -8 | publishes prior `+8` (`totalElevationGainSmoothed: 8 -> 16`); new pending `-8`, anchor `1008` |
| 5 | 1008 | +8 | publishes prior `-8` (`totalElevationLossSmoothed: 8 -> 16`); new pending `+8`, anchor `1000` |

After all 6 points: `totalElevationGainSmoothed = 16`, `totalElevationLossSmoothed = 16`,
`pendingDelta = +8` still outstanding (the final `1000 -> 1008` step, unconfirmed).
`finalElevationGain = 16 + max(8, 0) = 24`. `finalElevationLoss = 16 + max(-8, 0) = 16`.

`elevationGain = 24`, `elevationLoss = 16` — matches the existing TS regression suite's already-
documented expectation for this exact fixture shape (`gpx-metrics-computation.test.ts:189-190`).

### distance

```
$ node -e 'function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
let lats=[]; for(let i=0;i<6;i++) lats.push(47+i*0.0009);
let total=0; for(let i=1;i<6;i++) total+=haversine(lats[i-1],11.0,lats[i],11.0);
console.log(total);'
500.37716990053326
```

Each ~100.075 m hop clears the 5 m threshold, so the smoothed total equals the raw sum:
`distance = 500.37716990053326 m`.

### boundingBox / centroid

- `minLat = 47.0` (i=0), `maxLat = 47 + 5*0.0009 = 47.0045` (i=5)
- `minLon = maxLon = 11.0`
- `centroid.lat` = mean of `47 + i*0.0009` for `i=0..5` = `47 + 0.0009*(0+1+2+3+4+5)/6` =
  `47 + 0.0009*2.5 = 47.00225`
- `centroid.lon = 11.0`

### durationMs / pointCount

No `<time>` elements: `durationMs = 0`. Six `<trkpt>` elements: `pointCount = 6`.

## No implementation-under-test was executed to obtain these values

Elevation gain/loss was hand-traced through the documented state machine rules against this
fixture's own point-by-point elevation sequence — not by running `GpxMetricsComputation`.
Distance came from an independently-transcribed haversine formula. Bounding box and centroid
came from plain arithmetic.
