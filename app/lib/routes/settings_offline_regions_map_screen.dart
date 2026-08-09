import 'dart:convert';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/trail_collection_map.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/region_geometry.dart';
import 'package:wanderer/provider/region/region_geometry_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/region/file_path.dart';

/// Full-screen map showing a downloadable region's boundary polygon,
/// reached from the map icon on `settings_offline_regions_screen.dart`'s
/// active-row list. Fits immediately to the region's locally-cached bbox
/// (no network wait) and draws the boundary once/if the cached geometry
/// fetch resolves — a failure surfaces as a toast only, never blocking the
/// map.
class SettingsOfflineRegionsMapScreen extends ConsumerStatefulWidget {
  /// The region's materialized path (e.g. `canada.alberta.south`).
  final String path;

  const SettingsOfflineRegionsMapScreen({super.key, required this.path});

  @override
  ConsumerState<SettingsOfflineRegionsMapScreen> createState() =>
      _SettingsOfflineRegionsMapScreenState();
}

class _SettingsOfflineRegionsMapScreenState
    extends ConsumerState<SettingsOfflineRegionsMapScreen> {
  ml.MapController? _controller;
  ml.StyleController? _style;
  bool _polygonAdded = false;
  bool _toastShown = false;

  /// Fits the camera to the region's LOCALLY CACHED catalog bbox — never a
  /// network read. If no row matches (e.g. a stale/invalid deep link), this
  /// no-ops rather than crashing or falling back to a network fetch.
  void _fitToBbox() {
    final controller = _controller;
    if (controller == null) return;

    final region = ref
        .read(regionListNotifierProvider)
        .firstWhereOrNull((r) => r.path == widget.path);
    if (region == null) return;

    final bounds = ml.LngLatBounds(
      longitudeEast: region.maxLon,
      longitudeWest: region.minLon,
      latitudeNorth: region.maxLat,
      latitudeSouth: region.minLat,
    );

    // Never Duration.zero — it crashes the Android native binding.
    controller.fitBounds(
      bounds: bounds,
      padding: const EdgeInsets.all(40),
      nativeDuration: const Duration(milliseconds: 1),
    );
  }

  /// The single draw path for the boundary polygon, called from both
  /// `onStyleLoaded` and the geometry listener below — either resolution
  /// order (style-before-geometry or geometry-before-style) reaches this
  /// same guarded call.
  Future<void> _maybeDrawPolygon() async {
    if (_polygonAdded) return;
    final style = _style;
    if (style == null) return;

    final geometry = ref.read(regionGeometryProvider(widget.path)).value;
    if (geometry == null) return;

    _polygonAdded = true;

    try {
      await style.addSource(
        ml.GeoJsonSource(
          id: 'region-outline',
          data: jsonEncode({
            'type': 'Feature',
            'geometry': geometry.polygon,
            'properties': <String, Object>{},
          }),
        ),
      );

      // Paint values ported verbatim from
      // db/routes/regions_ext/regions_ui.html — identical in light and dark
      // themes, deliberately NOT derived from Theme.of(context).
      await style.addLayer(
        const ml.FillStyleLayer(
          id: 'region-outline-fill',
          sourceId: 'region-outline',
          paint: <String, Object>{
            'fill-color': '#1055c9',
            'fill-opacity': 0.18,
          },
        ),
      );
      await style.addLayer(
        const ml.LineStyleLayer(
          id: 'region-outline-line',
          sourceId: 'region-outline',
          paint: <String, Object>{'line-color': '#1055c9', 'line-width': 2},
        ),
      );
    } catch (e) {
      debugPrint(
        'settings_offline_regions_map_screen: failed to draw region outline — $e',
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final hasPath = widget.path.isNotEmpty && isValidRegionPath(widget.path);

    if (hasPath) {
      ref.listen<AsyncValue<RegionGeometry>>(
        regionGeometryProvider(widget.path),
        (
          AsyncValue<RegionGeometry>? previous,
          AsyncValue<RegionGeometry> next,
        ) {
          if (next.hasValue) {
            _maybeDrawPolygon();
          } else if (next.hasError && !_toastShown) {
            _toastShown = true;
            ref
                .read(toastProvider.notifier)
                .add(
                  ToastMessage(
                    type: ToastType.error,
                    icon: FontAwesomeIcons.circleExclamation,
                    text: AppLocalizations.of(
                      context,
                    )!.regions_map_geometry_failed,
                  ),
                );
          }
        },
      );
    }

    return Scaffold(
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: IconButton(
          icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
          onPressed: () => context.pop(),
          style: IconButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.surface,
          ),
          tooltip: AppLocalizations.of(context)!.regions_map_back_label,
        ),
      ),
      body: TrailCollectionMap(
        onMapCreated: (controller) => _controller = controller,
        onStyleLoaded: (style) {
          _style = style;
          _fitToBbox();
          _maybeDrawPolygon();
        },
      ),
    );
  }
}
