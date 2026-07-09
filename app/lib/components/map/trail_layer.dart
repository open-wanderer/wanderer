import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/trail.dart';
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
  8, 250,
  12, 160,
  16, 90,
];

/// Adds the GPX track (white casing under a colored route line) and the static
/// directional arrows as native GL style layers via [ml.StyleController]
/// (TRAIL-01 / TRAIL-02).
///
/// Call this from `onStyleLoaded` so the layers are re-added after every style
/// swap (CORE-02 theme toggle rebuilds the style and drops added layers/images).
///
/// The `arrow` icon is registered here via [ml.StyleController.addImageFromIconData]
/// rather than relying on the Protomaps style sprite: 15-03 found `file://`
/// sprite resolution unreliable on device, so shipping our own image makes the
/// arrows deterministic regardless of the sprite (D-04).
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
  // symbol layer never depends on the style sprite providing `arrow`.
  await style.addImageFromIconData(
    id: 'arrow',
    iconData: Icons.navigation,
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
      layout: const <String, Object>{
        'line-cap': 'round',
        'line-join': 'round',
      },
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
        'icon-image': 'arrow',
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
