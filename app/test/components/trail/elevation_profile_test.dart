import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/util/gpx_conversion_util.dart';
import 'package:wanderer/util/gpx_util.dart';

/// A slow, jittery track: ~1 m of real eastward progress per sample with GPS
/// noise of the same order, so most hops fall UNDER the 5 m smoothing
/// threshold. Measured on this fixture: raw haversine 232.6 m vs smoothed
/// 80.6 m — a 152 m gap over 100 points.
///
/// This is the shape CONV-05's smoothing exists for, and where the chart's old
/// raw accumulation disagreed most visibly with the distance shown everywhere
/// else in the app.
Gpx _jitteryTrack({int points = 100}) {
  final trkpts = <Wpt>[];
  for (var i = 0; i < points; i++) {
    final jitter = (i.isEven ? 1 : -1) * 0.00001;
    trkpts.add(
      Wpt(
        lat: 47.0 + jitter,
        lon: 11.0 + i * 0.00001,
        ele: 1000 + (i % 5) * 2.0,
        time: DateTime.utc(2024, 1, 1, 10).add(Duration(seconds: i * 10)),
      ),
    );
  }
  return Gpx()
    ..trks = [
      Trk(trksegs: [Trkseg(trkpts: trkpts)]),
    ];
}

void main() {
  group('buildElevationTrackPoints distance agrees with the trail', () {
    // The defect this pins: the chart's x-axis and its distance stat both read
    // the last TrackPoint's distanceM, while trail_panel/trail cards/the
    // persisted trail.distance all come from computeTrailMetrics. When the
    // chart accumulated raw haversine, one track showed two different lengths.
    test('final distanceM equals computeTrailMetrics().distance', () {
      final gpx = _jitteryTrack();

      final points = buildElevationTrackPoints(gpx, 1);
      final metrics = computeTrailMetrics(gpx);

      expect(points, isNotEmpty);
      expect(points.last.distanceM, closeTo(metrics.distance, 1e-9));
    });

    test('the fixture is one where raw and smoothed genuinely differ', () {
      // Non-vacuity: without this, the assertion above would still pass on a
      // track so clean that raw and smoothed coincide, proving nothing.
      final gpx = _jitteryTrack();

      var raw = 0.0;
      final pts = gpx.allWaypoints;
      for (var i = 1; i < pts.length; i++) {
        raw += haversineMeters(pts[i - 1], pts[i]);
      }

      final smoothed = computeTrailMetrics(gpx).distance;
      expect(raw, greaterThan(smoothed));
      // Measured: raw 232.6 m vs smoothed 80.6 m on this fixture.
      expect(raw - smoothed, greaterThan(100.0));
    });

    test('distanceM is non-decreasing along the track', () {
      final points = buildElevationTrackPoints(_jitteryTrack(), 1);
      for (var i = 1; i < points.length; i++) {
        expect(
          points[i].distanceM,
          greaterThanOrEqualTo(points[i - 1].distanceM),
        );
      }
    });

    test('a point with no usable coordinate is skipped but still counted', () {
      // The vendored reader leaves lat/lon null for a <trkpt> missing them,
      // so this is reachable. It must not be plotted, and must not desync the
      // chart's total from the trail's.
      final gpx = _jitteryTrack(points: 20);
      gpx.trks.single.trksegs.single.trkpts.insert(
        10,
        Wpt(lat: null, lon: null, ele: 1000),
      );

      final points = buildElevationTrackPoints(gpx, 1);
      final metrics = computeTrailMetrics(gpx);

      expect(points.every((p) => p.lonlat.lat.isFinite), isTrue);
      expect(points.last.distanceM, closeTo(metrics.distance, 1e-9));
    });

    test('an empty GPX yields no points', () {
      expect(buildElevationTrackPoints(Gpx(), 1), isEmpty);
    });
  });
}
