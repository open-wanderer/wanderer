import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:latlong2/latlong.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailLayer extends StatelessWidget {
  final Trail trail;
  final Color routeColor;
  final double strokeWidth;
  final bool showWaypoints;

  TrailLayer({
    super.key,
    required this.trail,
    this.routeColor = Colors.blue,
    this.strokeWidth = 5.0,
    this.showWaypoints = true,
  }) : assert(
         trail.expand?.gpx != null,
         'TrailLayer requires expanded GPX data.',
       );

  @override
  Widget build(BuildContext context) {
    final gpx = trail.expand!.gpx!;

    final pathPoints = gpx.allPoints;

    final List<Marker> markers = [];

    // 2. Add Start Marker (Green)
    markers.add(
      Marker(
        point: pathPoints.first,
        width: 28,
        height: 28,
        child: _buildCircularMarker(FontAwesomeIcons.bullseye),
      ),
    );

    markers.add(
      Marker(
        point: pathPoints.last,
        width: 28,
        height: 28,
        child: _buildCircularMarker(FontAwesomeIcons.flagCheckered),
      ),
    );

    if (showWaypoints && trail.expand?.waypointsViaTrail != null) {
      for (var wp in trail.expand!.waypointsViaTrail!) {
        markers.add(
          Marker(
            point: LatLng(wp.lat, wp.lon),
            width: 28,
            height: 28,
            child: GestureDetector(
              onTap: () => _showWaypointDetails(context, wp),
              child: _buildCircularMarker(wp.icon),
            ),
          ),
        );
      }
    }

    return Stack(
      children: [
        PolylineLayer(
          polylines: [
            Polyline(
              points: pathPoints,
              color: routeColor,
              strokeWidth: strokeWidth,
            ),
          ],
        ),
        if (showWaypoints) MarkerLayer(markers: markers),
      ],
    );
  }

  void _showWaypointDetails(BuildContext context, Waypoint wp) {
    showModalBottomSheet(
      context: context,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              wp.name ?? "",
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            if (wp.description != null) Text(wp.description!),
          ],
        ),
      ),
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
        border: Border.all(color: color, width: 2),
      ),
      child: Center(child: FaIcon(faIcon, color: Colors.white, size: 16)),
    );
  }
}
