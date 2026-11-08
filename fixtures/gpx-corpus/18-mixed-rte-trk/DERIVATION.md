# Derivation: 18-mixed-rte-trk

The file has a `<rte>` (three points near Sydney, first in the file) and a `<trk>` (four points in Tyrol). Any `<trk>` element means the file is measured from its tracks only, so the route contributes nothing: not to the distance, elevation, duration, point count, bounding box, centroid, start point, date or name. Nothing carries a `<metadata><name>`, so the name comes from the first track.

## Distance

Haversine, `R = 6371000`, evaluated in a scratch Python script outside the repository, over the four track points:

| Hop | Metres |
|-----|--------|
| 1 (47.2000, 11.3000 to 47.2008, 11.3006) | 99.83967439825399 |
| 2 (to 47.2016, 11.3012) | 99.83936406713745 |
| 3 (to 47.2024, 11.3018) | 99.83905373629277 |

Total: `299.5180922016842`.

## Elevation

Hand-traced from the documented defer-then-publish rules (threshold 5 m). The first point (ele 1500) is the anchor.

| Point | Ele | Jump from last filtered | Decision |
|-------|-----|-------------------------|----------|
| 1 | 1507 | +7 | nothing pending: becomes pending (+7) |
| 2 | 1514 | +7 | same sign, cannot cancel: publishes the +7 (gain 7), becomes pending (+7) |
| 3 | 1509 | -5 | opposite sign and within 5 m of the pending anchor (1507), but 99.8 m from the last filtered point: no cancel. Publishes the +7 (gain 14), becomes pending (-5) |

The track ends with the -5 still pending, and the reported totals include it: gain `14`, loss `5`.

## Remaining values

- durationMs: 07:00:00 to 07:01:30, `90000`. The route's 2023 times play no part.
- pointCount: `4`.
- Bounding box: minLat 47.2, maxLat 47.2024, minLon 11.3, maxLon 11.3018. Centroid: mean latitude 47.2012, mean longitude 11.3009. No route latitude (about -33.85) or longitude (151.2) appears anywhere.
- trail.lat/lon: the first track point, 47.2, 11.3. trail.date: `2024-06-02`. trail.name: `Mixed Track Wins`. trail.description: null. No waypoints.

No expected value came from executing the implementation under test.
