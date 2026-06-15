import 'dart:math' as math;

import 'package:latlong2/latlong.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';

part 'navigation_provider.g.dart';

class NavigationState {
  const NavigationState({
    required this.response,
    required this.currentManeuverIndex,
    required this.breadcrumb,
  });

  final NavigateResponse response;

  /// Never decrements — advancement is forward-only.
  final int currentManeuverIndex;

  /// Session-only; discarded on screen exit (never persisted to disk or GPX).
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

@riverpod
class Navigation extends _$Navigation {
  static const _kManeuverAdvanceThresholdMeters = 30.0;

  // Never looks backward — advances only as the user moves forward along the
  // route. Stored as a class field (not in state) because the UI doesn't need
  // to know the exact shape segment; only currentManeuverIndex is UI-relevant.
  int _currentShapeIndex = 0;

  final _distance = const Distance();

  /// Precomputed once in [build]; never recomputed per GPS fix.
  /// `_shapeCumulativeMeters[i]` = total Haversine path length from shape[0] to shape[i].
  late final List<double> _shapeCumulativeMeters;

  /// Precomputed once in [build].
  /// `_maneuverCumulativeMeters[m]` = `_shapeCumulativeMeters[clampedBeginShapeIndex(m)]`.
  late final List<double> _maneuverCumulativeMeters;

  @override
  NavigationState build(NavigateResponse response) {
    final shape = response.shapeAsLatLng;

    if (shape.isEmpty) {
      _shapeCumulativeMeters = const [];
      _maneuverCumulativeMeters = const [];
    } else {
      final shapeCumulative = List<double>.filled(shape.length, 0.0);
      for (var i = 1; i < shape.length; i++) {
        shapeCumulative[i] = shapeCumulative[i - 1] +
            _distance.as(LengthUnit.Meter, shape[i - 1], shape[i]);
      }
      _shapeCumulativeMeters = shapeCumulative;

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

  /// Projects [pos] onto the route polyline starting at [fromShapeIndex].
  /// Never looks back; the search window is also capped at 1 km ahead of
  /// [fromShapeIndex] to prevent a loop trail's endpoint segment (which
  /// terminates near the trailhead) from winning on the very first GPS fix.
  /// Returns the along-track distance from the route start to the nearest
  /// projected foot, and the index of the winning shape segment.
  (double, int) _projectAlongTrack(
      LatLng pos, List<LatLng> shape, int fromShapeIndex) {
    if (shape.length < 2) {
      return (shape.isEmpty ? 0.0 : _shapeCumulativeMeters[0], 0);
    }

    const metersPerDegree = 111320.0;
    // Limits how far ahead the search scans. Prevents the loop-trail bug where
    // the endpoint segment (coordinates ≈ trailhead) wins on the first fix.
    const maxLookAheadMeters = 1000.0;

    final start = fromShapeIndex.clamp(0, shape.length - 2).toInt();
    final startCumulative = _shapeCumulativeMeters[start];

    var bestCrossTrack = double.infinity;
    var bestSegmentStart = start;
    var bestT = 0.0;
    var bestSegmentMeters = 0.0;

    for (var i = start; i < shape.length - 1; i++) {
      if (_shapeCumulativeMeters[i] > startCumulative + maxLookAheadMeters) {
        break;
      }

      final a = shape[i];
      final b = shape[i + 1];

      final midLatRad = ((a.latitude + b.latitude) / 2.0) * math.pi / 180.0;
      final cosMidLat = math.cos(midLatRad);

      final abX = (b.longitude - a.longitude) * metersPerDegree * cosMidLat;
      final abY = (b.latitude - a.latitude) * metersPerDegree;

      final apX = (pos.longitude - a.longitude) * metersPerDegree * cosMidLat;
      final apY = (pos.latitude - a.latitude) * metersPerDegree;

      final segLenSq = abX * abX + abY * abY;

      double t;
      if (segLenSq <= 0.0) {
        t = 0.0;
      } else {
        t = (apX * abX + apY * abY) / segLenSq;
        t = t.clamp(0.0, 1.0);
      }

      final footX = apX - t * abX;
      final footY = apY - t * abY;
      final crossTrack = math.sqrt(footX * footX + footY * footY);

      if (crossTrack < bestCrossTrack) {
        bestCrossTrack = crossTrack;
        bestSegmentStart = i;
        bestT = t;
        // Geodesic length keeps this consistent with the cumulative table.
        bestSegmentMeters = _distance.as(LengthUnit.Meter, a, b);
      }
    }

    return (
      _shapeCumulativeMeters[bestSegmentStart] + bestT * bestSegmentMeters,
      bestSegmentStart,
    );
  }

  void onPosition(LatLng pos) {
    state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);

    final shape = state.response.shapeAsLatLng;
    if (shape.isEmpty || _maneuverCumulativeMeters.isEmpty) return;

    final maneuvers = state.response.maneuvers;
    final current = state.currentManeuverIndex;
    // Use the higher of our confirmed shape progress or the current maneuver's
    // declared start — this prevents the projection from scanning segments
    // behind where we've already been confirmed to be.
    final fromShapeIndex = math.max(
      _currentShapeIndex,
      maneuvers[current].beginShapeIndex.clamp(0, shape.length - 1).toInt(),
    );

    final (atd, bestSegment) = _projectAlongTrack(pos, shape, fromShapeIndex);
    _currentShapeIndex = math.max(_currentShapeIndex, bestSegment);

    // Advance to the highest maneuver reached; break on the first not-yet-reached
    // one so a single fix can skip multiple maneuvers if the user shortcutted.
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

    if (newIndex > current) {
      state = state.copyWith(currentManeuverIndex: newIndex);
    }
  }
}
