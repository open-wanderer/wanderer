import 'dart:math' as math;

import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';

/// A stable, index-independent identity for a [RouteSegment], derived from
/// its bounding anchor ids rather than array position — indices drift under
/// insert/delete/reorder (RESEARCH.md Pitfall 3), but an anchor-id pair does
/// not. Both the routing engine's in-flight/generation race-guard maps
/// (`route_anchor_provider.dart`) and 19-03's GeoJSON feature properties key
/// off this value.
String segmentKey(String beforeAnchorId, String afterAnchorId) =>
    '${beforeAnchorId}_$afterAnchorId';

/// Splits [segment] into two, inserting [newAnchorId] as the shared boundary
/// anchor at the point on the existing polyline closest to [tapPoint].
///
/// Never calls Valhalla — a plain WAYP-03 tap on a routed/straight segment
/// geometrically splits the already-rendered path rather than re-routing
/// (Open Question 1, resolved to the geometric-split strategy). Both halves
/// inherit [segment]'s state unchanged.
///
/// The nearest point is found by projecting [tapPoint] onto every existing
/// sub-edge of the polyline (not just snapping to an existing vertex) so a
/// straight, 2-point (un-routed) segment — the common case for a freshly
/// appended or dragged anchor pair — still splits at a genuine point between
/// its two endpoints rather than degenerating to one of them.
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
/// approximated in a local planar (equirectangular) projection — mirrors the
/// cross-track projection math in `navigation_provider.dart`'s
/// `_projectAlongTrack`, adequate for the short, single-segment distances
/// involved here.
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

  return Geographic(lat: a.lat + t * (b.lat - a.lat), lon: a.lon + t * (b.lon - a.lon));
}
