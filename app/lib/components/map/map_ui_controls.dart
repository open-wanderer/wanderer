import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:pointer_interceptor/pointer_interceptor.dart';

/// App-local replacements for maplibre 0.3.5's `MapCompass`/`MapScalebar`.
///
/// Both package widgets live in [ml.MapLibreMap.children], where any read of
/// `MapController`/`MapCamera` subscribes to a model whose
/// `updateShouldNotify` is unconditionally true — so they rebuild on every
/// native camera frame during a pan. That part is unavoidable; what these
/// replacements fix is what each rebuild COSTS:
///
///  * `MapCompass` called `controller.getCamera()` — a full JNI camera fetch
///    — per frame, per mounted map. This one reads the Flutter-side
///    [ml.MapCamera] (already tracked, no JNI) and returns an identical
///    cached subtree while the bearing hasn't visibly changed.
///  * `MapScalebar` called `controller.toLngLat` + `getMetersPerPixelAtLatitude`
///    (two JNI round-trips) and allocated a fresh painter per frame. This one
///    re-derives meters-per-pixel only when zoom/latitude have moved enough
///    to change the result visibly, and returns a cached subtree otherwise.
///
/// Rendering and tap behavior deliberately mirror the package versions
/// (the compass needle painter is ported from maplibre 0.3.5, BSD-licensed).

class WandererMapCompass extends StatefulWidget {
  const WandererMapCompass({
    super.key,
    this.hideIfRotatedNorth = false,
    this.rotateNorthOnPressed = true,
    this.onPressed,
    this.alignment = Alignment.topRight,
    this.padding = const EdgeInsets.all(10),
  });

  /// Hide the compass while the map is not rotated.
  final bool hideIfRotatedNorth;

  /// Rotate the map back to north when tapped (the package default).
  final bool rotateNorthOnPressed;

  /// Additional/overriding tap behavior.
  final VoidCallback? onPressed;

  final Alignment alignment;
  final EdgeInsets padding;

  @override
  State<WandererMapCompass> createState() => _WandererMapCompassState();
}

class _WandererMapCompassState extends State<WandererMapCompass> {
  static const _radius = 22.0;

  /// Bearing granularity below which a rotation is imperceptible at compass
  /// size — camera frames that stay within it reuse the cached subtree.
  static const _bearingEpsilon = 0.25;

  double? _builtBearing;
  Widget? _built;

  @override
  Widget build(BuildContext context) {
    final controller = ml.MapController.maybeOf(context);
    final camera = ml.MapCamera.maybeOf(context);
    if (controller == null || camera == null) return const SizedBox.shrink();

    if (widget.hideIfRotatedNorth && camera.bearing == 0) {
      return const SizedBox.shrink();
    }

    final cached = _built;
    if (cached != null &&
        _builtBearing != null &&
        (camera.bearing - _builtBearing!).abs() < _bearingEpsilon) {
      return cached;
    }

    _builtBearing = camera.bearing;
    return _built = Container(
      alignment: widget.alignment,
      padding: widget.padding,
      child: Transform.rotate(
        angle: -camera.bearing * math.pi / 180,
        child: PointerInterceptor(
          child: InkWell(
            onTap: () {
              if (widget.rotateNorthOnPressed) {
                controller.animateCamera(
                  bearing: 0,
                  nativeDuration: const Duration(milliseconds: 200),
                );
              }
              widget.onPressed?.call();
            },
            child: CustomPaint(
              painter: _CompassPainter(radius: _radius),
              child: const SizedBox.square(dimension: _radius * 2),
            ),
          ),
        ),
      ),
    );
  }
}

/// Needle rendering ported from maplibre 0.3.5's private `_CompassPainter`.
class _CompassPainter extends CustomPainter {
  _CompassPainter({required this.radius});

  final double radius;

  @override
  void paint(Canvas canvas, Size size) {
    const needleHeight = 13.0;
    const needleWidth = 5.0;
    const needleStrokeWidth = 1.5;
    const borderWidth = 3.0;

    canvas.drawCircle(
      Offset(radius, radius),
      radius,
      Paint()..color = Colors.white60,
    );
    canvas.drawCircle(
      Offset(radius, radius),
      radius - (borderWidth / 2),
      Paint()
        ..color = Colors.black
        ..strokeWidth = borderWidth
        ..style = PaintingStyle.stroke,
    );
    canvas.drawPath(
      Path()
        ..moveTo(radius - needleWidth, radius)
        ..lineTo(radius, radius + needleHeight)
        ..lineTo(radius + needleWidth, radius),
      Paint()
        ..style = PaintingStyle.stroke
        ..color = Colors.red
        ..strokeCap = StrokeCap.butt
        ..strokeJoin = StrokeJoin.bevel
        ..strokeWidth = needleStrokeWidth,
    );
    const halfStrokeWidth = needleStrokeWidth / 2;
    canvas.drawVertices(
      Vertices(VertexMode.triangles, [
        Offset(radius - needleWidth - halfStrokeWidth, radius),
        Offset(radius, radius - needleHeight - halfStrokeWidth),
        Offset(radius + needleWidth + halfStrokeWidth, radius),
      ]),
      BlendMode.color,
      Paint()..color = Colors.red,
    );
  }

  @override
  bool shouldRepaint(covariant _CompassPainter oldDelegate) =>
      radius != oldDelegate.radius;
}

/// Metric-only (this app never mounts the imperial variant).
class WandererMapScalebar extends StatefulWidget {
  const WandererMapScalebar({
    super.key,
    this.alignment = Alignment.bottomLeft,
    this.padding = const EdgeInsets.all(12),
  });

  final Alignment alignment;
  final EdgeInsets padding;

  @override
  State<WandererMapScalebar> createState() => _WandererMapScalebarState();
}

class _WandererMapScalebarState extends State<WandererMapScalebar> {
  static const _height = 22.0;

  /// Camera deltas below which the bar's chosen length/label cannot visibly
  /// change — frames inside them reuse the cached subtree, skipping the two
  /// JNI calls (`toLngLat` + `getMetersPerPixelAtLatitude`).
  static const _zoomEpsilon = 0.005;
  static const _latEpsilon = 0.02;

  double? _builtZoom;
  double? _builtCenterLat;
  ThemeData? _builtTheme;
  Widget? _built;

  @override
  Widget build(BuildContext context) {
    final controller = ml.MapController.maybeOf(context);
    final camera = ml.MapCamera.maybeOf(context);
    if (controller == null || camera == null) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final cached = _built;
    if (cached != null &&
        identical(theme, _builtTheme) &&
        _builtZoom != null &&
        (camera.zoom - _builtZoom!).abs() < _zoomEpsilon &&
        (camera.center.lat - _builtCenterLat!).abs() < _latEpsilon) {
      return cached;
    }
    _builtZoom = camera.zoom;
    _builtCenterLat = camera.center.lat;
    _builtTheme = theme;

    // Accurate at the scalebar's own on-screen latitude, like the package
    // version (matters at low zoom where the viewport spans many degrees).
    final screenSize = MediaQuery.sizeOf(context);
    final totalPadding = widget.padding + MediaQuery.viewPaddingOf(context);
    final paddingOffset = Offset(totalPadding.left, totalPadding.top);
    final paddedSize =
        screenSize - Offset(totalPadding.right, totalPadding.bottom) as Size;
    final scalebarCenter =
        widget.alignment.alongSize(paddedSize) + paddingOffset;
    final scalebarLat = controller.toLngLat(scalebarCenter).lat;
    final metersPerPixel = controller.getMetersPerPixelAtLatitude(scalebarLat);

    final painter = _ScaleBarPainter(metersPerPixel, theme);
    return _built = SafeArea(
      child: Container(
        alignment: widget.alignment,
        padding: widget.padding,
        child: CustomPaint(
          painter: painter,
          size: Size(painter.width, _height),
        ),
      ),
    );
  }
}

/// Metric bar-length selection and rendering ported from maplibre 0.3.5's
/// `ScaleBarPainter` (not exported by the package).
class _ScaleBarPainter extends CustomPainter {
  _ScaleBarPainter(this.metersPerPixel, this.theme) {
    meters = switch (metersPerPixel) {
      >= 300000 => 50000000,
      >= 200000 => 30000000,
      >= 100000 => 20000000,
      >= 75000 => 10000000,
      >= 50000 => 5000000,
      >= 30000 => 3000000,
      >= 15000 => 2000000,
      >= 10000 => 1000000,
      >= 5000 => 500000,
      >= 3000 => 300000,
      >= 2000 => 200000,
      >= 1000 => 100000,
      >= 500 => 50000,
      >= 300 => 30000,
      >= 200 => 20000,
      >= 100 => 10000,
      >= 50 => 5000,
      >= 30 => 3000,
      >= 20 => 2000,
      >= 10 => 1000,
      >= 5 => 500,
      >= 3 => 300,
      >= 2 => 200,
      >= 1 => 100,
      >= 0.5 => 50,
      >= 0.3 => 30,
      >= 0.2 => 20,
      >= 0.1 => 10,
      >= 0.05 => 5,
      >= 0.03 => 3,
      >= 0.02 => 2,
      >= 0.01 => 1,
      _ => metersPerPixel * 100,
    };
  }

  final double metersPerPixel;
  final ThemeData theme;

  late final double meters;
  late final double width = meters / metersPerPixel;

  late final _linePaint = Paint()
    ..color = Colors.black
    ..strokeWidth = 1.5;
  late final _backgroundPaint = Paint()..color = Colors.white60;

  @override
  void paint(Canvas canvas, Size size) {
    final useKm = meters >= 1000;
    canvas.drawVertices(
      Vertices.raw(
        VertexMode.triangles,
        Float32List.fromList([
          0,
          22,
          0,
          0,
          width,
          22,
          0,
          0,
          width,
          0,
          width,
          22,
        ]),
      ),
      BlendMode.color,
      _backgroundPaint,
    );
    canvas.drawRawPoints(
      PointMode.lines,
      Float32List.fromList([
        0,
        22,
        0,
        0,
        0,
        22,
        width,
        22,
        width,
        0,
        width,
        22,
      ]),
      _linePaint,
    );

    final textPainter = TextPainter(
      text: TextSpan(
        style: theme.textTheme.bodySmall?.copyWith(color: Colors.black),
        text: useKm ? '${(meters / 1000).round()} km' : '${meters.round()} m',
      ),
      textAlign: TextAlign.left,
      textDirection: TextDirection.ltr,
    );
    textPainter.layout();
    textPainter.paint(canvas, const Offset(5, 5));
  }

  @override
  bool shouldRepaint(covariant _ScaleBarPainter oldDelegate) =>
      metersPerPixel != oldDelegate.metersPerPixel;
}
