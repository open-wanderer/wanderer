import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/offline_trail_filter_bounds.dart';

// ---------------------------------------------------------------------------
// Pure-input tests for offline_trail_filter_bounds.dart's two public pure
// functions. No Store, no mocks, no source greps -- `readLocalTrailMetrics`
// is deliberately left uncovered here (see its own doc comment).
// ---------------------------------------------------------------------------

void main() {
  group('nextBoundAbove', () {
    test('rounds an ordinary observed max up to the next step', () {
      expect(nextBoundAbove(812.4, 5000), 5000);
      expect(nextBoundAbove(41000, 5000), 45000);
    });

    test('moves an exact-multiple observed max up one full step', () {
      // The rounding boundary: an observed max that already sits on a
      // multiple of the step still moves up one step, so the longest trail
      // is never pinned to the slider's extreme.
      expect(nextBoundAbove(40000, 5000), 45000);
    });

    test('a flat trail (0) yields exactly one step', () {
      expect(nextBoundAbove(0, 250), 250);
    });

    test('guards against negative, NaN and infinite input', () {
      expect(nextBoundAbove(-5, 5000), 5000);
      expect(nextBoundAbove(double.nan, 5000), 5000);
      expect(nextBoundAbove(double.infinity, 5000), 5000);
    });
  });

  group('computeOfflineTrailFilterValues', () {
    test('an empty store returns kOfflineTrailFilterValues exactly', () {
      const empty = (
        distances: <double>[],
        elevationGains: <double>[],
        elevationLosses: <double>[],
      );

      // Both types are @freezed, so this is value equality -- the
      // fresh-install floor: a device with nothing on it still gets a
      // usable filter sheet.
      expect(
        computeOfflineTrailFilterValues(empty),
        kOfflineTrailFilterValues,
      );
    });

    test('a single short trail yields the first-step bound on every axis', () {
      const metrics = (
        distances: [812.4],
        elevationGains: [37.0],
        elevationLosses: [41.0],
      );

      final result = computeOfflineTrailFilterValues(metrics);

      expect(result.maxDistance, 5000);
      expect(result.maxElevationGain, 250);
      expect(result.maxElevationLoss, 250);
    });

    test('a longer trail yields larger rounded bounds, distinct from the '
        'empty-store constant', () {
      // Task 2 reuses this exact fixture, so this assertion is what stops
      // that test from silently degenerating into a constant-vs-constant
      // comparison.
      const metrics = (
        distances: [12345.0],
        elevationGains: [321.0],
        elevationLosses: [98.0],
      );

      final result = computeOfflineTrailFilterValues(metrics);

      expect(result.maxDistance, 15000);
      expect(result.maxElevationGain, 500);
      expect(result.maxElevationLoss, 250);
      expect(result, isNot(kOfflineTrailFilterValues));
    });

    test('per-axis independence: data on one axis does not drag the others '
        'off the empty-store floor', () {
      const metrics = (
        distances: [12345.0],
        elevationGains: <double>[],
        elevationLosses: <double>[],
      );

      final result = computeOfflineTrailFilterValues(metrics);

      expect(result.maxDistance, 15000);
      expect(result.maxElevationGain, kOfflineTrailFilterValues.maxElevationGain);
      expect(result.maxElevationLoss, kOfflineTrailFilterValues.maxElevationLoss);
    });

    test('minima are always 0 and maxDuration always the constant, for '
        'empty and non-empty input alike', () {
      const empty = (
        distances: <double>[],
        elevationGains: <double>[],
        elevationLosses: <double>[],
      );
      const nonEmpty = (
        distances: [12345.0],
        elevationGains: [321.0],
        elevationLosses: [98.0],
      );

      for (final metrics in [empty, nonEmpty]) {
        final result = computeOfflineTrailFilterValues(metrics);
        expect(result.minDistance, 0);
        expect(result.minElevationGain, 0);
        expect(result.minElevationLoss, 0);
        expect(result.minDuration, 0);
        expect(result.maxDuration, kOfflineTrailFilterValues.maxDuration);
      }
    });
  });
}
