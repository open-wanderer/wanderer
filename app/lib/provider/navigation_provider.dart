import 'package:collection/collection.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/util/route/map_matcher.dart';

part 'navigation_provider.g.dart';

class NavigationState {
  const NavigationState({
    required this.response,
    required this.currentManeuverIndex,
    required this.breadcrumb,
    required this.breadcrumbLength,
  });

  final NavigateResponse response;

  /// Never decrements — advancement is forward-only.
  final int currentManeuverIndex;

  /// Session-only; discarded on screen exit (never persisted to disk or GPX).
  ///
  /// A stable unmodifiable VIEW over the notifier's internal grow-in-place
  /// list — the same object across every state emission, so appends are O(1)
  /// instead of an O(n) copy per GPS fix (a multi-hour hike made the old
  /// spread-copy quadratic). Consumers must key change detection on
  /// [breadcrumbLength], never on this list's identity.
  final List<Wpt> breadcrumb;

  /// Change signal for [breadcrumb] — compare this across emissions to detect
  /// appends (the list itself keeps a stable identity by design).
  final int breadcrumbLength;

  NavigationState copyWith({
    NavigateResponse? response,
    int? currentManeuverIndex,
    List<Wpt>? breadcrumb,
    int? breadcrumbLength,
  }) {
    return NavigationState(
      response: response ?? this.response,
      currentManeuverIndex: currentManeuverIndex ?? this.currentManeuverIndex,
      breadcrumb: breadcrumb ?? this.breadcrumb,
      breadcrumbLength: breadcrumbLength ?? this.breadcrumbLength,
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

  /// Whether the route has any shape at all — precomputed in [build] so
  /// [onPosition] never re-materializes `shapeAsGeographic` (an O(n) list
  /// allocation) per GPS fix just for an emptiness check.
  bool _hasShape = false;

  /// Internal grow-in-place backing list for [NavigationState.breadcrumb] —
  /// appends mutate this list directly; the state only ever exposes one
  /// stable [UnmodifiableListView] over it (see [NavigationState.breadcrumb]).
  List<Wpt> _breadcrumb = [];

  @override
  NavigationState build(
    NavigateResponse response, {
    int? resumeManeuverIndex,
    List<Wpt>? resumeBreadcrumb,
  }) {
    final shape = response.shapeAsGeographic;
    _hasShape = shape.isNotEmpty;

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

    _breadcrumb = [...?resumeBreadcrumb];
    return NavigationState(
      response: response,
      currentManeuverIndex: initialManeuverIndex,
      breadcrumb: UnmodifiableListView(_breadcrumb),
      breadcrumbLength: _breadcrumb.length,
    );
  }

  /// Feeds a GPS fix into the map matcher and advances
  /// [NavigationState.currentManeuverIndex] as far as the matched along-track
  /// distance warrants. [heading]/[headingAccuracy]/[speed]/[accuracy] are
  /// optional; passing them improves the matcher's accuracy.
  ///
  /// [recordBreadcrumb] gates only the breadcrumb append (the data persisted
  /// to disk and exported as the saved trail's GPX) — pass `false` while
  /// navigation is frozen (manually paused or auto-stationary) so paused
  /// intervals are excluded from the saved track. Maneuver-advance detection
  /// below is unrelated bookkeeping and always runs regardless of this flag,
  /// so pausing during turn-by-turn navigation does not stop progress
  /// tracking.
  ///
  /// Returns whether [NavigationState.currentManeuverIndex] advanced — so the
  /// caller (holding a cached notifier reference) never needs a before/after
  /// pair of provider reads, each of which deep-hashes the family argument's
  /// full route shape.
  bool onPosition(
    Geographic pos, {
    double? heading,
    double? headingAccuracy,
    double? speed,
    double? altitude,
    double? accuracy,
    bool recordBreadcrumb = true,
  }) {
    if (recordBreadcrumb) {
      // In-place append + length-only state bump: the view object in
      // `state.breadcrumb` stays identity-stable, listeners key off
      // `breadcrumbLength` (see NavigationState docs).
      _breadcrumb.add(
        Wpt(lat: pos.lat, lon: pos.lon, ele: altitude, time: DateTime.now()),
      );
      state = state.copyWith(breadcrumbLength: _breadcrumb.length);
    }

    if (!_hasShape || _maneuverCumulativeMeters.isEmpty) return false;

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
      return true;
    }
    return false;
  }
}
