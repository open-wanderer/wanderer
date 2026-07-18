import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/gpx_util.dart';

void main() {
  group('buildNavShape', () {
    // Geographic clamps latitude to [-90, 90] and wraps longitude to
    // (-180, 180], so fixtures scale the index down to stay within bounds
    // while still uniquely encoding it.
    const scale = 0.0001;

    test('2-point input returns 2 shape maps without downsampling', () {
      final points = List.generate(
        2,
        (i) => Geographic(lat: i * scale, lon: i * scale),
      );
      final result = buildNavShape(points);

      expect(result.length, 2);
      expect(result.first, {'lat': 0.0, 'lon': 0.0});
      expect(result.last, {'lat': 1 * scale, 'lon': 1 * scale});
    });

    test('500-point input returns all 500 entries unchanged', () {
      final points = List.generate(
        500,
        (i) => Geographic(lat: i * scale, lon: i * scale),
      );
      final result = buildNavShape(points);

      expect(result.length, 500);
      expect(result.first, {'lat': 0.0, 'lon': 0.0});
      expect(result.last, {'lat': 499 * scale, 'lon': 499 * scale});
    });

    test(
      '1000-point input is downsampled to ≤500 with first and last preserved',
      () {
        final points = List.generate(
          1000,
          (i) => Geographic(lat: i * scale, lon: i * scale),
        );
        final result = buildNavShape(points);

        expect(result.length, lessThanOrEqualTo(500));
        expect(result.first, {'lat': 0.0, 'lon': 0.0});
        expect(result.last, {'lat': 999 * scale, 'lon': 999 * scale});
      },
    );

    test(
      '501-point input is downsampled to ≤500 and last point appears exactly once',
      () {
        // step = ceil(501/499) = 2, so modulo sampling already includes index
        // 500 (the last point) — the dedup guard must not append it again.
        final points = List.generate(
          501,
          (i) => Geographic(lat: i * scale, lon: i * scale),
        );
        final result = buildNavShape(points);

        expect(result.length, lessThanOrEqualTo(500));

        final lastPointCount = result
            .where((m) => m['lat'] == 500 * scale)
            .length;
        expect(lastPointCount, 1);

        expect(result.first, {'lat': 0.0, 'lon': 0.0});
      },
    );
  });

  group('buildGpxFromPoints', () {
    test('empty input returns an empty Gpx (allPoints empty, no crash)', () {
      final gpx = buildGpxFromPoints(const []);

      expect(gpx.allPoints, isEmpty);
    });

    test(
      'ordered points round-trip via a single Trk > single Trkseg > Wpt list, '
      'with no ele set',
      () {
        final points = [
          const Geographic(lat: 47.000, lon: 9.000),
          const Geographic(lat: 47.001, lon: 9.001),
          const Geographic(lat: 47.002, lon: 9.002),
        ];

        final gpx = buildGpxFromPoints(points);

        expect(gpx.trks, hasLength(1));
        expect(gpx.trks.single.trksegs, hasLength(1));
        expect(gpx.allPoints, points);
        expect(
          gpx.allWaypoints.every((wpt) => wpt.ele == null),
          isTrue,
        );
      },
    );
  });
}
