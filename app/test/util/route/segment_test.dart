import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/util/route/segment.dart';

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

  group('buildSegmentsGeoJson', () {
    const a = Geographic(lat: 47.000, lon: 9.000);
    const b = Geographic(lat: 47.002, lon: 9.000);
    const c = Geographic(lat: 47.004, lon: 9.000);

    test(
      'produces one Feature per segment, with beforeAnchorId/afterAnchorId/'
      'state properties and lon/lat-ordered coordinates matching the '
      'source polyline',
      () {
        final segments = [
          RouteSegment(
            beforeAnchorId: 'a1',
            afterAnchorId: 'a2',
            polyline: const [a, b],
            state: SegmentState.routed,
          ),
          RouteSegment(
            beforeAnchorId: 'a2',
            afterAnchorId: 'a3',
            polyline: const [b, c],
            state: SegmentState.blocked,
          ),
        ];

        final decoded =
            jsonDecode(buildSegmentsGeoJson(segments)) as Map<String, Object?>;
        expect(decoded['type'], 'FeatureCollection');
        final features = decoded['features'] as List<Object?>;
        expect(features.length, 2);

        final first = features[0] as Map<String, Object?>;
        final firstProps = first['properties'] as Map<String, Object?>;
        expect(firstProps['beforeAnchorId'], 'a1');
        expect(firstProps['afterAnchorId'], 'a2');
        expect(firstProps['state'], 'routed');
        final firstGeometry = first['geometry'] as Map<String, Object?>;
        expect(firstGeometry['type'], 'LineString');
        expect(firstGeometry['coordinates'], [
          [a.lon, a.lat],
          [b.lon, b.lat],
        ]);

        final second = features[1] as Map<String, Object?>;
        final secondProps = second['properties'] as Map<String, Object?>;
        expect(secondProps['beforeAnchorId'], 'a2');
        expect(secondProps['afterAnchorId'], 'a3');
        expect(secondProps['state'], 'blocked');
        final secondGeometry = second['geometry'] as Map<String, Object?>;
        expect(secondGeometry['coordinates'], [
          [b.lon, b.lat],
          [c.lon, c.lat],
        ]);
      },
    );

    test(
      'an empty segment list produces a valid FeatureCollection with an '
      'empty features array (not null, does not throw)',
      () {
        final decoded =
            jsonDecode(buildSegmentsGeoJson(const [])) as Map<String, Object?>;
        expect(decoded['type'], 'FeatureCollection');
        expect(decoded['features'], isNotNull);
        expect(decoded['features'], isEmpty);
      },
    );

    test('output is valid JSON parseable by jsonDecode', () {
      final segments = [
        RouteSegment(
          beforeAnchorId: 'a1',
          afterAnchorId: 'a2',
          polyline: const [a, b],
          state: SegmentState.straight,
        ),
      ];

      expect(
        () => jsonDecode(buildSegmentsGeoJson(segments)),
        returnsNormally,
      );
      final decoded =
          jsonDecode(buildSegmentsGeoJson(segments)) as Map<String, Object?>;
      expect(decoded['type'], 'FeatureCollection');
    });
  });
}
