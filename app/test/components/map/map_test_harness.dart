// `MapLibreInheritedModel` is what `MapController.maybeOf`/`MapCamera.maybeOf`
// resolve through, but the package does not export it. Widget tests have no
// platform view, so they provide the model themselves.
// ignore_for_file: depend_on_referenced_packages, implementation_imports
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:maplibre_platform_interface/src/widget/inherited_model.dart';

/// A map at pitch 0: a fixed Web-Mercator screen mapping ([project]) plus a pan
/// [shift], which counts native projection calls.
class FakeMapController implements ml.MapController {
  int projections = 0;
  Offset shift = Offset.zero;

  /// Geographic -> screen at "pitch 0": Web-Mercator on a 36000 px world,
  /// (lon 0, lat 0) at the middle of the default 800x600 test surface. About
  /// 100 px per degree near the equator, north up.
  static Offset project(ml.Geographic g) {
    const worldPx = 36000.0;
    double x(double lon) => (lon + 180) / 360;
    double y(double lat) {
      final s = math.sin(lat * math.pi / 180);
      return 0.5 - math.log((1 + s) / (1 - s)) / (4 * math.pi);
    }

    return Offset(
      400 + (x(g.lon) - x(0)) * worldPx,
      300 + (y(g.lat) - y(0)) * worldPx,
    );
  }

  @override
  Offset toScreenLocation(ml.Geographic lngLat) {
    projections++;
    return project(lngLat) + shift;
  }

  /// Inverse of [toScreenLocation] (including the pan [shift]).
  @override
  ml.Geographic toLngLat(Offset screenLocation) {
    const worldPx = 36000.0;
    final p = screenLocation - shift;
    final x = (p.dx - 400) / worldPx + 0.5;
    final y = (p.dy - 300) / worldPx + 0.5;
    final n = math.pi * (1 - 2 * y);
    final lat = math.atan((math.exp(n) - math.exp(-n)) / 2) * 180 / math.pi;
    return ml.Geographic(lon: x * 360 - 180, lat: lat);
  }

  @override
  List<Offset> toScreenLocations(List<ml.Geographic> lngLats) =>
      lngLats.map(toScreenLocation).toList(growable: false);

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Counts how often it is painted. Stands in for anything else on the page
/// (a docked sheet, the app bar) that must NOT repaint on a camera frame.
class PaintCounter extends CustomPainter {
  PaintCounter(this.onPaint);

  final VoidCallback onPaint;

  @override
  void paint(Canvas canvas, Size size) => onPaint();

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

/// What the map host looks like to the layers under test: an inherited model
/// that, like the real one, notifies every dependent on every update.
class MapHost extends StatelessWidget {
  const MapHost({
    super.key,
    required this.controller,
    required this.camera,
    required this.child,
  });

  final FakeMapController controller;
  final ml.MapCamera camera;
  final Widget child;

  @override
  Widget build(BuildContext context) => MapLibreInheritedModel(
    mapController: controller,
    mapCamera: camera,
    child: Stack(children: [child]),
  );
}

ml.MapCamera cameraFor({double zoom = 10, double pitch = 0}) => ml.MapCamera(
  center: ml.Geographic(lon: 0, lat: 0),
  zoom: zoom,
  bearing: 0,
  pitch: pitch,
);
