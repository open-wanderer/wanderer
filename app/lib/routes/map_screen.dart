import 'package:flutter/foundation.dart' show kDebugMode;
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_animations/flutter_map_animations.dart';
import 'package:flutter_map_location_marker/flutter_map_location_marker.dart';
import 'package:flutter_map_marker_cluster/flutter_map_marker_cluster.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/map_compass.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/map_camera_provider.dart';
import 'package:wanderer/provider/map_style_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_polyline_provider.dart';
import 'package:wanderer/util/icon_util.dart';
import 'package:skeletonizer/skeletonizer.dart';

class MapScreen extends ConsumerStatefulWidget {
  final LatLng? initialCenter;
  final double? initialZoom;

  const MapScreen({super.key, this.initialCenter, this.initialZoom});

  @override
  ConsumerState<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends ConsumerState<MapScreen>
    with TickerProviderStateMixin {
  late final _animatedMapController = AnimatedMapController(vsync: this);

  TrailSearchResult? _selectedTrail;
  Polyline? _selectedPolyline;

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
      _animatedMapController
          .animateTo(
            dest: newCenter,
            zoom: widget.initialZoom ?? 13.0,
            duration: const Duration(milliseconds: 750),
            curve: Curves.easeInOut,
          )
          .then((_) {
            if (!mounted) return;
            final bounds =
                _animatedMapController.mapController.camera.visibleBounds;
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
    _animatedMapController.dispose();
    _sheetController.removeListener(_onSheetSizeChanged);
    _sheetController.dispose();
    _sheetSize.dispose();
    _mapButtonController.dispose();
    _searchAreaController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final savedCamera = ref.read(mapCameraProvider);
    final styleAsync = ref.watch(mapStyleProvider);
    final isRefreshing = styleAsync.isLoading && styleAsync.hasValue;
    final style = isRefreshing ? null : styleAsync.value;

    ref.listen(mapStyleProvider, (previous, next) {
      if (previous?.value == null && next.value != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) {
            final bounds =
                _animatedMapController.mapController.camera.visibleBounds;
            ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds);
          }
        });
      }
    });

    final filterAsync = ref.watch(trailFilterProvider);
    final activeFilterCount = filterAsync.hasValue
        ? _countActiveFilters(
            filterAsync.value!,
            ref.read(trailFilterProvider.notifier).defaultFilter,
          )
        : 0;

    final searchResultAsync = ref.watch(mapTrailSearchProvider);
    final trails = searchResultAsync.value ?? [];

    final markers = trails.map((trail) {
      return Marker(
        key: ValueKey(trail.id),
        point: LatLng(trail.geo.lat, trail.geo.lng),
        width: 36,
        height: 36,
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
          child: Center(child: getTrailIcon(trail.category)),
        ),
      );
    }).toList();

    return Stack(
      children: [
        FlutterMap(
          key: ObjectKey(style),
          mapController: _animatedMapController.mapController,
          options: MapOptions(
            backgroundColor: Theme.of(context).colorScheme.surface,
            initialCenter:
                widget.initialCenter ??
                savedCamera?.center ??
                const LatLng(0, 0),
            initialZoom: widget.initialZoom ?? savedCamera?.zoom ?? 3,
            interactionOptions: InteractionOptions(
              enableMultiFingerGestureRace: true,
            ),
            maxZoom: 22,
            onMapEvent: (event) {
              if (event is MapEventMoveEnd) {
                ref
                    .read(mapCameraProvider.notifier)
                    .save(event.camera.center, event.camera.zoom);
                const userGestures = {
                  MapEventSource.dragEnd,
                  MapEventSource.flingAnimationController,
                  MapEventSource.multiFingerEnd,
                  MapEventSource.doubleTap,
                  MapEventSource.doubleTapZoomAnimationController,
                  MapEventSource.scrollWheel,
                };
                if (userGestures.contains(event.source)) {
                  _searchAreaController.forward();
                }
              }
            },
            onTap: (tapPosition, point) {
              if (_sheetController.isAttached) {
                _sheetController.animateTo(
                  sheetMinSize,
                  curve: Curves.easeOut,
                  duration: Duration(milliseconds: 300),
                );
              }

              setState(() {
                _selectedTrail = null;
                _selectedPolyline = null;
              });
            },
          ),
          children: [
            if (style != null)
              SizedBox.expand(
                child: VectorTileLayer(
                  tileProviders: style.providers,
                  theme: style.theme,
                  tileOffset: TileOffset.DEFAULT,
                  concurrency: kDebugMode
                      ? 0
                      : VectorTileLayer.defaultConcurrency,
                ),
              ),
            const CurrentLocationLayer(),

            if (_selectedPolyline != null)
              PolylineLayer(polylines: [_selectedPolyline!]),
            Positioned(
              top: 124,
              right: 8,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [const MapCompass(hideIfRotatedNorth: true)],
              ),
            ),
            MarkerClusterLayerWidget(
              options: MarkerClusterLayerOptions(
                maxClusterRadius: 45,
                size: const Size(40, 40),
                alignment: Alignment.center,
                padding: const EdgeInsets.all(50),
                maxZoom: 15,
                markers: markers,
                centerMarkerOnClick: false,
                showPolygon: false,
                onMarkerTap: (marker) {
                  final trailId = (marker.key as ValueKey<String>).value;
                  final trail = trails.firstWhere((t) => t.id == trailId);
                  setState(() {
                    _selectedTrail = trail;
                    _selectedPolyline = null;
                  });

                  ref.read(trailPolylineProvider(trailId).future).then((
                    polyline,
                  ) {
                    if (!mounted || _selectedTrail?.id != trailId) return;
                    if (polyline != null) {
                      _animatedMapController.animatedFitCamera(
                        cameraFit: CameraFit.bounds(
                          bounds: LatLngBounds.fromPoints(polyline.points),
                          padding: EdgeInsets.fromLTRB(40, 56, 40, 248),
                        ),
                        duration: Duration(milliseconds: 750),
                      );
                    }
                    setState(() => _selectedPolyline = polyline);
                  });
                },
                builder: (context, markers) {
                  return Container(
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(20),
                      color: Theme.of(context).primaryColor,
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
                      child: Text(
                        markers.length.toString(),
                        style: const TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
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
                    final bounds = _animatedMapController
                        .mapController
                        .camera
                        .visibleBounds;
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
                child: Skeletonizer(
                  enabled: searchResultAsync.isLoading,
                  child: ListView.builder(
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
