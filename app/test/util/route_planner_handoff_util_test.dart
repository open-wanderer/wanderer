import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/util/route_planner_handoff_util.dart';

// ---------------------------------------------------------------------------
// Tests for the pure handoff helpers (no network/navigation): buildDraftTrail
// and mergeHeightsIntoGpx. finishPlanning's orchestration (network fetch +
// pendingImportedTrail + navigation) is intentionally not unit-tested here —
// it has no pure/synchronous seam without a WidgetRef/BuildContext harness.
// ---------------------------------------------------------------------------

void main() {
  group('buildDraftTrail', () {
    Gpx buildSampleGpx() {
      final gpx = Gpx();
      gpx.trks = [
        Trk(
          trksegs: [
            Trkseg(
              trkpts: [
                Wpt(lat: 47.000, lon: 9.000, ele: 400),
                Wpt(lat: 47.001, lon: 9.001, ele: 410),
              ],
            ),
          ],
        ),
      ];
      return gpx;
    }

    test('sets a non-empty expand.gpxData containing "<gpx" (Pitfall 1)', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.expand?.gpxData, isNotNull);
      expect(result.expand!.gpxData, isNotEmpty);
      expect(result.expand!.gpxData, contains('<gpx'));
    });

    test('leaves expand.waypointsViaTrail empty (D-07)', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.expand?.waypointsViaTrail, isEmpty);
    });

    test('keeps expand.gpx identical to the finalGpx passed in', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(identical(result.expand!.gpx, gpx), isTrue);
    });

    test('passes a supplied category id through', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx, category: 'bike-id');

      expect(result.category, 'bike-id');
    });

    test('leaves category null when none is supplied', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.category, isNull);
    });

    test('sets lat/lon/bounds from the track so the map centers on the '
        'route instead of null island', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.maxLat, 47.001);
      expect(result.minLat, 47.000);
      expect(result.maxLon, 9.001);
      expect(result.minLon, 9.000);
      expect(result.lat, closeTo(47.0005, 1e-9));
      expect(result.lon, closeTo(9.0005, 1e-9));
    });

    test('leaves lat/lon null and bounds zeroed for an empty track', () {
      final result = buildDraftTrail(Gpx());

      expect(result.lat, isNull);
      expect(result.lon, isNull);
      expect(result.maxLat, 0);
      expect(result.minLat, 0);
      expect(result.maxLon, 0);
      expect(result.minLon, 0);
    });
  });

  group('mergeHeightsIntoGpx', () {
    test('aligns ele to shape indices', () {
      final shape = [
        {'lat': 47.000, 'lon': 9.000},
        {'lat': 47.001, 'lon': 9.001},
        {'lat': 47.002, 'lon': 9.002},
      ];
      final heights = [400, 410, 420];

      final gpx = mergeHeightsIntoGpx(shape, heights);
      final points = gpx.trks.single.trksegs.single.trkpts;

      expect(points, hasLength(3));
      expect(points[0].lat, 47.000);
      expect(points[0].lon, 9.000);
      expect(points[0].ele, 400.0);
      expect(points[1].ele, 410.0);
      expect(points[2].ele, 420.0);
    });

    test('assigns null ele when heights is shorter than shape', () {
      final shape = [
        {'lat': 47.000, 'lon': 9.000},
        {'lat': 47.001, 'lon': 9.001},
        {'lat': 47.002, 'lon': 9.002},
      ];
      final heights = [400, 410];

      final gpx = mergeHeightsIntoGpx(shape, heights);
      final points = gpx.trks.single.trksegs.single.trkpts;

      expect(points[0].ele, 400.0);
      expect(points[1].ele, 410.0);
      expect(points[2].ele, isNull);
    });

    test('returns an empty Gpx (no tracks) for an empty shape', () {
      final gpx = mergeHeightsIntoGpx(const [], const []);

      expect(gpx.trks, isEmpty);
    });
  });
}
