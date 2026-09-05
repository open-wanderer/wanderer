import 'dart:convert';
import 'dart:math' as math;

import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';

/// A stable, index-independent identity for a [RouteSegment], derived from
/// its bounding anchor ids rather than array position — indices drift under
/// insert/delete/reorder, but an anchor-id pair does not.
String segmentKey(String beforeAnchorId, String afterAnchorId) =>
    '${beforeAnchorId}_$afterAnchorId';

/// Serializes [segments] into a GeoJSON `FeatureCollection` string for
/// [RouteSegmentLayer]'s `GeoJsonSource`.
///
/// Each feature's `properties` carries [RouteSegment.beforeAnchorId]/
/// [RouteSegment.afterAnchorId] directly (not an index), so a tap handler
/// can read them without reconstructing a [segmentKey] string. `state` is
/// used as the `filter` discriminator by [RouteSegmentLayer]'s paint variants.
///
/// Coordinates use `[lon, lat]` ordering. An empty [segments] list produces
/// a valid `FeatureCollection` with an empty `features` array.
String buildSegmentsGeoJson(List<RouteSegment> segments) {
  return jsonEncode(<String, Object?>{
    'type': 'FeatureCollection',
    'features': <Map<String, Object?>>[
      for (final s in segments)
        <String, Object?>{
          'type': 'Feature',
          'properties': <String, Object?>{
            'beforeAnchorId': s.beforeAnchorId,
            'afterAnchorId': s.afterAnchorId,
            'state': s.state.name,
          },
          'geometry': <String, Object?>{
            'type': 'LineString',
            'coordinates': <List<double>>[
              for (final p in s.polyline) <double>[p.lon, p.lat],
            ],
          },
        },
    ],
  });
}

/// Splits [segment] into two, inserting [newAnchorId] as the shared boundary
/// anchor at the point on the existing polyline closest to [tapPoint].
///
/// Never calls Valhalla — a tap geometrically splits the already-rendered
/// path rather than re-routing. Both halves inherit [segment]'s state unchanged.
///
/// The nearest point is found by projecting [tapPoint] onto every sub-edge of
/// the polyline (not just snapping to a vertex), so a straight, 2-point
/// segment still splits at a genuine point between its endpoints.
(RouteSegment, RouteSegment) splitSegmentAt(
  RouteSegment segment,
  String newAnchorId,
  Geographic tapPoint,
) {
  final points = segment.polyline;

  var bestIndex = 1;
  var bestDistance = double.infinity;
  var splitPoint = points.length > 1 ? points[1] : tapPoint;

  for (var i = 0; i < points.length - 1; i++) {
    final projected = _closestPointOnSegment(
      points[i],
      points[i + 1],
      tapPoint,
    );
    final distance = SphericalGreatCircle(projected).distanceTo(tapPoint);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = i + 1;
      splitPoint = projected;
    }
  }

  final first = RouteSegment(
    beforeAnchorId: segment.beforeAnchorId,
    afterAnchorId: newAnchorId,
    polyline: [...points.sublist(0, bestIndex), splitPoint],
    state: segment.state,
  );
  final second = RouteSegment(
    beforeAnchorId: newAnchorId,
    afterAnchorId: segment.afterAnchorId,
    polyline: [splitPoint, ...points.sublist(bestIndex)],
    state: segment.state,
  );
  return (first, second);
}

/// The closest point to [p] on the straight line between [a] and [b],
/// approximated via a local planar (equirectangular) projection — adequate
/// for the short, single-segment distances involved here.
Geographic _closestPointOnSegment(Geographic a, Geographic b, Geographic p) {
  const metersPerDegree = 111320.0;
  final midLatRad = ((a.lat + b.lat) / 2.0) * math.pi / 180.0;
  final cosMidLat = math.cos(midLatRad);

  final abX = (b.lon - a.lon) * metersPerDegree * cosMidLat;
  final abY = (b.lat - a.lat) * metersPerDegree;
  final apX = (p.lon - a.lon) * metersPerDegree * cosMidLat;
  final apY = (p.lat - a.lat) * metersPerDegree;

  final segLenSq = abX * abX + abY * abY;
  double t;
  if (segLenSq <= 0.0) {
    t = 0.0;
  } else {
    t = (apX * abX + apY * abY) / segLenSq;
    t = t.clamp(0.0, 1.0);
  }

  return Geographic(
    lat: a.lat + t * (b.lat - a.lat),
    lon: a.lon + t * (b.lon - a.lon),
  );
}
