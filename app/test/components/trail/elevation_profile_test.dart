import 'dart:collection';

import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';

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

    test('the axis is raw, and now coincides with the trail distance', () {
      // These two agree today because the REPORTED distance became raw
      // (quick-260801-opr superseded the 5 m gate, which measured
      // -3.29% against FIT ground truth while raw measured +0.54%). They
      // agree by coincidence of source, not because they mean the same
      // thing: the axis is a plotting COORDINATE saying where each sample
      // sits, while the reported distance is a trail LENGTH.
      //
      // So the original warning still stands even though the inequality it
      // guarded is gone: if a denoised length estimate is ever reintroduced,
      // do NOT drive this axis from it. A smoothed accumulator advances only
      // once per threshold, so consecutive samples share an x and the
      // gradient divides by zero — measured at 46 of 60 points reading 0.0%
      // on a constant 10% grade. Scale the axis instead and keep computing
      // the gradient from raw deltas.
      final gpx = _jitteryTrack();

      final axisMax = buildElevationTrackPoints(gpx, 1).last.distanceM;
      final reported = computeTrailMetrics(gpx).distance;

      var raw = 0.0;
      final pts = gpx.allWaypoints;
      for (var i = 1; i < pts.length; i++) {
        raw += haversineMeters(pts[i - 1], pts[i]);
      }

      expect(axisMax, closeTo(raw, 1e-9));
      expect(reported, closeTo(raw, 1e-9));
      expect(axisMax, closeTo(reported, 1e-9));
    });

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

    test('a point with no usable elevation is skipped, not plotted at 0', () {
      // REGRESSION GUARD. This was `wpt.ele ?? 0`, which drew a cliff from sea
      // level to the device's real altitude at the start of every live
      // recording that seeded from an already-resolved map marker: that first
      // breadcrumb point carries no altitude reading at all.
      final gpx = _constantGradeTrack(points: 20);
      gpx.trks.single.trksegs.single.trkpts.insert(
        0,
        Wpt(lat: 47.0, lon: 11.0, ele: null, time: DateTime.utc(2024, 1, 1, 9)),
      );

      final points = buildElevationTrackPoints(gpx, 1);

      expect(points, hasLength(20));
      expect(points.every((p) => p.elevationM >= 1000), isTrue);
    });

    test('a GPX with no elevation at all yields no points', () {
      // Better an empty state than a fake flat line at sea level.
      final gpx = Gpx()
        ..trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.0, lon: 11.0),
                  Wpt(lat: 47.0, lon: 11.00002),
                ],
              ),
            ],
          ),
        ];

      expect(buildElevationTrackPoints(gpx, 1), isEmpty);
    });
  });

  // REGRESSION GUARD for the live elevation profile shown while RECORDING
  // (navigation_screen's _buildElevationPage). ElevationProfile only re-parses
  // in didUpdateWidget when `oldWidget.gpx != widget.gpx`, and gpx 2.3.0's
  // Gpx/Trkseg `==` delegates to ListEquality, which short-circuits on
  // `identical`. NavigationState.breadcrumb is an identity-stable
  // UnmodifiableListView over a grow-in-place list, so a Gpx built directly
  // over that view is EQUAL to the previous one no matter how many GPS fixes
  // landed — the chart renders once and then freezes. The recording path must
  // therefore pass a per-rebuild COPY.
  group('live breadcrumb Gpx must not alias the recording breadcrumb', () {
    Wpt point(int i) => Wpt(
      lat: 47.0,
      lon: 11.0 + i * 0.00002,
      ele: 1000 + i * 0.15,
      time: DateTime.utc(2024, 1, 1, 10).add(Duration(seconds: i)),
    );

    test('a Gpx over the live view compares equal after the list grows', () {
      final backing = <Wpt>[point(0), point(1)];
      final liveView = UnmodifiableListView(backing);

      final before = buildGpxFromPoints(liveView);
      backing.add(point(2));
      final after = buildGpxFromPoints(liveView);

      // The trap being guarded against, asserted so the reason the copy
      // exists stays documented and provable rather than folklore.
      expect(
        after == before,
        isTrue,
        reason:
            'aliasing the identity-stable breadcrumb view makes successive '
            'Gpx objects compare equal, so didUpdateWidget never re-parses',
      );
    });

    test('a Gpx over a per-rebuild copy compares unequal after growth', () {
      final backing = <Wpt>[point(0), point(1)];
      final liveView = UnmodifiableListView(backing);

      final before = buildGpxFromPoints(List<Wpt>.of(liveView));
      backing.add(point(2));
      final after = buildGpxFromPoints(List<Wpt>.of(liveView));

      expect(after == before, isFalse);
      expect(after.allPoints, hasLength(3));
      // And the parsed chart series actually advances with the new fix.
      expect(
        buildElevationTrackPoints(after, 1).length,
        greaterThan(buildElevationTrackPoints(before, 1).length),
      );
    });
  });
}
