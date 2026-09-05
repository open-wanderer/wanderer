# Derivation: 01-two-point-segment

## What this fixture is

A single `<trkseg>` with exactly two track points, no elevation, no time. The minimal
reproduction of the segment-first-point defect.

## Defect pinned

`gpx.ts`'s pre-fix `getTotals()` looped `for (let i = 1; i < pointLength; i++)`,
skipping index 0 of every segment. For a 2-point segment, `metrics.addAndFilter(point)` was
therefore called exactly once, on the segment's second point (`b`). That call is
`GpxMetricsComputation`'s *first-ever* invocation, which only initializes its internal anchors
(`lastPointXY`, `lastFilteredPointXY`) — no distance is added on that call. The reported
distance was therefore exactly **0**, regardless of how far apart the two points actually are.

The fix starts the loop at `i = 0`, so both points are fed to `addAndFilter()`: the first call
initializes anchors, the second computes and adds the real hop distance.

## Pre-fix value

`distance = 0` (see `gpx.test.ts:9-20`, "Pre-fix value was exactly 0").

## Derivation of expected values

### distance

Haversine formula, evaluated independently (not via `GPX.getTotals()`/`GpxMetricsComputation`):

```
R = 6371000 m
lat1, lon1 = 47.0, 11.0
lat2, lon2 = 47.001, 11.001
dLat = (lat2 - lat1) * pi/180
dLon = (lon2 - lon1) * pi/180
a = sin(dLat/2)^2 + cos(lat1*pi/180)*cos(lat2*pi/180)*sin(dLon/2)^2
c = 2 * atan2(sqrt(a), sqrt(1-a))
distance = R * c
```

Evaluated via a scratch `node -e` one-liner implementing exactly this formula (fresh
transcription, not the app's `haversineDistance`/`GpxMetricsComputation`):

```
$ node -e 'function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
console.log(haversine(47.0,11.0,47.001,11.001));'
134.59240148587796
```

Both points clear the smoothing threshold (5 m) by a wide margin, so the smoothed distance
equals the raw hop: **134.59240148587796 m**.

### elevationGain / elevationLoss

Neither point carries `<ele>`, so `parseElevation()` returns `undefined` for both — no elevation
data exists to accumulate a diff from. `elevationGain = 0`, `elevationLoss = 0`.

### durationMs

Neither point carries `<time>`. `gpx.ts`'s duration calc only runs `if (startTime && endTime)`;
both are absent, so `totalDuration` stays its initialized `0`. `durationMs = 0`.

### pointCount

Two `<trkpt>` elements in the one `<trkseg>`. `pointCount = 2`.

### boundingBox / centroid

Plain arithmetic over the fixture's own two coordinates:

- `minLat = min(47.0, 47.001) = 47.0`, `maxLat = max(47.0, 47.001) = 47.001`
- `minLon = min(11.0, 11.001) = 11.0`, `maxLon = max(11.0, 11.001) = 11.001`
- `centroid.lat = (47.0 + 47.001) / 2 = 47.0005`
- `centroid.lon = (11.0 + 11.001) / 2 = 11.0005` (IEEE-754 summation yields
  `11.000499999999999`, which the fixture records verbatim since `expected.json` compares to
  1e-9 degrees, not exact string equality)

## No implementation-under-test was executed to obtain these values

Every number above comes from a fresh, independent transcription of the haversine formula and
plain arithmetic over the fixture's own coordinates — never from calling `GPX.getTotals()`,
`gpx2trail()`, or `GpxMetricsComputation`.
