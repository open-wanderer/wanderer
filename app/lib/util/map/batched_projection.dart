import 'dart:math' as math;
import 'dart:ui' show Offset;

import 'package:maplibre/maplibre.dart' show Geographic;

/// Projects [points] to screen offsets with a bounded number of [project]
/// calls instead of one per point.
///
/// [project] is the platform's own projection (`MapController.toScreenLocation`),
/// which on Android is several synchronous JNI round trips per point. The stock
/// `WidgetLayer` makes N of them on every camera frame; this makes five.
///
/// How: with the camera at pitch 0, a map is an orthographic view of the
/// Web-Mercator plane, so screen position is an exact affine function of
/// mercator (x, y) — whatever the zoom, bearing, content insets or platform.
/// The affine map is read off the platform itself, not re-derived from camera
/// parameters: the four corners of the points' mercator bounding box are
/// projected natively, and every point is then bilinearly interpolated between
/// them (`a`, `b` are in [0, 1], so float error never amplifies). Two more
/// facts verify the model before it is trusted — the fourth corner and the
/// centre must land within [tolerancePx] of what the affine map predicts. Any
/// doubt (pitch, a span over 180 degrees of longitude where the platform
/// unwraps across the antimeridian, non-finite input, float32 precision at
/// extreme zoom) falls back to projecting every point natively, so the result
/// is never worse than per-point projection, only slower.
///
/// The centre check matters: for a tall, narrow set of points under a tilted
/// camera the corner mismatch is tiny while the mid-span error is large.
///
/// Callers must still gate on camera pitch (see `ScreenMarkerLayer`): the
/// validation is a safety net, the pitch gate is what makes the model exact.
List<Offset> projectBatched(
  List<Geographic> points,
  Offset Function(Geographic point) project, {
  int minBatchSize = 8,
  double tolerancePx = 0.5,
}) {
  final n = points.length;
  if (n < minBatchSize) return _projectEach(points, project);

  final xs = List<double>.filled(n, 0);
  final ys = List<double>.filled(n, 0);
  var minX = double.infinity;
  var maxX = double.negativeInfinity;
  var minY = double.infinity;
  var maxY = double.negativeInfinity;
  for (var i = 0; i < n; i++) {
    final lon = points[i].lon;
    final lat = points[i].lat;
    // `num.clamp` maps NaN to the upper limit, so reject it before clamping.
    if (!lon.isFinite || !lat.isFinite) return _projectEach(points, project);
    final x = _mercatorX(lon);
    final y = _mercatorY(lat);
    xs[i] = x;
    ys[i] = y;
    if (x < minX) minX = x;
    if (x > maxX) maxX = x;
    if (y < minY) minY = y;
    if (y > maxY) maxY = y;
  }

  // The platform unwraps longitude around the camera centre; beyond half the
  // world the bounding box no longer describes one contiguous patch.
  if (maxX - minX > 0.5) return _projectEach(points, project);

  var spanX = maxX - minX;
  var spanY = maxY - minY;
  var originX = minX;
  var originY = minY;
  const degenerate = 1e-12;
  if (spanX < degenerate && spanY < degenerate) {
    return _projectEach(points, project);
  }
  // Points on one meridian/parallel: widen the flat axis so the reference
  // quad is not degenerate. Every point then sits at 0.5 on that axis.
  if (spanX < degenerate) {
    spanX = spanY;
    originX = minX - spanX / 2;
  } else if (spanY < degenerate) {
    spanY = spanX;
    originY = minY - spanY / 2;
  }

  Offset at(double a, double b) => project(
    Geographic(
      lon: _lonFromMercatorX(originX + a * spanX),
      lat: _latFromMercatorY(originY + b * spanY),
    ),
  );

  final p00 = at(0, 0);
  final e1 = at(1, 0) - p00;
  final e2 = at(0, 1) - p00;
  final corner = at(1, 1);
  final centre = at(0.5, 0.5);

  final affineHolds =
      (p00 + e1 + e2 - corner).distance <= tolerancePx &&
      (p00 + e1 * 0.5 + e2 * 0.5 - centre).distance <= tolerancePx;
  if (!affineHolds) return _projectEach(points, project);

  return List<Offset>.generate(n, (i) {
    final a = (xs[i] - originX) / spanX;
    final b = (ys[i] - originY) / spanY;
    return p00 + e1 * a + e2 * b;
  }, growable: false);
}

List<Offset> _projectEach(
  List<Geographic> points,
  Offset Function(Geographic point) project,
) => List<Offset>.generate(
  points.length,
  (i) => project(points[i]),
  growable: false,
);

// Web-Mercator, normalised so the world is the unit square with y pointing
// south. Latitude is clamped to the projection's own limit so a polar input
// stays finite.
const _maxMercatorLat = 85.0511287798066;

double _mercatorX(double lon) => (lon + 180) / 360;

double _mercatorY(double lat) {
  final clamped = lat.clamp(-_maxMercatorLat, _maxMercatorLat);
  final s = math.sin(clamped * math.pi / 180);
  return 0.5 - math.log((1 + s) / (1 - s)) / (4 * math.pi);
}

double _lonFromMercatorX(double x) => x * 360 - 180;

double _latFromMercatorY(double y) {
  final n = math.pi * (1 - 2 * y);
  final sinhN = (math.exp(n) - math.exp(-n)) / 2;
  return math.atan(sinhN) * 180 / math.pi;
}
