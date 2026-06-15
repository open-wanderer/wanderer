import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:latlong2/latlong.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/navigation_provider.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Build a [NavigateResponse] with three maneuvers and a known shape so that
/// the 30 m advancement threshold is deterministic in tests.
///
/// Shape layout (latitude increases by ~0.001 ≈ 111 m per step):
///   shape[0] = LatLng(47.000, 9.000)   ← maneuver 0 start
///   shape[3] = LatLng(47.003, 9.000)   ← maneuver 1 begin_shape_index
///   shape[6] = LatLng(47.006, 9.000)   ← maneuver 2 begin_shape_index
NavigateResponse _buildSampleResponse() {
  return NavigateResponse(
    shape: [
      [47.000, 9.000],
      [47.001, 9.000],
      [47.002, 9.000],
      [47.003, 9.000], // index 3 — maneuver 1 target
      [47.004, 9.000],
      [47.005, 9.000],
      [47.006, 9.000], // index 6 — maneuver 2 target
    ],
    maneuvers: const [
      NavigateManeuver(
        instruction: 'Start',
        length: 300.0,
        beginShapeIndex: 0,
        bearing: 0.0,
      ),
      NavigateManeuver(
        instruction: 'Turn right',
        length: 300.0,
        beginShapeIndex: 3,
        bearing: 90.0,
      ),
      NavigateManeuver(
        instruction: 'Arrive',
        length: 0.0,
        beginShapeIndex: 6,
        bearing: 0.0,
      ),
    ],
  );
}

/// A position very close to shape[3] (maneuver 1 target) — within 30 m.
///
/// shape[3] = LatLng(47.003, 9.000). Offset by ~0.0001° lat ≈ 11 m.
const _nearManeuver1 = LatLng(47.00301, 9.000);

/// A position far from shape[3] — clearly > 30 m away.
///
/// shape[3] = LatLng(47.003, 9.000). Offset by 0.01° lat ≈ 1110 m.
const _farFromManeuver1 = LatLng(47.010, 9.000);

/// A position very close to shape[6] (maneuver 2 begin) — within 30 m.
///
/// shape[6] = LatLng(47.006, 9.000). Offset by ~0.00001° lat ≈ 1 m. Its
/// projected along-track distance is ~667 m, past maneuver 1's ~333 m.
const _nearManeuver2 = LatLng(47.00601, 9.000);

/// A position projecting to roughly the route midpoint between maneuver 0
/// (~0 m) and maneuver 1 (~333 m) — about ~178 m along-track. It lies on the
/// route between shape[1] and shape[2] and is > 30 m short of maneuver 1.
const _midpointBeforeManeuver1 = LatLng(47.0016, 9.000);

/// A position exactly on shape[6] (the maneuver 2 vertex). Its along-track
/// projection must equal maneuver 2's cumulative distance (~667 m).
const _exactlyManeuver2 = LatLng(47.006, 9.000);

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('NavigationNotifier', () {
    late ProviderContainer container;
    late NavigateResponse response;

    setUp(() {
      response = _buildSampleResponse();
      container = ProviderContainer();
    });

    tearDown(() {
      container.dispose();
    });

    test('initial state: currentManeuverIndex is 0 and breadcrumb is empty',
        () {
      final state = container.read(navigationProvider(response));
      expect(state.currentManeuverIndex, 0);
      expect(state.breadcrumb, isEmpty);
    });

    test(
        'onPosition far from maneuver 1 target: index stays at 0, one breadcrumb appended',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      notifier.onPosition(_farFromManeuver1);

      final state = container.read(navigationProvider(response));
      expect(state.currentManeuverIndex, 0);
      expect(state.breadcrumb.length, 1);
      expect(state.breadcrumb[0], _farFromManeuver1);
    });

    test(
        'onPosition within 30 m of shape[maneuver1.beginShapeIndex] advances index to 1',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      notifier.onPosition(_nearManeuver1);

      final state = container.read(navigationProvider(response));
      expect(state.currentManeuverIndex, 1);
    });

    test(
        'onPosition at last maneuver does not throw and does not advance further',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      // Advance to last maneuver (index 2) by calling onPosition twice near
      // each successive target.
      const nearManeuver1 = LatLng(47.00301, 9.000);
      const nearManeuver2 = LatLng(47.00601, 9.000);
      notifier.onPosition(nearManeuver1); // advances to 1
      notifier.onPosition(nearManeuver2); // advances to 2 (last)

      expect(container.read(navigationProvider(response)).currentManeuverIndex,
          2);

      // Calling onPosition again at the last maneuver must not throw and must
      // not change the index beyond the last.
      expect(
        () => notifier.onPosition(const LatLng(47.009, 9.000)),
        returnsNormally,
      );
      expect(container.read(navigationProvider(response)).currentManeuverIndex,
          2);
    });

    test('advancement is forward-only: near earlier maneuver never decrements',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      // Advance to index 1 first.
      notifier.onPosition(_nearManeuver1);
      expect(
          container.read(navigationProvider(response)).currentManeuverIndex, 1);

      // A position near shape[0] (maneuver 0 target) must NOT decrement index.
      notifier.onPosition(const LatLng(47.000, 9.000));
      expect(
          container.read(navigationProvider(response)).currentManeuverIndex, 1);
    });

    test(
        'onPosition near maneuver 2 from index 0 skips maneuver 1 and advances directly to index 2',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      // Single fix near shape[6] (maneuver 2 begin) while still at index 0.
      // Along-track projection must advance past the skipped maneuver 1.
      notifier.onPosition(_nearManeuver2);

      expect(container.read(navigationProvider(response)).currentManeuverIndex,
          2);
    });

    test(
        'onPosition projecting to route midpoint before maneuver 1 does not advance',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      // ~178 m along-track — well under maneuver 1's ~333 m minus the 30 m
      // along-track buffer.
      notifier.onPosition(_midpointBeforeManeuver1);

      expect(container.read(navigationProvider(response)).currentManeuverIndex,
          0);
    });

    test(
        'onPosition exactly on maneuver 2 vertex projects to its cumulative distance and advances to index 2',
        () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      // A position exactly on shape[6] projects to ~667 m along-track (within
      // a few meters of maneuver 2's cumulative distance), reaching maneuver 2.
      notifier.onPosition(_exactlyManeuver2);

      expect(container.read(navigationProvider(response)).currentManeuverIndex,
          2);
    });

    test('each onPosition call appends exactly one LatLng to breadcrumb', () {
      final notifier =
          container.read(navigationProvider(response).notifier);

      const p1 = LatLng(47.001, 9.001);
      const p2 = LatLng(47.002, 9.002);
      const p3 = LatLng(47.003, 9.003);

      notifier.onPosition(p1);
      notifier.onPosition(p2);
      notifier.onPosition(p3);

      final breadcrumb =
          container.read(navigationProvider(response)).breadcrumb;
      expect(breadcrumb.length, 3);
      expect(breadcrumb[0], p1);
      expect(breadcrumb[1], p2);
      expect(breadcrumb[2], p3);
    });
  });
}
