import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' show Offset, Size;

import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' show Geographic;
import 'package:wanderer/util/map/batched_projection.dart';

/// A stand-in for the platform's `toScreenLocation`: Web-Mercator onto a
/// screen, with the camera's centre/zoom/bearing, optional perspective
/// ([tilt] is a homography strength, 0 = pitch 0), and float32 output like
/// Android's `PointF`. Counts calls so tests can pin the native-call budget.
class _SimulatedPlatform {
  _SimulatedPlatform({
    this.center = const Geographic(lon: 11.0, lat: 47.0),
    this.zoom = 13,
    this.bearingDeg = 0,
    this.tilt = 0,
  });

  final Geographic center;
  final double zoom;
  final double bearingDeg;
  final double tilt;
  static const size = Size(412, 892);
  int calls = 0;

  static double _mx(double lon) => (lon + 180) / 360;
  static double _my(double lat) {
    final s = math.sin(lat * math.pi / 180);
    return 0.5 - math.log((1 + s) / (1 - s)) / (4 * math.pi);
  }

  static double _f32(double v) => (Float32List(1)..[0] = v)[0];

  Offset call(Geographic p) {
    calls++;
    final worldPx = 512 * math.pow(2, zoom);
    final dx = (_mx(p.lon) - _mx(center.lon)) * worldPx;
    final dy = (_my(p.lat) - _my(center.lat)) * worldPx;
    final t = -bearingDeg * math.pi / 180;
    var x = dx * math.cos(t) - dy * math.sin(t);
    var y = dx * math.sin(t) + dy * math.cos(t);
    if (tilt != 0) {
      final w = 1 + tilt * y / size.height;
      x /= w;
      y /= w;
    }
    return Offset(_f32(size.width / 2 + x), _f32(size.height / 2 + y));
  }
}

List<Geographic> _scatter(
  int n, {
  required Geographic around,
  required double spreadDeg,
  int seed = 7,
}) {
  final rnd = math.Random(seed);
  return [
    for (var i = 0; i < n; i++)
      Geographic(
        lon: around.lon + (rnd.nextDouble() - 0.5) * spreadDeg,
        lat: around.lat + (rnd.nextDouble() - 0.5) * spreadDeg,
      ),
  ];
}

double _worstError(List<Offset> a, List<Offset> b) {
  expect(a.length, b.length);
  var worst = 0.0;
  for (var i = 0; i < a.length; i++) {
    worst = math.max(worst, (a[i] - b[i]).distance);
  }
  return worst;
}

void main() {
  group('projectBatched', () {
    test('projects a large set with five native calls, matching per-point', () {
      final points = _scatter(
        60,
        around: const Geographic(lon: 11.0, lat: 47.0),
        spreadDeg: 0.2,
      );
      final batched = _SimulatedPlatform();
      final direct = _SimulatedPlatform();

      final got = projectBatched(points, batched.call);
      final want = [for (final p in points) direct.call(p)];

      expect(batched.calls, 5, reason: 'N must not drive native calls');
      expect(_worstError(got, want), lessThan(0.05));
    });

    test(
      'is exact across zoom levels and bearings (affine, not approximate)',
      () {
        final points = _scatter(
          25,
          around: const Geographic(lon: -73.98, lat: 40.75),
          spreadDeg: 0.05,
        );
        for (final zoom in [3.0, 8.0, 12.5, 16.0, 19.0]) {
          for (final bearing in [0.0, 37.0, 90.0, 181.0, 300.0]) {
            final batched = _SimulatedPlatform(
              center: const Geographic(lon: -73.98, lat: 40.75),
              zoom: zoom,
              bearingDeg: bearing,
            );
            final direct = _SimulatedPlatform(
              center: const Geographic(lon: -73.98, lat: 40.75),
              zoom: zoom,
              bearingDeg: bearing,
            );

            final got = projectBatched(points, batched.call);
            final want = [for (final p in points) direct.call(p)];

            expect(
              _worstError(got, want),
              lessThanOrEqualTo(0.5),
              reason: 'zoom $zoom bearing $bearing',
            );
          }
        }
      },
    );

    test('markers far off screen are projected correctly too', () {
      // A long route: most markers are thousands of pixels off screen.
      final points = _scatter(
        40,
        around: const Geographic(lon: 11.0, lat: 47.0),
        spreadDeg: 2,
      );
      final batched = _SimulatedPlatform(zoom: 10);
      final direct = _SimulatedPlatform(zoom: 10);

      final got = projectBatched(points, batched.call);
      final want = [for (final p in points) direct.call(p)];

      expect(_worstError(got, want), lessThanOrEqualTo(0.5));
    });

    test('below the batch threshold every point is projected natively', () {
      final points = _scatter(
        7,
        around: const Geographic(lon: 11.0, lat: 47.0),
        spreadDeg: 0.1,
      );
      final platform = _SimulatedPlatform();

      final got = projectBatched(points, platform.call);

      expect(platform.calls, 7);
      expect(got, [for (final p in points) _SimulatedPlatform().call(p)]);
    });

    test('a perspective camera falls back to exact per-point projection', () {
      final points = _scatter(
        30,
        around: const Geographic(lon: 11.0, lat: 47.0),
        spreadDeg: 0.1,
      );
      final batched = _SimulatedPlatform(tilt: 0.3);
      final direct = _SimulatedPlatform(tilt: 0.3);

      final got = projectBatched(points, batched.call);
      final want = [for (final p in points) direct.call(p)];

      expect(got, want, reason: 'validation must reject the affine model');
      expect(batched.calls, 5 + points.length);
    });

    test('a tall, narrow set under tilt is still rejected (centre check)', () {
      // A north-south line: the corner mismatch is only a few pixels across
      // the narrow width while the mid-span error along the length is large.
      // Only the centre probe sees it.
      final points = [
        for (var i = 0; i < 20; i++)
          Geographic(lon: 11.0 + i * 0.00002, lat: 46.95 + i * 0.005),
      ];
      final batched = _SimulatedPlatform(zoom: 12, tilt: 0.05);
      final direct = _SimulatedPlatform(zoom: 12, tilt: 0.05);

      final got = projectBatched(points, batched.call);
      final want = [for (final p in points) direct.call(p)];

      expect(_worstError(got, want), lessThanOrEqualTo(0.5));
      expect(
        batched.calls,
        5 + points.length,
        reason: 'fell back after the centre probe failed',
      );
    });

    test('points on a single parallel or meridian still batch exactly', () {
      final sameLat = [
        for (var i = 0; i < 12; i++)
          Geographic(lon: 11.0 + i * 0.003, lat: 47.0),
      ];
      final sameLon = [
        for (var i = 0; i < 12; i++)
          Geographic(lon: 11.0, lat: 47.0 + i * 0.003),
      ];
      for (final points in [sameLat, sameLon]) {
        final batched = _SimulatedPlatform(bearingDeg: 20);
        final direct = _SimulatedPlatform(bearingDeg: 20);

        final got = projectBatched(points, batched.call);
        final want = [for (final p in points) direct.call(p)];

        expect(batched.calls, 5);
        expect(_worstError(got, want), lessThan(0.05));
      }
    });

    test('coincident points fall back instead of dividing by zero', () {
      final points = List.filled(10, const Geographic(lon: 11.0, lat: 47.0));
      final platform = _SimulatedPlatform();

      final got = projectBatched(points, platform.call);

      expect(platform.calls, 10);
      expect(got.every((o) => o.dx.isFinite && o.dy.isFinite), isTrue);
    });

    test('a set spanning the antimeridian is projected natively', () {
      final points = [
        for (var i = 0; i < 10; i++)
          Geographic(lon: i.isEven ? 179.5 : -179.5, lat: 10.0 + i * 0.01),
      ];
      final platform = _SimulatedPlatform(
        center: const Geographic(lon: 180, lat: 10),
      );

      projectBatched(points, platform.call);

      expect(platform.calls, points.length);
    });

    test('non-finite input falls back without throwing', () {
      final points = [
        for (var i = 0; i < 10; i++)
          Geographic(lon: 11.0 + i * 0.01, lat: i == 3 ? double.nan : 47.0),
      ];
      final platform = _SimulatedPlatform();

      expect(() => projectBatched(points, platform.call), returnsNormally);
      expect(platform.calls, points.length);
    });

    test('empty input makes no native calls', () {
      final platform = _SimulatedPlatform();

      expect(projectBatched(const [], platform.call), isEmpty);
      expect(platform.calls, 0);
    });
  });
}
