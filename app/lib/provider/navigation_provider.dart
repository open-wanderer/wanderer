import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/util/route_map_matcher.dart';

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
  final List<Geographic> breadcrumb;

  NavigationState copyWith({
    NavigateResponse? response,
    int? currentManeuverIndex,
    List<Geographic>? breadcrumb,
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

  /// Resolves each GPS fix's position along the route. See [RouteMapMatcher].
  late final RouteMapMatcher _matcher;

  /// Precomputed once in [build]; never recomputed per GPS fix.
  /// `_shapeCumulativeMeters[i]` = total Haversine path length from shape[0] to shape[i].
  late final List<double> _shapeCumulativeMeters;

  /// Precomputed once in [build].
  /// `_maneuverCumulativeMeters[m]` = `_shapeCumulativeMeters[clampedBeginShapeIndex(m)]`.
  late final List<double> _maneuverCumulativeMeters;

  @override
  NavigationState build(
    NavigateResponse response, {
    int? resumeManeuverIndex,
    List<Geographic>? resumeBreadcrumb,
  }) {
    final shape = response.shapeAsGeographic;

    if (shape.isEmpty) {
      _shapeCumulativeMeters = const [];
      _maneuverCumulativeMeters = const [];
    } else {
      final shapeCumulative = List<double>.filled(shape.length, 0.0);
      for (var i = 1; i < shape.length; i++) {
        final calculator = SphericalGreatCircle(shape[i - 1]);
        double distanceInMeters = calculator.distanceTo(shape[i]);

        shapeCumulative[i] = shapeCumulative[i - 1] + distanceInMeters;
      }
      _shapeCumulativeMeters = shapeCumulative;

      _maneuverCumulativeMeters = response.maneuvers
          .map((m) {
            final clamped = m.beginShapeIndex
                .clamp(0, shape.length - 1)
                .toInt();
            return shapeCumulative[clamped];
          })
          .toList(growable: false);
    }

    var initialManeuverIndex = 0;
    double? seedAlongTrackMeters;
    if (resumeManeuverIndex != null) {
      initialManeuverIndex = resumeManeuverIndex;
      final maneuvers = response.maneuvers;
      if (maneuvers.isNotEmpty && shape.isNotEmpty) {
        final clampedManeuver = resumeManeuverIndex.clamp(
          0,
          maneuvers.length - 1,
        );
        seedAlongTrackMeters = _maneuverCumulativeMeters[clampedManeuver];
      }
    }

    _matcher = RouteMapMatcher(
      shape: shape,
      shapeCumulativeMeters: _shapeCumulativeMeters,
      initialAlongTrackMeters: seedAlongTrackMeters,
    );

    return NavigationState(
      response: response,
      currentManeuverIndex: initialManeuverIndex,
      breadcrumb: resumeBreadcrumb ?? const [],
    );
  }

  /// Feeds a GPS fix into the map matcher and advances
  /// [NavigationState.currentManeuverIndex] as far as the matched along-track
  /// distance warrants. [heading]/[headingAccuracy]/[speed]/[accuracy] are
  /// optional; passing them improves the matcher's accuracy.
  void onPosition(
    Geographic pos, {
    double? heading,
    double? headingAccuracy,
    double? speed,
    double? accuracy,
  }) {
    state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);

    final shape = state.response.shapeAsGeographic;
    if (shape.isEmpty || _maneuverCumulativeMeters.isEmpty) return;

    final result = _matcher.update(
      pos: pos,
      heading: heading,
      headingAccuracy: headingAccuracy,
      speed: speed,
      accuracy: accuracy,
    );
    final atd = result.alongTrackMeters;

    final maneuvers = state.response.maneuvers;
    final current = state.currentManeuverIndex;

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
