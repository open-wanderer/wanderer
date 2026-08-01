import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/util/gpx_conversion_util.dart';
import 'package:wanderer/util/gpx_util.dart';

/// A track on a true, constant 10% grade sampled every ~1.5 m — roughly what a
/// 1 Hz recording at walking pace produces.
Gpx _constantGradeTrack({int points = 60}) {
  final trkpts = <Wpt>[];
  for (var i = 0; i < points; i++) {
    trkpts.add(
      Wpt(
        lat: 47.0,
        lon: 11.0 + i * 0.00002, // ~1.5 m per step at 47N
        ele: 1000 + i * 0.15, // 0.15 m rise per 1.5 m run = 10%
        time: DateTime.utc(2024, 1, 1, 10).add(Duration(seconds: i)),
      ),
    );
  }
  return Gpx()
    ..trks = [
      Trk(trksegs: [Trkseg(trkpts: trkpts)]),
    ];
}

/// A slow, jittery track: ~1 m of real progress per sample with GPS noise of
/// the same order, so most hops fall under the 5 m smoothing threshold.
Gpx _jitteryTrack({int points = 100}) {
  final trkpts = <Wpt>[];
  for (var i = 0; i < points; i++) {
    final jitter = (i.isEven ? 1 : -1) * 0.00001;
    trkpts.add(
      Wpt(
        lat: 47.0 + jitter,
        lon: 11.0 + i * 0.00001,
        ele: 1000 + (i % 5) * 2.0,
      ),
    );
  }
  return Gpx()
    ..trks = [
      Trk(trksegs: [Trkseg(trkpts: trkpts)]),
    ];
}

void main() {
  group('buildElevationTrackPoints', () {
    // REGRESSION GUARD. The chart's distanceM was once switched to the SMOOTHED
    // accumulator so the axis maximum would equal the trail's reported
    // distance. That broke this: the smoothed accumulator only advances once
    // per ~5 m, so consecutive samples shared an x and the gradient below
    // divided a one-sample elevation delta by a zero or multi-sample distance
    // delta. On this exact fixture it produced 46 zeroes out of 60 and a max of
    // 2.5%, all inside _gradientColor's "flat" bucket — the chart's gradient
    // colouring was destroyed for any 1 Hz recording.
    test('reports the true grade on a constant 10% climb', () {
      final points = buildElevationTrackPoints(_constantGradeTrack(), 1);

      // Point 0 has no predecessor, so its gradient is 0 by definition.
      final gradients = points.skip(1).map((p) => p.gradient).toList();

      expect(gradients, isNotEmpty);
      for (final g in gradients) {
        expect(g, closeTo(10.0, 0.5));
      }
    });

    test('distanceM advances on every sample, never as a step function', () {
      final points = buildElevationTrackPoints(_constantGradeTrack(), 1);

      for (var i = 1; i < points.length; i++) {
        expect(
          points[i].distanceM,
          greaterThan(points[i - 1].distanceM),
          reason:
              'sample $i shares an x with its predecessor — this is what '
              'breaks the gradient and the waypoint markers',
        );
      }
    });

    test(
      'the axis is raw, and legitimately differs from the trail distance',
      () {
        // Documents a deliberate decision, not an oversight. The smoothed total
        // is a denoised estimate of trail LENGTH; the axis is a plotting
        // COORDINATE saying where each sample sits. Conflating them is what
        // broke the gradient. If the axis maximum must match the reported
        // distance, scale the axis and keep computing the gradient from raw
        // deltas — do not swap the accumulator.
        final gpx = _jitteryTrack();

        final axisMax = buildElevationTrackPoints(gpx, 1).last.distanceM;
        final reported = computeTrailMetrics(gpx).distance;

        var raw = 0.0;
        final pts = gpx.allWaypoints;
        for (var i = 1; i < pts.length; i++) {
          raw += haversineMeters(pts[i - 1], pts[i]);
        }

        expect(axisMax, closeTo(raw, 1e-9));
        expect(axisMax, greaterThan(reported));
      },
    );

    test('a point with no usable coordinate is skipped, not plotted', () {
      // Reachable since the vendored reader leaves lat/lon null for a <trkpt>
      // missing them rather than throwing.
      final gpx = _constantGradeTrack(points: 20);
      gpx.trks.single.trksegs.single.trkpts.insert(
        10,
        Wpt(lat: null, lon: null, ele: 1000),
      );

      final points = buildElevationTrackPoints(gpx, 1);

      expect(points, hasLength(20));
      expect(points.every((p) => p.lonlat.lat.isFinite), isTrue);
      for (var i = 1; i < points.length; i++) {
        expect(points[i].distanceM, greaterThan(points[i - 1].distanceM));
      }
    });

    test('an empty GPX yields no points', () {
      expect(buildElevationTrackPoints(Gpx(), 1), isEmpty);
    });
  });
}
