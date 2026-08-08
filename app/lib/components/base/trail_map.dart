import 'package:wanderer/components/map/map_ui_controls.dart';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/platform_view_pop_guard.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/map/location_marker_layer.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/glyph_sprite_cache_paths.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/region/tile_proxy_provider.dart';
import 'package:wanderer/util/region/offline_style_rewriter.dart';

/// Native MapLibre GL map host for a single [Trail]. Swaps light/dark styles
/// live via [ml.MapController.setStyle] (no remount/flash) and hosts the
/// elevation-scrub and interim-location markers.
///
/// Single-trail detail views only — use `TrailCollectionMap` for screens
/// showing a collection of trails.
class TrailMap extends ConsumerStatefulWidget {
  final Trail trail;

  /// Hands the native [ml.MapController] back to the caller once created.
  final void Function(ml.MapController controller)? onMapCreated;

  final bool disabled;
  final bool offline;
  final List<Widget>? controls;
  final ml.Geographic? elevationMarkerPosition;
  final EdgeInsets initialCameraFitPadding;

  final bool showTrail;
  final bool showLocation;
  final Waypoint? selectedWaypoint;

  final void Function(ml.Geographic point)? onTap;
  final void Function(ml.MapEvent event)? onMapEvent;
  final void Function(Waypoint wp)? onWaypointTap;
  final void Function(Waypoint wp, ml.Geographic point)? onWaypointDragEnd;

  const TrailMap({
    super.key,
    required this.trail,
    this.onMapCreated,
    this.onTap,
    this.onWaypointTap,
    this.onWaypointDragEnd,
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

class _TrailMapState extends ConsumerState<TrailMap>
    with PlatformViewPopGuard<TrailMap> {
  static const _trailLayer = TrailLayer();

  ml.MapController? _controller;

  /// Buffers a style-loaded event that arrives before [_controller] is set —
  /// the native channel doesn't always fire `onMapCreated` first despite docs.
  ml.StyleController? _pendingStyle;

  /// Last resolved style JSON, cached so a provider refresh (theme toggle)
  /// swaps in place via [ml.MapController.setStyle] instead of remounting.
  String? _lastStyleJson;

  bool _cacheWarmed = false;

  @override
  Widget build(BuildContext context) {
    // Warms the shared glyph/sprite cache on first open (online only) so a
    // later offline open renders from disk; idempotent against the
    // trail-download trigger. Skipped when offline — the warm is a network
    // download and the cache is already populated by the time it is needed.
    if (!_cacheWarmed && !widget.offline) {
      _cacheWarmed = true;
      ref.read(glyphSpriteCacheProvider.future).ignore();
    }

    // Swap the style in place on theme toggle, or once the offline
    // glyph/sprite cache finishes warming — no remount, no flash. Region
    // coverage is resolved by the loopback tile proxy per-tile,
    // so no separate region-change listener is needed here. Offline reads the
    // network-free providers so no `/map/style-sources` call is ever made.
    if (widget.offline) {
      ref.listen(offlineMapStyleJsonProvider, (_, _) => _swapStyle());
      ref.listen(offlineGlyphSpritePathsProvider, (_, _) => _swapStyle());
    } else {
      ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
    }

    final baseAsync = widget.offline
        ? ref.watch(offlineMapStyleJsonProvider)
        : ref.watch(mapStyleJsonProvider);
    final baseJson = baseAsync.value;
    Object? error = baseAsync.error;

    // Offline: rewrite the style so glyphs/sprite/tiles resolve from local
    // file:// / .pmtiles caches instead of the network.
    GlyphSpriteCachePaths? cache;
    if (widget.offline) {
      final cacheAsync = ref.watch(offlineGlyphSpritePathsProvider);
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

  /// Composes the style JSON: [baseJson] as-is when online, or rewritten via
  /// [rewriteStyleForProxy] when offline. Returns null while an input is
  /// still resolving or the rewrite rejects it.
  ///
  /// Offline tiles resolve through the loopback tile proxy — a
  /// single static XYZ source baked into every composed style, with
  /// per-tile region coverage resolved server-side
  /// (`resolveRegionForTile`) rather than pre-queried here. This means
  /// `TrailMap` now serves tiles for any downloaded region the (fixed,
  /// trail-bounded) camera happens to render, not only the trail's own
  /// bbox — intentional, and harmless because the camera stays fit to the
  /// trail. Uncovered tiles resolve to a blank basemap via the proxy's own
  /// 404 responses, so no banner or empty state is added here.
  String? _composeStyle(String? baseJson, GlyphSpriteCachePaths? cache) {
    if (baseJson == null) return null;
    if (!widget.offline) return baseJson;
    if (cache == null) return null;
    try {
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      final offlineStyle = rewriteStyleForProxy(
        decoded,
        cacheRoot: cache.root,
        proxyBaseUrl: ref.read(tileProxyBaseUrlProvider),
        dark:
            effectiveBrightness(ref.read(themeModeProvider)) == Brightness.dark,
      );
      return jsonEncode(offlineStyle);
    } catch (e) {
      debugPrint('TrailMap: offline style rewrite failed — $e');
      return null;
    }
  }

  void _swapStyle() {
    final controller = _controller;
    if (controller == null) return;
    final baseJson = widget.offline
        ? ref.read(offlineMapStyleJsonProvider).value
        : ref.read(mapStyleJsonProvider).value;
    final cache = widget.offline
        ? ref.read(offlineGlyphSpritePathsProvider).value
        : null;
    final json = _composeStyle(baseJson, cache);
    if (json != null && json != _lastStyleJson) {
      _lastStyleJson = json;
      controller.setStyle(json);
    }
  }

  Widget _buildMap(BuildContext context, String styleJson) {
    // Drop the native surface as the pop begins so it cannot outlive the
    // transition — see [PlatformViewPopGuard]. Same colour as the map's
    // androidForegroundLoadColor, so the swap reads as the map simply being
    // covered rather than as a flash.
    if (platformViewPopping) {
      return ColoredBox(color: Theme.of(context).colorScheme.surface);
    }

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
        // MapLibre's texture mode renders into a TextureView, costing a
        // GPU→CPU→GPU copy per frame — measured at 76% of a core on the
        // TextureViewRend thread alone while panning. SurfaceView renders
        // direct. `hc` is required alongside it: with textureMode off, the
        // default `tlhc_vd` falls back to Virtual Display (its own slow
        // path), whereas Hybrid Composition keeps correct z-ordering for
        // the Flutter marker layers drawn over the map.
        androidTextureMode: false,
        androidMode: ml.AndroidPlatformViewMode.hc,
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
        // Tappable waypoint + start/finish markers, with a 36px proximity nudge.
        if (widget.showTrail && widget.trail.expand?.gpx != null)
          TrailMarkerLayer(
            trail: widget.trail,
            selectedWaypoint: widget.selectedWaypoint,
            onWaypointTap: widget.onWaypointTap,
            onWaypointDragEnd: widget.onWaypointDragEnd,
          ),

        if (widget.elevationMarkerPosition != null) _buildElevationMarker(),

        if (widget.showLocation) const LocationMarkerLayer(),

        const WandererMapScalebar(alignment: Alignment.topLeft),
        const WandererAttribution(
          alignment: Alignment.topLeft,
          padding: EdgeInsets.symmetric(horizontal: 10, vertical: 44),
        ),

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

  /// Runs the initial camera fit and (re)adds the trail track, buffered via
  /// [_pendingStyle] if `onStyleLoaded` fires before `onMapCreated`.
  void _onStyleLoaded(ml.StyleController style) {
    _fitInitialCamera().ignore();
    // Re-add track + arrows after every style load — setStyle drops them.
    if (widget.showTrail && widget.trail.expand?.gpx != null) {
      _trailLayer.add(style, widget.trail).ignore();
    }
  }

  /// Reacts to [TrailMap.showTrail] flipping and to the trail's track being
  /// replaced in place (e.g. after a route-planner edit) — `_onStyleLoaded`
  /// only re-runs on a style swap, not a plain widget rebuild, so neither
  /// case would otherwise reach the mounted native GL layer.
  @override
  void didUpdateWidget(covariant TrailMap oldWidget) {
    super.didUpdateWidget(oldWidget);
    final style = _controller?.style;
    if (style == null) return;

    if (oldWidget.showTrail != widget.showTrail) {
      if (widget.showTrail && widget.trail.expand?.gpx != null) {
        _trailLayer.add(style, widget.trail).ignore();
      } else {
        _trailLayer.remove(style).ignore();
      }
      return;
    }

    // Identity compare: only a route edit produces a new Gpx instance;
    // unrelated trail edits keep the same gpx reference via copyWith.
    if (widget.showTrail &&
        !identical(oldWidget.trail.expand?.gpx, widget.trail.expand?.gpx)) {
      _trailLayer.remove(style).then((_) {
        if (widget.trail.expand?.gpx != null) {
          _trailLayer.add(style, widget.trail).ignore();
        }
      }).ignore();
      _fitInitialCamera().ignore();
    }
  }

  Future<void> _fitInitialCamera() async {
    final controller = _controller;
    if (controller == null) return;

    // `min/max_lat/lon` are populated on every trail record, so read
    // bounds directly rather than deriving them from the GPX track.
    final bounds = widget.trail.bounds;
    final hasExtent =
        bounds.latitudeNorth != bounds.latitudeSouth ||
        bounds.longitudeEast != bounds.longitudeWest;

    if (hasExtent) {
      // Duration.zero is avoided: the Android binding passes it to
      // `animateCamera` as null, which throws.
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

  /// Elevation-profile scrub marker, driven by [TrailMap.elevationMarkerPosition].
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
