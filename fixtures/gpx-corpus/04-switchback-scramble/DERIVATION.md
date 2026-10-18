# Derivation: 04-switchback-scramble

## What this fixture is

Twelve track points at `lat = 47 + i * 0.0000036` for `i = 0..11`, `lon = 11.0` throughout,
`ele = 1000 + i * 8` — a steep 88 m climb spread over only ~4.4 m of horizontal movement (a
scramble/via-ferrata-shaped stretch: lots of vertical gain, almost no lateral travel).

## Defect pinned

Elevation gain/loss must be sampled independently of the horizontal distance threshold.
Pre-fix, the smoothed elevation diff was gated behind the same horizontal-movement threshold used
for distance smoothing — since this stretch's cumulative horizontal movement (~4.4 m) never
clears the 5 m threshold, the pre-fix code never registered any of the climb.

## Pre-fix value

`elevationGain = 0` (per the plan's own fixture description and
`gpx-metrics-computation.test.ts:96-99`: "the smoothed elevation diff was gated behind the
horizontal threshold, which this stretch never clears").

## Derivation of expected values

### elevationGain — the single most important derivation in this corpus

Reasoning over the algorithm's stated rules (threshold 5 m on elevation, defer-then-publish,
discard only on horizontal-stillness — `gpx-metrics-computation.ts:97-235`). Each consecutive
step is `ele[i] - ele[i-1] = 8`, which is `>= 5` (the threshold), so **every** step becomes a
confirming move for whatever was pending, publishing it, then itself becoming the new pending
excursion:

| i | ele | diff from lastFilteredZ | action |
|---|-----|--------------------------|--------|
| 0 | 1000 | — | establishes `lastFilteredZ = 1000` |
| 1 | 1008 | +8 | first excursion: `pendingDelta = 8`, `pendingAnchorZ = 1000`, `lastFilteredZ = 1008` |
| 2 | 1016 | +8 | confirms & publishes prior +8 (`totalElevationGainSmoothed: 0 -> 8`); new pending `+8`, anchor `1008` |
| 3 | 1024 | +8 | publishes (`8 -> 16`); new pending `+8`, anchor `1016` |
| 4 | 1032 | +8 | publishes (`16 -> 24`); new pending `+8`, anchor `1024` |
| 5 | 1040 | +8 | publishes (`24 -> 32`); new pending `+8`, anchor `1032` |
| 6 | 1048 | +8 | publishes (`32 -> 40`); new pending `+8`, anchor `1040` |
| 7 | 1056 | +8 | publishes (`40 -> 48`); new pending `+8`, anchor `1048` |
| 8 | 1064 | +8 | publishes (`48 -> 56`); new pending `+8`, anchor `1056` |
| 9 | 1072 | +8 | publishes (`56 -> 64`); new pending `+8`, anchor `1064` |
| 10 | 1080 | +8 | publishes (`64 -> 72`); new pending `+8`, anchor `1072` |
| 11 | 1088 | +8 | publishes (`72 -> 80`); new pending `+8`, anchor `1080` |

After all 12 points: `totalElevationGainSmoothed = 80` (10 confirmed +8 steps: point 1 through
point 10's publish events — 10 publishes x 8 = 80), and `pendingDelta = +8` is still outstanding
(the point-10-to-point-11 step, never confirmed by a 13th point).

**`totalElevationGainSmoothed` (80) is the monotonic running total — it is NOT what this corpus
asserts.** The value this corpus asserts is `finalElevationGain = totalElevationGainSmoothed +
max(pendingDelta, 0) = 80 + 8 = 88`, because the track is complete: the last 8 m step is real,
confirmed climbing, just not yet published into the monotonic total (which only publishes on the
*next* move, and there is no next move on a finished track). The corpus's
`elevationGain` field is the `final*` semantics, never `total*Smoothed` — porting the wrong one
of this pair is called out as "the single most likely way to fail the corpus."

**The corpus asserts 88, not 80.**

`elevationLoss = 0` throughout (every diff was positive, so `totalElevationLossSmoothed` never
incremented, and `pendingDelta` is positive at the end so `max(-pendingDelta, 0) = 0`).

### distance

Direct verification that the cumulative horizontal movement never clears the 5 m threshold:

```
$ node -e 'function haversine(lat1,lon1,lat2,lon2){const R=6371000;const toRad=d=>d*Math.PI/180;
const dLat=toRad(lat2-lat1);const dLon=toRad(lon2-lon1);
const a=Math.sin(dLat/2)**2+Math.cos(toRad(lat1))*Math.cos(toRad(lat2))*Math.sin(dLon/2)**2;
const c=2*Math.atan2(Math.sqrt(a),Math.sqrt(1-a));return R*c;}
console.log("first-to-last direct distance:", haversine(47,11.0,47+11*0.0000036,11.0));'
first-to-last direct distance: 4.403319095226456
```

The direct first-to-last hop is ~4.4 m, under the 5 m threshold, and each intermediate hop is
even smaller (~0.4 m). Since distance smoothing was retired, the reported distance is the
raw accumulator (`totalDistance`), which sums every hop regardless of the 5 m threshold: the
eleven ~0.4003 m meridian hops sum to `4.403319095226456` m (agreeing with the direct
first-to-last haversine above to within ~1e-12, as expected along a meridian). **The corpus
asserts 4.403319095226456, not 0.**

The now-unreported smoothed accumulator (`totalDistanceSmoothed`) tells the superseded story:
its anchor (`lastFilteredPointXY`) never advances past point 0 because no single hop from it
ever reaches 5 m, so it would have stayed exactly `0` throughout — that `0` is the counterfactual
the superseded 5 m gate would have produced, not the value this corpus asserts. This is precisely
what proves the elevation threshold and the distance threshold are gated independently of each
other — the elevation climb registers in full regardless of which distance accumulator is read.

### boundingBox / centroid

Plain arithmetic over the 12 generated latitudes (`lat = 47 + i*0.0000036`, `lon = 11.0`
throughout):

- `minLat = 47.0` (i=0), `maxLat = 47 + 11*0.0000036 = 47.0000396` (i=11)
- `minLon = maxLon = 11.0`
- `centroid.lat` = mean of `47 + i*0.0000036` for `i=0..11` = `47 + (0.0000036 * (0+1+...+11)/12)` =
  `47 + 0.0000036 * (66/12)` = `47 + 0.0000036*5.5` = `47.0000198`
- `centroid.lon = 11.0`

### durationMs / pointCount

No `<time>` elements: `durationMs = 0`. Twelve `<trkpt>` elements: `pointCount = 12`.

## No implementation-under-test was executed to obtain these values

Elevation gain/loss was hand-traced through the documented state machine rules (threshold
comparison, defer/publish/discard) against this fixture's own point-by-point elevation sequence
— not by running `GpxMetricsComputation`. Distance came from an independently-transcribed
haversine formula. Bounding box and centroid came from plain arithmetic.
