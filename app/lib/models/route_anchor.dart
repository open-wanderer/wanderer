import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:maplibre/maplibre.dart' show Geographic, SphericalGreatCircle;
import 'package:wanderer/util/reverse_geocode_util.dart';

part 'route_anchor.freezed.dart';

/// A single tap/drag point along an in-progress, unsaved route plan.
///
/// This is a deliberately distinct, new in-memory type (D-01) — it does not
/// reuse the model backing a persisted `Trail`'s route points, which is a
/// different data type/concept. `RouteAnchor` backs the route planner's
/// in-memory state only.
@freezed
abstract class RouteAnchor with _$RouteAnchor {
  const factory RouteAnchor({
    required String id, // generated via UniqueKey().toString() at creation
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

    /// Estimated travel time for this segment, in seconds. Valhalla's own
    /// `trip.summary.time` when [state] is [SegmentState.routed]; `null` for
    /// a [SegmentState.straight]/[SegmentState.blocked] segment, which never
    /// made a successful Valhalla call — callers estimate those from
    /// distance + travel profile instead (`valhalla_util.dart`'s
    /// `estimateSegmentDurationSeconds`).
    double? durationSeconds,

    /// A downsampled (`buildNavShape`-capped) subset of [polyline], used only
    /// as the point set queried against `/valhalla/height`. Deliberately
    /// distinct from [polyline] (which drives map rendering at full
    /// resolution) — `null` until [elevations] has been resolved for the
    /// current [polyline].
    List<Geographic>? elevationProfile,

    /// Heights (meters), one per [elevationProfile] point, fetched from
    /// `/valhalla/height` by `route_anchor_provider.dart`'s
    /// `_resolveElevation` — fired fire-and-forget after every segment
    /// creation/update so it never blocks the segment's own polyline from
    /// rendering. `null` until that fetch resolves.
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
  /// deltas between consecutive [elevations] samples. `0` while
  /// [elevations] is still unresolved (a fresh segment, mid-fetch) rather
  /// than blocking on the height lookup.
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
