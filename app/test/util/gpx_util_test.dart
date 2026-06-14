import 'package:flutter_test/flutter_test.dart';
import 'package:latlong2/latlong.dart';
import 'package:wanderer/util/gpx_util.dart';

// ---------------------------------------------------------------------------
// Tests for buildNavShape downsampling guarantees (Phase 05, plan 05-01).
//
// Verifies: passthrough for ≤500 points, cap at ≤500 for >500 points,
// first/last point preservation, and last-point deduplication.
// ---------------------------------------------------------------------------

void main() {
  group('buildNavShape', () {
    test('2-point input returns 2 shape maps without downsampling', () {
      final points = List.generate(2, (i) => LatLng(i.toDouble(), i.toDouble()));
      final result = buildNavShape(points);

      expect(result.length, 2);
      expect(result.first, {'lat': 0.0, 'lon': 0.0});
      expect(result.last, {'lat': 1.0, 'lon': 1.0});
    });

    test('500-point input returns all 500 entries unchanged', () {
      final points =
          List.generate(500, (i) => LatLng(i.toDouble(), i.toDouble()));
      final result = buildNavShape(points);

      expect(result.length, 500);
      expect(result.first, {'lat': 0.0, 'lon': 0.0});
      expect(result.last, {'lat': 499.0, 'lon': 499.0});
    });

    test('1000-point input is downsampled to ≤500 with first and last preserved',
        () {
      final points =
          List.generate(1000, (i) => LatLng(i.toDouble(), i.toDouble()));
      final result = buildNavShape(points);

      expect(result.length, lessThanOrEqualTo(500));
      expect(result.first, {'lat': 0.0, 'lon': 0.0});
      expect(result.last, {'lat': 999.0, 'lon': 999.0});
    });

    test(
        '501-point input is downsampled to ≤500 and last point appears exactly once',
        () {
      // step = ceil(501/499) = 2
      // modulo sampling at step=2 visits indices 0,2,4,...,500 — so index 500
      // (the last point) IS already included by the modulo loop. The dedup
      // guard must not append it a second time.
      final points =
          List.generate(501, (i) => LatLng(i.toDouble(), i.toDouble()));
      final result = buildNavShape(points);

      expect(result.length, lessThanOrEqualTo(500));

      // Last point (lat=500.0, lon=500.0) must appear exactly once.
      final lastPointCount =
          result.where((m) => m['lat'] == 500.0).length;
      expect(lastPointCount, 1);

      // First point must be preserved.
      expect(result.first, {'lat': 0.0, 'lon': 0.0});
    });
  });
}
