import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show UniqueKey;
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

  /// Pushes the current (pre-mutation) anchors/segments onto the undo stack
  /// and clears the redo stack (D-11: any new action clears redo). Called
  /// FIRST, before mutating, by every mutation method below.
  void _pushUndo() {
    state = state.copyWith(
      undoStack: [
        ...state.undoStack,
        RouteAnchorsSnapshot(anchors: state.anchors, segments: state.segments),
      ],
      redoStack: const [],
    );
  }

  /// WAYP-01: appends a new anchor to the end of the route. If a previous
  /// last anchor existed, also creates the connecting segment (straight by
  /// default, auto-resolved via Valhalla if auto-routing is on).
  void appendAnchor(Geographic point) {
    _pushUndo();

    final newAnchor = RouteAnchor(
      id: UniqueKey().toString(),
      lat: point.lat,
      lon: point.lon,
    );
    final previousLast = state.anchors.isNotEmpty ? state.anchors.last : null;

    state = state.copyWith(anchors: [...state.anchors, newAnchor]);

    if (previousLast != null) {
      final newSegment = RouteSegment(
        beforeAnchorId: previousLast.id,
        afterAnchorId: newAnchor.id,
        polyline: [previousLast.point, newAnchor.point],
        state: SegmentState.straight,
      );
      state = state.copyWith(segments: [...state.segments, newSegment]);

      if (state.autoRoutingEnabled) {
        _resolveSegment(
          previousLast.id,
          newAnchor.id,
          previousLast,
          newAnchor,
        ).ignore();
      }
    }
  }

  /// WAYP-02: repositions [anchorId] and re-resolves only its ≤2 adjacent
  /// segments. Invoked once per drag gesture at `onPanEnd` (D-05), never
  /// during `onPanUpdate`.
  void dragAnchor(String anchorId, Geographic newPoint) {
    _pushUndo();

    final anchors = [
      for (final a in state.anchors)
        if (a.id == anchorId)
          a.copyWith(lat: newPoint.lat, lon: newPoint.lon)
        else
          a,
    ];
    state = state.copyWith(anchors: anchors);

    final anchorsById = {for (final a in anchors) a.id: a};
    final touched = state.segments.where(
      (s) => s.beforeAnchorId == anchorId || s.afterAnchorId == anchorId,
    );

    for (final segment in touched) {
      final a = anchorsById[segment.beforeAnchorId]!;
      final b = anchorsById[segment.afterAnchorId]!;
      if (state.autoRoutingEnabled) {
        _resolveSegment(
          segment.beforeAnchorId,
          segment.afterAnchorId,
          a,
          b,
        ).ignore();
      } else {
        _applySegment(
          segmentKey(segment.beforeAnchorId, segment.afterAnchorId),
          points: [a.point, b.point],
          segmentState: SegmentState.straight,
        );
      }
    }
  }

  /// WAYP-03: on a plain (non-blocked) segment tap, geometrically splits it
  /// and inserts the new anchor between its two endpoint anchors — never
  /// calls Valhalla (Open Question 1, resolved). The new anchor's list
  /// position IS its D-02 display-number renumbering; numbers are always
  /// derived from index, never stored.
  void insertAnchorOnSegment(
    String beforeAnchorId,
    String afterAnchorId,
    Geographic tapPoint,
  ) {
    _pushUndo();

    final key = segmentKey(beforeAnchorId, afterAnchorId);
    final targetSegment = state.segments.firstWhere(
      (s) => segmentKey(s.beforeAnchorId, s.afterAnchorId) == key,
    );

    final newAnchor = RouteAnchor(
      id: UniqueKey().toString(),
      lat: tapPoint.lat,
      lon: tapPoint.lon,
    );

    final (first, second) = splitSegmentAt(targetSegment, newAnchor.id, tapPoint);

    final beforeIndex = state.anchors.indexWhere((a) => a.id == beforeAnchorId);
    final insertAt = beforeIndex + 1;
    final anchors = [
      ...state.anchors.sublist(0, insertAt),
      newAnchor,
      ...state.anchors.sublist(insertAt),
    ];

    final segments = [
      for (final s in state.segments)
        if (segmentKey(s.beforeAnchorId, s.afterAnchorId) == key) ...[
          first,
          second,
        ] else
          s,
    ];

    state = state.copyWith(anchors: anchors, segments: segments);
  }

  /// ROUTE-04: restores the immediately prior anchors/segments snapshot.
  /// No-ops if the undo stack is empty (defensive — the app-bar button is
  /// also disabled per D-11).
  void undo() {
    final stack = state.undoStack;
    if (stack.isEmpty) return;

    final previous = stack.last;
    final redoSnapshot = RouteAnchorsSnapshot(
      anchors: state.anchors,
      segments: state.segments,
    );
    state = state.copyWith(
      anchors: previous.anchors,
      segments: previous.segments,
      undoStack: stack.sublist(0, stack.length - 1),
      redoStack: [...state.redoStack, redoSnapshot],
    );
  }

  /// ROUTE-04: re-applies the most recently undone mutation. No-ops if the
  /// redo stack is empty.
  void redo() {
    final stack = state.redoStack;
    if (stack.isEmpty) return;

    final next = stack.last;
    final undoSnapshot = RouteAnchorsSnapshot(
      anchors: state.anchors,
      segments: state.segments,
    );
    state = state.copyWith(
      anchors: next.anchors,
      segments: next.segments,
      redoStack: stack.sublist(0, stack.length - 1),
      undoStack: [...state.undoStack, undoSnapshot],
    );
  }
}
