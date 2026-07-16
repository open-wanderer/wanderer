import 'package:flutter_test/flutter_test.dart';
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
}
