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
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/util/offline_style_rewriter.dart';

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

class _TrailMapState extends ConsumerState<TrailMap> {
  static const _trailLayer = TrailLayer();

  ml.MapController? _controller;

  /// Buffers a style-loaded event that arrives before [_controller] is set —
  /// the native channel doesn't always fire `onMapCreated` first despite docs.
  ml.StyleController? _pendingStyle;

  /// Last resolved style JSON, cached so a provider refresh (theme toggle)
  /// swaps in place via [ml.MapController.setStyle] instead of remounting.
  String? _lastStyleJson;

  bool _cacheWarmed = false;

  /// Region source/layer ids currently materialized in the mounted style —
  /// seeded from the baked style on every full load ([_seedRegionTracking])
  /// and grown by [_addRegionComposition]'s incremental add-only reconcile.
  final Set<String> _addedSourceIds = {};
  final Set<String> _addedLayerIds = {};

  @override
  Widget build(BuildContext context) {
    // Warms the shared glyph/sprite cache on first open; idempotent
    // against the trail-download trigger.
    if (!_cacheWarmed) {
      _cacheWarmed = true;
      ref.read(glyphSpriteCacheProvider.future).ignore();
    }

    // Swap the style in place on theme toggle, or once the offline
    // glyph/sprite cache finishes warming — no remount, no flash. A region
    // finishing download mid-session is applied incrementally instead
    // (RENDER-03 option-b) — never a setStyle reload (D-06).
    ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());
    if (widget.offline) {
      ref.listen(glyphSpriteCacheProvider, (_, _) => _swapStyle());
      ref.listen(regionListNotifierProvider, (_, _) => _addRegionComposition());
    }

    final baseAsync = ref.watch(mapStyleJsonProvider);
    final baseJson = baseAsync.value;
    Object? error = baseAsync.error;

    // Offline: rewrite the style so glyphs/sprite/tiles resolve from local
    // file:// / .pmtiles caches instead of the network.
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

  /// Composes the style JSON: [baseJson] as-is when online, or rewritten via
  /// [rewriteStyleForOffline] when offline. Returns null while an input is
  /// still resolving or the rewrite rejects it.
  ///
  /// Offline tiles are sourced from the region registry
  /// (`TileRepositoryManager.localTilePathsForBounds`, RENDER-01) rather than
  /// the trail's own legacy tile-path fields — re-queried fresh on every call
  /// against the trail's fixed bounds (D-05), so this always reflects
  /// whichever regions are downloaded right now. An uncovered viewport (no
  /// vector paths) returns null → the existing `build()` passthrough renders
  /// a blank basemap with no banner (D-01/D-02) instead of
  /// [rewriteStyleForOffline] throwing on empty `cellPaths`.
  String? _composeStyle(String? baseJson, GlyphSpriteCachePaths? cache) {
    if (baseJson == null) return null;
    if (!widget.offline) return baseJson;
    if (cache == null) return null;
    try {
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      final tiles = ref
          .read(tileRepositoryManagerProvider)
          .localTilePathsForBounds(widget.trail.bounds);
      if (tiles.vectorPaths.isEmpty) return null;
      final offlineStyle = rewriteStyleForOffline(
        decoded,
        cacheRoot: cache.root,
        cellPaths: tiles.vectorPaths,
        demCellPaths: tiles.demPaths,
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
    final baseJson = ref.read(mapStyleJsonProvider).value;
    final cache = ref.read(glyphSpriteCacheProvider).value;
    final json = _composeStyle(baseJson, cache);
    if (json != null && json != _lastStyleJson) {
      _lastStyleJson = json;
      controller.setStyle(json);
    }
  }

  /// (Re)seeds [_addedSourceIds]/[_addedLayerIds] from a just-loaded composed
  /// style JSON's `sources` map and the layer ids that reference them —
  /// excludes the source-less `background` chrome layer, which is never a
  /// region layer and must never be reconciled. Called after every full
  /// style load (initial mount or a theme-toggle `setStyle`) so the tracking
  /// sets stay consistent with whatever region ids are actually baked in.
  void _seedRegionTracking(String? composedJson) {
    if (composedJson == null) return;
    final decoded = jsonDecode(composedJson) as Map<String, dynamic>;
    final sources = decoded['sources'];
    final sourceIds = sources is Map<String, dynamic>
        ? sources.keys.toSet()
        : const <String>{};

    _addedSourceIds
      ..clear()
      ..addAll(sourceIds);
    _addedLayerIds.clear();

    final layers = decoded['layers'];
    if (layers is! List) return;
    for (final layer in layers) {
      if (layer is! Map) continue;
      final source = layer['source'];
      final id = layer['id'];
      if (source is String && id is String && sourceIds.contains(source)) {
        _addedLayerIds.add(id);
      }
    }
  }

  /// Builds a typed [ml.Source] from a composed style's raw JSON source
  /// entry. `raster-dem` entries (hillshade) become [ml.RasterDemSource]
  /// with the Terrarium encoding [rewriteStyleForOffline] already assumes;
  /// everything else becomes an [ml.VectorSource]. Returns null (and logs)
  /// if the entry is missing its `url`.
  ml.Source? _sourceFromJson(String id, Map source) {
    final url = source['url'] as String?;
    if (url == null) {
      debugPrint('TrailMap: region source "$id" missing url — skipped');
      return null;
    }
    if (source['type'] == 'raster-dem') {
      return ml.RasterDemSource(
        id: id,
        url: url,
        maxZoom: (source['maxzoom'] as num).toDouble(),
        tileSize: (source['tileSize'] as num?)?.toInt() ?? 512,
        encoding: const ml.RasterDemTerrariumEncoding(),
      );
    }
    return ml.VectorSource(
      id: id,
      url: url,
      maxZoom: (source['maxzoom'] as num?)?.toDouble() ?? 14,
    );
  }

  /// Builds a typed [ml.StyleLayer] from a composed style's raw JSON layer
  /// entry. Source-less `background` layers (never a region layer) and any
  /// unsupported layer type return null so a future style-layer type
  /// degrades gracefully instead of crashing.
  ml.StyleLayer? _layerFromJson(Map layer) {
    final type = layer['type'];
    if (type == 'background') return null;

    final id = layer['id'] as String;
    final sourceId = layer['source'] as String;
    final sourceLayerId = layer['source-layer'] as String?;
    final paint = (layer['paint'] as Map?)?.cast<String, Object>() ?? const {};
    final layout =
        (layer['layout'] as Map?)?.cast<String, Object>() ?? const {};
    final filter = (layer['filter'] as List?)?.cast<Object>();
    final minZoom = (layer['minzoom'] as num?)?.toDouble() ?? 0;
    final maxZoom = (layer['maxzoom'] as num?)?.toDouble() ?? 24;

    switch (type) {
      case 'fill':
        return ml.FillStyleLayer(
          id: id,
          sourceId: sourceId,
          sourceLayerId: sourceLayerId,
          paint: paint,
          layout: layout,
          filter: filter,
          minZoom: minZoom,
          maxZoom: maxZoom,
        );
      case 'line':
        return ml.LineStyleLayer(
          id: id,
          sourceId: sourceId,
          sourceLayerId: sourceLayerId,
          paint: paint,
          layout: layout,
          filter: filter,
          minZoom: minZoom,
          maxZoom: maxZoom,
        );
      case 'symbol':
        return ml.SymbolStyleLayer(
          id: id,
          sourceId: sourceId,
          sourceLayerId: sourceLayerId,
          paint: paint,
          layout: layout,
          filter: filter,
          minZoom: minZoom,
          maxZoom: maxZoom,
        );
      case 'hillshade':
        return ml.HillshadeStyleLayer(
          id: id,
          sourceId: sourceId,
          paint: paint,
          layout: layout,
          minZoom: minZoom,
          maxZoom: maxZoom,
        );
      default:
        debugPrint('TrailMap: unsupported region layer type "$type" — skipped');
        return null;
    }
  }

  /// Incremental ADD-ONLY reconcile (RENDER-03 option-b, D-06): recomputes
  /// the composed style for [widget.trail.bounds] and adds any
  /// newly-relevant region source/layer via [ml.StyleController.addSource]/
  /// [ml.StyleController.addLayer] — never a full-style [ml.MapController.
  /// setStyle] reload, and never a remove (TrailMap's bounds never change,
  /// so a region only ever adds coverage). Hillshade layers are inserted
  /// below the first vector layer so relief renders underneath the basemap
  /// (25-01 finding 2).
  Future<void> _addRegionComposition() async {
    final style = _controller?.style;
    if (style == null) return;

    final tiles = ref
        .read(tileRepositoryManagerProvider)
        .localTilePathsForBounds(widget.trail.bounds);
    if (tiles.vectorPaths.isEmpty) return;

    final baseJson = ref.read(mapStyleJsonProvider).value;
    final cache = ref.read(glyphSpriteCacheProvider).value;
    if (baseJson == null || cache == null) return;

    Map<String, dynamic> composed;
    try {
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      composed = rewriteStyleForOffline(
        decoded,
        cacheRoot: cache.root,
        cellPaths: tiles.vectorPaths,
        demCellPaths: tiles.demPaths,
        dark:
            effectiveBrightness(ref.read(themeModeProvider)) == Brightness.dark,
      );
    } catch (e) {
      debugPrint('TrailMap: incremental region compose failed — $e');
      return;
    }

    final sources = composed['sources'];
    final layers = composed['layers'];
    if (sources is! Map<String, dynamic> || layers is! List) return;

    final firstVectorLayer = layers.whereType<Map>().firstWhere(
      (layer) => const {'fill', 'line', 'symbol'}.contains(layer['type']),
      orElse: () => const {},
    );
    final firstVectorLayerId = firstVectorLayer['id'] as String?;

    for (final entry in sources.entries) {
      final id = entry.key;
      if (_addedSourceIds.contains(id)) continue;
      final source = entry.value;
      if (source is! Map) continue;
      try {
        final src = _sourceFromJson(id, source);
        if (src != null) {
          await style.addSource(src);
          _addedSourceIds.add(id);
        }
      } catch (e) {
        debugPrint('TrailMap: incremental addSource failed for "$id" — $e');
      }
    }

    for (final layer in layers) {
      if (layer is! Map) continue;
      final source = layer['source'];
      final id = layer['id'];
      if (source is! String || id is! String) continue;
      if (!sources.containsKey(source)) continue;
      if (_addedLayerIds.contains(id)) continue;

      final layerObj = _layerFromJson(layer);
      if (layerObj == null) continue;
      try {
        if (layer['type'] == 'hillshade') {
          await style.addLayer(layerObj, belowLayerId: firstVectorLayerId);
        } else {
          await style.addLayer(layerObj);
        }
        _addedLayerIds.add(id);
      } catch (e) {
        debugPrint('TrailMap: incremental addLayer failed for "$id" — $e');
      }
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

        const ml.MapScalebar(alignment: Alignment.topLeft),
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
    // Reseed the region tracking sets from whatever's baked into the
    // freshly-loaded style (initial mount or a theme-toggle setStyle).
    _seedRegionTracking(_lastStyleJson);
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
