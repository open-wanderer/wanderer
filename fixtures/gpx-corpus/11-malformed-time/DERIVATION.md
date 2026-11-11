# Derivation: 11-malformed-time

## What this fixture is

A single `<trkseg>` with three track points and no elevation. The first point carries a
non-empty but unparseable `<time>N/A</time>`, the second an empty `<time></time>`, and the
third a genuine ISO-8601 instant.

## Defect pinned

The two implementations disagreed on a **non-empty** unparseable `<time>` body.

- **TypeScript (pre-fix):** `Waypoint`'s constructor did `if (object.time) this.time = new
  Date(object.time)`. A garbage string is truthy, so `this.time` became an `Invalid Date` —
  which is itself a truthy object. `gpx.ts`'s `getTotals()` guard `if (startTime && endTime)`
  therefore passed and computed `endTime.getTime() - startTime.getTime()`, i.e. `NaN`. The
  trail's whole `duration` came out `NaN`.
- **Dart:** `sanitizeGpxNumericAndTime` rewrites any `<time>` body that fails
  `DateTime.tryParse` to a self-closing `<time/>`, which `GpxReader` reads as `null`. The
  segment contributed `0`.

Both languages already agreed on the *empty* `<time></time>` form (xml2js yields `''`, which is
falsy), which is why the second point is included: it pins the case that already worked
alongside the one that did not, in the same document.

The fix gates the TS side on the grammar Dart's `DateTime.parse` accepts (transcribed verbatim
from the Dart SDK) plus an `Invalid Date` check, so an unparseable body is "no time" in both
languages.

## Pre-fix value

`duration = NaN` on the TS side (`trail.duration` and `metrics.durationMs` both), `0` on the
Dart side. There was no corpus fixture exercising this input class, so the parity suite could
not see the divergence.

## Derivation of expected values

### distance

Haversine formula, evaluated independently (not via `GPX.getTotals()`/`GpxMetricsComputation`),
via a fresh scratch `node -e` transcription with `R = 6371000`:

```
$ node -e 'function hav(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
const h1=hav(47.0,11.0,47.001,11.001), h2=hav(47.001,11.001,47.002,11.002);
console.log(h1, h2, h1+h2);'
134.59240148587796 134.59160176690574 269.18400325278367
```

Both hops clear the 5 m smoothing threshold by a wide margin, so each is published in full and
the smoothing anchor advances to the just-accepted point after each one. The smoothed total is
therefore the plain sum of the two raw hops, accumulated in traversal order:
**269.18400325278367 m**.

### elevationGain / elevationLoss

No point carries `<ele>`, so `parseElevation()` yields `undefined` throughout and no elevation
diff is ever available to accumulate. `elevationGain = 0`, `elevationLoss = 0`.

### durationMs

The segment has three points, so the `points.length >= 2` guard is satisfied and the duration
calc reads `points[0].time` and `points[2].time`.

- `points[0]`'s body is `N/A`, which is not a valid instant in either language, so its `time`
  is absent.
- The guard is `startTime && endTime` (TS) / `startTime != null && endTime != null` (Dart).
  With `startTime` absent it fails, and `totalDuration` keeps its initialized `0`.

`durationMs = 0`. This is the value the fixture exists to pin: it is `NaN` on the pre-fix TS
side and `0` on the Dart side.

### pointCount

Three `<trkpt>` elements in the one `<trkseg>`. `pointCount = 3`.

### boundingBox / centroid

Plain arithmetic over the fixture's own three coordinates:

- `minLat = 47.0`, `maxLat = 47.002`, `minLon = 11.0`, `maxLon = 11.002`
- `centroid.lat = (47.0 + 47.001 + 47.002) / 3` — IEEE-754 summation in traversal order yields
  `47.001000000000005`, recorded verbatim (the corpus compares centroid to 1e-9 degrees)
- `centroid.lon = (11.0 + 11.001 + 11.002) / 3 = 11.001`

### trail.date

`gpx2trail()` / `trailFromGpx()` set the date only when the first segment's first AND last
point both carry a time. The first point's time is absent, so `date = null`.

### trail.name / trail.description

`metadata.name` is `Malformed Time`; there is no `<metadata><desc>`, so `description` is null
(the Dart model's non-nullable field defaults to `''`, which the corpus helper treats as
equivalent — same convention as fixture 01).

## Known residual asymmetry (deliberate)

Aligning the two runtimes exactly on date parsing is not achievable in general: Dart's
`DateTime` normalises out-of-range components (`2024-13-45`) and accepts 6-digit years, both of
which V8's `Date` rejects. In those cases the TS side now reports "no time" while Dart reports a
real instant — the reverse of the defect this fixture pins, and unreachable from any GPX a
conforming exporter produces (GPX 1.1 mandates ISO 8601 for `<time>`). This fixture deliberately
does not exercise that class.

## No implementation-under-test was executed to obtain these values

Every number above comes from a fresh, independent transcription of the haversine formula, plain
arithmetic over the fixture's own coordinates, and the documented guard rules — never from
calling `GPX.getTotals()`, `gpx2trail()`, `GpxMetricsComputation`, `computeTrailMetrics`, or
`trailFromGpx`.
