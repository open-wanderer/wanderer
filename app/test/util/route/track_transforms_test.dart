import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/route/track_transforms.dart';

void main() {
  group('withTransformedTrack', () {
    Gpx track(List<(double, double, double)> points) => Gpx()
      ..trks = [
        Trk(
          trksegs: [
            Trkseg(
              trkpts: [
                for (final (lat, lon, ele) in points)
                  Wpt(lat: lat, lon: lon, ele: ele),
              ],
            ),
          ],
        ),
      ];

    final waypoint = Waypoint(
      id: '',
      localKey: 'wp-1',
      lat: 47.0,
      lon: 11.0,
      created: DateTime(2026),
      updated: DateTime(2026),
    );

    Trail buildCapture() => Trail(
      id: '',
      name: 'Morning loop',
      description: 'typed by the user',
      created: DateTime(2026),
      updated: DateTime(2026),
      movingDuration: 1800,
      category: 'hiking',
      originalGpxData: '<gpx>original</gpx>',
      expand: TrailExpand(
        gpxData: '<gpx>original</gpx>',
        waypointsViaTrail: [waypoint],
      ),
    );

    test('takes the track and its metrics, keeps everything the user set', () {
      final gpx = track([(47.0, 11.0, 500), (47.01, 11.0, 600)]);
      final transformed = (
        gpx: gpx,
        xml: '<gpx>heights</gpx>',
        followsRoads: false,
        heightsRecalculated: true,
      );
      final measured = trailFromGpx(gpx);

      final result = withTransformedTrack(
        buildCapture(),
        transformed,
        measured: measured,
      );

      expect(result.expand?.gpxData, '<gpx>heights</gpx>');
      expect(result.expand?.gpx, same(gpx));
      expect(result.distance, measured.distance);
      expect(result.elevationGain, measured.elevationGain);
      expect(result.maxLat, measured.maxLat);
      expect(result.trackHeightsRecalculated, isTrue);
      expect(result.trackFollowsRoads, isFalse);

      expect(result.name, 'Morning loop');
      expect(result.description, 'typed by the user');
      expect(result.category, 'hiking');
      expect(result.movingDuration, 1800);
      expect(result.expand?.waypointsViaTrail, [waypoint]);
      expect(result.originalGpxData, '<gpx>original</gpx>');
    });

    test('switching both off again restores the original metrics', () {
      final original = track([(47.0, 11.0, 500), (47.01, 11.0, 520)]);
      final heights = track([(47.0, 11.0, 500), (47.01, 11.0, 700)]);
      final capture = buildCapture();

      final on = withTransformedTrack(capture, (
        gpx: heights,
        xml: '<gpx>heights</gpx>',
        followsRoads: false,
        heightsRecalculated: true,
      ), measured: trailFromGpx(heights));
      final off = withTransformedTrack(on, (
        gpx: original,
        xml: '<gpx>original</gpx>',
        followsRoads: false,
        heightsRecalculated: false,
      ), measured: trailFromGpx(original));

      expect(off.elevationGain, trailFromGpx(original).elevationGain);
      expect(off.expand?.gpxData, '<gpx>original</gpx>');
      expect(off.trackHeightsRecalculated, isFalse);
    });
  });
}
