import 'package:wanderer/components/map/map_ui_controls.dart';
import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/trail_collection_map.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/map/map_marker_gestures.dart';
import 'package:wanderer/components/map/trail_layer.dart'
    show TrailLayer, kTrailRouteColor;
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/list_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/components/category/category_icon.dart';
import 'package:wanderer/util/geo/polyline.dart';

class ListDetailMapScreen extends ConsumerStatefulWidget {
  final String id;
  const ListDetailMapScreen({super.key, required this.id});

  @override
  ConsumerState<ListDetailMapScreen> createState() =>
      _ListDetailMapScreenState();
}

class _ListDetailMapScreenState extends ConsumerState<ListDetailMapScreen> {
  static const _trailLayer = TrailLayer();

  ml.MapController? _controller;
  Trail? _selectedTrail;
  ml.LngLatBounds? _combinedBoundsCache;

  void _deselect() {
    setState(() => _selectedTrail = null);
    final bounds = _combinedBoundsCache;
    if (bounds != null) {
      _controller?.fitBounds(
        bounds: bounds,
        padding: const EdgeInsets.all(40),
        nativeDuration: const Duration(milliseconds: 750),
      );
    }
  }

  void _onMarkerTap(Trail trail) {
    setState(() => _selectedTrail = trail);

    final bounds = ml.LngLatBounds(
      longitudeEast: trail.maxLon,
      longitudeWest: trail.minLon,
      latitudeNorth: trail.maxLat,
      latitudeSouth: trail.minLat,
    );

    _controller?.fitBounds(
      bounds: bounds,
      padding: const EdgeInsets.fromLTRB(40, 56, 40, 120),
      nativeDuration: const Duration(milliseconds: 750),
    );
  }

  @override
  Widget build(BuildContext context) {
    final listAsync = ref.watch(listProvider(widget.id));

    return listAsync.when(
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (err, stack) => Scaffold(
        body: WandererError(err: err, stack: stack),
      ),
      data: (list) {
        final trails = list.expand?.trails ?? [];
        final allCategories = ref.watch(categoryProvider).value ?? [];
        final allSubcategories = ref.watch(subcategoryProvider);

        final polylines = trails
            .where((t) => t.polyline != null && t.polyline!.isNotEmpty)
            .map(
              (t) => ml.Feature<ml.LineString>(
                geometry: ml.LineString.from(PolylineUtil.decode(t.polyline!)),
              ),
            )
            .toList();

        final lines = trails
            .where((t) => t.polyline != null && t.polyline!.isNotEmpty)
            .map((t) => PolylineUtil.decode(t.polyline!))
            .toList();

        final markers = trails.where((t) => t.lat != null && t.lon != null).map(
          (t) {
            final Category? category = t.categoryId != null
                ? allCategories.firstWhereOrNull((c) => c.id == t.categoryId)
                : null;
            final subcategory = t.subcategoryId != null
                ? allSubcategories.firstWhereOrNull(
                    (s) => s.id == t.subcategoryId,
                  )
                : null;
            final isSelected = _selectedTrail?.id == t.id;
            return ml.Marker(
              point: ml.Geographic(lat: t.lat!, lon: t.lon!),
              size: const Size(36, 36),
              child: MapMarkerGestures(
                onTap: () => _onMarkerTap(t),
                child: Container(
                  decoration: BoxDecoration(
                    color: isSelected
                        ? Theme.of(context).colorScheme.secondary
                        : Theme.of(context).primaryColor,
                    shape: BoxShape.circle,
                    border: Border.all(color: Colors.white, width: 2),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withValues(alpha: 0.2),
                        blurRadius: 4,
                        offset: const Offset(0, 2),
                      ),
                    ],
                  ),
                  child: Center(
                    child: trailCategoryIcon(
                      category,
                      subcategory: subcategory,
                      color: isSelected
                          ? Theme.of(context).colorScheme.onSurface
                          : Theme.of(context).colorScheme.onPrimary,
                    ),
                  ),
                ),
              ),
            );
          },
        ).toList();

        final combinedBounds = _combinedBounds(trails);
        _combinedBoundsCache = combinedBounds;

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
            ),
          ),
          body: Stack(
            children: [
              TrailCollectionMap(
                onMapCreated: (controller) => _controller = controller,
                onStyleLoaded: (style) {
                  final bounds = combinedBounds;
                  if (bounds != null) {
                    _controller?.fitBounds(
                      bounds: bounds,
                      padding: const EdgeInsets.all(40),
                      // Duration.zero crashes the Android native binding
                      // (animateCamera receives a null duration).
                      nativeDuration: const Duration(milliseconds: 1),
                    );
                  }
                  _trailLayer.addArrows(style, lines).ignore();
                },
                onMapEvent: (event) {
                  if (event is ml.MapEventClick) _deselect();
                },
                layers: polylines.isNotEmpty
                    ? [
                        ml.PolylineLayer(
                          polylines: polylines,
                          color: kTrailRouteColor,
                          width: 5,
                        ),
                      ]
                    : null,
                children: [
                  if (markers.isNotEmpty)
                    ml.WidgetLayer(allowInteraction: true, markers: markers),
                  const SafeArea(
                    child: WandererMapCompass(
                      hideIfRotatedNorth: true,
                      padding: EdgeInsets.only(top: 0, right: 8),
                    ),
                  ),
                  const WandererMapScalebar(),
                  const WandererAttribution(),
                ],
              ),

              if (_selectedTrail != null)
                Positioned(
                  bottom: 16,
                  left: 16,
                  right: 16,
                  child: Dismissible(
                    key: ValueKey(_selectedTrail!.id),
                    direction: DismissDirection.down,
                    onDismissed: (_) => _deselect(),
                    child: TrailListItem(
                      trail: _selectedTrail!,
                      onTrailSelect: () =>
                          context.push('/trail/${_selectedTrail!.id}'),
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  ml.LngLatBounds? _combinedBounds(List<Trail> trails) {
    final withBounds = trails.where(
      (t) => t.minLat != 0 || t.maxLat != 0 || t.minLon != 0 || t.maxLon != 0,
    );
    if (withBounds.isEmpty) return null;

    double minLat = double.infinity;
    double maxLat = double.negativeInfinity;
    double minLon = double.infinity;
    double maxLon = double.negativeInfinity;

    for (final t in withBounds) {
      if (t.minLat < minLat) minLat = t.minLat;
      if (t.maxLat > maxLat) maxLat = t.maxLat;
      if (t.minLon < minLon) minLon = t.minLon;
      if (t.maxLon > maxLon) maxLon = t.maxLon;
    }

    return ml.LngLatBounds(
      longitudeEast: maxLon,
      longitudeWest: minLon,
      latitudeNorth: maxLat,
      latitudeSouth: minLat,
    );
  }
}
