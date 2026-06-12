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

  @override
  NavigationState build(NavigateResponse response) => NavigationState(
        response: response,
        currentManeuverIndex: 0,
        breadcrumb: const [],
      );

  /// Called on each GPS position event.
  ///
  /// Per call this method:
  /// 1. Appends [pos] to the in-memory breadcrumb (D-18).
  /// 2. Checks whether the user is within [_kManeuverAdvanceThresholdMeters]
  ///    of the next maneuver's begin-shape point and, if so, advances
  ///    [currentManeuverIndex] by 1 (forward-only, D-12).
  /// 3. Returns early without error when already at the last maneuver (D-14).
  void onPosition(LatLng pos) {
    // (1) Append position to breadcrumb (D-18 / NAV-08).
    state = state.copyWith(breadcrumb: [...state.breadcrumb, pos]);

    // (2) Compute index of the next maneuver.
    final next = state.currentManeuverIndex + 1;

    // (3) Completion guard (D-14 / Pitfall 3): already at or past the last
    //     maneuver — do nothing.
    if (next >= state.response.maneuvers.length) return;

    // Clamp beginShapeIndex to valid range (Pitfall 3 out-of-bounds guard,
    // T-02-01 mitigation).
    final rawIndex = state.response.maneuvers[next].beginShapeIndex;
    final clampedIndex =
        rawIndex.clamp(0, state.response.shape.length - 1).toInt();

    final targetLatLng = state.response.shapeAsLatLng[clampedIndex];
    final meters =
        _distance.as(LengthUnit.Meter, pos, targetLatLng);

    // (4) Advance forward-only (T-02-02 mitigation: never decrement).
    if (meters <= _kManeuverAdvanceThresholdMeters) {
      state = state.copyWith(currentManeuverIndex: next);
    }
  }
}
