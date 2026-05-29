import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:latlong2/latlong.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailLayer extends StatefulWidget {
  final Trail trail;
  final Color routeColor;
  final double strokeWidth;
  final bool showWaypoints;

  final Function(Waypoint wp)? onWaypointTap;

  TrailLayer({
    super.key,
    required this.trail,
    this.routeColor = const Color(0xff3549bb),
    this.strokeWidth = 5.0,
    this.showWaypoints = true,
    this.onWaypointTap,
  }) : assert(
         trail.expand?.gpx != null,
         'TrailLayer requires expanded GPX data.',
       );

  @override
  State<TrailLayer> createState() => _TrailLayerState();
}

class _TrailLayerState extends State<TrailLayer>
    with SingleTickerProviderStateMixin {
  late AnimationController _animationController;
  final Distance _distanceCalculator = const Distance();

  @override
  void initState() {
    super.initState();
    _animationController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 12),
    )..repeat();
  }

  @override
  void dispose() {
    _animationController.dispose();
    super.dispose();
  }

  void _addArrowsAlongPath({
    required List<LatLng> points,
    required double targetSpacingMeters,
    required double animationFraction,
    required List<Marker> outputList,
  }) {
    if (points.length < 2) return;

    double accumulatedDistance = 0.0;

    double nextArrowTarget = targetSpacingMeters * animationFraction;

    for (int i = 0; i < points.length - 1; i++) {
      final LatLng p1 = points[i];
      final LatLng p2 = points[i + 1];

      final double segmentDistance = _distanceCalculator.as(
        LengthUnit.Meter,
        p1,
        p2,
      );

      if (segmentDistance < 0.1) continue;

      while (accumulatedDistance + segmentDistance >= nextArrowTarget) {
        final double remainingDistanceToTarget =
            nextArrowTarget - accumulatedDistance;
        final double fractionOfSegment =
            remainingDistanceToTarget / segmentDistance;

        final double lat =
            p1.latitude + (p2.latitude - p1.latitude) * fractionOfSegment;
        final double lon =
            p1.longitude + (p2.longitude - p1.longitude) * fractionOfSegment;
        final LatLng arrowPosition = LatLng(lat, lon);

        final double bearingInDegrees = _distanceCalculator.bearing(p1, p2);
        final double bearingInRadians = bearingInDegrees * (math.pi / 180.0);

        outputList.add(
          Marker(
            point: arrowPosition,
            width: 18,
            height: 18,
            child: Transform.rotate(
              angle: bearingInRadians,
              child: const Icon(Icons.circle, color: Colors.white, size: 6),
            ),
          ),
        );

        nextArrowTarget += targetSpacingMeters;
      }

      accumulatedDistance += segmentDistance;
    }
  }

  @override
  Widget build(BuildContext context) {
    final camera = MapCamera.of(context);
    final double currentZoom = camera.zoom;

    final gpx = widget.trail.expand!.gpx!;
    final pathPoints = gpx.allPoints;
    final List<Marker> staticMarkers = [];

    if (pathPoints.isNotEmpty) {
      staticMarkers.add(
        Marker(
          point: pathPoints.first,
          width: 28,
          height: 28,
          child: _buildCircularMarker(
            FontAwesomeIcons.bullseye,
            color: Colors.green,
          ),
        ),
      );
      staticMarkers.add(
        Marker(
          point: pathPoints.last,
          width: 28,
          height: 28,
          child: _buildCircularMarker(
            FontAwesomeIcons.flagCheckered,
            color: Colors.red,
          ),
        ),
      );
    }

    if (widget.showWaypoints &&
        widget.trail.expand?.waypointsViaTrail != null) {
      for (var wp in widget.trail.expand!.waypointsViaTrail!) {
        staticMarkers.add(
          Marker(
            point: LatLng(wp.lat, wp.lon),
            width: 28,
            height: 28,
            child: GestureDetector(
              onTap: () => widget.onWaypointTap?.call(wp),
              child: _buildCircularMarker(wp.icon),
            ),
          ),
        );
      }
    }

    double arrowSpacingMeters = 100.0;
    if (currentZoom >= 16) {
      arrowSpacingMeters = 240.0;
    } else if (currentZoom >= 14) {
      arrowSpacingMeters = 600.0;
    } else if (currentZoom >= 12) {
      arrowSpacingMeters = 2000.0;
    } else if (currentZoom >= 10) {
      arrowSpacingMeters = 5000.0;
    } else {
      arrowSpacingMeters = 10000.0;
    }

    bool showArrows = currentZoom > 8.0;
    showArrows = false;
    return Stack(
      children: [
        // Base Trail Line Layer
        PolylineLayer(
          polylines: [
            Polyline(
              points: pathPoints,
              color: Colors.white,
              strokeWidth: widget.strokeWidth + 2,
            ),
            Polyline(
              points: pathPoints,
              color: widget.routeColor,
              strokeWidth: widget.strokeWidth,
            ),
          ],
        ),

        if (pathPoints.length > 1 && showArrows)
          AnimatedBuilder(
            animation: _animationController,
            builder: (context, child) {
              final List<Marker> directionalArrows = [];

              _addArrowsAlongPath(
                points: pathPoints,
                targetSpacingMeters: arrowSpacingMeters,
                animationFraction: _animationController.value,
                outputList: directionalArrows,
              );

              return MarkerLayer(markers: directionalArrows);
            },
          ),

        // Static Overlay Layer
        if (widget.showWaypoints || pathPoints.isNotEmpty)
          MarkerLayer(markers: staticMarkers),
      ],
    );
  }

  Widget _buildCircularMarker(
    FaIconData faIcon, {
    Color color = Colors.blueGrey,
  }) {
    return Container(
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: .2),
            blurRadius: 4,
            offset: const Offset(0, 2),
          ),
        ],
        border: Border.all(color: Colors.white, width: 2),
      ),
      child: Center(child: FaIcon(faIcon, color: Colors.white, size: 14)),
    );
  }
}
