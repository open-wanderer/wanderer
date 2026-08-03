import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/util/offline_trail_filter_bounds.dart';

// ---------------------------------------------------------------------------
// Pure-input tests for trail_filter_provider.dart's fallback filter and
// retry policy. No widget pump, no ProviderScope, no Store. Imports
// computeOfflineTrailFilterValues/kOfflineTrailFilterValues from
// offline_trail_filter_bounds.dart to build the `computed` fixture, so this
// file and offline_trail_filter_bounds_test.dart stay in step by
// construction rather than by duplicated literals.
// ---------------------------------------------------------------------------

void main() {
  // (15000, 500, 250) -- differs from kOfflineTrailFilterValues on all three
  // axes.
  final computed = computeOfflineTrailFilterValues((
    distances: [12345.0],
    elevationGains: [321.0],
    elevationLosses: [98.0],
  ));

  // A server-shaped fixture, matching the payload
  // test/provider/trail_filter_provider_test.dart already stubs.
  const server = TrailFilterValues(
    minDistance: 0,
    maxDistance: 42000,
    minElevationGain: 0,
    maxElevationGain: 3000,
    minElevationLoss: 0,
    maxElevationLoss: 2500,
    minDuration: 0,
    maxDuration: 86400,
  );

  group('buildDefaultTrailFilter', () {
    test('max == limit on every axis, proven with COMPUTED values -- this '
        'is the property the entire spinner fix rests on', () {
      final filter = buildDefaultTrailFilter(computed);

      expect(filter.distanceMax, filter.distanceLimit);
      expect(filter.elevationGainMax, filter.elevationGainLimit);
      expect(filter.elevationLossMax, filter.elevationLossLimit);
    });

    test('max == limit also holds for the constant', () {
      final filter = buildDefaultTrailFilter(kOfflineTrailFilterValues);

      expect(filter.distanceMax, filter.distanceLimit);
      expect(filter.elevationGainMax, filter.elevationGainLimit);
      expect(filter.elevationLossMax, filter.elevationLossLimit);
    });

    test('the invariant observed through filter text: computed and server '
        'bounds produce byte-identical filter text', () {
      // Guard against silently degenerating: prove the equality across
      // three genuinely different bound triples, not three copies of one.
      expect(computed, isNot(kOfflineTrailFilterValues));
      expect(computed.maxDistance, isNot(server.maxDistance));

      expect(
        buildDefaultTrailFilter(computed).toFilterText(),
        buildDefaultTrailFilter(server).toFilterText(),
      );
    });

    test('the fallback never narrows a search', () {
      final text = buildDefaultTrailFilter(computed).toFilterText();

      expect(text, contains('distance >= 0'));
      expect(text, isNot(contains('distance <=')));
      expect(text, isNot(contains('elevation_gain <=')));
      expect(text, isNot(contains('elevation_loss <=')));
    });

    test('empty-store slider bounds are 100 km / 5000 m / 5000 m -- the '
        'user-visible half. On a device with no trails, these are the max: '
        'of every filter sheet slider app-wide while offline.', () {
      final filter = buildDefaultTrailFilter(kOfflineTrailFilterValues);

      expect(filter.distanceLimit, 100000);
      expect(filter.elevationGainLimit, 5000);
      expect(filter.elevationLossLimit, 5000);
    });
  });

  group('trailFilterRetry', () {
    test('retries the first two attempts', () {
      expect(trailFilterRetry(0, Exception('x')), isNotNull);
      expect(trailFilterRetry(1, Exception('x')), isNotNull);
    });

    test('exhausts the retry budget after two attempts', () {
      expect(trailFilterRetry(2, Exception('x')), isNull);
    });
  });
}
