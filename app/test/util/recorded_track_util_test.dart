import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/recorded_track_util.dart';

// Tests for buildRecordedTrackTrail, the pure breadcrumb -> stub Trail
// helper. Mirrors the structure of route_planner_handoff_util_test.dart.

void main() {
  group('buildRecordedTrackTrail', () {
    test('returns null for an empty breadcrumb', () {
      expect(buildRecordedTrackTrail(const []), isNull);
    });

    test('returns null for a single-point breadcrumb', () {
      final breadcrumb = [const Geographic(lat: 47.000, lon: 9.000)];

      expect(buildRecordedTrackTrail(breadcrumb), isNull);
    });

    group('with a >=2-point breadcrumb', () {
      final breadcrumb = [
        const Geographic(lat: 47.000, lon: 9.000),
        const Geographic(lat: 47.001, lon: 9.001),
        const Geographic(lat: 47.002, lon: 9.0005),
      ];

      test('expand.gpx.allPoints round-trips the breadcrumb in order', () {
        final result = buildRecordedTrackTrail(breadcrumb);

        expect(result, isNotNull);
        final allPoints = result!.expand!.gpx!.allPoints;
        expect(allPoints, hasLength(3));
        for (var i = 0; i < breadcrumb.length; i++) {
          expect(allPoints[i].lat, breadcrumb[i].lat);
          expect(allPoints[i].lon, breadcrumb[i].lon);
        }
      });

      test('sets a non-empty expand.gpxData containing "<gpx"', () {
        final result = buildRecordedTrackTrail(breadcrumb);

        expect(result!.expand?.gpxData, isNotNull);
        expect(result.expand!.gpxData, isNotEmpty);
        expect(result.expand!.gpxData, contains('<gpx'));
      });

      test('leaves expand.waypointsViaTrail empty', () {
        final result = buildRecordedTrackTrail(breadcrumb);

        expect(result!.expand?.waypointsViaTrail, isEmpty);
      });

      test(
        'derives non-null lat/lon and bounds bracketing the input points',
        () {
          final result = buildRecordedTrackTrail(breadcrumb);

          expect(result!.lat, isNotNull);
          expect(result.lon, isNotNull);
          expect(result.maxLat, 47.002);
          expect(result.minLat, 47.000);
          expect(result.maxLon, 9.001);
          expect(result.minLon, 9.000);
          expect(
            result.lat! >= result.minLat && result.lat! <= result.maxLat,
            isTrue,
          );
          expect(
            result.lon! >= result.minLon && result.lon! <= result.maxLon,
            isTrue,
          );
        },
      );

      test('sets duration from durationSeconds when supplied', () {
        final result = buildRecordedTrackTrail(
          breadcrumb,
          durationSeconds: 123.0,
        );

        expect(result!.duration, 123.0);
      });

      test('defaults duration to 0 when durationSeconds is omitted', () {
        final result = buildRecordedTrackTrail(breadcrumb);

        expect(result!.duration, 0);
      });

      test(
        'sets elevationGain/elevationLoss from the given meters when supplied',
        () {
          final result = buildRecordedTrackTrail(
            breadcrumb,
            elevationGainMeters: 42.5,
            elevationLossMeters: 17.0,
          );

          expect(result!.elevationGain, 42.5);
          expect(result.elevationLoss, 17.0);
        },
      );

      test(
        'defaults elevationGain/elevationLoss to 0 when omitted',
        () {
          final result = buildRecordedTrackTrail(breadcrumb);

          expect(result!.elevationGain, 0);
          expect(result.elevationLoss, 0);
        },
      );
    });
  });
}
