import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/routes/navigation_screen.dart';

void main() {
  group('liveElevationChartRevision', () {
    test('is full fidelity for a short track — one step per fix', () {
      for (var n = 0; n <= 300; n++) {
        expect(liveElevationChartRevision(n), n);
      }
    });

    test('holds steady across consecutive fixes once the track is long', () {
      // The whole point: 10 fixes in a row must NOT trigger 10 rebuilds, each
      // of which costs two O(n) passes over the entire recording so far.
      final revisions = <int>{
        for (var n = 1000; n < 1010; n++) liveElevationChartRevision(n),
      };
      expect(revisions, hasLength(1));

      // ...and the next stride boundary does move it.
      expect(
        liveElevationChartRevision(1010),
        isNot(liveElevationChartRevision(1009)),
      );
    });

    test('never goes backwards as the recording grows', () {
      var previous = liveElevationChartRevision(0);
      for (var n = 1; n <= 20000; n++) {
        final current = liveElevationChartRevision(n);
        expect(
          current,
          greaterThanOrEqualTo(previous),
          reason: 'revision went backwards at n=$n',
        );
        previous = current;
      }
    });

    test('does not collide across the full-fidelity boundary', () {
      // A collision here would silently drop the first refresh after the
      // threshold, freezing the chart at the boundary.
      expect(
        liveElevationChartRevision(301),
        greaterThan(liveElevationChartRevision(300)),
      );
    });

    test('cuts refreshes for a 4 h 1 Hz recording by ~10x', () {
      // Distinct revisions over the last hour of a 4 h recording ≈ how many
      // times the chart re-parses in that hour.
      final distinct = <int>{
        for (var n = 10800; n <= 14400; n++) liveElevationChartRevision(n),
      };
      expect(distinct.length, closeTo(360, 2)); // was 3601
    });
  });
}
