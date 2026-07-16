import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/util/route_segment_util.dart';

void main() {
  group('segmentKey', () {
    test('derives a stable, anchor-id-pair-based key', () {
      expect(segmentKey('a1', 'a2'), 'a1_a2');
    });

    test('two different pairs never collide', () {
      expect(segmentKey('a1', 'a2') == segmentKey('a2', 'a1'), isFalse);
    });
  });

  group('splitSegmentAt', () {
    const a = Geographic(lat: 47.000, lon: 9.000);
    const b = Geographic(lat: 47.002, lon: 9.000);
    const tapPoint = Geographic(lat: 47.001, lon: 9.000); // midpoint

    test(
      'given a straight 2-point segment, splits at the tap point '
      '(combined polylines cover the original endpoints with the split '
      'point inserted between them)',
      () {
        final segment = RouteSegment(
          beforeAnchorId: 'a1',
          afterAnchorId: 'a2',
          polyline: const [a, b],
          state: SegmentState.straight,
        );

        final (first, second) = splitSegmentAt(segment, 'new1', tapPoint);

        // Identity: the new anchor becomes the shared boundary.
        expect(first.beforeAnchorId, 'a1');
        expect(first.afterAnchorId, 'new1');
        expect(second.beforeAnchorId, 'new1');
        expect(second.afterAnchorId, 'a2');

        // Both halves inherit the original segment's state unchanged.
        expect(first.state, SegmentState.straight);
        expect(second.state, SegmentState.straight);

        // Combined polylines cover the original endpoints...
        expect(first.polyline.first, a);
        expect(second.polyline.last, b);
        // ...with a shared split point genuinely between them (not snapped
        // to either original endpoint).
        expect(first.polyline.last, second.polyline.first);
        final splitPoint = first.polyline.last;
        expect(splitPoint, isNot(a));
        expect(splitPoint, isNot(b));
        expect(splitPoint.lat, closeTo(tapPoint.lat, 0.0005));
        expect(splitPoint.lon, closeTo(tapPoint.lon, 0.0005));
      },
    );

    test('inherits a blocked segment\'s state on both halves', () {
      final segment = RouteSegment(
        beforeAnchorId: 'a1',
        afterAnchorId: 'a2',
        polyline: const [a, b],
        state: SegmentState.blocked,
      );

      final (first, second) = splitSegmentAt(segment, 'new1', tapPoint);

      expect(first.state, SegmentState.blocked);
      expect(second.state, SegmentState.blocked);
    });
  });
}
