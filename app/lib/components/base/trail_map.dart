import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/map/location_marker_layer.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/glyph_sprite_cache_paths.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/util/offline_style_rewriter.dart';

/// A native MapLibre GL map host for a single [Trail]. Renders the
/// Protomaps basemap from [mapStyleJsonProvider] via native GL, swaps
/// light/dark styles live with [ml.MapController.setStyle] (no
/// remount/flash), fits the trail bounds in `onStyleLoaded`, shows
/// the built-in scale bar + attribution, and hosts the
/// elevation-scrub and interim-location markers.
///
/// Single-trail detail views only — for screens that render a collection of
/// trails (search results, list previews), use `TrailCollectionMap` instead.
class TrailMap extends ConsumerStatefulWidget {
  final Trail trail;

  /// Hands the native [ml.MapController] back to the caller once the map is
  /// created. The controller is created by the native map (it cannot be
  /// free-standing), so callers hold it as a nullable field set from here.
  final void Function(ml.MapController controller)? onMapCreated;

  final bool disabled;
  final bool offline;
  final List<Widget>? controls;
  final ml.Geographic? elevationMarkerPosition;
  final EdgeInsets initialCameraFitPadding;

  final bool showTrail;
  final bool showLocation;
  final Waypoint? selectedWaypoint;

  /// Called with the tapped geographic point on a map click (was flutter_map's
  /// `TapCallback`; now driven by the native `onEvent` click event).
  final void Function(ml.Geographic point)? onTap;
  final void Function(ml.MapEvent event)? onMapEvent;
  final void Function(Waypoint wp)? onWaypointTap;

  const TrailMap({
    super.key,
    required this.trail,
    this.onMapCreated,
    this.onTap,
    this.onWaypointTap,
    this.onMapEvent,
    this.disabled = false,
    this.offline = false,
    this.controls = const [],
    this.showTrail = true,
    this.showLocation = false,
    this.selectedWaypoint,
    this.elevationMarkerPosition,
    this.initialCameraFitPadding = const EdgeInsets.all(40),
  });

  @override
  ConsumerState<TrailMap> createState() => _TrailMapState();
}

class _TrailMapState extends ConsumerState<TrailMap> {
  static const _trailLayer = TrailLayer();

  ml.MapController? _controller;

  /// Buffers a style-loaded event that arrives before [_controller] is set.
  /// The native platform channel does not always fire `onMapCreated` before
  /// `onStyleLoaded` despite the package docs implying that order — callers
  /// that call back into [ml.MapController] from `onStyleLoaded` (e.g.
  /// `fitBounds`) would silently no-op on a still-null controller otherwise.
  ml.StyleController? _pendingStyle;

  /// The last successfully-resolved style JSON. Cached so a provider refresh
  /// (e.g. a theme toggle) never drops us back to the loading state and
  /// remounts the map — the live swap goes through [ml.MapController.setStyle]
  /// instead.
  String? _lastStyleJson;

  bool _cacheWarmed = false;

  @override
  Widget build(BuildContext context) {
    // Warms the shared app-wide glyph/sprite cache on first map open.
    // Idempotent against the trail-download trigger.
    if (!_cacheWarmed) {
      _cacheWarmed = true;
      ref.read(glyphSpriteCacheProvider.future).ignore();
    }

    // Live style swap: when the style JSON changes (theme toggle) —
    // or, for a downloaded trail, when the offline glyph/sprite cache finishes
    // warming — swap the composed style in place on the already-mounted map, no
    // ObjectKey remount, no flash. The offline branch reruns the
    // rewrite so the swap keeps resolving from file:// + pmtiles://file://.
    ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
    if (widget.offline) {
      ref.listen(glyphSpriteCacheProvider, (_, _) => _swapStyle());
    }

    final baseAsync = ref.watch(mapStyleJsonProvider);
    final baseJson = baseAsync.value;
    Object? error = baseAsync.error;

    // Offline: the style is rewritten so glyphs/sprite resolve
    // from the app-wide file:// cache and the protomaps tiles resolve
    // from the trail's local .pmtiles cells. Online trails use the base JSON
    // unchanged.
    GlyphSpriteCachePaths? cache;
    if (widget.offline) {
      final cacheAsync = ref.watch(glyphSpriteCacheProvider);
      cache = cacheAsync.value;
      error ??= cacheAsync.error;
    }

    final composed = _composeStyle(baseJson, cache);
    if (composed != null) _lastStyleJson = composed;
    final styleJson = _lastStyleJson;

    if (styleJson == null) {
      if (error != null) {
        return Center(child: Text(error.toString()));
      }
      return ColoredBox(color: Theme.of(context).colorScheme.surface);
    }

    return _buildMap(context, styleJson);
  }

  /// Composes the style JSON to hand to the map from the two resolved inputs.
  ///
  /// Online: the [baseJson] as-is. Offline: [baseJson] rewritten
  /// via [rewriteStyleForOffline] so `glyphs`/`sprite` resolve from [cache] and
  /// the protomaps tiles resolve from `trail.pmTiles` (`pmtiles://file://`).
  /// Returns null while a required input is still resolving or if the rewrite
  /// rejects an input — the caller then shows the loading passthrough.
  String? _composeStyle(String? baseJson, GlyphSpriteCachePaths? cache) {
    if (baseJson == null) return null;
    if (!widget.offline) return baseJson;
    if (cache == null) return null;
    try {
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      final offlineStyle = rewriteStyleForOffline(
        decoded,
        cacheRoot: cache.root,
        cellPaths: widget.trail.pmTiles,
        demCellPaths: widget.trail.demPmTiles,
        dark:
            effectiveBrightness(ref.read(themeModeProvider)) == Brightness.dark,
      );
      return jsonEncode(offlineStyle);
    } catch (e) {
      debugPrint('TrailMap: offline style rewrite failed — $e');
      return null;
    }
  }

  /// Recomposes the (possibly offline-rewritten) style from current provider
  /// state and swaps it onto the mounted controller in place.
  void _swapStyle() {
    final controller = _controller;
    if (controller == null) return;
    final baseJson = ref.read(mapStyleJsonProvider).value;
    final cache = ref.read(glyphSpriteCacheProvider).value;
    final json = _composeStyle(baseJson, cache);
    if (json != null && json != _lastStyleJson) {
      _lastStyleJson = json;
      controller.setStyle(json);
    }
  }

  Widget _buildMap(BuildContext context, String styleJson) {
    final center = ml.Geographic(
      lat: widget.trail.lat ?? 0,
      lon: widget.trail.lon ?? 0,
    );

    return ml.MapLibreMap(
      options: ml.MapOptions(
        initStyle: styleJson,
        initCenter: center,
        initZoom: 18,
        gestures: widget.disabled
            ? const ml.MapGestures.none()
            : const ml.MapGestures.all(),
        androidForegroundLoadColor: Theme.of(context).colorScheme.surface,
      ),
      onMapCreated: (controller) {
        _controller = controller;
        widget.onMapCreated?.call(controller);
        final pending = _pendingStyle;
        if (pending != null) {
          _pendingStyle = null;
          _onStyleLoaded(pending);
        }
      },
      onStyleLoaded: (style) {
        if (_controller == null) {
          _pendingStyle = style;
          return;
        }
        _onStyleLoaded(style);
      },
      onEvent: (event) {
        widget.onMapEvent?.call(event);
        if (event is ml.MapEventClick) {
          widget.onTap?.call(event.point);
        }
      },
      layers: const [],
      children: [
        // Interactive trail markers (tappable waypoints + start/finish
        // pins with the 36px proximity nudge) as a WidgetLayer.
        if (widget.showTrail && widget.trail.expand?.gpx != null)
          TrailMarkerLayer(
            trail: widget.trail,
            selectedWaypoint: widget.selectedWaypoint,
            onWaypointTap: widget.onWaypointTap,
          ),

        if (widget.elevationMarkerPosition != null) _buildElevationMarker(),

        if (widget.showLocation) const LocationMarkerLayer(),

        const ml.MapScalebar(
          alignment: Alignment.topLeft,
        ), // default bottom-left
        const WandererAttribution(
          alignment: Alignment.topLeft,
          padding: EdgeInsets.symmetric(horizontal: 10, vertical: 44),
        ), // default bottom-right (ODbL)

        Align(
          alignment: Alignment.topRight,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: widget.controls ?? const [],
          ),
        ),
      ],
    );
  }

  /// (re)runs the style-loaded work: the initial camera fit plus (re)adding
  /// the trail track + static arrows. Buffered/replayed via [_pendingStyle]
  /// when the native platform channel fires `onStyleLoaded` before
  /// `onMapCreated` — otherwise `_fitInitialCamera`'s null-`_controller`
  /// early-return would silently no-op the initial fit.
  void _onStyleLoaded(ml.StyleController style) {
    _fitInitialCamera().ignore();
    // (Re)adds the trail track + static arrows after every style
    // load, so they survive the theme swap (setStyle drops them).
    if (widget.showTrail && widget.trail.expand?.gpx != null) {
      _trailLayer.add(style, widget.trail).ignore();
    }
  }

  /// Reacts to [TrailMap.showTrail] flipping after the initial style load —
  /// `_onStyleLoaded` only re-runs on a style swap (e.g. theme toggle), not
  /// on a plain widget rebuild, so the trail track layers would otherwise
  /// never be added/removed in response to the toggle.
  @override
  void didUpdateWidget(covariant TrailMap oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.showTrail == widget.showTrail) return;
    final style = _controller?.style;
    if (style == null) return;
    if (widget.showTrail && widget.trail.expand?.gpx != null) {
      _trailLayer.add(style, widget.trail).ignore();
    } else {
      _trailLayer.remove(style).ignore();
    }
  }

  Future<void> _fitInitialCamera() async {
    final controller = _controller;
    if (controller == null) return;

    // `min/max_lat/lon` ARE populated on every trail record (including the
    // single-trail `GET /trail/:id`), so read the record's bounds directly
    // rather than deriving bounds from the GPX track.
    final bounds = widget.trail.bounds;
    final hasExtent =
        bounds.latitudeNorth != bounds.latitudeSouth ||
        bounds.longitudeEast != bounds.longitudeWest;

    if (hasExtent) {
      // Near-instant initial fit to the trail bounds. Duration.zero is
      // avoided — the Android native binding passes a zero duration to
      // the underlying Java `animateCamera` as null, which throws.
      await controller.fitBounds(
        bounds: bounds,
        padding: widget.initialCameraFitPadding,
        nativeDuration: const Duration(milliseconds: 1),
      );
    } else {
      await controller.moveCamera(
        center: ml.Geographic(
          lat: widget.trail.lat ?? 0,
          lon: widget.trail.lon ?? 0,
        ),
        zoom: 18,
      );
    }
  }

  /// Elevation-profile scrub marker: a 12px white dot with a 2px
  /// black border, driven by [TrailMap.elevationMarkerPosition].
  Widget _buildElevationMarker() {
    return ml.WidgetLayer(
      markers: [
        ml.Marker(
          point: widget.elevationMarkerPosition!,
          size: const Size(12, 12),
          child: Container(
            decoration: BoxDecoration(
              color: Colors.white,
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withValues(alpha: .2),
                  blurRadius: 4,
                  offset: const Offset(0, 2),
                ),
              ],
              border: Border.all(color: Colors.black, width: 2),
            ),
          ),
        ),
      ],
    );
  }

}
