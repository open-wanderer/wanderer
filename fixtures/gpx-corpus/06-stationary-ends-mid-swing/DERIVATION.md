# Derivation: 06-stationary-ends-mid-swing

## What this fixture is

The same generator as fixture 05, all points at `(47.0, 11.0)`, elevation alternating
`1000, 1007, ...`, but truncated to 60 points (`i = 0..59`) instead of 61 — since `i = 59` is
odd, the track ends at `1007`, mid-swing, with the final excursion never cancelled.

## Defect pinned

A track that ends mid-excursion must report the genuine net displacement of its
final, un-cancelled swing — not `0` (which would under-report a real, if small, elevation
change) and not the flat `210`/`203` the pre-fix ratchet produced.

## Pre-fix value

`elevationGain = 210`, `elevationLoss = 203` (`gpx-metrics-computation.test.ts:150`).

## Derivation of expected values

### elevationGain / elevationLoss

Identical point-by-point reasoning to fixture 05 (see `05-stationary-noise-returns/DERIVATION.md`
for the full per-point discard trace) for `i = 0..58`: every odd-index excursion (`+7`) is
discarded by the very next even-index point (`-7`, returning exactly to the pending anchor with
zero horizontal movement), leaving `totalElevationGainSmoothed = totalElevationLossSmoothed = 0`
and `pendingDelta = 0` after processing point 58 (`i=58`, even, `ele=1000`).

The fixture stops one point short of fixture 05's cancelling pair: point 59 (`i=59`, `ele=1007`,
diff `+7` from `lastFilteredZ=1000`) becomes a *new* pending excursion (`pendingDelta = +7`,
`pendingAnchorZ = 1000`) — and there is no point 60 to confirm or cancel it. The track ends with
this excursion still pending.

`totalElevationGainSmoothed = 0` (never published), `pendingDelta = +7`, so:
`finalElevationGain = totalElevationGainSmoothed + max(pendingDelta, 0) = 0 + 7 = 7`.
`totalElevationLossSmoothed = 0`, and `max(-pendingDelta, 0) = max(-7, 0) = 0`, so
`finalElevationLoss = 0`.

This is exactly the genuine net displacement of the whole 60-point track: it started at `1000`
and ends at `1007`, a real `+7` m climb that the monotonic `totalElevationGainSmoothed` field
alone cannot see (it stays `0` because the excursion was never confirmed by a subsequent point),
but which `finalElevationGain` correctly surfaces — this is precisely the "track ends
mid-swing" case that the corpus must not confuse with the monotonic smoothed pair.

### distance / boundingBox / centroid

Identical reasoning to fixture 05: every point shares the coordinate `(47.0, 11.0)`, so
`distance = 0`, `boundingBox = {minLat: 47.0, maxLat: 47.0, minLon: 11.0, maxLon: 11.0}`,
`centroid = {lat: 47.0, lon: 11.0}`.

### durationMs / pointCount

No `<time>` elements: `durationMs = 0`. 60 `<trkpt>` elements: `pointCount = 60`.

## No implementation-under-test was executed to obtain these values

Elevation gain/loss was hand-traced through the documented discard-condition rules against this
fixture's own point sequence — not by running `GpxMetricsComputation`.
