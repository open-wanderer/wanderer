# Derivation: 19-descent-out-and-back

A synthetic out-and-back that stands in for open-wanderer/wanderer#670. Nothing here derives from the issue's attached user file.

## Construction

121 points on one meridian (lon 11.400000), three seconds apart from 06:00:00Z, so the last point is 06:06:00Z.

- Out leg, `k = i` for `i = 0..60`: lat `47.3 + i * 0.000036` written with six decimals (47.300000 up to 47.302160), ele `1200 + i + nOut[i mod 4]` with `nOut = [0, 1, 0, -1]`.
- Back leg, `k = 60 + j` for `j = 1..60`, at position `p = 60 - j`, which retraces the latitude of out point `p`: ele `1200 + p + nBack[j mod 4]` with `nBack = [0, -1, 0, 1]`.

That is a real 60 m climb, an equal descent and +/-1 m noise, at about 4.003 m per hop.

## Distance

Haversine, `R = 6371000`, summed over the 120 consecutive hops (each between 4.003017358 m and 4.003017359 m), evaluated in a scratch Python script outside the repository: `480.362083105392`.

## Elevation

Hand-traced from the documented defer-then-publish rules (threshold 5 m; a jump cancels only if it has the opposite sign, returns within 5 m of the pending anchor elevation, and lies within 5 m horizontally of the last filtered point). The first point (ele 1200) is the anchor.

The noise never reaches 5 m on its own; the ramp does, every 5 to 7 points. That gives 20 jumps of exactly 6 m, 10 up and 10 down. Between any two consecutive jumps the track moves at least 20.0 m horizontally, so the 5 m horizontal-stillness test can never hold and nothing is ever cancelled. Every jump below is therefore published by the next one, and the last one is still pending when the track ends.

| Point | Ele | Jump from last filtered | Decision |
|-------|-----|-------------------------|----------|
| 5 | 1206 | +6 | nothing pending: becomes pending (+6) |
| 12 | 1212 | +6 | same sign, cannot cancel: publishes the +6 (published gain 6, loss 0), becomes pending (+6) |
| 17 | 1218 | +6 | same sign, cannot cancel: publishes the +6 (published gain 12, loss 0), becomes pending (+6) |
| 24 | 1224 | +6 | same sign, cannot cancel: publishes the +6 (published gain 18, loss 0), becomes pending (+6) |
| 29 | 1230 | +6 | same sign, cannot cancel: publishes the +6 (published gain 24, loss 0), becomes pending (+6) |
| 36 | 1236 | +6 | same sign, cannot cancel: publishes the +6 (published gain 30, loss 0), becomes pending (+6) |
| 41 | 1242 | +6 | same sign, cannot cancel: publishes the +6 (published gain 36, loss 0), becomes pending (+6) |
| 48 | 1248 | +6 | same sign, cannot cancel: publishes the +6 (published gain 42, loss 0), becomes pending (+6) |
| 53 | 1254 | +6 | same sign, cannot cancel: publishes the +6 (published gain 48, loss 0), becomes pending (+6) |
| 60 | 1260 | +6 | same sign, cannot cancel: publishes the +6 (published gain 54, loss 0), becomes pending (+6) |
| 65 | 1254 | -6 | opposite sign and back at the pending anchor elevation, but 20.0 m from the last filtered point: no cancel. Publishes the +6 (published gain 60, loss 0), becomes pending (-6) |
| 72 | 1248 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 6), becomes pending (-6) |
| 77 | 1242 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 12), becomes pending (-6) |
| 84 | 1236 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 18), becomes pending (-6) |
| 89 | 1230 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 24), becomes pending (-6) |
| 96 | 1224 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 30), becomes pending (-6) |
| 101 | 1218 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 36), becomes pending (-6) |
| 108 | 1212 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 42), becomes pending (-6) |
| 113 | 1206 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 48), becomes pending (-6) |
| 120 | 1200 | -6 | same sign, cannot cancel: publishes the -6 (published gain 60, loss 54), becomes pending (-6) |

The track ends with the final -6 pending, and the reported totals include it: gain `60`, loss `54 + 6 = 60`. Loss equals gain and is not 0.

## Remaining values

- durationMs: 06:00:00 to 06:06:00, `360000`.
- pointCount: `121`.
- Bounding box, from the written strings: minLat 47.3, maxLat 47.30216, minLon 11.4, maxLon 11.4. Centroid: longitude 11.4, latitude the mean of the 121 latitudes, `47.3 + 0.000036 * 3600 / 121 = 47.30107107438017`.
- trail.lat/lon: the first point, 47.3, 11.4. trail.date: `2024-07-20`. trail.name: `Out And Back Descent`. trail.description: the metadata description. No waypoints.

No expected value came from executing the implementation under test.
