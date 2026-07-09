import 'dart:convert';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_map_location_marker/flutter_map_location_marker.dart'
    show LocationMarkerPosition;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/search_map.dart';
import 'package:wanderer/components/map/cluster_layer.dart';
import 'package:wanderer/components/map/trail_layer.dart' show kTrailRouteColor;
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/map_camera_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/map_cluster_search_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_polyline_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';

class MapScreen extends ConsumerStatefulWidget {
  final ml.Geographic? initialCenter;
  final double? initialZoom;

  const MapScreen({super.key, this.initialCenter, this.initialZoom});

  @override
  ConsumerState<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends ConsumerState<MapScreen>
    with TickerProviderStateMixin {
  ml.MapController? _controller;
  bool _initialSearchDone = false;

  TrailSearchResult? _selectedTrail;
  List<ml.Geographic>? _selectedPolyline;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();
  late final ValueNotifier<double> _sheetSize;
  late final AnimationController _mapButtonController;
  late final Animation<double> _mapButtonScale;
  late final AnimationController _searchAreaController;
  late final Animation<double> _searchAreaScale;
  ScrollController? _sheetScrollController;

  final sheetMinSize = 0.2;
  final sheetMediumsize = 0.7;
  final sheetMaxSize = 1.0;

  @override
  void initState() {
    super.initState();
    _sheetController.addListener(_onSheetSizeChanged);
    _sheetSize = ValueNotifier<double>(sheetMinSize);
    _mapButtonController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
      reverseDuration: const Duration(milliseconds: 0),
    );
    _mapButtonScale = CurvedAnimation(
      parent: _mapButtonController,
      curve: Curves.elasticOut,
      reverseCurve: Curves.easeOut,
    );
    _searchAreaController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
      reverseDuration: const Duration(milliseconds: 0),
    );
    _searchAreaScale = CurvedAnimation(
      parent: _searchAreaController,
      curve: Curves.elasticOut,
      reverseCurve: Curves.easeOut,
    );
  }

  @override
  void didUpdateWidget(MapScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    final newCenter = widget.initialCenter;
    if (newCenter != null && newCenter != oldWidget.initialCenter) {
      final controller = _controller;
      if (controller == null) return;
      controller
          .animateCamera(
            center: newCenter,
            zoom: widget.initialZoom ?? 13.0,
            nativeDuration: const Duration(milliseconds: 750),
          )
          .then((_) {
            if (!mounted) return;
            final bounds = controller.getVisibleRegion();
            final zoom = controller.getCamera().zoom;
            ref
                .read(mapClusterSearchProvider.notifier)
                .searchInBounds(bounds, zoom);
            ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds);
          });
    }
  }

  void _onSheetSizeChanged() {
    _sheetSize.value = _sheetController.size;
    if (_sheetController.size >= sheetMaxSize) {
      if (!_mapButtonController.isAnimating &&
          !_mapButtonController.isCompleted) {
        _mapButtonController.forward();
      }
    } else {
      if (_mapButtonController.value > 0) {
        _mapButtonController.reverse();
      }
    }
  }

  @override
  void dispose() {
    _sheetController.removeListener(_onSheetSizeChanged);
    _sheetController.dispose();
    _sheetSize.dispose();
    _mapButtonController.dispose();
    _searchAreaController.dispose();
    super.dispose();
  }

  /// Fetch-then-fit trail selection (D-02): selects [trailId] immediately
  /// (metadata resolved from the parallel `mapTrailSearchProvider` results,
  /// via `firstWhereOrNull` — T-16-01, the id is untrusted input), then fits
  /// the camera to the trail's polyline once it resolves.
  void _selectTrail(String trailId) {
    final trails = ref.read(mapTrailSearchProvider).value ?? [];
    setState(() {
      _selectedTrail = trails.firstWhereOrNull((t) => t.id == trailId);
      _selectedPolyline = null;
    });

    ref.read(trailPolylineProvider(trailId).future).then((polyline) {
      if (!mounted || _selectedTrail?.id != trailId) return;
      if (polyline != null) {
        _controller?.fitBounds(
          bounds: ml.LngLatBounds.fromPoints(polyline),
          padding: const EdgeInsets.fromLTRB(40, 56, 40, 248),
          nativeDuration: const Duration(milliseconds: 750),
        );
      }
      setState(() => _selectedPolyline = polyline);
    });
  }

  @override
  Widget build(BuildContext context) {
    final savedCamera = ref.read(mapCameraProvider);

    // Re-query rendering (CLUS-04): swap the native cluster source's data in
    // place on every new mapClusterSearchProvider result — never remove and
    // re-add the source.
    ref.listen(mapClusterSearchProvider, (previous, next) {
      final style = _controller?.style;
      final data = next.value;
      if (style == null || data == null || next.isLoading) return;
      updateClusterSource(style, jsonEncode(data)).ignore();
    });

    final filterAsync = ref.watch(trailFilterProvider('map'));
    final activeFilterCount = filterAsync.hasValue
        ? _countActiveFilters(
            filterAsync.value!,
            ref.read(trailFilterProvider('map').notifier).defaultFilter,
          )
        : 0;

    // KEEP mapTrailSearchProvider watched exactly as today — it still powers
    // the bottom-sheet TrailCard list and the tapped-trail metadata lookup
    // (the cluster endpoint's attributesToRetrieve is only id/_geo/
    // bounding_box_diagonal, Pitfall 2).
    final searchResultAsync = ref.watch(mapTrailSearchProvider);
    final trails = searchResultAsync.value ?? [];
    final allCategories = ref.watch(categoryProvider).value ?? [];
    final allSubcategories = ref.watch(subcategoryProvider);

    // mapClusterSearchProvider drives the native cluster circle/count layers
    // (cluster_layer.dart) AND the unclustered category-icon WidgetLayer
    // markers below (D-05).
    final clusterAsync = ref.watch(mapClusterSearchProvider);
    final featureCollection = clusterAsync.value;
    final features =
        (featureCollection?['features'] as List<dynamic>?) ?? const [];

    final unclusteredMarkers = <ml.Marker>[];
    for (final feature in features) {
      if (feature is! Map) continue;
      final properties = feature['properties'];
      if (properties is! Map) continue;

      final pointCount = properties['point_count'];
      if (pointCount is! num || pointCount.toInt() != 1) continue;
      // is_large is intentionally NOT filtered here (user decision, post-
      // checkpoint fix): the server marks the top MAP_MAX_POLYLINES trails by
      // size as is_large once zoom passes clusteringMaxZoom (~11) — with
      // fewer trails in view than that threshold, this can mean ALL visible
      // trails, not just rare huge ones. Full-polyline rendering for is_large
      // trails is still deferred (FUT-01); for now any unclustered point
      // (point_count == 1) renders as a category-icon marker regardless.

      final trailId = properties['id'];
      if (trailId is! String) continue;
      // T-16-01: the feature's id is untrusted input — firstWhereOrNull, not
      // firstWhere, so a missing match skips the marker instead of crashing.
      final trail = trails.firstWhereOrNull((t) => t.id == trailId);
      if (trail == null) continue;

      final geometry = feature['geometry'];
      if (geometry is! Map) continue;
      final coordinates = geometry['coordinates'];
      if (coordinates is! List || coordinates.length < 2) continue;
      final lon = (coordinates[0] as num).toDouble();
      final lat = (coordinates[1] as num).toDouble();

      final Category? category = trail.categoryId != null
          ? allCategories.firstWhereOrNull((c) => c.id == trail.categoryId)
          : null;
      final subcategory = trail.subcategoryId != null
          ? allSubcategories.firstWhereOrNull(
              (s) => s.id == trail.subcategoryId,
            )
          : null;

      unclusteredMarkers.add(
        ml.Marker(
          point: ml.Geographic(lat: lat, lon: lon),
          size: const Size(36, 36),
          child: GestureDetector(
            onTap: () => _selectTrail(trailId),
            child: Container(
              decoration: BoxDecoration(
                color: Theme.of(context).primaryColor,
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
                  color: Colors.white,
                ),
              ),
            ),
          ),
        ),
      );
    }

    return Stack(
      children: [
        SearchMap(
          initCenter: widget.initialCenter ?? savedCamera?.center,
          initZoom: widget.initialZoom ?? savedCamera?.zoom,
          onMapCreated: (controller) => _controller = controller,
          onStyleLoaded: (style) async {
            // T-16-02: fail soft on a malformed/oversized cluster response —
            // never crash the map.
            try {
              final data =
                  ref.read(mapClusterSearchProvider).value ??
                  const {'type': 'FeatureCollection', 'features': <Object>[]};
              await addClusterLayers(style, jsonEncode(data));
            } catch (e) {
              debugPrint('map_screen: failed to add cluster layers — $e');
            }

            // Initial-load search trigger (replaces the old mapStyleProvider
            // post-frame-callback pattern) — fires once per widget lifetime,
            // not on every theme-swap style reload.
            if (!_initialSearchDone) {
              _initialSearchDone = true;
              final controller = _controller;
              if (controller != null) {
                final bounds = controller.getVisibleRegion();
                final zoom = controller.getCamera().zoom;
                ref
                    .read(mapClusterSearchProvider.notifier)
                    .searchInBounds(bounds, zoom);
                ref
                    .read(mapTrailSearchProvider.notifier)
                    .searchInBounds(bounds);
              }
            }
          },
          onMapEvent: (event) {
            if (event is ml.MapEventStartMoveCamera &&
                event.reason == ml.CameraChangeReason.apiGesture) {
              _searchAreaController.forward();
              return;
            }

            if (event is ml.MapEventClick) {
              final controller = _controller;
              if (controller == null) return;

              // CLUS-03: only the native `clusters` circle layer needs
              // featuresAtPoint — unclustered points are WidgetLayer markers
              // that handle their own onTap directly (D-05).
              final clusterHits = controller.featuresAtPoint(
                event.screenPoint,
                layerIds: const ['clusters'],
              );
              if (clusterHits.isNotEmpty) {
                final currentZoom = controller.getCamera().zoom;
                controller
                    .animateCamera(
                      center: event.point,
                      zoom: currentZoom + 2,
                      nativeDuration: const Duration(milliseconds: 400),
                    )
                    .then((_) {
                      if (!mounted) return;
                      final bounds = controller.getVisibleRegion();
                      final zoom = controller.getCamera().zoom;
                      ref
                          .read(mapClusterSearchProvider.notifier)
                          .searchInBounds(bounds, zoom);
                      ref
                          .read(mapTrailSearchProvider.notifier)
                          .searchInBounds(bounds);
                    });
                return;
              }

              // Background tap (not on a cluster or a marker widget) —
              // deselect + collapse the sheet, matching today's onTap.
              if (_sheetController.isAttached) {
                _sheetController.animateTo(
                  sheetMinSize,
                  curve: Curves.easeOut,
                  duration: const Duration(milliseconds: 300),
                );
              }
              setState(() {
                _selectedTrail = null;
                _selectedPolyline = null;
              });
              return;
            }

            if (event is ml.MapEventCameraIdle) {
              final camera = _controller?.getCamera();
              if (camera != null) {
                ref
                    .read(mapCameraProvider.notifier)
                    .save(camera.center, camera.zoom);
              }
            }
          },
          layers: _selectedPolyline != null
              ? [
                  ml.PolylineLayer(
                    polylines: [
                      ml.Feature<ml.LineString>(
                        geometry: ml.LineString.from(_selectedPolyline!),
                      ),
                    ],
                    color: kTrailRouteColor,
                    width: 5,
                  ),
                ]
              : null,
          children: [
            if (unclusteredMarkers.isNotEmpty)
              ml.WidgetLayer(
                allowInteraction: true,
                markers: unclusteredMarkers,
              ),
            _LocationLayer(),
            Positioned(
              top: 124,
              right: 8,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: const [ml.MapCompass(hideIfRotatedNorth: true)],
              ),
            ),
            const ml.MapScalebar(),
            const ml.SourceAttribution(),
          ],
        ),

        if (_selectedTrail == null)
          Positioned(
            bottom: MediaQuery.of(context).size.height * sheetMinSize + 12,
            left: 0,
            right: 0,
            child: Center(
              child: ScaleTransition(
                scale: _searchAreaScale,
                child: FilledButton.icon(
                  onPressed: () {
                    _searchAreaController.reverse();
                    final controller = _controller;
                    if (controller == null) return;
                    final bounds = controller.getVisibleRegion();
                    final zoom = controller.getCamera().zoom;
                    ref
                        .read(mapClusterSearchProvider.notifier)
                        .searchInBounds(bounds, zoom);
                    ref
                        .read(mapTrailSearchProvider.notifier)
                        .searchInBounds(bounds);
                  },
                  icon: const FaIcon(FontAwesomeIcons.mapLocationDot, size: 14),
                  label: Text(AppLocalizations.of(context)!.search_this_area),
                ),
              ),
            ),
          ),

        if (_selectedTrail == null)
          DraggableScrollableSheet(
            controller: _sheetController,
            initialChildSize: sheetMinSize,
            minChildSize: sheetMinSize,
            maxChildSize: sheetMaxSize,
            snap: true,
            snapSizes: [sheetMinSize, sheetMediumsize, sheetMaxSize],
            builder: (context, scrollController) {
              _sheetScrollController = scrollController;
              return ValueListenableBuilder<double>(
                valueListenable: _sheetSize,
                builder: (context, size, child) => Container(
                  decoration: BoxDecoration(
                    color: Theme.of(context).canvasColor,
                    borderRadius: BorderRadius.vertical(
                      top: Radius.circular(size >= sheetMaxSize ? 0 : 16),
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withValues(alpha: 0.15),
                        blurRadius: 10,
                        offset: const Offset(0, -2),
                      ),
                    ],
                  ),
                  child: Padding(
                    padding: size >= sheetMediumsize
                        ? EdgeInsets.fromLTRB(0, _getDynamicPadding(size), 0, 0)
                        : EdgeInsets.zero,
                    child: child,
                  ),
                ),
                child: AsyncLoader(
                  asyncValue: searchResultAsync,
                  mockData: List.generate(5, (_) => TrailSearchResult.mock()),
                  builder: (trails) => ListView.builder(
                    itemCount: trails.length + 2,
                    padding: EdgeInsets.fromLTRB(16, 8, 16, 16),
                    controller: scrollController,
                    itemBuilder: (context, index) {
                      if (index == 0 || index == 1) {
                        return ValueListenableBuilder<double>(
                          valueListenable: _sheetSize,
                          builder: (context, size, child) {
                            double fadeStart = sheetMediumsize;
                            double opacity = 1.0;

                            if (size > fadeStart) {
                              opacity =
                                  1.0 - ((size - fadeStart) / (1 - fadeStart));
                              opacity = opacity.clamp(0.0, 1.0);
                            }

                            if (opacity == 0.0) return const SizedBox.shrink();

                            Widget child = index == 0
                                ? Center(
                                    child: Container(
                                      width: 30,
                                      height: 5,
                                      decoration: BoxDecoration(
                                        color: Colors.grey[300],
                                        borderRadius: BorderRadius.circular(10),
                                      ),
                                    ),
                                  )
                                : Padding(
                                    padding: const EdgeInsets.all(8.0),
                                    child: Center(
                                      child: Text(
                                        "${trails.length}${trails.length == 100 ? '+' : ''} ${AppLocalizations.of(context)!.trail(trails.length)}",
                                        style: Theme.of(
                                          context,
                                        ).textTheme.labelLarge,
                                      ),
                                    ),
                                  );

                            return Opacity(opacity: opacity, child: child);
                          },
                        );
                      }
                      final trail = trails[index - 2];
                      return TrailCard(
                        trail: trail,
                        onTrailSelect: () =>
                            context.push("/trail/${trail.id}", extra: trail),
                      );
                    },
                  ),
                ),
              );
            },
          ),

        SafeArea(
          child: Align(
            alignment: Alignment.topCenter,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24.0),

              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  GestureDetector(
                    onTap: () => context.push('/search'),
                    child: Material(
                      color: Theme.of(context).colorScheme.surface,
                      elevation: 4,
                      shadowColor: Colors.black26,
                      borderRadius: BorderRadius.circular(30),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 13,
                        ),
                        child: Row(
                          children: [
                            Icon(
                              Icons.search,
                              color: Theme.of(context).colorScheme.onSurface,
                            ),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                AppLocalizations.of(context)!.search,
                                style: const TextStyle(
                                  color: Colors.grey,
                                  fontSize: 16,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Stack(
                        clipBehavior: Clip.none,
                        children: [
                          ActionChip(
                            onPressed: () => context.push('/trail/filter'),
                            avatar: FaIcon(
                              FontAwesomeIcons.filter,
                              size: 14,
                              color: Theme.of(context).colorScheme.onSurface,
                            ),
                            shape: StadiumBorder(
                              side: BorderSide(color: Colors.transparent),
                            ),
                            label: const Text('Filter'),
                            backgroundColor: Theme.of(
                              context,
                            ).colorScheme.surface,
                            elevation: 4,
                          ),
                          if (activeFilterCount > 0)
                            Positioned(
                              top: -2,
                              right: 3,
                              child: Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 5,
                                  vertical: 1,
                                ),
                                decoration: BoxDecoration(
                                  color: Theme.of(context).primaryColor,
                                  borderRadius: BorderRadius.circular(10),
                                ),
                                child: Text(
                                  '$activeFilterCount',
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 10,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ),
                            ),
                        ],
                      ),
                      const SizedBox(width: 8),
                      ActionChip(
                        onPressed: () => context.push('/trail/sort'),
                        avatar: FaIcon(
                          FontAwesomeIcons.arrowDownShortWide,
                          size: 14,
                          color: Theme.of(context).colorScheme.onSurface,
                        ),
                        shape: StadiumBorder(
                          side: BorderSide(color: Colors.transparent),
                        ),
                        label: Text(AppLocalizations.of(context)!.sort),
                        backgroundColor: Theme.of(context).colorScheme.surface,
                        elevation: 4,
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),

        Positioned(
          bottom: 24,
          left: 0,
          right: 0,
          child: Center(
            child: ScaleTransition(
              scale: _mapButtonScale,
              child: FilledButton.icon(
                onPressed: () {
                  if (_sheetScrollController?.hasClients == true) {
                    _sheetScrollController!.jumpTo(0);
                  }
                  _sheetController.animateTo(
                    sheetMinSize,
                    curve: Curves.easeOut,
                    duration: const Duration(milliseconds: 300),
                  );
                },
                icon: const FaIcon(FontAwesomeIcons.map),
                label: Text(AppLocalizations.of(context)!.map),
              ),
            ),
          ),
        ),

        if (_selectedTrail != null)
          Positioned(
            bottom: 16,
            left: 16,
            right: 16,
            child: Dismissible(
              key: ValueKey(_selectedTrail!.id),
              direction: DismissDirection.down,
              onDismissed: (_) {
                setState(() {
                  _selectedTrail = null;
                  _selectedPolyline = null;
                });
              },
              child: TrailListItem(
                trail: _selectedTrail!,
                onTrailSelect: () {
                  context.push('/trail/${_selectedTrail!.id}');
                },
              ),
            ),
          ),
      ],
    );
  }

  double _getDynamicPadding(double currentSize) {
    const double minPadding = 0.0;
    const double maxPadding = 156.0;

    double startThreshold = sheetMediumsize;
    double endThreshold = sheetMaxSize;

    if (currentSize <= startThreshold) return minPadding;

    if (currentSize >= endThreshold) return maxPadding;

    double percentage =
        (currentSize - startThreshold) / (endThreshold - startThreshold);

    return minPadding + (percentage * (maxPadding - minPadding));
  }

  int _countActiveFilters(TrailFilter current, TrailFilter defaultFilter) {
    int count = 0;
    if (current.category.isNotEmpty) count++;
    if (current.tags.isNotEmpty) count++;
    if (current.difficulty.length != defaultFilter.difficulty.length ||
        !current.difficulty.every(defaultFilter.difficulty.contains)) {
      count++;
    }
    if (current.author != null) count++;
    if (current.public != defaultFilter.public) count++;
    if (current.private != defaultFilter.private) count++;
    if (current.shared != defaultFilter.shared) count++;
    if (current.liked != defaultFilter.liked) count++;
    if (current.distanceMin > 0) count++;
    if (current.distanceMax < current.distanceLimit) count++;
    if (current.elevationGainMin > 0) count++;
    if (current.elevationGainMax < current.elevationGainLimit) count++;
    if (current.elevationLossMin > 0) count++;
    if (current.elevationLossMax < current.elevationLossLimit) count++;
    if (current.startDate != null) count++;
    if (current.endDate != null) count++;
    return count;
  }
}

/// Interim device-location marker (mirrors `WandererMap._buildLocationLayer`)
/// — a simple native puck driven by [foregroundPositionStreamProvider].
/// Replaces the old flutter_map-only `CurrentLocationLayer` widget, which
/// cannot render outside a flutter_map `FlutterMap` widget tree.
class _LocationLayer extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final positionStream = ref.watch(foregroundPositionStreamProvider);
    return StreamBuilder<LocationMarkerPosition?>(
      stream: positionStream,
      builder: (context, snapshot) {
        final position = snapshot.data;
        if (position == null) return const SizedBox.shrink();
        return ml.WidgetLayer(
          markers: [
            ml.Marker(
              point: ml.Geographic(
                lat: position.latitude,
                lon: position.longitude,
              ),
              size: const Size(18, 18),
              child: Container(
                decoration: BoxDecoration(
                  color: Colors.blue,
                  shape: BoxShape.circle,
                  border: Border.all(color: Colors.white, width: 3),
                  boxShadow: [
                    BoxShadow(
                      color: Colors.black.withValues(alpha: .3),
                      blurRadius: 4,
                      offset: const Offset(0, 1),
                    ),
                  ],
                ),
              ),
            ),
          ],
        );
      },
    );
  }
}
