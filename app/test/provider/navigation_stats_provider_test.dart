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
///
/// [altitudeAccuracy] defaults to 5.0 (a plausible real GPS vertical accuracy)
/// rather than 0.0. "No altitude reading available" is the BOTH-zero sentinel —
/// `altitude == 0` together with `altitudeAccuracy == 0`, exactly what
/// `seedPositionFrom` fabricates — not zero accuracy alone; see
/// `hasUsableAltitude`. Tests simulating an unavailable reading must therefore
/// pass `altitude: 0, altitudeAccuracy: 0` (see the seed-position regression
/// test below).
Position _pos({
  required double lat,
  required double lon,
  double altitude = 0.0,
  double altitudeAccuracy = 5.0,
  double speed = 0.0,
}) {
  return Position(
    latitude: lat,
    longitude: lon,
    timestamp: DateTime(2026, 1, 1),
    accuracy: 5.0,
    altitude: altitude,
    altitudeAccuracy: altitudeAccuracy,
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
        container.read(navigationStatsProvider(response).notifier);

    NavigationStats read() => container.read(navigationStatsProvider(response));

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

    test(
      'first onPosition only anchors reference: distance stays 0, no throw',
      () {
        final n = notifier();
        expect(
          () => n.onPosition(_pos(lat: 47.000, lon: 9.000)),
          returnsNormally,
        );
        expect(read().distanceMeters, 0);
      },
    );

    test('two positions ~111 m apart accumulate distance ≈ 111 m', () {
      final n = notifier();
      // 0.001° latitude ≈ 111 m.
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.onPosition(_pos(lat: 47.001, lon: 9.000));
      expect(read().distanceMeters, closeTo(111, 5));
    });

    test(
      'altitude delta below noise floor (+1.0 m) does not accumulate gain',
      () {
        final n = notifier();
        n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 100.0));
        n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 101.0));
        expect(read().elevationGainMeters, 0);
      },
    );

    test(
      'a zero-altitude/zero-accuracy seed fix (mirrors '
      'TraceletPositionSource.seedPositionFrom) does not anchor the '
      'elevation reference — the next real fix must not register as gain',
      () {
        final n = notifier();
        // Mirrors seedPositionFrom's placeholder: altitude/altitudeAccuracy
        // both zeroed because the map marker's LocationMarkerPosition never
        // tracks altitude. This must be treated as "no reading available",
        // not anchored as a genuine sea-level fix.
        n.onPosition(
          _pos(lat: 47.000, lon: 9.000, altitude: 0.0, altitudeAccuracy: 0.0),
        );
        expect(read().elevationGainMeters, 0);

        // First REAL fix (e.g. tracelet's first GPS lock at ~500 m, as in
        // Munich): must anchor the reference here, not diff against the
        // fabricated 0.0 seed above.
        n.onPosition(
          _pos(lat: 47.000, lon: 9.000, altitude: 500.0, altitudeAccuracy: 5.0),
        );
        expect(read().elevationGainMeters, 0);
        expect(read().elevationLossMeters, 0);

        // A genuine subsequent climb still accumulates normally.
        n.onPosition(
          _pos(lat: 47.000, lon: 9.000, altitude: 505.0, altitudeAccuracy: 5.0),
        );
        expect(read().elevationGainMeters, closeTo(5.0, 0.001));
      },
    );

    test('a real altitude with zero altitudeAccuracy still accumulates — '
        'Android only reports vertical accuracy on API 26+, and this app '
        'supports API 21+', () {
      // REGRESSION GUARD. Gating elevation on `altitudeAccuracy > 0` alone
      // was tried and reverted: on any device that never reports vertical
      // accuracy it silently disables elevation tracking for the whole
      // recording, which is worse than the phantom-gain bug it was meant to
      // prevent. Only the both-zero sentinel means "no reading".
      final n = notifier();
      n.onPosition(
        _pos(lat: 47.000, lon: 9.000, altitude: 100.0, altitudeAccuracy: 0.0),
      );
      n.onPosition(
        _pos(lat: 47.000, lon: 9.000, altitude: 105.0, altitudeAccuracy: 0.0),
      );
      expect(read().elevationGainMeters, closeTo(5.0, 0.001));
    });

    test(
      'a non-finite altitude is skipped rather than poisoning the reference',
      () {
        final n = notifier();
        n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 100.0));
        n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: double.nan));
        n.onPosition(_pos(lat: 47.000, lon: 9.000, altitude: 105.0));

        expect(read().elevationGainMeters, closeTo(5.0, 0.001));
        expect(read().elevationGainMeters.isFinite, isTrue);
      },
    );

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

    test('setStationary(true) sets isStationary and freezes accumulation', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.setStationary(true);
      expect(read().isStationary, true);

      // While stationary, distance/elevation must NOT accumulate and speed
      // is forced to 0.
      n.onPosition(_pos(lat: 47.010, lon: 9.000, altitude: 200.0, speed: 5.0));
      expect(read().distanceMeters, 0);
      expect(read().elevationGainMeters, 0);
      expect(read().currentSpeedKmh, 0);
    });

    test('setStationary(false) after setStationary(true) re-anchors so no '
        'distance jump occurs', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.setStationary(true);
      // User "moves" while stationary-frozen (no accumulation).
      n.onPosition(_pos(lat: 47.010, lon: 9.000));
      n.setStationary(false);
      expect(read().isStationary, false);

      // First post-resume fix anchors; distance still 0 (no jump from the
      // frozen interval).
      n.onPosition(_pos(lat: 47.010, lon: 9.000));
      expect(read().distanceMeters, 0);

      // A subsequent fix ~111 m away accumulates normally.
      n.onPosition(_pos(lat: 47.011, lon: 9.000));
      expect(read().distanceMeters, closeTo(111, 5));
    });

    test(
      'manual togglePause sets isPaused independently of isStationary; '
      'setStationary(false) while manually paused leaves accumulation frozen',
      () {
        final n = notifier();
        n.onPosition(_pos(lat: 47.000, lon: 9.000));
        n.togglePause(); // manual pause
        expect(read().isPaused, true);
        expect(read().isStationary, false);

        n.setStationary(false); // no-op: was already false, idempotent
        expect(read().isPaused, true);

        // Still frozen — manual pause holds regardless of isStationary.
        n.onPosition(_pos(lat: 47.010, lon: 9.000, speed: 5.0));
        expect(read().distanceMeters, 0);
        expect(read().currentSpeedKmh, 0);
      },
    );

    test(
      'setStationary toggling before the first GPS fix does not inflate '
      'pausedAccum (regression: negative elapsed once the session starts)',
      () async {
        // Autodispose provider: without an active listener it would be
        // torn down once the synchronous setup below returns, before the
        // `await` resumes — keep it alive for the duration of the test.
        container.listen(navigationStatsProvider(response), (_, _) {});
        final n = notifier();
        // No onPosition yet — _start is still null. This mirrors a
        // trail-less recording session waiting for its first GPS lock while
        // tracelet's motion engine has already reported the device as
        // stationary (physically still) and then moving.
        n.setStationary(true);
        await Future<void>.delayed(const Duration(milliseconds: 50));
        n.setStationary(false);

        // The frozen interval occurred entirely before the stopwatch
        // started, so it must not be folded into pausedAccum — there is no
        // elapsed interval yet for it to belong to. Before the fix this
        // accumulated ~50ms of real wall-clock time regardless.
        expect(n.pausedAccum, Duration.zero);

        // Once the session actually starts, the first fix must anchor
        // normally (no residual frozen state left over).
        n.onPosition(_pos(lat: 47.000, lon: 9.000));
        expect(read().distanceMeters, 0);
      },
    );

    test('overlapping manual pause + stationary: paused time not '
        'double-counted, resumes cleanly with a single re-anchor', () {
      final n = notifier();
      n.onPosition(_pos(lat: 47.000, lon: 9.000));
      n.togglePause(); // manual pause starts freeze
      n.setStationary(true); // stationary while already manually paused
      n.setStationary(false); // still frozen — manual pause still holds
      expect(read().isPaused, true);
      expect(read().isStationary, false);

      // Frozen throughout — no accumulation from any of the above.
      n.onPosition(_pos(lat: 47.010, lon: 9.000, speed: 5.0));
      expect(read().distanceMeters, 0);
      expect(read().currentSpeedKmh, 0);

      n.togglePause(); // resume — the only unfreeze
      expect(read().isPaused, false);

      // First post-resume fix anchors; distance still 0 (single re-anchor,
      // no jump from the combined frozen interval).
      n.onPosition(_pos(lat: 47.010, lon: 9.000));
      expect(read().distanceMeters, 0);

      // A subsequent fix ~111 m away accumulates normally.
      n.onPosition(_pos(lat: 47.011, lon: 9.000));
      expect(read().distanceMeters, closeTo(111, 5));
    });
  });
}
