import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/polyline_util.dart';

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

        // Longitude (not latitude) because Geographic clamps latitude to
        // [-90, 90], which would mask the ~10x divergence this test asserts.
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
