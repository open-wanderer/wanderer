# Derivation: 17-route-only-promotion

The file has no `<trk>` element, so it is measured from its two `<rte>` elements, one segment per route. Route 1's first `<rtept>` has no `lon`; it is dropped before anything is measured, which leaves six usable points (three per route). Nothing carries a `<metadata><name>`, so the trail name comes from the first route.

## Distance

Haversine, `R = 6371000`, evaluated in a scratch Python script outside the repository, over the six usable points in file order. The hop from the last point of route 1 into the first point of route 2 is counted (one shared meter, no reset, as in fixture 09).

| Hop | From | To | Metres |
|-----|------|----|--------|
| 1 | 47.1000, 11.2000 | 47.1010, 11.2000 | 111.1949266442689 |
| 2 | 47.1010, 11.2000 | 47.1020, 11.2000 | 111.1949266442689 |
| 3 (route 1 into route 2) | 47.1020, 11.2000 | 47.1020, 11.2015 | 113.5347955864054 |
| 4 | 47.1020, 11.2015 | 47.1010, 11.2015 | 111.1949266442689 |
| 5 | 47.1010, 11.2015 | 47.1000, 11.2015 | 111.1949266442689 |

Total: `558.314502163481`.

## Elevation

Hand-traced from the documented defer-then-publish rules (threshold 5 m; a jump cancels only if it has the opposite sign, returns within 5 m of the pending anchor elevation, and lies within 5 m horizontally of the last filtered point). The first usable point (ele 1000) is the anchor.

| Point | Ele | Jump from last filtered | Decision |
|-------|-----|-------------------------|----------|
| 1 | 1010 | +10 | nothing pending: becomes pending (+10) |
| 2 | 1020 | +10 | same sign, cannot cancel: publishes the +10 (gain 10), becomes pending (+10) |
| 3 | 1012 | -8 | opposite sign and within 5 m of the pending anchor (1010), but 113.5 m from the last filtered point: no cancel. Publishes the +10 (gain 20), becomes pending (-8) |
| 4 | 1004 | -8 | same sign: publishes the -8 (loss 8), becomes pending (-8) |
| 5 | 996 | -8 | same sign: publishes the -8 (loss 16), becomes pending (-8) |

The track ends with the -8 still pending, and the reported totals include it: gain `20`, loss `16 + 8 = 24`.

## Remaining values

- durationMs: route 1 spans 08:00:00 to 08:02:00 (120 s), route 2 spans 09:00:00 to 09:02:00 (120 s), 240 s in total: `240000`. The positionless point's 23:59 time is dropped with it. The gap between the routes is not counted.
- pointCount: `6`.
- Bounding box: minLat 47.1, maxLat 47.102, minLon 11.2, maxLon 11.2015. Centroid: mean latitude 47.101, mean longitude 11.20075.
- trail.lat/lon: the first usable point, 47.1, 11.2. trail.date: `2024-05-01`, from the first usable point's time.
- trail.name: `Route Only North`. trail.description: null. One waypoint, `Route Camp`, with icon `campground`.

No expected value came from executing the implementation under test.
