import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/route/map_matcher.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

List<double> _cumulativeMeters(List<Geographic> shape) {
  final cum = List<double>.filled(shape.length, 0.0);
  for (var i = 1; i < shape.length; i++) {
    cum[i] =
        cum[i - 1] + SphericalGreatCircle(shape[i - 1]).distanceTo(shape[i]);
  }
  return cum;
}

/// A straight route running north, points ~11.1 m apart (0.0001° latitude).
List<Geographic> _straightShape({int points = 30}) {
  return List.generate(
    points,
    (i) => Geographic(lat: 47.0000 + 0.0001 * i, lon: 9.0000),
  );
}

/// A tight hairpin: an "entry" leg climbing north, then an "exit" leg
/// descending back south only ~15 m away (longitude offset 0.0002° at this
/// latitude). Every point on the entry leg has a spatially-close counterpart
/// on the exit leg that is hundreds of metres away *along the route* — the
/// exact ambiguity that fooled a nearest-point projection.
List<Geographic> _hairpinShape() {
  const startLat = 47.0000;
  const stepLat = 0.0001; // ~11.1 m
  const entryLon = 9.0000;
  const exitLon = 9.0002; // ~15.2 m east at this latitude

  final shape = <Geographic>[];
  for (var i = 0; i <= 50; i++) {
    shape.add(Geographic(lat: startLat + stepLat * i, lon: entryLon));
  }
  for (var i = 50; i >= 0; i--) {
    shape.add(Geographic(lat: startLat + stepLat * i, lon: exitLon));
  }
  return shape;
}

/// Two parallel legs joined by a ~78 m north-south connector — wide enough
/// that a genuine cross-country shortcut between them pushes cross-track
/// distance from the outbound leg's local window past the off-route
/// threshold, unlike the tight hairpin above.
List<Geographic> _wideGapLoopShape() {
  final shape = <Geographic>[];
  // Outbound leg: east along lat 47.0000, lon 9.0000 -> 9.0050.
  for (var i = 0; i <= 25; i++) {
    shape.add(Geographic(lat: 47.0000, lon: 9.0000 + 0.0002 * i));
  }
  // Inbound leg: ~78 m north, west along lon 9.0050 -> 9.0000.
  for (var i = 0; i <= 25; i++) {
    shape.add(Geographic(lat: 47.0007, lon: 9.0050 - 0.0002 * i));
  }
  return shape;
}

/// A ~330 m out-and-back spur: the return leg retraces the outbound leg's
/// exact coordinates, so every point before the apex has a spatially
/// *identical* twin on the return leg a couple hundred metres away
/// along-track — inside the off-route recovery window, unlike
/// [_hairpinShape] (offset ~15 m, never truly coincident) or
/// [_wideGapLoopShape] (a different, unconnected leg ~78 m away). This is
/// the geometry that produced the reported false "route finished" while
/// biking: `shape[j] == shape[60 - j]` for `j <= 29`.
List<Geographic> _narrowSpurShape({int legPoints = 30}) {
  const startLat = 47.0000;
  const lon = 9.0000;
  const stepLat = 0.0001; // ~11.1 m
  final outbound = List.generate(
    legPoints + 1,
    (i) => Geographic(lat: startLat + stepLat * i, lon: lon),
  );
  final inbound = outbound.reversed.skip(1).toList();
  return [...outbound, ...inbound];
}

/// Linearly interpolates [steps] intermediate points from [a] to [b]
/// (inclusive of [b], exclusive of [a]).
List<Geographic> _interpolate(Geographic a, Geographic b, int steps) {
  return List.generate(steps, (i) {
    final t = (i + 1) / steps;
    return Geographic(
      lat: a.lat + (b.lat - a.lat) * t,
      lon: a.lon + (b.lon - a.lon) * t,
    );
  });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('RouteMapMatcher', () {
    test('straight route: sequential fixes track along-track progress', () {
      final shape = _straightShape();
      final cumulative = _cumulativeMeters(shape);
      final matcher = RouteMapMatcher(
        shape: shape,
        shapeCumulativeMeters: cumulative,
      );

      MapMatchResult? result;
      for (var i = 0; i < shape.length; i++) {
        result = matcher.update(pos: shape[i]);
      }

      expect(result!.alongTrackMeters, closeTo(cumulative.last, 2.0));
      expect(result.shapeIndex, greaterThanOrEqualTo(shape.length - 2));
    });

    test(
      'hairpin regression: a single noisy fix near the apex does not snap '
      'to the spatially-close-but-far-along-route exit leg',
      () {
        final shape = _hairpinShape();
        final cumulative = _cumulativeMeters(shape);
        final matcher = RouteMapMatcher(
          shape: shape,
          shapeCumulativeMeters: cumulative,
        );

        // Walk the first 10 points of the entry leg for real (on-line).
        for (var i = 0; i <= 9; i++) {
          matcher.update(pos: shape[i]);
        }

        // One noisy fix near shape[10], offset toward the exit leg by ~11 m
        // — enough that it is spatially *closer* to the exit leg's
        // same-latitude point than to the entry leg it's actually on
        // (typical outdoor GPS error under canopy).
        final apex = shape[10];
        final noisyFix = Geographic(lat: apex.lat, lon: apex.lon + 0.00015);
        final result = matcher.update(pos: noisyFix);

        // Confirm the ambiguity is real: the exit leg's same-latitude point
        // is far away along the route.
        final exitLegIndex = 101 - 10; // mirror of entry index 10
        final alongRouteGap =
            cumulative[exitLegIndex] - cumulative[10];
        expect(alongRouteGap, greaterThan(400));

        // The matcher must not make that jump on one noisy fix.
        expect(
          (result.alongTrackMeters - cumulative[10]).abs(),
          lessThan(60),
        );
      },
    );

    test(
      'genuine shortcut: a sustained sequence of fixes cutting across a wide '
      'gap advances the match onto the other leg',
      () {
        final shape = _wideGapLoopShape();
        final cumulative = _cumulativeMeters(shape);
        final matcher = RouteMapMatcher(
          shape: shape,
          shapeCumulativeMeters: cumulative,
        );

        // Walk the first part of the outbound leg for real.
        for (var i = 0; i <= 15; i++) {
          matcher.update(pos: shape[i]);
        }
        final beforeShortcut = cumulative[15];

        // Cut diagonally across the ~79 m gap to a point on the inbound leg,
        // over several fixes (not a single teleport).
        const targetIndex = 35;
        final steps = _interpolate(shape[15], shape[targetIndex], 6);
        MapMatchResult? result;
        for (final step in steps) {
          result = matcher.update(pos: step);
        }
        // A jump this large must keep winning for several consecutive fixes
        // before it's committed (see commitConfirmFixes) — a real shortcut
        // does exactly this, so confirm it by staying at the target.
        for (var i = 0; i < 3; i++) {
          result = matcher.update(pos: shape[targetIndex]);
        }

        // This genuinely saves several hundred metres of route distance for
        // ~80 m of cross-country walking — confirm the setup is meaningful.
        expect(cumulative[targetIndex] - beforeShortcut, greaterThan(300));

        // The matcher should have re-acquired the inbound leg near the
        // target, not stayed stuck near the outbound leg.
        expect(
          result!.alongTrackMeters,
          closeTo(cumulative[targetIndex], 80.0),
        );
        expect(result.alongTrackMeters, greaterThan(beforeShortcut + 250));
      },
    );

    test('repeated near-identical fixes (stationary) do not drift', () {
      final shape = _straightShape();
      final cumulative = _cumulativeMeters(shape);
      final matcher = RouteMapMatcher(
        shape: shape,
        shapeCumulativeMeters: cumulative,
      );

      matcher.update(pos: shape[10]);
      final committed = matcher.update(pos: shape[10]).alongTrackMeters;

      // A handful of near-identical fixes (GPS jitter at rest) shouldn't
      // walk the along-track distance away from where the user is standing.
      for (var i = 0; i < 5; i++) {
        final jittered = Geographic(
          lat: shape[10].lat + (i.isEven ? 0.0000005 : -0.0000005),
          lon: shape[10].lon,
        );
        matcher.update(pos: jittered);
      }
      final after = matcher.update(pos: shape[10]).alongTrackMeters;

      expect((after - committed).abs(), lessThan(5.0));
    });

    test(
      'narrow out-and-back spur: sustained wide-cornering fixes at bike '
      'speed do not commit onto the spatially-identical return leg',
      () {
        final shape = _narrowSpurShape();
        final cumulative = _cumulativeMeters(shape);

        // Confirm the fixture reproduces the reported bug's geometry:
        // identical coordinates, along-track gap inside the recovery window
        // but outside the normal (non-recovery) window.
        expect(shape[20], shape[40]);
        final gap = cumulative[40] - cumulative[20];
        expect(gap, lessThan(400));
        expect(gap, greaterThan(150));

        final matcher = RouteMapMatcher(
          shape: shape,
          shapeCumulativeMeters: cumulative,
        );

        // Walk the outbound leg for real, at bike speed.
        for (var i = 0; i <= 18; i++) {
          matcher.update(pos: shape[i], speed: 6.0);
        }

        // A burst of wide-cornering fixes through the danger zone — offset
        // enough to exceed even the speed-adjusted recovery trigger for
        // several consecutive fixes, as a bike's wider turning radius
        // plausibly would on a spur originally recorded on foot.
        for (var i = 19; i <= 22; i++) {
          final wide = Geographic(
            lat: shape[i].lat,
            lon: shape[i].lon + 0.0009, // ~68 m east
          );
          matcher.update(pos: wide, speed: 6.0);
        }

        // Resume tracking correctly along the real outbound leg.
        MapMatchResult? result;
        for (var i = 23; i <= 29; i++) {
          result = matcher.update(pos: shape[i], speed: 6.0);
        }

        // Must reflect forward progress on the true (outbound) leg, not a
        // false snap onto the spatially-identical return leg.
        expect(result!.alongTrackMeters, closeTo(cumulative[29], 80.0));
        expect(result.alongTrackMeters, lessThan(cumulative[35]));
      },
    );

    test(
      'bike cornering: moderate cross-track fixes at bike speed stay in the '
      'normal window instead of forcing recovery',
      () {
        final shape = _straightShape();
        final cumulative = _cumulativeMeters(shape);
        final matcher = RouteMapMatcher(
          shape: shape,
          shapeCumulativeMeters: cumulative,
        );

        matcher.update(pos: shape[5], speed: 6.0);

        // ~46 m offset: past the walking-tuned threshold (40 m) but under
        // the bike-speed-adjusted one (40 + 3*6 = 58 m) — a wide corner, not
        // a genuine excursion.
        MapMatchResult? result;
        for (var i = 6; i <= 9; i++) {
          final wide = Geographic(
            lat: shape[i].lat,
            lon: shape[i].lon + 0.00042,
          );
          result = matcher.update(pos: wide, speed: 6.0);
        }

        // Should have kept tracking forward along the route, not stalled or
        // jumped — the wide cornering shouldn't have been treated as
        // off-route at all.
        expect(result!.alongTrackMeters, greaterThan(cumulative[5]));
        expect(result.alongTrackMeters, closeTo(cumulative[9], 15.0));
      },
    );

    test('a poor-accuracy fix does not throw and still returns a finite match', () {
      final shape = _straightShape();
      final cumulative = _cumulativeMeters(shape);
      final matcher = RouteMapMatcher(
        shape: shape,
        shapeCumulativeMeters: cumulative,
      );

      matcher.update(pos: shape[5]);
      final result = matcher.update(
        pos: Geographic(lat: shape[6].lat, lon: shape[6].lon + 0.0003),
        accuracy: 50.0,
      );

      expect(result.alongTrackMeters.isFinite, isTrue);
      expect(result.alongTrackMeters, greaterThanOrEqualTo(0.0));
    });
  });
}
