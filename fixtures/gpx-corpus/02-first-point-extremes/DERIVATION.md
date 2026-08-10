# Derivation: 02-first-point-extremes

## What this fixture is

Three track points where the FIRST point is deliberately the geographic extreme on every axis:
`(40.0, 10.0)`, `(47.0, 11.0)`, `(48.0, 12.0)`.

## Defect pinned

The bounding box and the centroid divisor. Pre-fix, the `i = 1` loop bound skipped
the segment's own first point when accumulating `minLat`/`maxLat`/`minLon`/`maxLon` and the
centroid's `totalLat`/`totalLon`/`summedPointCount`. Since the skipped point here is also the
geographic extreme, the pre-fix bounding box reported the *second* point's coordinates
(`47.0`/`11.0`) as the minimum, instead of the true minimum (`40.0`/`10.0`). The centroid's
numerator summed only 2 points; depending on the exact pre-fix code path for the divisor, this
produced a mismatched divisor and numerator — the corpus's headline pre-fix figure is
`31.3343.../7.3343...`, i.e. a centroid pulled toward the two later, more northeasterly points
while an inconsistent divisor further skewed it away from the plain 3-point mean.

## Pre-fix values

`boundingBox.minLat = 47.0`, `boundingBox.minLon = 11.0` (should be `40.0`/`10.0`).
`centroid = (31.3343.., 7.3343..)` (should be `(45.0, 11.0)`).

## Derivation of expected values

### boundingBox

Plain min/max over the fixture's own three coordinates:

- `minLat = min(40.0, 47.0, 48.0) = 40.0`
- `maxLat = max(40.0, 47.0, 48.0) = 48.0`
- `minLon = min(10.0, 11.0, 12.0) = 10.0`
- `maxLon = max(10.0, 11.0, 12.0) = 12.0`

### centroid (invariant: divide by exactly the count summed)

- `totalLat = 40.0 + 47.0 + 48.0 = 135.0`, `summedPointCount = 3`
- `centroid.lat = 135.0 / 3 = 45.0`
- `totalLon = 10.0 + 11.0 + 12.0 = 33.0`
- `centroid.lon = 33.0 / 3 = 11.0`

This is simply the plain arithmetic mean of all three points — the invariant is
that the divisor must equal the number of points actually summed into the numerator, which for
a 3-point segment with no dropped points is trivially 3.

### distance

Haversine formula, evaluated independently, hop by hop:

```
$ node -e 'function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
console.log("h1", haversine(40.0,10.0,47.0,11.0));
console.log("h2", haversine(47.0,11.0,48.0,12.0));'
h1 782513.2058901277
h2 134189.88119072223
```

Both hops vastly exceed the 5 m smoothing threshold, so the smoothed total equals the raw sum:
`distance = 782513.2058901277 + 134189.88119072223 = 916703.0870808499 m`.

### elevationGain / elevationLoss / durationMs

No `<ele>` or `<time>` on any point: `elevationGain = 0`, `elevationLoss = 0`, `durationMs = 0`.

### pointCount

Three `<trkpt>` elements. `pointCount = 3`.

## No implementation-under-test was executed to obtain these values

Bounding box and centroid come from plain min/max/mean arithmetic over the fixture's own
coordinates; distance comes from an independently-transcribed haversine formula. None of
`GPX.getTotals()`, `gpx2trail()`, or `GpxMetricsComputation` was called.
