import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/map_marker_gestures.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx/gpx.dart';

/// Default GPX route line color (`#3549bb`). Overridable per call.
const Color kTrailRouteColor = Color(0xff3549bb);

/// Zoom-interpolated `symbol-spacing` (screen pixels) for the directional
/// arrows — denser (smaller spacing) at higher zoom. Static: there is no
/// motion, just density-by-zoom. Meter→pixel is approximate, so tune on
/// device.
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

/// A `trail-` prefixed id — the basemap style's `roads_oneway` layer
/// also references a real Protomaps sprite icon literally named `arrow`.
/// MapLibre's native `addImage` (both Android and iOS) replaces an existing
/// same-id image rather than throwing, so registering under the bare `arrow`
/// id silently overwrites the basemap's one-way-road icon whenever a trail
/// is on screen.
const String _kTrailArrowImageId = 'trail-arrow';

/// Owns the native MapLibre GL style layers/source that draw a trail route
/// line (via [add]/[remove]) and, separately, the directional-arrow overlay
/// used both by [add] and by list/collection screens (via [addArrows]) that
/// draw their own route lines declaratively.
///
/// [showArrows] is fixed at construction — it is not reactive. A caller that
/// needs to change whether arrows are shown must construct a new
/// [TrailLayer] rather than mutating an existing instance.
class TrailLayer {
  const TrailLayer({
    this.sourceId = 'trail',
    this.casingLayerId = 'trail-casing',
    this.routeLayerId = 'trail-route',
    this.arrowsLayerId = 'trail-arrows',
    this.showArrows = true,
  });

  final String sourceId;
  final String casingLayerId;
  final String routeLayerId;
  final String arrowsLayerId;
  final bool showArrows;

  /// Adds the GPX track (white casing under a colored route line) and,
  /// if [showArrows], the static directional arrows as native GL style
  /// layers via [ml.StyleController].
  ///
  /// Call this from `onStyleLoaded` so the layers are re-added after every
  /// style swap (a theme toggle rebuilds the style and drops added
  /// layers/images).
  Future<void> add(
    ml.StyleController style,
    Trail trail, {
    Color routeColor = kTrailRouteColor,
  }) async {
    final gpx = trail.expand?.gpx;
    if (gpx == null) return;
    final points = gpx.allPoints;
    if (points.length < 2) return;

    // Serialize the track as a single GeoJSON LineString Feature. jsonEncode
    // (not string concatenation) keeps the geometry structurally isolated
    // from the style document.
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

    await style.addSource(ml.GeoJsonSource(id: sourceId, data: data));

    // Draw order = add order: casing (9px white) first, colored route (5px)
    // on top, so 2px of white shows as an outline around the route.
    await style.addLayer(
      ml.LineStyleLayer(
        id: casingLayerId,
        sourceId: sourceId,
        layout: const <String, Object>{
          'line-cap': 'round',
          'line-join': 'round',
        },
        paint: const <String, Object>{'line-color': '#ffffff', 'line-width': 9},
      ),
    );
    await style.addLayer(
      ml.LineStyleLayer(
        id: routeLayerId,
        sourceId: sourceId,
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

    if (!showArrows) return;

    // The directional-arrow icon is registered here via
    // [ml.StyleController.addImageFromIconData] rather than relying on the
    // Protomaps style sprite: `file://` sprite resolution is unreliable
    // on device, so shipping our own image makes the arrows deterministic
    // regardless of the sprite. Uses [_kTrailArrowImageId], NOT the bare
    // `arrow` id the basemap's own sprite icon uses — see that constant's
    // doc.
    await style.addImageFromIconData(
      id: _kTrailArrowImageId,
      iconData: Icons.arrow_left,
      size: 32,
      color: Colors.white,
    );

    // Static directional arrows: native line placement + rotation, no
    // animation loop. minZoom 8 gates visibility below `zoom > 8`.
    await style.addLayer(
      ml.SymbolStyleLayer(
        id: arrowsLayerId,
        sourceId: sourceId,
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

  /// Removes the layers/source added by [add], in reverse order (layers
  /// before the source they depend on). Safe to call even if [add] was
  /// never called or already removed — each removal is individually
  /// guarded since [ml.StyleController.removeLayer]/`removeSource` throw on
  /// an unknown id. Deliberately leaves the shared arrow image registered.
  Future<void> remove(ml.StyleController style) async {
    if (showArrows) {
      try {
        await style.removeLayer(arrowsLayerId);
      } catch (_) {}
    }
    for (final id in [routeLayerId, casingLayerId]) {
      try {
        await style.removeLayer(id);
      } catch (_) {}
    }
    try {
      await style.removeSource(sourceId);
    } catch (_) {}
  }

  /// Adds a native directional-arrow symbol layer over a set of already-drawn
  /// polylines (list surfaces). Unlike [add] this does NOT draw the
  /// route/casing lines themselves — the list screens keep drawing those via
  /// the declarative `ml.PolylineLayer`; this only adds arrows on top,
  /// reusing the exact same arrow image id and zoom-interpolated spacing so
  /// the list arrows are visually identical to the single-trail detail
  /// arrows. No-ops if [showArrows] is false.
  ///
  /// Call this from `onStyleLoaded` (after the existing `fitBounds` call) so
  /// the layer is re-added after every style swap, same as [add].
  Future<void> addArrows(
    ml.StyleController style,
    List<List<ml.Geographic>> lines, {
    String sourceId = 'list-trail-arrows',
    String layerId = 'list-trail-arrows-symbols',
  }) async {
    if (!showArrows) return;
    final validLines = lines.where((line) => line.length >= 2).toList();
    if (validLines.isEmpty) return;

    // Same arrow image id as [add] — see _kTrailArrowImageId's doc for why
    // this must stay distinct from the basemap's own `arrow` icon.
    await style.addImageFromIconData(
      id: _kTrailArrowImageId,
      iconData: Icons.arrow_left,
      size: 32,
      color: Colors.white,
    );

    // jsonEncode (not string concatenation) keeps the geometry structurally
    // isolated from the style document.
    final data = jsonEncode(<String, Object?>{
      'type': 'FeatureCollection',
      'features': <Map<String, Object?>>[
        for (final line in validLines)
          <String, Object?>{
            'type': 'Feature',
            'properties': <String, Object?>{},
            'geometry': <String, Object?>{
              'type': 'LineString',
              'coordinates': <List<double>>[
                for (final p in line) <double>[p.lon, p.lat],
              ],
            },
          },
      ],
    });

    await style.addSource(ml.GeoJsonSource(id: sourceId, data: data));

    await style.addLayer(
      ml.SymbolStyleLayer(
        id: layerId,
        sourceId: sourceId,
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
}

/// Serializes a [Color] to a `#rrggbb` Style-Spec color string.
String _colorToHex(Color color) {
  int channel(double v) => (v * 255).round().clamp(0, 255);
  String hex(int v) => v.toRadixString(16).padLeft(2, '0');
  return '#${hex(channel(color.r))}${hex(channel(color.g))}${hex(channel(color.b))}';
}

/// Interactive trail markers rendered as Flutter widgets over the native map
/// via a single [ml.WidgetLayer]: tappable waypoints with
/// the [AnimatedScale] selection animation, and the start/finish pins with the
/// 36-screen-pixel proximity nudge.
///
/// Place this in [ml.MapLibreMap.children]. It reads [ml.MapController]/
/// [ml.MapCamera] from context so the nudge recomputes on every camera move.
class TrailMarkerLayer extends StatefulWidget {
  final Trail trail;
  final bool showWaypoints;
  final Waypoint? selectedWaypoint;
  final void Function(Waypoint wp)? onWaypointTap;
  final void Function(Waypoint wp, ml.Geographic point)? onWaypointDragEnd;

  const TrailMarkerLayer({
    super.key,
    required this.trail,
    this.showWaypoints = true,
    this.selectedWaypoint,
    this.onWaypointTap,
    this.onWaypointDragEnd,
  });

  @override
  State<TrailMarkerLayer> createState() => _TrailMarkerLayerState();
}

class _TrailMarkerLayerState extends State<TrailMarkerLayer> {
  String? _draggingWaypointId;
  Offset? _dragOffset;

  /// Memoized waypoint markers and the inputs they were built from — the
  /// same pattern (and rationale) as `RouteAnchorLayer`: this widget rebuilds
  /// on every camera frame (`MapController/MapCamera.maybeOf` subscribe to a
  /// model whose `updateShouldNotify` is unconditionally true), and what's
  /// avoidable is rebuilding every marker's subtree per frame —
  /// [ml.WidgetLayer] repositions markers itself, so handing it the same
  /// `Widget` instances lets element diffing skip the subtrees entirely.
  List<ml.Marker>? _cachedWaypointMarkers;
  List<Waypoint>? _cachedWaypoints;
  String? _cachedSelectedId;
  Color? _cachedPrimary;

  /// Start/finish pin children never depend on theme or camera — built once.
  late final Widget _startPin = _buildCircularMarker(
    FontAwesomeIcons.bullseye,
    color: Colors.greenAccent,
  );
  late final Widget _endPin = _buildCircularMarker(
    FontAwesomeIcons.flagCheckered,
    color: Colors.redAccent,
  );

  /// Endpoints + their great-circle separation, cached on gpx identity so the
  /// O(n) `allPoints` materialization runs once per trail instead of once per
  /// camera frame.
  Object? _cachedGpx;
  ml.Geographic? _firstPoint;
  ml.Geographic? _lastPoint;
  double? _endpointsMeters;

  void _clearDrag() {
    setState(() {
      _draggingWaypointId = null;
      _dragOffset = null;
      _cachedWaypointMarkers = null;
    });
  }

  void _resolveEndpoints() {
    final gpx = widget.trail.expand?.gpx;
    if (identical(gpx, _cachedGpx)) return;
    _cachedGpx = gpx;
    final points = gpx?.allPoints ?? const <ml.Geographic>[];
    _firstPoint = points.isEmpty ? null : points.first;
    _lastPoint = points.isEmpty ? null : points.last;
    _endpointsMeters = (points.length > 1)
        ? ml.SphericalGreatCircle(points.first).distanceTo(points.last)
        : null;
  }

  ml.Marker _buildWaypointMarker(
    Waypoint wp, {
    required bool isSelected,
    required bool isDragging,
    required bool isDraggable,
    required ml.Geographic point,
    required Color primary,
  }) {
    return ml.Marker(
      point: point,
      size: const Size(32, 32),
      child: MapMarkerGestures(
        onTap: () => widget.onWaypointTap?.call(wp),
        onPanStart: !isDraggable
            ? null
            : (details) {
                final c = ml.MapController.maybeOf(context);
                if (c == null) return;
                setState(() {
                  _draggingWaypointId = wp.id;
                  _dragOffset = c.toScreenLocation(
                    ml.Geographic(lon: wp.lon, lat: wp.lat),
                  );
                  _cachedWaypointMarkers = null;
                });
              },
        onPanUpdate: !isDraggable
            ? null
            : (details) {
                if (_draggingWaypointId != wp.id || _dragOffset == null) {
                  return;
                }
                setState(() => _dragOffset = _dragOffset! + details.delta);
              },
        onPanEnd: !isDraggable
            ? null
            : (details) {
                if (_draggingWaypointId != wp.id) return;
                final c = ml.MapController.maybeOf(context);
                final offset = _dragOffset;
                _clearDrag();
                if (c != null && offset != null) {
                  widget.onWaypointDragEnd?.call(wp, c.toLngLat(offset));
                }
              },
        onPanCancel: !isDraggable
            ? null
            : () {
                if (_draggingWaypointId == wp.id) _clearDrag();
              },
        child: AnimatedScale(
          scale: (isSelected || isDragging) ? 1.0 : 0.875,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOutBack,
          child: _buildCircularMarker(
            wp.icon,
            color: primary,
            selected: isSelected || isDragging,
          ),
        ),
      ),
    );
  }

  List<ml.Marker> _waypointMarkers(
    ml.MapController? controller,
    Color primary,
  ) {
    final waypoints = widget.trail.expand?.waypointsViaTrail;
    if (!widget.showWaypoints || waypoints == null) return const [];

    // A drag re-projects a fixed screen offset through the live camera, so
    // its markers rebuild per frame. Only the static case caches.
    final cached = _cachedWaypointMarkers;
    if (_draggingWaypointId == null &&
        cached != null &&
        identical(waypoints, _cachedWaypoints) &&
        _cachedSelectedId == widget.selectedWaypoint?.id &&
        _cachedPrimary == primary) {
      return cached;
    }

    // Drag handlers are registered only where dragging is actually wired up
    // (the trail create/edit flow). Elsewhere they would take single-finger
    // drags away from the map for nothing.
    final isDraggable = widget.onWaypointDragEnd != null;
    final markers = <ml.Marker>[
      for (final wp in waypoints)
        _buildWaypointMarker(
          wp,
          isSelected: widget.selectedWaypoint?.id == wp.id,
          isDragging: _draggingWaypointId == wp.id,
          isDraggable: isDraggable,
          point:
              (_draggingWaypointId == wp.id &&
                  controller != null &&
                  _dragOffset != null)
              ? controller.toLngLat(_dragOffset!)
              : ml.Geographic(lon: wp.lon, lat: wp.lat),
          primary: primary,
        ),
    ];

    if (_draggingWaypointId == null) {
      _cachedWaypointMarkers = markers;
      _cachedWaypoints = waypoints;
      _cachedSelectedId = widget.selectedWaypoint?.id;
      _cachedPrimary = primary;
    } else {
      _cachedWaypointMarkers = null;
    }
    return markers;
  }

  @override
  Widget build(BuildContext context) {
    final controller = ml.MapController.maybeOf(context);
    // Subscribe to camera changes so the start/finish nudge recomputes as the
    // user pans/zooms (the pins may cross the 36px threshold).
    final camera = ml.MapCamera.maybeOf(context);

    _resolveEndpoints();

    final markers = <ml.Marker>[
      ..._waypointMarkers(controller, Theme.of(context).primaryColor),
    ];

    final first = _firstPoint;
    final last = _lastPoint;
    if (first != null && last != null) {
      var startAlignment = Alignment.center;
      var endAlignment = Alignment.center;

      // Screen distance between the pins, from pure camera math rather than
      // the two per-frame JNI projections this used to cost: MapLibre zoom is
      // 512px-tile web mercator, so meters-per-logical-pixel at the camera
      // center is (equator circumference / 512) * cos(lat) / 2^zoom. Using
      // the camera-center latitude instead of the pins' is well within
      // tolerance for a 36px overlap heuristic.
      final meters = _endpointsMeters;
      if (meters != null && camera != null) {
        final metersPerPixel =
            78271.51696 *
            math.cos(camera.center.lat * math.pi / 180) /
            math.pow(2, camera.zoom);
        if (meters / metersPerPixel < 36) {
          startAlignment = const Alignment(1, 0);
          endAlignment = const Alignment(-1, 0);
        }
      }

      markers.add(
        ml.Marker(
          point: first,
          size: const Size(28, 28),
          alignment: startAlignment,
          child: _startPin,
        ),
      );
      markers.add(
        ml.Marker(
          point: last,
          size: const Size(28, 28),
          alignment: endAlignment,
          child: _endPin,
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
