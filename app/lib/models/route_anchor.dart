import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:maplibre/maplibre.dart' show Geographic;

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
  }) = _RouteSegment;
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
