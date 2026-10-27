// ignore_for_file: depend_on_referenced_packages
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/route_segment_layer.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/util/route/segment.dart';

/// Records what the segment layer asks the native style to do. Every
/// `updateGeoJsonSource` is a whole-route string handed to MapLibre over JNI
/// (about 2 ms at rest and up to 27 ms inside a creation burst on a Pixel for a
/// ~570-point route), so how often it is called is the thing under test.
class _RecordingStyle extends ml.StyleController {
  final calls = <String>[];
  final uploads = <String>[];

  /// When true the next [updateGeoJsonSource] throws once.
  bool failNextUpload = false;

  @override
  Future<void> addSource(ml.Source source) async => calls.add('addSource');

  @override
  Future<void> addLayer(
    ml.StyleLayer layer, {
    String? belowLayerId,
    String? aboveLayerId,
    int? atIndex,
  }) async => calls.add('addLayer');

  @override
  Future<void> removeLayer(String id) async => calls.add('removeLayer');

  @override
  Future<void> removeSource(String id) async => calls.add('removeSource');

  @override
  Future<void> updateGeoJsonSource({
    required String id,
    required String data,
  }) async {
    if (failNextUpload) {
      failNextUpload = false;
      throw StateError('native upload failed');
    }
    calls.add('upload');
    uploads.add(data);
  }

  int count(String call) => calls.where((c) => c == call).length;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

List<ml.Geographic> _line(double lon, int points) => [
  for (var i = 0; i < points; i++)
    ml.Geographic(lon: lon + i * 1e-4, lat: 47 + i * 1e-4),
];

RouteSegment _segment(
  String before,
  String after, {
  List<ml.Geographic>? polyline,
  SegmentState state = SegmentState.routed,
}) => RouteSegment(
  beforeAnchorId: before,
  afterAnchorId: after,
  polyline: polyline ?? _line(9, 30),
  state: state,
);

void main() {
  late _RecordingStyle style;
  late RouteSegmentLayer layer;

  setUp(() {
    style = _RecordingStyle();
    layer = RouteSegmentLayer();
  });

  test(
    'the first update adds the source and layers and uploads nothing else',
    () async {
      await layer.update(style, [_segment('a', 'b')]);

      expect(style.count('addSource'), 1);
      expect(style.count('addLayer'), 5);
      expect(style.count('upload'), 0);
    },
  );

  test('an elevation result for unchanged geometry is not uploaded again '
      '(it carries no GeoJSON the map has not already got)', () async {
    final route = _segment('a', 'b');
    await layer.update(style, [route]);

    // What _resolveElevation does: same ids, state and polyline, new heights.
    final withHeights = route.copyWith(
      elevationProfile: [ml.Geographic(lat: 47, lon: 9)],
      elevations: const [123.0],
      durationSeconds: 600,
    );
    await layer.update(style, [withHeights]);
    await layer.update(style, [
      withHeights.copyWith(elevations: const [5]),
    ]);

    expect(style.count('upload'), 0);
  });

  test('the same route rebuilt into fresh lists (equal GeoJSON, different '
      'instances) is not uploaded again either', () async {
    await layer.update(style, [_segment('a', 'b')]);
    await layer.update(style, [_segment('a', 'b', polyline: _line(9, 30))]);
    await layer.update(style, [_segment('a', 'b', polyline: _line(9, 30))]);

    expect(style.count('upload'), 0);
  });

  test(
    'a changed polyline is uploaded, with exactly the new GeoJSON',
    () async {
      await layer.update(style, [_segment('a', 'b')]);

      final routed = [_segment('a', 'b', polyline: _line(9, 80))];
      await layer.update(style, routed);

      expect(style.uploads, [buildSegmentsGeoJson(routed)]);
    },
  );

  test('a changed segment state is uploaded', () async {
    final route = _segment('a', 'b');
    await layer.update(style, [route]);

    await layer.update(style, [route.copyWith(state: SegmentState.blocked)]);

    expect(style.count('upload'), 1);
    expect(style.uploads.single, contains('"state":"blocked"'));
  });

  test('an added or removed segment is uploaded', () async {
    final first = _segment('a', 'b');
    final second = _segment('b', 'c', polyline: _line(9.1, 30));
    await layer.update(style, [first]);

    await layer.update(style, [first, second]);
    await layer.update(style, [first]);

    expect(style.count('upload'), 2);
    expect(style.uploads.last, buildSegmentsGeoJson([first]));
  });

  test('skipping an unchanged update does not make the layer forget the '
      'geometry: the next real change still reaches the map', () async {
    final route = _segment('a', 'b');
    await layer.update(style, [route]);
    await layer.update(style, [
      route.copyWith(elevations: const [1.0]),
    ]);

    final moved = [_segment('a', 'b', polyline: _line(9.5, 30))];
    await layer.update(style, moved);
    // And a return to the original geometry is a change relative to `moved`.
    await layer.update(style, [route]);

    expect(style.uploads, [
      buildSegmentsGeoJson(moved),
      buildSegmentsGeoJson([route]),
    ]);
  });

  test(
    'a different style is never skipped (it has none of our layers yet)',
    () async {
      final route = _segment('a', 'b');
      await layer.update(style, [route]);

      final reloaded = _RecordingStyle();
      await layer.update(reloaded, [route]);

      expect(reloaded.count('upload'), 1);
    },
  );

  test(
    'after remove() the next update starts over instead of being skipped',
    () async {
      final route = _segment('a', 'b');
      await layer.update(style, [route]);
      await layer.remove(style);

      await layer.update(style, [route]);

      expect(style.count('addSource'), 2);
      expect(style.count('addLayer'), 10);
    },
  );

  test('a failed upload is retried by the next identical update', () async {
    final route = _segment('a', 'b');
    await layer.update(style, [route]);

    final changed = [_segment('a', 'b', polyline: _line(9, 80))];
    style.failNextUpload = true;
    await expectLater(layer.update(style, changed), throwsStateError);

    await layer.update(style, changed);

    expect(style.uploads, [buildSegmentsGeoJson(changed)]);
  });
}
