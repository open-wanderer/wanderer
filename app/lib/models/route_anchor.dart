import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:maplibre/maplibre.dart' show Geographic, SphericalGreatCircle;
import 'package:wanderer/util/geo/reverse_geocode.dart';

part 'route_anchor.freezed.dart';

/// A single tap/drag point along an in-progress, unsaved route plan.
///
/// Deliberately distinct from the model backing a persisted `Trail`'s route
/// points — `RouteAnchor` backs the route planner's in-memory state only.
@freezed
abstract class RouteAnchor with _$RouteAnchor {
  const factory RouteAnchor({
    required String id,
    required double lat,
    required double lon,
    ReverseLocationResult? location,
  }) = _RouteAnchor;

  const RouteAnchor._();

  /// The anchor's position as a MapLibre [Geographic] point.
  Geographic get point => Geographic(lat: lat, lon: lon);
}

/// The resolution state of a [RouteSegment].
enum SegmentState {
  /// Resolved via a Valhalla route call.
  routed,

  /// A direct straight line between the two bounding anchors.
  straight,

  /// Auto-routing was requested but Valhalla could not resolve a path.
  blocked,
}

/// The connecting geometry between two consecutive [RouteAnchor]s.
///
/// Identity is tracked via explicit [beforeAnchorId]/[afterAnchorId] rather
/// than array position, so a segment's identity stays stable across
/// renumber/insert/delete/reorder operations on the anchor list.
@freezed
abstract class RouteSegment with _$RouteSegment {
  const factory RouteSegment({
    required String beforeAnchorId,
    required String afterAnchorId,
    required List<Geographic> polyline,
    required SegmentState state,

    /// Estimated travel time in seconds. Valhalla's `trip.summary.time` when
    /// [state] is [SegmentState.routed]; `null` otherwise (callers estimate
    /// from distance + travel profile instead).
    double? durationSeconds,

    /// A downsampled subset of [polyline] used as the point set queried
    /// against `/valhalla/height`. `null` until [elevations] resolves.
    List<Geographic>? elevationProfile,

    /// Heights (meters), one per [elevationProfile] point. Fetched
    /// fire-and-forget so it never blocks the polyline from rendering.
    /// `null` until that fetch resolves.
    List<double>? elevations,
  }) = _RouteSegment;

  const RouteSegment._();

  /// This segment's own length, in meters — the great-circle sum along
  /// [polyline] (not [elevationProfile], which may be downsampled).
  double get distanceMeters {
    var total = 0.0;
    for (var i = 1; i < polyline.length; i++) {
      total += SphericalGreatCircle(polyline[i - 1]).distanceTo(polyline[i]);
    }
    return total;
  }

  /// This segment's own elevation gain, in meters: the sum of positive
  /// deltas between consecutive [elevations] samples. `0` while unresolved.
  double get elevationGainMeters {
    final elevs = elevations;
    if (elevs == null || elevs.length < 2) return 0;

    var gain = 0.0;
    for (var i = 1; i < elevs.length; i++) {
      final delta = elevs[i] - elevs[i - 1];
      if (delta > 0) gain += delta;
    }
    return gain;
  }
}

/// An immutable, deep snapshot of the full anchor/segment state, used as a
/// single unit on the route planner's undo/redo stacks.
@freezed
abstract class RouteAnchorsSnapshot with _$RouteAnchorsSnapshot {
  const factory RouteAnchorsSnapshot({
    required List<RouteAnchor> anchors,
    required List<RouteSegment> segments,
  }) = _RouteAnchorsSnapshot;
}
