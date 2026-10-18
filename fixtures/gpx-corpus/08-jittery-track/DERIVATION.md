# Derivation: 08-jittery-track

## What this fixture is

Sixteen track points, no elevation: start at `lat = 47.0`, then 5 iterations each emitting three
points — `lat += 0.00018` (a real ~20 m forward hop), `lat += 0.000009` (a ~1 m jitter forward),
`lat -= 0.000009` (a ~1 m jitter back to the same spot as the forward hop), `lon = 11.0`
throughout.

## Defect pinned (distance smoothing, since retired)

The corpus originally required the reported distance to be the smoothed accumulator (only advancing
on a hop that clears the 5 m threshold), not the raw sum of every consecutive-pair haversine
distance, so GPS jitter that never actually moves the device forward would not inflate the
reported distance. That rule is superseded: ground truth from a real FIT recording showed the 5 m
gate chord-shortcuts real switchback sampling density by far more than jitter ever inflates a raw
sum. The reported
distance is now the raw accumulator; this fixture still exists to pin the two figures against
each other so a regression in either accumulator is caught.

## Derivation of expected values

### distance — both the raw sum and the smoothed value

Both figures come from an independently-transcribed haversine formula plus a hand-simulation of
the documented smoothing rule (`smoothedDistance >= thresholdXY_m (5 m)` gates whether the
smoothed anchor advances) — this is the same distance-threshold rule stated in
`gpx-metrics-computation.ts`'s comments, transcribed fresh, not executed via the class itself.

```
$ node -e '
function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
const points=[[47.0,11.0]]; let lat=47.0;
for(let i=0;i<5;i++){
  lat+=0.00018; points.push([lat,11.0]);
  lat+=0.000009; points.push([lat,11.0]);
  lat-=0.000009; points.push([lat,11.0]);
}
let lastXY=null, lastFilteredXY=null, raw=0, smoothed=0;
for (const p of points) {
  if (lastXY===null){ lastXY=p; lastFilteredXY=p; continue; }
  raw += haversine(lastXY[0],lastXY[1],p[0],p[1]);
  const sHop = haversine(lastFilteredXY[0],lastFilteredXY[1],p[0],p[1]);
  lastXY=p;
  if (sHop>=5){ smoothed+=sHop; lastFilteredXY=p; }
}
console.log("raw",raw,"smoothed",smoothed);'
raw 110.08297737671097 smoothed 100.07543398026468
```

Per-hop breakdown for one iteration (all five iterations are identical by construction):
forward hop (~20.015 m, clears threshold, anchor advances), jitter-forward hop (~1.001 m, under
threshold, anchor does not advance — but the *raw* per-pair distance is still counted), jitter-
back hop measured from the still-unmoved smoothed anchor (~0 m, since the jitter-back point
lands back at the smoothed anchor's own coordinate) while its *raw* consecutive-pair distance is
still ~1.001 m (jitter-forward point to jitter-back point).

**Raw sum ≈ 110.08297737671097 m. Smoothed accumulator ≈ 100.07543398026468 m.** The corpus
asserts the raw value (`distance` in `metrics`/`trail`), because that is what
`GPX.getTotals()` reports as `distance` since distance smoothing was retired
(`gpx.ts:148`, `totalDistance = metrics.totalDistance`). The smoothed accumulator is the
counterfactual the superseded 5 m gate would have produced — it is no longer part of the public
contract, though `totalDistanceSmoothed` itself survives, unreported, on the class (the corpus also
still excludes `cumulativeDistance`'s raw per-point array from the Dart port, since its only
consumer — the web trail-edit crop slider — has no Dart-side equivalent).

### elevationGain / elevationLoss / durationMs

No `<ele>` or `<time>` elements on any point: `elevationGain = 0`, `elevationLoss = 0`,
`durationMs = 0`.

### boundingBox / centroid

Plain arithmetic over the 16 generated latitudes (all share `lon = 11.0`): `minLat = 47.0`
(first point), `maxLat = 47.000909` (last point, the highest cumulative latitude reached);
`centroid.lat` is the mean of all 16 latitude values, `centroid.lon = 11.0`.

### pointCount

Sixteen `<trkpt>` elements: `pointCount = 16`.

## No implementation-under-test was executed to obtain these values

Both the raw and smoothed distance figures come from an independently-transcribed haversine
formula plus a fresh hand-transcription of the documented smoothing-threshold rule — not from
calling `GPX.getTotals()`, `gpx2trail()`, or `GpxMetricsComputation`.
