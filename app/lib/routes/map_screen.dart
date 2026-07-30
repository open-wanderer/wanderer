import 'dart:async';
import 'dart:convert';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/trail_collection_map.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/base/wanderer_offline_state.dart';
import 'package:wanderer/components/map/cluster_layer.dart';
import 'package:wanderer/components/map/location_marker_layer.dart';
import 'package:wanderer/components/map/trail_layer.dart' show kTrailRouteColor;
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/map_camera_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/map_cluster_search_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_polyline_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';

/// Zoom used when centering on a specific point (GPS fix or saved home
/// location).
const double _kPointZoom = 13.0;

/// World-view zoom for the (0,0) fallback when no other location is known.
const double _kWorldZoom = 3.0;

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

  /// A GPS fix resolved after first build, used to re-center the camera once
  /// available. Only takes effect when there is neither an explicit
  /// [MapScreen.initialCenter] nor a [mapCameraProvider] save to defer to.
  ml.Geographic? _resolvedGpsCenter;
  StreamSubscription<LocationMarkerPosition?>? _gpsSub;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();
  late final ValueNotifier<double> _sheetSize;
  late final AnimationController _mapButtonController;
  late final Animation<double> _mapButtonScale;
  late final AnimationController _searchAreaController;
  late final Animation<double> _searchAreaScale;
  ScrollController? _sheetScrollController;

  double sheetMinSize = 0.5;
  final sheetMediumsize = 0.5;
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

    // Only chase a GPS fix when neither an explicit route target nor a
    // saved camera already decides the initial focus.
    if (widget.initialCenter == null && ref.read(mapCameraProvider) == null) {
      _gpsSub = ref.read(foregroundPositionStreamProvider).listen((position) {
        if (position == null || _resolvedGpsCenter != null) return;
        _gpsSub?.cancel();
        final center = ml.Geographic(
          lat: position.latitude,
          lon: position.longitude,
        );
        setState(() => _resolvedGpsCenter = center);
        _controller
            ?.animateCamera(
              center: center,
              zoom: _kPointZoom,
              nativeDuration: const Duration(milliseconds: 750),
            )
            .then((_) {
              if (!mounted) return;
              final controller = _controller;
              if (controller == null) return;
              final bounds = controller.getVisibleRegion();
              final zoom = controller.getCamera().zoom;
              ref
                  .read(mapClusterSearchProvider.notifier)
                  .searchInBounds(bounds, zoom);
              ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds);
            });
      }, onError: (_) {});
    }
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
    _gpsSub?.cancel();
    _sheetController.removeListener(_onSheetSizeChanged);
    _sheetController.dispose();
    _sheetSize.dispose();
    _mapButtonController.dispose();
    _searchAreaController.dispose();
    super.dispose();
  }

  /// Fetch-then-fit trail selection: selects [trailId] immediately (metadata
  /// resolved from the parallel `mapTrailSearchProvider` results via
  /// `firstWhereOrNull`, since the id is untrusted input), then fits the
  /// camera to the trail's polyline once it resolves.
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

  /// Re-probes connectivity, and if it's back, invalidates the keepAlive
  /// providers `TrailCollectionMap` depends on (they cache their offline
  /// error otherwise) and re-runs the last bounds search so the sheet
  /// repopulates without an app restart.
  Future<void> _retryOnline() async {
    final online = await ref.read(onlineStatusProvider.notifier).refresh();
    if (!online) return;

    ref.invalidate(mapStyleSourcesProvider);
    ref.invalidate(mapStyleJsonProvider);

    // TrailCollectionMap (and its native controller) was torn out of the
    // tree while offline, so `_controller` is a stale/disposed reference —
    // calling native methods on it here would hit a disposed platform
    // channel. Clear it and reset the initial-search flag so the fresh
    // onMapCreated/onStyleLoaded cycle (fired when the map remounts now
    // that isOnline flips true) re-runs the bounds search itself.
    _controller = null;
    _initialSearchDone = false;
  }

  /// Offline stand-in for the trail-results list inside the draggable sheet.
  ///
  /// MUST attach [scrollController] and stay scrollable: the sheet drives its
  /// drag from the attached scroll position, so a non-scrollable child (which
  /// is what `AsyncLoader` substitutes on error — a bare `WandererError`) leaves
  /// the sheet frozen as well as showing a raw exception. Carries no retry CTA
  /// because the map takeover behind it already owns that affordance.
  Widget _buildSheetOfflineState(ScrollController scrollController) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final muted = theme.colorScheme.onSurface.withValues(alpha: 0.5);

    return ListView(
      controller: scrollController,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: EdgeInsets.fromLTRB(
        16,
        8,
        16,
        kBottomNavigationBarHeight + 16 + 32,
      ),
      children: [
        Center(
          child: Container(
            width: 30,
            height: 5,
            margin: const EdgeInsets.only(top: 4),
            decoration: BoxDecoration(
              color: theme.colorScheme.outline,
              borderRadius: BorderRadius.circular(10),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: 64),
          child: Column(
            children: [
              FaIcon(FontAwesomeIcons.linkSlash, size: 32, color: muted),
              const SizedBox(height: 16),
              Text(
                l10n.offline_title,
                style: theme.textTheme.labelLarge!.copyWith(color: muted),
              ),
              const SizedBox(height: 8),
              Text(
                l10n.offline_trail_search_body,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodySmall?.copyWith(color: muted),
              ),
            ],
          ),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final savedCamera = ref.read(mapCameraProvider);
    final isOnline = ref.watch(onlineStatusProvider);

    sheetMinSize =
        (56 + kBottomNavigationBarHeight + 48) /
        MediaQuery.of(context).size.height;

    // Initial-focus fallback chain, lowest priority first: (0,0) world view,
    // then the user's saved home location, then a resolved GPS fix. Each of
    // these only matters when there's neither an explicit route-provided
    // center nor a previously saved camera position (both handled below).
    ml.Geographic fallbackCenter = const ml.Geographic(lat: 0, lon: 0);
    double fallbackZoom = _kWorldZoom;
    final settingsLocation = ref.watch(settingsProvider)?.location;
    if (settingsLocation != null) {
      fallbackCenter = ml.Geographic(
        lat: settingsLocation.lat,
        lon: settingsLocation.lon,
      );
      fallbackZoom = _kPointZoom;
    }
    if (_resolvedGpsCenter != null) {
      fallbackCenter = _resolvedGpsCenter!;
      fallbackZoom = _kPointZoom;
    }

    // Swap the native cluster source's data in place on every new
    // mapClusterSearchProvider result — never remove and re-add the source.
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

    // mapTrailSearchProvider still powers the bottom-sheet TrailCard list and
    // the tapped-trail metadata lookup — the cluster endpoint's
    // attributesToRetrieve only includes id/_geo/bounding_box_diagonal.
    final searchResultAsync = ref.watch(mapTrailSearchProvider);
    final trails = searchResultAsync.value ?? [];
    final allCategories = ref.watch(categoryProvider).value ?? [];
    final allSubcategories = ref.watch(subcategoryProvider);

    // mapClusterSearchProvider drives the native cluster circle/count layers
    // (cluster_layer.dart) AND the unclustered category-icon WidgetLayer
    // markers below.
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
      // is_large not filtered here: full-polyline rendering for is_large
      // trails isn't implemented yet, so every unclustered point still
      // renders as a category-icon marker.

      final trailId = properties['id'];
      if (trailId is! String) continue;
      // The feature's id is untrusted input, so use firstWhereOrNull, not
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
        if (!isOnline)
          Container(
            color: Theme.of(context).colorScheme.surface,
            child: WandererOfflineState(
              title: AppLocalizations.of(context)!.offline_title,
              body: AppLocalizations.of(context)!.offline_map_body,
              retryLabel: AppLocalizations.of(context)!.offline_try_again,
              onRetry: _retryOnline,
            ),
          )
        else
          TrailCollectionMap(
            initCenter:
                widget.initialCenter ?? savedCamera?.center ?? fallbackCenter,
            initZoom: widget.initialZoom ?? savedCamera?.zoom ?? fallbackZoom,
            onMapCreated: (controller) => _controller = controller,
            onStyleLoaded: (style) async {
              // Fail soft on a malformed/oversized cluster response — never
              // crash the map.
              try {
                final data =
                    ref.read(mapClusterSearchProvider).value ??
                    const {'type': 'FeatureCollection', 'features': <Object>[]};
                await addClusterLayers(style, jsonEncode(data));
              } catch (e) {
                debugPrint('map_screen: failed to add cluster layers — $e');
              }

              // Initial-load search trigger — fires once per widget lifetime,
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

                // Only the native `clusters` circle layer needs
                // featuresAtPoint — unclustered points are WidgetLayer markers
                // that handle their own onTap directly.
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
                // deselect and collapse the sheet.
                if (_sheetController.isAttached) {
                  _sheetController.animateTo(
                    sheetMediumsize,
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
              const LocationMarkerLayer(),
              Positioned(
                top: 124,
                right: 8,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: const [ml.MapCompass(hideIfRotatedNorth: true)],
                ),
              ),
              const ml.MapScalebar(
                alignment: Alignment.topLeft,
                padding: EdgeInsets.only(left: 24, top: 112),
              ),
              WandererAttribution(
                alignment: Alignment.bottomLeft,
                padding: EdgeInsets.only(
                  left: 10,
                  bottom: kBottomNavigationBarHeight + 45,
                ),
              ),
            ],
          ),

        // Rides up with the sheet as it drags from min to medium size, then
        // holds still at that height and gets covered as the sheet expands
        // past medium. Hidden while offline — neither the FAB nor the
        // search-this-area button below it can do anything useful without
        // connectivity.
        if (_selectedTrail == null && isOnline)
          ValueListenableBuilder<double>(
            valueListenable: _sheetSize,
            builder: (context, sheetSize, child) {
              final clampedSize = sheetSize < sheetMediumsize
                  ? sheetSize
                  : sheetMediumsize;
              return Positioned(
                bottom: MediaQuery.of(context).size.height * clampedSize + 12,
                right: 16,
                child: child!,
              );
            },
            child: StreamBuilder<LocationMarkerPosition?>(
              stream: ref.watch(foregroundPositionStreamProvider),
              builder: (context, snapshot) {
                final position = snapshot.data;
                return Material(
                  color: Theme.of(context).colorScheme.surface,
                  elevation: 4,
                  shadowColor: Colors.black26,
                  shape: const CircleBorder(),
                  child: IconButton(
                    icon: const FaIcon(
                      FontAwesomeIcons.locationCrosshairs,
                      size: 18,
                    ),
                    tooltip: AppLocalizations.of(
                      context,
                    )!.center_on_my_location,
                    onPressed: position == null
                        ? null
                        : () {
                            _controller?.animateCamera(
                              center: ml.Geographic(
                                lat: position.latitude,
                                lon: position.longitude,
                              ),
                              zoom: _kPointZoom,
                              nativeDuration: const Duration(milliseconds: 750),
                            );
                          },
                  ),
                );
              },
            ),
          ),

        if (_selectedTrail == null && isOnline)
          ValueListenableBuilder(
            valueListenable: _sheetSize,
            builder: (context, sheetSize, child) {
              final clampedSize = sheetSize < sheetMediumsize
                  ? sheetSize
                  : sheetMediumsize;
              return Positioned(
                bottom: MediaQuery.of(context).size.height * clampedSize + 12,
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
                      icon: const FaIcon(
                        FontAwesomeIcons.mapLocationDot,
                        size: 14,
                      ),
                      label: Text(
                        AppLocalizations.of(context)!.search_this_area,
                      ),
                    ),
                  ),
                ),
              );
            },
          ),

        Opacity(
          key: const ValueKey('trail_sheet'),
          opacity: _selectedTrail == null ? 1 : 0,
          child: DraggableScrollableSheet(
            controller: _sheetController,
            initialChildSize: sheetMediumsize,
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
                  child: child,
                ),
                child: (searchResultAsync.hasError && !isOnline)
                    ? _buildSheetOfflineState(scrollController)
                    : AsyncLoader(
                        asyncValue: searchResultAsync,
                        mockData: List.generate(
                          5,
                          (_) => TrailSearchResult.mock(),
                        ),
                        builder: (trails) => ListView(
                          padding: EdgeInsets.fromLTRB(
                            16,
                            8,
                            16,
                            kBottomNavigationBarHeight + 16 + 32,
                          ),
                          controller: scrollController,
                          children: [
                            ValueListenableBuilder<double>(
                              valueListenable: _sheetSize,
                              builder: (context, size, child) {
                                final opacity = _sheetHeaderOpacity(size);

                                return Column(
                                  children: [
                                    if (opacity > 0.0)
                                      Opacity(
                                        opacity: opacity,
                                        child: Column(
                                          children: [
                                            Center(
                                              child: Container(
                                                width: 30,
                                                height: 5,
                                                margin: const EdgeInsets.only(
                                                  top: 4,
                                                ),
                                                decoration: BoxDecoration(
                                                  color: Theme.of(
                                                    context,
                                                  ).colorScheme.outline,
                                                  borderRadius:
                                                      BorderRadius.circular(10),
                                                ),
                                              ),
                                            ),

                                            Padding(
                                              padding: const EdgeInsets.all(
                                                8.0,
                                              ),
                                              child: Center(
                                                child: Opacity(
                                                  opacity: trails.isNotEmpty
                                                      ? 1
                                                      : (1 - size / sheetMediumsize)
                                                            .clamp(0, 1),
                                                  child: Text(
                                                    "${trails.length}${trails.length == 100 ? '+' : ''} ${AppLocalizations.of(context)!.trail(trails.length)}",
                                                    style: Theme.of(
                                                      context,
                                                    ).textTheme.labelLarge,
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ],
                                        ),
                                      ),
                                    SizedBox(height: _getDynamicPadding(size)),
                                  ],
                                );
                              },
                            ),
                            if (trails.isNotEmpty) ...{
                              ...trails.map(
                                (t) => TrailCard(
                                  trail: t,
                                  onTrailSelect: () =>
                                      context.push("/trail/${t.id}", extra: t),
                                ),
                              ),
                            } else ...{
                              Padding(
                                padding: EdgeInsetsGeometry.only(top: 64),
                                child: Column(
                                  children: [
                                    SvgPicture.asset(
                                      "assets/svgs/empty_state_search_${Theme.of(context).brightness.name}.svg",
                                      semanticsLabel:
                                          'wanderer comment empty state',
                                      height: 120,
                                    ),
                                    const SizedBox(height: 16),
                                    Text(
                                      AppLocalizations.of(
                                        context,
                                      )!.no_trails_found,
                                      style: Theme.of(context)
                                          .textTheme
                                          .labelLarge!
                                          .copyWith(
                                            color: Theme.of(context)
                                                .colorScheme
                                                .onSurface
                                                .withValues(alpha: 0.5),
                                          ),
                                    ),
                                  ],
                                ),
                              ),
                            },
                          ],
                        ),
                      ),
              );
            },
          ),
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
                            label: Text(AppLocalizations.of(context)!.filter),
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
          bottom: kBottomNavigationBarHeight + 64,
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
            bottom: kBottomNavigationBarHeight + 16 + 45,
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

  /// Fades the sheet's drag handle and trail-count header out as the sheet
  /// is dragged open past [sheetMediumsize], fully gone by max size.
  double _sheetHeaderOpacity(double currentSize) {
    if (currentSize <= sheetMediumsize) return 1.0;

    final opacity =
        1.0 - ((currentSize - sheetMediumsize) / (1 - sheetMediumsize));
    return opacity.clamp(0.0, 1.0);
  }

  double _getDynamicPadding(double currentSize) {
    const double minPadding = 0.0;
    const double maxTopPadding = 156.0;
    const double maxBottomPadding = 64.0;

    double startThreshold = sheetMediumsize;
    double endTopThreshold = sheetMaxSize;
    double endBottomThreshold = sheetMinSize;

    if (currentSize >= endTopThreshold) return maxTopPadding;
    if (currentSize <= endBottomThreshold) return maxBottomPadding;

    if (currentSize <= startThreshold) {
      double percentage =
          (startThreshold - currentSize) /
          (startThreshold - endBottomThreshold);
      return minPadding + (percentage * (maxBottomPadding - minPadding));
    } else {
      double percentage =
          (currentSize - startThreshold) / (endTopThreshold - startThreshold);

      return minPadding + (percentage * (maxTopPadding - minPadding));
    }
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
