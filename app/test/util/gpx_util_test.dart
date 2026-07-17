import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/util/gpx_util.dart';

// ---------------------------------------------------------------------------
// Tests for buildNavShape downsampling guarantees.
//
// Verifies: passthrough for ≤500 points, cap at ≤500 for >500 points,
// first/last point preservation, and last-point deduplication.
// ---------------------------------------------------------------------------

void main() {
  group('buildNavShape', () {
    // Geographic() clamps latitude to [-90, 90] and wraps longitude to
    // (-180, 180], unlike the old LatLng which stored raw doubles verbatim.
    // Fixtures scale the index down so every generated point stays within
    // valid coordinate bounds while still uniquely encoding its index.
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
        // step = ceil(501/499) = 2
        // modulo sampling at step=2 visits indices 0,2,4,...,500 — so index 500
        // (the last point) IS already included by the modulo loop. The dedup
        // guard must not append it a second time.
        final points = List.generate(
          501,
          (i) => Geographic(lat: i * scale, lon: i * scale),
        );
        final result = buildNavShape(points);

        expect(result.length, lessThanOrEqualTo(500));

        // Last point (lat=500*scale, lon=500*scale) must appear exactly once.
        final lastPointCount = result
            .where((m) => m['lat'] == 500 * scale)
            .length;
        expect(lastPointCount, 1);

        // First point must be preserved.
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

  group('categoryForTravelProfile', () {
    test('a bike-named category returns its id for bicycle', () {
      const categories = [
        Category(id: 'hike-id', name: 'Hiking Trails'),
        Category(id: 'bike-id', name: 'Mountain Bike Trails'),
      ];

      expect(
        categoryForTravelProfile('bicycle', categories),
        'bike-id',
      );
    });

    test('a hike-named category returns its id for pedestrian', () {
      const categories = [
        Category(id: 'hike-id', name: 'Hiking Trails'),
        Category(id: 'bike-id', name: 'Mountain Bike Trails'),
      ];

      expect(
        categoryForTravelProfile('pedestrian', categories),
        'hike-id',
      );
    });

    test('a list with no matching category returns null', () {
      const categories = [Category(id: 'other-id', name: 'Wildlife')];

      expect(categoryForTravelProfile('bicycle', categories), isNull);
      expect(categoryForTravelProfile('pedestrian', categories), isNull);
    });

    test('an empty category list returns null', () {
      expect(categoryForTravelProfile('bicycle', const []), isNull);
      expect(categoryForTravelProfile('pedestrian', const []), isNull);
    });

    test(
      'symmetry check: a category named to match costingForCategory\'s '
      '"bike" branch is returned by categoryForTravelProfile for bicycle',
      () {
        const category = Category(id: 'cycle-id', name: 'Cycling Routes');

        // Forward direction (existing heuristic).
        expect(costingForCategory(category.name), 'bicycle');
        // Reverse direction (this task's new heuristic) agrees.
        expect(
          categoryForTravelProfile('bicycle', const [category]),
          'cycle-id',
        );
      },
    );
  });
}
