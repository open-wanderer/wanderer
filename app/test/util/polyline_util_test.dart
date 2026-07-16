import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/polyline_util.dart';

// ---------------------------------------------------------------------------
// Tests for PolylineUtil's precision-parameterized encode/decode.
//
// Verifies: default-precision-5 round-trip (regression guard for existing
// trail-polyline callers), precision-6 round-trip (Valhalla /route shapes),
// that precision 5 vs precision 6 are genuinely distinct (the bug this fix
// prevents), and defensive handling of an empty/malformed shape string.
// ---------------------------------------------------------------------------

void main() {
  group('PolylineUtil', () {
    // Zürich-range fixture coordinates so precision-6 detail is meaningful.
    final points = [
      Geographic(lat: 47.3769, lon: 8.5417),
      Geographic(lat: 47.3780123, lon: 8.5430456),
      Geographic(lat: 47.3795987, lon: 8.5401234),
    ];

    test(
      'encode then decode round-trips within 1e-5 with default precision (5)',
      () {
        final encoded = PolylineUtil.encode(points);
        final decoded = PolylineUtil.decode(encoded);

        expect(decoded.length, points.length);
        for (var i = 0; i < points.length; i++) {
          expect(decoded[i].lat, closeTo(points[i].lat, 1e-5));
          expect(decoded[i].lon, closeTo(points[i].lon, 1e-5));
        }
      },
    );

    test('encode then decode round-trips within 1e-6 at precision 6', () {
      final encoded = PolylineUtil.encode(points, precision: 6);
      final decoded = PolylineUtil.decode(encoded, precision: 6);

      expect(decoded.length, points.length);
      for (var i = 0; i < points.length; i++) {
        expect(decoded[i].lat, closeTo(points[i].lat, 1e-6));
        expect(decoded[i].lon, closeTo(points[i].lon, 1e-6));
      }
    });

    test(
      'decoding a precision-6-encoded string with default precision (5) '
      'yields coordinates off by ~10x, proving the two precisions are '
      'genuinely distinct',
      () {
        final encoded = PolylineUtil.encode(points, precision: 6);
        final decodedWrong = PolylineUtil.decode(encoded);

        // At precision 6, decoding as if precision 5 divides accumulated
        // integer deltas by the wrong factor (1e5 instead of 1e6), scaling
        // the coordinate value by roughly 10x versus the true input.
        // Longitude is used (not latitude) because Geographic clamps
        // latitude to [-90, 90] and the 10x-scaled fixture latitude would
        // be clamped, masking the divergence this test asserts.
        final firstScaledLon = decodedWrong.first.lon;
        final ratio = firstScaledLon / points.first.lon;
        expect(ratio, closeTo(10, 1));
        expect(
          (firstScaledLon - points.first.lon).abs(),
          greaterThan(1e-3),
        );
      },
    );

    test('decode("") returns an empty list', () {
      expect(PolylineUtil.decode(''), isEmpty);
    });
  });
}
