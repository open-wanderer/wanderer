import 'dart:math' as math;

import 'package:latlong2/latlong.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';

part 'navigation_provider.g.dart';

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

/// Immutable snapshot of in-progress navigation state.
///
/// D-19: [breadcrumb] is session-only in-memory state. It is never persisted
/// to disk and is never written to GPX. Because the notifier is family-keyed
/// on [NavigateResponse], the entire breadcrumb list is discarded when the
/// provider family entry is disposed on screen exit.
class NavigationState {
  const NavigationState({
    required this.response,
    required this.currentManeuverIndex,
    required this.breadcrumb,
  });

  final NavigateResponse response;

  /// Index into [NavigateResponse.maneuvers] for the maneuver currently being
  /// executed. Always forward-only; never decrements.
  final int currentManeuverIndex;

  /// Actual GPS path traveled during this navigation session (D-18).
  ///
  /// Session-only; discarded on screen exit (D-19).
  final List<LatLng> breadcrumb;

  NavigationState copyWith({
    NavigateResponse? response,
    int? currentManeuverIndex,
    List<LatLng>? breadcrumb,
  }) {
    return NavigationState(
      response: response ?? this.response,
      currentManeuverIndex: currentManeuverIndex ?? this.currentManeuverIndex,
      breadcrumb: breadcrumb ?? this.breadcrumb,
    );
  }
}

// ---------------------------------------------------------------------------
// Notifier
// ---------------------------------------------------------------------------

/// Riverpod notifier that owns the live navigation state for a single
/// [NavigateResponse] session.
///
/// Family-keyed on [response] so each navigation session is isolated and
/// testable via a plain [ProviderContainer] (D-17).
@riverpod
class Navigation extends _$Navigation {
  /// Distance threshold (meters) from the next maneuver's begin-shape point at
  /// which [currentManeuverIndex] is advanced by 1 (D-12 / NAV-06).
  ///
  /// Named constant so it can be tuned in one place.
  static const _kManeuverAdvanceThresholdMeters = 30.0;

  /// latlong2 geodesic-distance primitive (same instance used in gpx_util.dart).
  final _distance = const Distance();

  /// Cumulative along-track Haversine distance (meters) from the route start to
  /// each entry in [NavigateResponse.shapeAsLatLng].
  ///
  /// `_shapeCumulativeMeters[i]` is the total path length walking
  /// `shapeAsLatLng[0] -> shapeAsLatLng[i]`. Precomputed once in [build] per the
  /// constraint that the table is never recomputed on a per-GPS-fix basis.
  late final List<double> _shapeCumulativeMeters;

  /// Cumulative along-track distance (meters) to each maneuver's begin-shape
  /// point. `_maneuverCumulativeMeters[m]` equals
  /// `_shapeCumulativeMeters[clampedBeginShapeIndex(m)]`.
  ///
  /// Precomputed once in [build]; used by [onPosition] for forward maneuver
  /// scanning against a projected along-track distance.
  late final List<double> _maneuverCumulativeMeters;

  @override
  NavigationState build(NavigateResponse response) {
    final shape = response.shapeAsLatLng;

    if (shape.isEmpty) {
      // Empty-shape guard: nothing to project against.
      _shapeCumulativeMeters = const [];
      _maneuverCumulativeMeters = const [];
    } else {
      // (1) Cumulative path length to each shape vertex.
      final shapeCumulative = List<double>.filled(shape.length, 0.0);
      for (var i = 1; i < shape.length; i++) {
        shapeCumulative[i] = shapeCumulative[i - 1] +
            _distance.as(LengthUnit.Meter, shape[i - 1], shape[i]);
      }
      _shapeCumulativeMeters = shapeCumulative;

      // (2) Cumulative distance to each maneuver's (clamped) begin-shape point.
      _maneuverCumulativeMeters = response.maneuvers.map((m) {
        final clamped = m.beginShapeIndex.clamp(0, shape.length - 1).toInt();
        return shapeCumulative[clamped];
      }).toList(growable: false);
    }

    return NavigationState(
      response: response,
      currentManeuverIndex: 0,
      breadcrumb: const [],
    );
  }

  /// Forward-only nearest-segment projection of [pos] onto the route polyline.
  ///
  /// Iterates segments `(shape[i], shape[i + 1])` starting at [fromShapeIndex]
  /// (so the projection can never snap back to an earlier leg the route doubles
  /// back near — research Failure mode B mitigation). For each segment the
  /// scalar projection parameter `t` of [pos] onto the segment is computed in a
  /// local equirectangular frame (lat/lon deltas converted to meters), clamped
  /// to `[0, 1]`, and the cross-track distance to the projected foot is
  /// measured. The segment with the smallest cross-track distance wins.
  ///
  /// Returns the along-track distance from the route start to the projected
  /// foot: `_shapeCumulativeMeters[bestSegmentStart] + t * segmentLengthMeters`.
  double _projectAlongTrack(LatLng pos, List<LatLng> shape, int fromShapeIndex) {
    if (shape.length < 2) {
      // Degenerate route: only the start vertex (if any) is reachable.
      return shape.isEmpty ? 0.0 : _shapeCumulativeMeters[0];
    }

    const metersPerDegree = 111320.0;

    final start = fromShapeIndex.clamp(0, shape.length - 2).toInt();

    var bestCrossTrack = double.infinity;
    var bestSegmentStart = start;
    var bestT = 0.0;
    var bestSegmentMeters = 0.0;

    for (var i = start; i < shape.length - 1; i++) {
      final a = shape[i];
      final b = shape[i + 1];

      // Local equirectangular projection around the segment midpoint.
      final midLatRad = ((a.latitude + b.latitude) / 2.0) * math.pi / 180.0;
      final cosMidLat = math.cos(midLatRad);

      // Segment vector (meters).
      final abX = (b.longitude - a.longitude) * metersPerDegree * cosMidLat;
      final abY = (b.latitude - a.latitude) * metersPerDegree;

      // Vector from segment start to the position (meters).
      final apX = (pos.longitude - a.longitude) * metersPerDegree * cosMidLat;
      final apY = (pos.latitude - a.latitude) * metersPerDegree;

      final segLenSq = abX * abX + abY * abY;

      double t;
      if (segLenSq <= 0.0) {
        // Zero-length segment: project onto its start vertex.
        t = 0.0;
      } else {
        t = (apX * abX + apY * abY) / segLenSq;
        t = t.clamp(0.0, 1.0);
      }

      // Cross-track distance from pos to the projected foot (meters).
      final footX = apX - t * abX;
      final footY = apY - t * abY;
      final crossTrack = math.sqrt(footX * footX + footY * footY);

      if (crossTrack < bestCrossTrack) {
        bestCrossTrack = crossTrack;
        bestSegmentStart = i;
        bestT = t;
        // Use the geodesic segment length for consistency with the cumulative
        // table (which is built from _distance.as).
        bestSegmentMeters = _distance.as(LengthUnit.Meter, a, b);
      }
    }

    return _shapeCumulativeMeters[bestSegmentStart] +
        bestT * bestSegmentMeters;
  }

  /// Called on each GPS position event.
  ///
  /// Per call this method:
  /// 1. Appends [pos] to the in-memory breadcrumb (D-18).
  /// 2. Projects [pos] orthogonally onto the route polyline (via
  ///    [_projectAlongTrack]) to obtain an along-track distance from the route
  ///    start, then advances [currentManeuverIndex] to the highest maneuver
  ///    whose cumulative along-track distance is at or before that projection
  ///    (within [_kManeuverAdvanceThresholdMeters] of along-track lookahead).
  ///    This lets a single fix skip past every maneuver the user shortcut past
  ///    (research Failure mode A), unlike the old single-next-waypoint
  ///    proximity check.
  /// 3. Advancement is forward-only — [currentManeuverIndex] never decrements
  ///    (D-12 / T-02-02) and never exceeds the last maneuver (D-14).
  void onPosition(LatLng pos) {
    // (1) Append position to breadcrumb (D-18 / NAV-08).
    state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);

    // (2) Empty-shape / uninitialized-table guard (Pitfall 3): nothing to
    //     project against.
    final shape = state.response.shapeAsLatLng;
    if (shape.isEmpty || _maneuverCumulativeMeters.isEmpty) return;

    // (3) Forward-only scan starting at the current maneuver's begin-shape
    //     index so the projection cannot snap to an earlier leg the route
    //     doubles back near (research Failure mode B mitigation).
    final maneuvers = state.response.maneuvers;
    final current = state.currentManeuverIndex;
    final fromShapeIndex = maneuvers[current]
        .beginShapeIndex
        .clamp(0, shape.length - 1)
        .toInt();

    final atd = _projectAlongTrack(pos, shape, fromShapeIndex);

    // (4) Advance to the highest maneuver reached by the projected along-track
    //     distance. Maneuvers are ordered by cumulative distance, so we can
    //     break on the first not-yet-reached one.
    var newIndex = current;
    final lastIndex = maneuvers.length - 1;
    for (var i = current + 1; i <= lastIndex; i++) {
      if (_maneuverCumulativeMeters[i] <=
          atd + _kManeuverAdvanceThresholdMeters) {
        newIndex = i;
      } else {
        break;
      }
    }

    // (5) Forward-only assignment (never decrement, never exceed last).
    if (newIndex > current) {
      state = state.copyWith(currentManeuverIndex: newIndex);
    }
  }
}
