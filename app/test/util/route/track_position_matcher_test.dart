import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/route/track_position_matcher.dart';

// ---------------------------------------------------------------------------
// Helpers — copied from map_matcher_test.dart (private there).
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

void main() {
  group('TrackPositionMatcher', () {
    test(
      'straight track: sequential fixes exactly on points return along-track '
      'values within 5 m of each point\'s cumulative distance',
      () {
        final shape = _straightShape();
        final cumulative = _cumulativeMeters(shape);
        final matcher = TrackPositionMatcher(
          shape: shape,
          cumulativeMeters: cumulative,
        );

        for (final i in [0, 5, 10, 15, 20]) {
          final result = matcher.update(shape[i]);
          expect(result, isNotNull);
          expect(result!, closeTo(cumulative[i], 5.0));
        }
      },
    );

    test(
      'a fix 200 m off-track after a warm fix on the track returns null',
      () {
        final shape = _straightShape();
        final cumulative = _cumulativeMeters(shape);
        final matcher = TrackPositionMatcher(
          shape: shape,
          cumulativeMeters: cumulative,
        );

        matcher.update(shape[10]);
        final farFix = Geographic(
          lat: shape[10].lat,
          lon: shape[10].lon + 0.0027,
        );
        final result = matcher.update(farFix);

        expect(result, isNull);
      },
    );

    test('pointAtAlongTrack at a half-segment offset lies within 1 m of the '
        'segment midpoint', () {
      final shape = _straightShape();
      final cumulative = _cumulativeMeters(shape);
      final matcher = TrackPositionMatcher(
        shape: shape,
        cumulativeMeters: cumulative,
      );

      final segLen = cumulative[11] - cumulative[10];
      final midpoint = matcher.pointAtAlongTrack(
        cumulative[10] + segLen / 2,
        10,
      );
      final expectedMidpoint = SphericalGreatCircle(
        shape[10],
      ).intermediatePointTo(shape[11], fraction: 0.5);

      expect(
        SphericalGreatCircle(midpoint).distanceTo(expectedMidpoint),
        lessThan(1.0),
      );
    });

    test('a zero-length first segment does not produce NaN from '
        'pointAtAlongTrack(0, 0)', () {
      final shape = [
        Geographic(lat: 47.0000, lon: 9.0000),
        Geographic(lat: 47.0000, lon: 9.0000),
        Geographic(lat: 47.0001, lon: 9.0000),
      ];
      final cumulative = _cumulativeMeters(shape);
      final matcher = TrackPositionMatcher(
        shape: shape,
        cumulativeMeters: cumulative,
      );

      final point = matcher.pointAtAlongTrack(0, 0);

      expect(point.lat.isNaN, isFalse);
      expect(point.lon.isNaN, isFalse);
      expect(point, shape[0]);
    });

    test('a matcher built over a single point returns null from update without '
        'throwing', () {
      final shape = [Geographic(lat: 47.0000, lon: 9.0000)];
      final matcher = TrackPositionMatcher(
        shape: shape,
        cumulativeMeters: [0.0],
      );

      expect(() => matcher.update(shape[0]), returnsNormally);
      expect(matcher.update(shape[0]), isNull);
    });

    test('a matcher built over empty lists returns null from update without '
        'throwing', () {
      final matcher = TrackPositionMatcher(shape: [], cumulativeMeters: []);

      expect(
        () => matcher.update(Geographic(lat: 47.0, lon: 9.0)),
        returnsNormally,
      );
      expect(matcher.update(Geographic(lat: 47.0, lon: 9.0)), isNull);
    });

    test('onTrackThresholdMeters is honoured', () {
      final shape = _straightShape();
      final cumulative = _cumulativeMeters(shape);

      final strictMatcher = TrackPositionMatcher(
        shape: shape,
        cumulativeMeters: cumulative,
        onTrackThresholdMeters: 5,
      );
      strictMatcher.update(shape[10]);
      final closeFix = Geographic(
        lat: shape[10].lat,
        lon: shape[10].lon + 0.000105,
      );
      expect(strictMatcher.update(closeFix), isNull);

      final defaultMatcher = TrackPositionMatcher(
        shape: shape,
        cumulativeMeters: cumulative,
      );
      defaultMatcher.update(shape[10]);
      final result = defaultMatcher.update(closeFix);
      expect(result, isNotNull);
    });
  });
}
