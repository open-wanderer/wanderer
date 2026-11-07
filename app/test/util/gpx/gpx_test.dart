import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:gpx/gpx.dart';

void main() {
  group('haversineMeters', () {
    // Characterisation tests for the extraction that retired the app's three
    // haversines. The expected values were captured from the PRE-extraction
    // implementations, so a drift in the shared function fails here rather
    // than silently moving a distance total or a chart axis.
    //
    // elevation_profile.dart previously hand-rolled its own with
    // `const r = 6371.0; ... r * 1000 * 2 * atan2(...)`. Measured against the
    // geobase-backed function it agreed to ~1e-10 m across these same cases,
    // which is what made the extraction safe — but they were free to drift
    // apart, which is why there is now only one.
    Wpt at(double lat, double lon) => Wpt(lat: lat, lon: lon);

    test('one degree of latitude', () {
      expect(
        haversineMeters(at(47.0, 11.0), at(47.001, 11.0)),
        closeTo(111.1949266442689, 1e-9),
      );
    });

    test('one degree of longitude at 47N', () {
      expect(
        haversineMeters(at(47.0, 11.0), at(47.0, 11.001)),
        closeTo(75.83475761838756, 1e-9),
      );
    });

    test('a long diagonal', () {
      expect(
        haversineMeters(at(47.0, 11.0), at(48.5, 12.7)),
        closeTo(209687.34671374125, 1e-6),
      );
    });

    test('southern hemisphere', () {
      expect(
        haversineMeters(at(-33.9, 151.2), at(-34.0, 151.3)),
        closeTo(14447.26231111224, 1e-6),
      );
    });

    test('identical points are exactly zero', () {
      expect(haversineMeters(at(0, 0), at(0, 0)), 0.0);
      expect(haversineMeters(at(47.0, 11.0), at(47.0, 11.0)), 0.0);
    });

    test('a null coordinate yields NaN, never a fabricated 0', () {
      // Load-bearing: addAndFilter guards on isFinite and SKIPS a NaN hop.
      // Returning 0 here would silently absorb the bad point into the total.
      //
      // Note Wpt's constructor DEFAULTS lat/lon to 0.0, so a programmatically
      // built point never has null coords — null has to be passed explicitly,
      // as done here. The path is reachable in production only through the
      // vendored reader, which leaves lat/lon null for a <trkpt> missing them
      // (LOCAL MODIFICATION 2) rather than throwing StateError.
      expect(
        haversineMeters(Wpt(lat: null, lon: 11.0), at(47.001, 11.0)).isNaN,
        isTrue,
      );
      expect(
        haversineMeters(at(47.0, 11.0), Wpt(lat: 47.0, lon: null)).isNaN,
        isTrue,
      );
      expect(
        haversineMeters(
          Wpt(lat: null, lon: null),
          Wpt(lat: null, lon: null),
        ).isNaN,
        isTrue,
      );
    });
  });

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
          Wpt(lat: 47.000, lon: 9.000),
          Wpt(lat: 47.001, lon: 9.001),
          Wpt(lat: 47.002, lon: 9.002),
        ];

        final gpx = buildGpxFromPoints(points);

        expect(gpx.trks, hasLength(1));
        expect(gpx.trks.single.trksegs, hasLength(1));
        expect(gpx.allWaypoints, points);
        expect(gpx.allWaypoints.every((wpt) => wpt.ele == null), isTrue);
      },
    );
  });
}
