import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_marker_cluster/flutter_map_marker_cluster.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

class MapScreen extends ConsumerStatefulWidget {
  const MapScreen({super.key});

  @override
  ConsumerState<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends ConsumerState<MapScreen> {
  Style? style;
  final MapController _mapController = MapController();
  TrailSearchResult? _selectedTrail;

  @override
  void initState() {
    super.initState();
    _initializeStyle();
  }

  Future<void> _initializeStyle() async {
    final originalStyle = await StyleReader.asset(
      'assets/styles/ofm.json',
    ).read();

    if (mounted) {
      setState(() {
        style = originalStyle;
      });

      // Query initial trails after layout is ready
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          final bounds = _mapController.camera.visibleBounds;
          ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds);
        }
      });
    }
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
          mapController: _mapController,
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
            MarkerClusterLayerWidget(
              options: MarkerClusterLayerOptions(
                maxClusterRadius: 45,
                size: const Size(40, 40),
                alignment: Alignment.center,
                padding: const EdgeInsets.all(50),
                maxZoom: 15,
                markers: markers,
                onMarkerTap: (marker) {
                  final trailId = (marker.key as ValueKey<String>).value;
                  final trail = trails.firstWhere((t) => t.id == trailId);
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

        SafeArea(
          child: Align(
            alignment: Alignment.topCenter,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24.0),

              child: Material(
                color: Colors.white,
                elevation: 4,
                shadowColor: Colors.black26,
                borderRadius: BorderRadius.circular(30),
                clipBehavior: Clip.antiAlias,

                child: WandererSearchBar(
                  onChanged: (value) {
                    ref
                        .read(trailFilterProvider.notifier)
                        .updateFilter((f) => f.copyWith(q: value));
                  },
                  decoration: InputDecoration(
                    hintText: AppLocalizations.of(context)!.search_trails,
                    hintStyle: const TextStyle(color: Colors.grey),
                    prefixIcon: const Icon(Icons.search, color: Colors.black),
                    suffixIcon: IconButton(
                      onPressed: () => context.push('/trail/filter'),
                      iconSize: 20,
                      icon: const FaIcon(FontAwesomeIcons.filter),
                    ),
                    border: InputBorder.none,
                    contentPadding: const EdgeInsets.symmetric(vertical: 15),
                  ),
                ),
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
