# Derivation: 05-stationary-noise-returns

## What this fixture is

61 track points, all at the identical coordinate `(47.0, 11.0)`, elevation alternating
`1000, 1007, 1000, 1007, ...` for `i = 0..60` — since `i = 60` is even, the track ends back at
`1000`, i.e. every swing round-trips to its starting elevation with zero horizontal movement.

## Defect pinned

A fully-stationary altimeter/GPS noise oscillation that returns to its starting
elevation must not fabricate gain/loss. Pre-fix, the flat threshold-commit rule ratcheted on
every +/-7 m swing even though the device never moved and the track's net elevation change is
exactly 0.

## Pre-fix value

`elevationGain = 210`, `elevationLoss = 210` (`gpx-metrics-computation.test.ts:132`).

## Derivation of expected values

### elevationGain / elevationLoss

Reasoning over the algorithm's stated rules (threshold 5 m, defer-then-publish, discard on
horizontal-stillness return): every step's diff is `+/-7`, whose absolute value (7) is `>= 5`
(above the noise floor), so every step is a candidate excursion. The discard condition
(`cancelsPending`) requires: a pending excursion exists, the new diff's sign is opposite the
pending diff's sign, the new elevation is within `threshold` of the pending excursion's *anchor*
elevation, and the horizontal return distance is under the threshold.

| i | ele | diff | pendingDelta before | cancelsPending? | pendingDelta after |
|---|-----|------|----------------------|-------------------|----------------------|
| 0 | 1000 | — | — | — | establishes `lastFilteredZ=1000`, pending 0 |
| 1 | 1007 | +7 | 0 | no (`pendingDelta===0`) | publish (no-op), new pending `+7`, anchor `1000` |
| 2 | 1000 | -7 | +7 | yes: sign(-7) = -sign(+7); `|1000 - pendingAnchorZ(1000)| = 0 < 5`; `returnDistance = 0 < 5` (same coordinate) | discard: `lastFilteredZ` rewound to `pendingAnchorZ` (1000), pending reset to `0` |
| 3 | 1007 | +7 | 0 | no | publish (no-op), new pending `+7`, anchor `1000` |
| 4 | 1000 | -7 | +7 | yes (identical reasoning to i=2) | discard, pending reset to `0` |
| ... | ... | ... | ... | ... | pattern repeats identically for every subsequent pair |
| 59 | 1007 | +7 | 0 | no | new pending `+7`, anchor `1000` |
| 60 | 1000 | -7 | +7 | yes | discard, pending reset to `0` |

Every odd-index excursion is discarded by the very next (even-index) point, because the
coordinate never changes (`returnDistance` is always exactly `0`, trivially under the 5 m
threshold) and the elevation always returns exactly to the pending anchor (`0 < 5` m
difference). Nothing is ever published into `totalElevationGainSmoothed`/`LossSmoothed`, and at
the end (`i = 60`, even, back at `1000`) `pendingDelta = 0` (the last discard reset it). So
`totalElevationGainSmoothed = totalElevationLossSmoothed = 0` and `pendingDelta = 0`, giving
`finalElevationGain = 0`, `finalElevationLoss = 0`.

### distance

All 61 points share the identical coordinate `(47.0, 11.0)`. Every haversine hop between
identical coordinates is exactly `0` (verifiable trivially: `dLat = dLon = 0` collapses the
formula's `a` term to `0`, so `c = 0`, `distance = 0`). `totalDistanceSmoothed` stays `0`
throughout — no hop ever reaches the 5 m threshold.

### boundingBox / centroid

Every point is `(47.0, 11.0)`: `minLat = maxLat = 47.0`, `minLon = maxLon = 11.0`,
`centroid = (47.0, 11.0)` trivially (mean of 61 identical values is that value).

### durationMs / pointCount

No `<time>` elements: `durationMs = 0`. 61 `<trkpt>` elements: `pointCount = 61`.

## No implementation-under-test was executed to obtain these values

Elevation gain/loss was hand-traced through the documented discard-condition rules against this
fixture's own repeating point sequence — not by running `GpxMetricsComputation`. Distance,
bounding box, and centroid follow from the trivial fact that every point shares one coordinate.
