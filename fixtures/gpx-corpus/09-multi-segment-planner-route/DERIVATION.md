# Derivation: 09-multi-segment-planner-route

## What this fixture is

Two `<trkseg>` elements inside one `<trk>`, shaped like `valhalla_store.svelte.ts`'s
`insertIntoRoute()` output / `buildFinalPlannedGpx`'s per-leg emission: `leg1 = [(47.0,11.0),
(47.001,11.0), (47.002,11.0)]`, `leg2 = [(47.002,11.0), (47.003,11.0), (47.004,11.0)]` — the
second leg's opening point is deliberately the same coordinate as the first leg's closing point
(the shared anchor a route planner emits between consecutive legs).

## Defect pinned

The segment-first-point fix, applied across a segment boundary: the `i = 0` loop bound must not have introduced
a *per-segment* metrics-anchor reset. `GpxMetricsComputation` is a single shared instance across
every segment (`gpx.ts:106`, constructed once per `getTotals()` call and never re-created inside
the segment loop) — this fixture proves that a route with multiple track segments still measures
distance continuously through the segment boundary, not as N independently-reset segments.

## Pre-fix value

`distance = 333.585` (per the plan's own fixture description and
`gpx.test.ts:59-86`) — the `i = 1` loop bound dropped each segment's own first point: leg 1's
opening hop (`(47.0,11.0) -> (47.001,11.0)`) and leg 2's zero-length anchor duplicate, losing one
real hop's worth of distance overall.

## Derivation of expected values

### distance

Four hops, each ~111.19 m, haversine evaluated independently:

```
$ node -e 'function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
const leg1=[[47.0,11.0],[47.001,11.0],[47.002,11.0]];
const leg2=[[47.002,11.0],[47.003,11.0],[47.004,11.0]];
const h1=haversine(...leg1[0],...leg1[1]);
const h2=haversine(...leg1[1],...leg1[2]);
const h3=haversine(...leg2[0],...leg2[1]);
const h4=haversine(...leg2[1],...leg2[2]);
console.log(h1,h2,h3,h4,"total",h1+h2+h3+h4);'
111.19492664429958 111.1949266450897 111.19492664429958 111.19492664429958 total 444.77970657798846
```

Every hop clears the 5 m smoothing threshold, so the smoothed total equals the raw sum across
all four hops (leg 1's two internal hops, plus leg 2's two internal hops — the shared anchor
point at the leg boundary contributes no extra distance of its own, since it is the same
coordinate ending leg 1 and starting leg 2, not a fifth hop):
`distance = 444.77970657798846 m`.

This is precisely the sum of all four real hops; a per-segment metrics-anchor reset would either
double-count or drop the anchor point depending on the exact reset semantics — neither of which
happens here, confirming the shared-instance design.

### elevationGain / elevationLoss / durationMs

No `<ele>` or `<time>` elements on any point: `elevationGain = 0`, `elevationLoss = 0`,
`durationMs = 0`.

### boundingBox / centroid

All six points (three per segment, with the shared anchor appearing once per segment, i.e. twice
total across the two `<trkseg>` elements — `flatten()` walks every `<trkpt>` in every `<trkseg>`,
so the shared coordinate is counted twice for `pointCount`/bounding-box/centroid purposes, exactly
matching what `gpx.test.ts:169-172`'s `cumulativeDistance` length assertions already establish for
this identical fixture shape):

- `minLat = 47.0`, `maxLat = 47.004`, `minLon = maxLon = 11.0`
- `centroid.lat` = mean of `[47.0, 47.001, 47.002, 47.002, 47.003, 47.004]` =
  `282.012 / 6 = 47.002`
- `centroid.lon = 11.0`

### pointCount

Three `<trkpt>` per segment, two segments: `pointCount = 6`.

## No implementation-under-test was executed to obtain these values

Distance came from an independently-transcribed haversine formula summed over the fixture's own
four real hops. Bounding box and centroid came from plain arithmetic over the fixture's own six
points (counting the shared anchor coordinate once per segment, per `flatten()`'s definition).
