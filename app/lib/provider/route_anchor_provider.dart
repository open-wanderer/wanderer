import 'package:dio/dio.dart';
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/polyline_util.dart';
import 'package:wanderer/util/route_segment_util.dart';

part 'route_anchor_provider.g.dart';

class RouteAnchorsState {
  const RouteAnchorsState({
    required this.anchors,
    required this.segments,
    required this.autoRoutingEnabled,
    required this.travelProfile,
    required this.undoStack,
    required this.redoStack,
  });

  final List<RouteAnchor> anchors;
  final List<RouteSegment> segments;
  final bool autoRoutingEnabled;

  /// Fixed for the notifier's lifetime (D-07) — `'pedestrian'` or `'bicycle'`,
  /// set once via the family argument, never switched mid-session.
  final String travelProfile;
  final List<RouteAnchorsSnapshot> undoStack;
  final List<RouteAnchorsSnapshot> redoStack;

  RouteAnchorsState copyWith({
    List<RouteAnchor>? anchors,
    List<RouteSegment>? segments,
    bool? autoRoutingEnabled,
    String? travelProfile,
    List<RouteAnchorsSnapshot>? undoStack,
    List<RouteAnchorsSnapshot>? redoStack,
  }) {
    return RouteAnchorsState(
      anchors: anchors ?? this.anchors,
      segments: segments ?? this.segments,
      autoRoutingEnabled: autoRoutingEnabled ?? this.autoRoutingEnabled,
      travelProfile: travelProfile ?? this.travelProfile,
      undoStack: undoStack ?? this.undoStack,
      redoStack: redoStack ?? this.redoStack,
    );
  }
}

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
/// argument fixed for the notifier's lifetime (D-07).
@riverpod
class RouteAnchors extends _$RouteAnchors {
  // Non-reactive bookkeeping — never in `state`, mirrors `_currentShapeIndex`
  // in `navigation_provider.dart`. Keyed by the stable `segmentKey`
  // (anchor-id pair), never array index (Pitfall 3).
  final Map<String, CancelToken> _inFlight = {};
  final Map<String, int> _generation = {};

  @override
  RouteAnchorsState build(String travelProfile) {
    return RouteAnchorsState(
      anchors: const [],
      segments: const [],
      autoRoutingEnabled: true,
      travelProfile: travelProfile,
      undoStack: const [],
      redoStack: const [],
    );
  }

  bool _isValidCoordinate(RouteAnchor anchor) {
    return anchor.lat >= -90 &&
        anchor.lat <= 90 &&
        anchor.lon >= -180 &&
        anchor.lon <= 180;
  }

  /// Resolves the segment between [a] and [b] via Valhalla, guarded by a
  /// per-segment `CancelToken` + monotonically increasing generation
  /// counter so an out-of-order (superseded) response is discarded rather
  /// than applied over a newer request's result.
  ///
  /// On failure (non-cancel `DioException`, or a malformed response), the
  /// segment is marked [SegmentState.blocked] and its prior polyline is left
  /// untouched — ROUTE-05 forbids ever silently falling back to a straight
  /// line while auto-routing is on.
  Future<void> _resolveSegment(
    String beforeAnchorId,
    String afterAnchorId,
    RouteAnchor a,
    RouteAnchor b,
  ) async {
    final key = segmentKey(beforeAnchorId, afterAnchorId);

    if (!_isValidCoordinate(a) || !_isValidCoordinate(b)) {
      // V5: reject out-of-range coordinates before ever calling Valhalla.
      _markBlocked(key);
      return;
    }

    _inFlight[key]?.cancel();
    final token = CancelToken();
    _inFlight[key] = token;
    final myGeneration = (_generation[key] ?? 0) + 1;
    _generation[key] = myGeneration;

    try {
      final response = await ref.read(apiProvider).post(
        '/valhalla/route',
        data: {
          'directions_type': 'none',
          'locations': [
            {'lat': a.lat, 'lon': a.lon},
            {'lat': b.lat, 'lon': b.lon},
          ],
          'costing': state.travelProfile,
        },
        cancelToken: token,
      );

      // Stale-response guard: a newer request may have started (and even
      // completed) while this one was in flight.
      if (_generation[key] != myGeneration) return;

      final trip = response.data is Map ? response.data['trip'] : null;
      final legs = trip is Map ? trip['legs'] : null;
      final leg = (legs is List && legs.isNotEmpty) ? legs[0] : null;
      final shape = leg is Map ? leg['shape'] : null;

      if (shape is! String) {
        _markBlocked(key);
        return;
      }

      final points = PolylineUtil.decode(shape, precision: 6);
      _applySegment(key, points: points, segmentState: SegmentState.routed);
    } on DioException catch (e) {
      if (e.type == DioExceptionType.cancel) return; // superseded, not a real failure
      if (_generation[key] != myGeneration) return;
      _markBlocked(key); // ROUTE-05: never revert to straight, never clear
    }
  }

  void _markBlocked(String key) {
    final segments = [
      for (final segment in state.segments)
        if (segmentKey(segment.beforeAnchorId, segment.afterAnchorId) == key)
          segment.copyWith(state: SegmentState.blocked)
        else
          segment,
    ];
    state = state.copyWith(segments: segments);
  }

  void _applySegment(
    String key, {
    required List<Geographic> points,
    required SegmentState segmentState,
  }) {
    final segments = [
      for (final segment in state.segments)
        if (segmentKey(segment.beforeAnchorId, segment.afterAnchorId) == key)
          segment.copyWith(polyline: points, state: segmentState)
        else
          segment,
    ];
    state = state.copyWith(segments: segments);
  }

  /// D-09: retry lives on the blocked segment itself. Re-dispatches
  /// [_resolveSegment] for the given anchor pair.
  Future<void> retrySegment(String beforeAnchorId, String afterAnchorId) {
    final a = state.anchors.firstWhere((x) => x.id == beforeAnchorId);
    final b = state.anchors.firstWhere((x) => x.id == afterAnchorId);
    return _resolveSegment(beforeAnchorId, afterAnchorId, a, b);
  }

  /// ROUTE-01/02: flips [RouteAnchorsState.autoRoutingEnabled].
  ///
  /// Turning auto-routing OFF leaves every EXISTING segment's polyline and
  /// state exactly as-is — no Valhalla calls, no straight-line rewrite of
  /// what's already on screen. Only segments created AFTER this point (via
  /// `appendAnchor`/`dragAnchor`/`insertAnchorOnSegment` while auto-routing
  /// is off) become straight 2-point lines — see the must_haves truth in
  /// this plan (corrected from this task's original prose; documented as a
  /// deviation in 19-02-SUMMARY.md). Turning it back ON re-resolves every
  /// existing segment via Valhalla in parallel (`Future.wait`, not
  /// sequential awaits).
  Future<void> toggleAutoRouting() async {
    final enabled = !state.autoRoutingEnabled;
    state = state.copyWith(autoRoutingEnabled: enabled);

    if (!enabled) return;

    final anchorsById = {for (final a in state.anchors) a.id: a};
    await Future.wait([
      for (final segment in state.segments)
        _resolveSegment(
          segment.beforeAnchorId,
          segment.afterAnchorId,
          anchorsById[segment.beforeAnchorId]!,
          anchorsById[segment.afterAnchorId]!,
        ),
    ]);
  }
}
