import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_marker_cluster/flutter_map_marker_cluster.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/map/map_compass.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/util/polyline_util.dart';
import 'package:flutter_map_animations/flutter_map_animations.dart';

class MapScreen extends ConsumerStatefulWidget {
  const MapScreen({super.key});

  @override
  ConsumerState<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends ConsumerState<MapScreen>
    with TickerProviderStateMixin {
  Style? style;
  late final _animatedMapController = AnimatedMapController(vsync: this);

  TrailSearchResult? _selectedTrail;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();
  late final ValueNotifier<double> _sheetSize;

  final sheetMinSize = 0.2;
  final sheetMediumsize = 0.7;
  final sheetMaxSize = 1.0;

  @override
  void initState() {
    super.initState();
    _initializeStyle();
    _sheetController.addListener(_onSheetSizeChanged);
    _sheetSize = ValueNotifier<double>(sheetMinSize);
  }

  Future<void> _initializeStyle() async {
    final originalStyle = await StyleReader.asset(
      'assets/styles/ofm.json',
    ).read();

    if (mounted) {
      setState(() {
        style = originalStyle;
      });

      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          final bounds =
              _animatedMapController.mapController.camera.visibleBounds;
          ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds);
        }
      });
    }
  }

  void _onSheetSizeChanged() {
    _sheetSize.value = _sheetController.size;
  }

  @override
  void dispose() {
    _animatedMapController.dispose();
    _sheetController.removeListener(_onSheetSizeChanged);
    _sheetController.dispose();
    _sheetSize.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (style == null) {
      return const SizedBox.expand(
        child: Center(child: CircularProgressIndicator()),
      );
    }

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
          child: Center(child: _getTrailIcon(trail)),
        ),
      );
    }).toList();

    return Stack(
      children: [
        FlutterMap(
          mapController: _animatedMapController.mapController,
          options: MapOptions(
            initialCenter: const LatLng(0, 0),
            initialZoom: 3,
            maxZoom: 22,
            onMapEvent: (event) {
              if (event is MapEventMoveEnd) {
                final bounds = event.camera.visibleBounds;
                ref
                    .read(mapTrailSearchProvider.notifier)
                    .searchInBounds(bounds);
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
              });
            },
          ),
          children: [
            SizedBox.expand(
              child: VectorTileLayer(
                tileProviders: style!.providers,
                theme: style!.theme,
                tileOffset: TileOffset.DEFAULT,
              ),
            ),
            if (_selectedTrail?.polyline != null)
              PolylineLayer(
                polylines: [PolylineTools.decode(_selectedTrail!.polyline!)],
              ),
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
                  if (trail.polyline != null) {
                    _animatedMapController.animatedFitCamera(
                      cameraFit: CameraFit.bounds(
                        bounds: LatLngBounds.fromPoints(
                          PolylineTools.decode(trail.polyline!).points,
                        ),
                        padding: EdgeInsets.fromLTRB(40, 56, 40, 248),
                      ),
                      duration: Duration(milliseconds: 750),
                    );
                  }

                  setState(() {
                    _selectedTrail = trail;
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

        if (_selectedTrail == null && trails.isNotEmpty)
          DraggableScrollableSheet(
            controller: _sheetController,
            initialChildSize: sheetMinSize,
            minChildSize: sheetMinSize,
            maxChildSize: sheetMaxSize,
            snap: true,
            snapSizes: [sheetMinSize, sheetMediumsize, sheetMaxSize],
            builder: (context, scrollController) {
              return Container(
                decoration: BoxDecoration(
                  color: Theme.of(context).canvasColor,
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(16),
                  ),
                  boxShadow: [
                    BoxShadow(
                      color: Colors.black.withValues(alpha: 0.15),
                      blurRadius: 10,
                      offset: const Offset(0, -2),
                    ),
                  ],
                ),
                child: ValueListenableBuilder(
                  valueListenable: _sheetSize,
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
                                        "${trails.length}${trails.length == 500 ? '+' : ''} ${AppLocalizations.of(context)!.trail(trails.length)}",
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
                  builder: (context, size, child) {
                    EdgeInsets e = EdgeInsets.zero;
                    if (size >= sheetMediumsize) {
                      e = EdgeInsets.fromLTRB(
                        0,
                        _getDynamicPadding(size),
                        0,
                        0,
                      );
                    }
                    return Padding(padding: e, child: child);
                  },
                ),
              );
            },
          ),

        SafeArea(
          child: Align(
            alignment: Alignment.topCenter,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24.0),

              child: Row(
                children: [
                  Expanded(
                    child: GestureDetector(
                      onTap: () => context.push('/search'),
                      child: Material(
                        color: Colors.white,
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
                              const Icon(Icons.search, color: Colors.black),
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
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    onPressed: () => context.push('/trail/filter'),
                    icon: const FaIcon(FontAwesomeIcons.filter, size: 18),
                    style: IconButton.styleFrom(
                      backgroundColor: Theme.of(context).canvasColor,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),

        if (searchResultAsync.isLoading)
          Positioned(
            top: 16,
            right: 16,
            child: Card(
              elevation: 4,
              shape: const CircleBorder(),
              child: Padding(
                padding: const EdgeInsets.all(8.0),
                child: SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    valueColor: AlwaysStoppedAnimation<Color>(
                      Theme.of(context).primaryColor,
                    ),
                  ),
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
    const double maxPadding = 128.0;

    double startThreshold = sheetMediumsize;
    double endThreshold = sheetMaxSize;

    if (currentSize <= startThreshold) return minPadding;

    if (currentSize >= endThreshold) return maxPadding;

    double percentage =
        (currentSize - startThreshold) / (endThreshold - startThreshold);

    return minPadding + (percentage * (maxPadding - minPadding));
  }

  Icon _getTrailIcon(TrailSearchResult trail) {
    final category = trail.category;

    switch (category) {
      case "Hiking":
        return FaIcon(
          FontAwesomeIcons.personHiking,
          color: Colors.white,
          size: 18,
        );
      case "Biking":
        return FaIcon(FontAwesomeIcons.bicycle, color: Colors.white, size: 18);
      case "Walking":
        return Icon(Icons.nordic_walking, color: Colors.white, size: 18);
      case "Climbing":
        return Icon(Icons.landscape, color: Colors.white, size: 18);
      case "Skiing":
        return FaIcon(
          FontAwesomeIcons.personSkiing,
          color: Colors.white,
          size: 18,
        );
      case "Canoeing":
        return Icon(Icons.kayaking, color: Colors.white, size: 18);
      default:
        return FaIcon(FontAwesomeIcons.route, color: Colors.white, size: 18);
    }
  }
}
