import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:geolocator/geolocator.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Minimal [NavigateResponse] used only as the provider family key — the stats
/// notifier does not read its shape/maneuvers, but the family requires a key.
NavigateResponse _buildSampleResponse() {
  return const NavigateResponse(
    shape: [
      [47.000, 9.000],
      [47.001, 9.000],
    ],
    maneuvers: [
      NavigateManeuver(
        instruction: 'Start',
        length: 0.0,
        beginShapeIndex: 0,
        bearing: 0.0,
      ),
    ],
  );
}

/// Builds a geolocator [Position]. The geolocator [Position] constructor
/// requires latitude, longitude, timestamp, accuracy, altitude,
/// altitudeAccuracy, heading, headingAccuracy, speed, and speedAccuracy — all
/// supplied here.
Position _pos({
  required double lat,
  required double lon,
  double altitude = 0.0,
  double speed = 0.0,
}) {
  return Position(
    latitude: lat,
    longitude: lon,
    timestamp: DateTime(2026, 1, 1),
    accuracy: 5.0,
    altitude: altitude,
    altitudeAccuracy: 5.0,
    heading: 0.0,
    headingAccuracy: 5.0,
    speed: speed,
    speedAccuracy: 1.0,
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('NavigationStatsNotifier', () {
    late ProviderContainer container;
    late NavigateResponse response;

    setUp(() {
      response = _buildSampleResponse();
      container = ProviderContainer();
    });

    tearDown(() {
      container.dispose();
    });

    NavigationStatsNotifier notifier() =>
        container.read(navigationStatsNotifierProvider(response).notifier);

    NavigationStats read() =>
        container.read(navigationStatsNotifierProvider(response));

    test('initial state has all-zero stats and is not paused', () {
      final s = read();
      expect(s.distanceMeters, 0);
      expect(s.elevationGainMeters, 0);
      expect(s.elevationLossMeters, 0);
      expect(s.currentSpeedKmh, 0);
      expect(s.averageSpeedKmh, 0);
      expect(s.elapsed, Duration.zero);
      expect(s.isPaused, false);
    });

    test('first onPosition only anchors reference: distance stays 0, no throw',
        () {
      final n = notifier();
      expect(
        () => n.onPosition(_pos(lat: 47.000, lon: 9.000)),
        returnsNormally,
      );
      expect(read().distanceMeters, 0);
    });

    test('two positions ~111 m apart accumulate distance ≈ 111 m', () {
      final n = notifier();
      // 0.001° latitude ≈ 111 m.
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.onPosition(_pos(lat: 47.001, lon: 9.000));
      expect(read().distanceMeters, closeTo(111, 5));
    });

    test('altitude delta below noise floor (+1.0 m) does not accumulate gain',
        () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 100.0));
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 101.0));
      expect(read().elevationGainMeters, 0);
    });

    test('altitude delta at/above noise floor accumulates gain then loss', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 100.0));
      // +5.0 m → gain ≈ 5.0
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 105.0));
      expect(read().elevationGainMeters, closeTo(5.0, 0.001));
      expect(read().elevationLossMeters, 0);
      // -5.0 m → loss ≈ 5.0 (stored positive)
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 100.0));
      expect(read().elevationLossMeters, closeTo(5.0, 0.001));
      expect(read().elevationGainMeters, closeTo(5.0, 0.001));
    });

    test('speed 10.0 m/s converts to 36.0 km/h', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000, speed: 10.0));
      expect(read().currentSpeedKmh, 36.0);
    });

    test('NaN speed is guarded to 0', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000, speed: double.nan));
      expect(read().currentSpeedKmh, 0);
    });

    test('negative speed is guarded to 0', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000, speed: -1.0));
      expect(read().currentSpeedKmh, 0);
    });

    test('togglePause sets isPaused true and freezes accumulation', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.togglePause();
      expect(read().isPaused, true);

      // While paused, distance/elevation must NOT accumulate and speed → 0.
      n.onPosition(_pos(lat: 47.010, lon: 9.000, altitude: 200.0, speed: 5.0));
      expect(read().distanceMeters, 0);
      expect(read().elevationGainMeters, 0);
      expect(read().currentSpeedKmh, 0);
    });

    test('resume re-anchors references so no distance jump occurs', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.togglePause(); // pause
      // User "moves" while paused (no accumulation).
      n.onPosition(_pos(lat: 47.010, lon: 9.000));
      n.togglePause(); // resume → re-anchor
      expect(read().isPaused, false);

      // First post-resume fix anchors; distance still 0 (no jump from the
      // paused interval).
      n.onPosition(_pos(lat: 47.010, lon: 9.000));
      expect(read().distanceMeters, 0);

      // A subsequent fix ~111 m away accumulates normally.
      n.onPosition(_pos(lat: 47.011, lon: 9.000));
      expect(read().distanceMeters, closeTo(111, 5));
    });

    test('resume re-anchors altitude so no elevation jump occurs', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 100.0));
      n.togglePause(); // pause
      n.togglePause(); // resume → altitude reference reset
      // First post-resume fix at a very different altitude must not add gain.
      n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 500.0));
      expect(read().elevationGainMeters, 0);
      expect(read().elevationLossMeters, 0);
    });
  });
}
