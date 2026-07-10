import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx_util.dart';

/// Default GPX route line color (`#3549bb`). Overridable per call.
const Color kTrailRouteColor = Color(0xff3549bb);

/// Zoom-interpolated `symbol-spacing` (screen pixels) for the directional
/// arrows — denser (smaller spacing) at higher zoom (TRAIL-02, D-05). Static:
/// the density-by-zoom logic of the old meter table is preserved, the motion
/// is not. Tune on device (RESEARCH Pitfall 5 — meter→pixel is approximate).
const List<Object> _kArrowSpacing = <Object>[
  'interpolate',
  <Object>['linear'],
  <Object>['zoom'],
  8,
  250,
  12,
  160,
  16,
  90,
];

/// Adds the GPX track (white casing under a colored route line) and the static
/// directional arrows as native GL style layers via [ml.StyleController]
/// (TRAIL-01 / TRAIL-02).
///
/// Call this from `onStyleLoaded` so the layers are re-added after every style
/// swap (CORE-02 theme toggle rebuilds the style and drops added layers/images).
///
/// A `trail-` prefixed id — the ported basemap style's `roads_oneway` layer
/// also references a real Protomaps sprite icon literally named `arrow`.
/// MapLibre's native `addImage` (both Android and iOS) replaces an existing
/// same-id image rather than throwing, so registering under the bare `arrow`
/// id silently overwrote the basemap's one-way-road icon whenever a trail
/// was on screen — found during Phase 15 goal verification, not on-device.
const String _kTrailArrowImageId = 'trail-arrow';

/// The directional-arrow icon is registered here via
/// [ml.StyleController.addImageFromIconData] rather than relying on the
/// Protomaps style sprite: 15-03 found `file://` sprite resolution unreliable
/// on device, so shipping our own image makes the arrows deterministic
/// regardless of the sprite (D-04). Uses [_kTrailArrowImageId], NOT the bare
/// `arrow` id the basemap's own sprite icon uses — see that constant's doc.
Future<void> addTrailTrackLayers(
  ml.StyleController style,
  Trail trail, {
  Color routeColor = kTrailRouteColor,
}) async {
  final gpx = trail.expand?.gpx;
  if (gpx == null) return;
  final points = gpx.allPoints;
  if (points.length < 2) return;

  // Register the directional-arrow image (a small white nav mark) so the
  // symbol layer never depends on the style sprite providing an icon, under
  // an id distinct from the basemap's own `arrow` sprite icon.
  await style.addImageFromIconData(
    id: _kTrailArrowImageId,
    iconData: Icons.arrow_left,
    size: 32,
    color: Colors.white,
  );

  // Serialize the track as a single GeoJSON LineString Feature. jsonEncode
  // (not string concatenation) keeps the geometry structurally isolated from
  // the style document (threat T-15-05-01).
  final data = jsonEncode(<String, Object?>{
    'type': 'Feature',
    'properties': <String, Object?>{},
    'geometry': <String, Object?>{
      'type': 'LineString',
      'coordinates': <List<double>>[
        for (final p in points) <double>[p.lon, p.lat],
      ],
    },
  });

  await style.addSource(ml.GeoJsonSource(id: 'trail', data: data));

  // Draw order = add order: casing (9px white) first, colored route (5px) on
  // top, so 2px of white shows as an outline around the route (TRAIL-01).
  await style.addLayer(
    const ml.LineStyleLayer(
      id: 'trail-casing',
      sourceId: 'trail',
      layout: <String, Object>{'line-cap': 'round', 'line-join': 'round'},
      paint: <String, Object>{'line-color': '#ffffff', 'line-width': 9},
    ),
  );
  await style.addLayer(
    ml.LineStyleLayer(
      id: 'trail-route',
      sourceId: 'trail',
      layout: const <String, Object>{'line-cap': 'round', 'line-join': 'round'},
      paint: <String, Object>{
        'line-color': _colorToHex(routeColor),
        'line-width': 5,
      },
    ),
  );

  // Static directional arrows: native line placement + rotation, no animation
  // loop (D-05). minZoom 8 mirrors the old `zoom > 8` visibility gate.
  await style.addLayer(
    const ml.SymbolStyleLayer(
      id: 'trail-arrows',
      sourceId: 'trail',
      minZoom: 8,
      layout: <String, Object>{
        'symbol-placement': 'line',
        'icon-image': _kTrailArrowImageId,
        'icon-rotation-alignment': 'map',
        'icon-allow-overlap': true,
        'icon-ignore-placement': true,
        'icon-size': 0.5,
        'symbol-spacing': _kArrowSpacing,
      },
    ),
  );
}

/// Serializes a [Color] to a `#rrggbb` Style-Spec color string.
String _colorToHex(Color color) {
  int channel(double v) => (v * 255).round().clamp(0, 255);
  String hex(int v) => v.toRadixString(16).padLeft(2, '0');
  return '#${hex(channel(color.r))}${hex(channel(color.g))}${hex(channel(color.b))}';
}

/// Interactive trail markers rendered as Flutter widgets over the native map
/// via a single [ml.WidgetLayer] (TRAIL-03 / TRAIL-04): tappable waypoints with
/// the [AnimatedScale] selection animation, and the start/finish pins with the
/// 36-screen-pixel proximity nudge.
///
/// Place this in [ml.MapLibreMap.children]. It reads [ml.MapController]/
/// [ml.MapCamera] from context so the nudge recomputes on every camera move.
class TrailMarkerLayer extends StatelessWidget {
  final Trail trail;
  final bool showWaypoints;
  final Waypoint? selectedWaypoint;
  final void Function(Waypoint wp)? onWaypointTap;

  const TrailMarkerLayer({
    super.key,
    required this.trail,
    this.showWaypoints = true,
    this.selectedWaypoint,
    this.onWaypointTap,
  });

  @override
  Widget build(BuildContext context) {
    final controller = ml.MapController.maybeOf(context);
    // Subscribe to camera changes so the start/finish nudge recomputes as the
    // user pans/zooms (the pins may cross the 36px threshold).
    ml.MapCamera.maybeOf(context);

    final markers = <ml.Marker>[];

    if (showWaypoints && trail.expand?.waypointsViaTrail != null) {
      for (final wp in trail.expand!.waypointsViaTrail!) {
        final isSelected = selectedWaypoint?.id == wp.id;
        markers.add(
          ml.Marker(
            point: ml.Geographic(lon: wp.lon, lat: wp.lat),
            size: const Size(32, 32),
            child: GestureDetector(
              onTap: () => onWaypointTap?.call(wp),
              child: AnimatedScale(
                scale: isSelected ? 1.0 : 0.875,
                duration: const Duration(milliseconds: 200),
                curve: Curves.easeOutBack,
                child: _buildCircularMarker(
                  wp.icon,
                  color: Theme.of(context).primaryColor,
                  selected: isSelected,
                ),
              ),
            ),
          ),
        );
      }
    }

    final points = trail.expand?.gpx?.allPoints ?? const [];
    if (points.isNotEmpty) {
      var startAlignment = Alignment.center;
      var endAlignment = Alignment.center;

      if (points.length > 1 && controller != null) {
        final offsets = controller.toScreenLocations([
          points.first,
          points.last,
        ]);
        final dx = offsets[0].dx - offsets[1].dx;
        final dy = offsets[0].dy - offsets[1].dy;
        if (math.sqrt(dx * dx + dy * dy) < 36) {
          startAlignment = const Alignment(1, 0);
          endAlignment = const Alignment(-1, 0);
        }
      }

      markers.add(
        ml.Marker(
          point: points.first,
          size: const Size(28, 28),
          alignment: startAlignment,
          child: _buildCircularMarker(
            FontAwesomeIcons.bullseye,
            color: Colors.greenAccent,
          ),
        ),
      );
      markers.add(
        ml.Marker(
          point: points.last,
          size: const Size(28, 28),
          alignment: endAlignment,
          child: _buildCircularMarker(
            FontAwesomeIcons.flagCheckered,
            color: Colors.redAccent,
          ),
        ),
      );
    }

    return ml.WidgetLayer(allowInteraction: true, markers: markers);
  }
}

/// Ported verbatim from the old flutter_map implementation: a circular pin with
/// a Font Awesome glyph, 2px border, and a soft drop shadow. Selected state
/// inverts to a white fill with a colored icon + border and a larger glyph.
Widget _buildCircularMarker(
  FaIconData faIcon, {
  Color color = Colors.blueGrey,
  bool selected = false,
}) {
  return Container(
    decoration: BoxDecoration(
      color: selected ? Colors.white : color,
      shape: BoxShape.circle,
      boxShadow: [
        BoxShadow(
          color: Colors.black.withValues(alpha: .2),
          blurRadius: 4,
          offset: const Offset(0, 2),
        ),
      ],
      border: Border.all(color: selected ? color : Colors.white, width: 2),
    ),
    child: Center(
      child: FaIcon(
        faIcon,
        color: selected ? color : Colors.white,
        size: selected ? 16 : 14,
      ),
    ),
  );
}
