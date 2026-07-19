import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/route_map_matcher.dart';

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
