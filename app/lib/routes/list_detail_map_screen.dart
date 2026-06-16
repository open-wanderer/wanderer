import 'package:flutter/foundation.dart' show kDebugMode;
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_animations/flutter_map_animations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/map_style_provider.dart';
import 'package:wanderer/provider/trail/list_provider.dart';
import 'package:wanderer/util/icon_util.dart';
import 'package:wanderer/util/polyline_util.dart';

class ListDetailMapScreen extends ConsumerStatefulWidget {
  final String id;
  const ListDetailMapScreen({super.key, required this.id});

  @override
  ConsumerState<ListDetailMapScreen> createState() =>
      _ListDetailMapScreenState();
}

class _ListDetailMapScreenState extends ConsumerState<ListDetailMapScreen>
    with TickerProviderStateMixin {
  late final _animatedMapController = AnimatedMapController(vsync: this);
  Trail? _selectedTrail;
  LatLngBounds? _combinedBoundsCache;

  @override
  void dispose() {
    _animatedMapController.dispose();
    super.dispose();
  }

  void _deselect() {
    setState(() => _selectedTrail = null);
    final bounds = _combinedBoundsCache;
    if (bounds != null) {
      _animatedMapController.animatedFitCamera(
        cameraFit: CameraFit.bounds(
          bounds: bounds,
          padding: const EdgeInsets.all(40),
        ),
        duration: const Duration(milliseconds: 750),
      );
    }
  }

  void _onMarkerTap(Trail trail) {
    setState(() => _selectedTrail = trail);

    final bounds = LatLngBounds(
      LatLng(trail.minLat, trail.minLon),
      LatLng(trail.maxLat, trail.maxLon),
    );

    _animatedMapController.animatedFitCamera(
      cameraFit: CameraFit.bounds(
        bounds: bounds,
        padding: const EdgeInsets.fromLTRB(40, 56, 40, 120),
      ),
      duration: const Duration(milliseconds: 750),
    );
  }

  @override
  Widget build(BuildContext context) {
    final listAsync = ref.watch(listProvider(widget.id));
    final styleAsync = ref.watch(mapStyleProvider);

    return listAsync.when(
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (err, stack) => Scaffold(
        body: WandererError(err: err, stack: stack),
      ),
      data: (list) {
        final trails = list.expand?.trails ?? [];

        final polylines = trails
            .where((t) => t.polyline != null && t.polyline!.isNotEmpty)
            .map((t) => PolylineTools.decode(t.polyline!))
            .toList();

        final markers = trails
            .where((t) => t.lat != null && t.lon != null)
            .map(
              (t) => Marker(
                key: ValueKey(t.id),
                point: LatLng(t.lat!, t.lon!),
                width: 36,
                height: 36,
                child: GestureDetector(
                  onTap: () => _onMarkerTap(t),
                  child: Container(
                    decoration: BoxDecoration(
                      color: _selectedTrail?.id == t.id
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
                      child: getTrailIcon(
                        t.summaryCategory,
                        color: _selectedTrail?.id == t.id
                            ? Theme.of(context).primaryColor
                            : Colors.white,
                      ),
                    ),
                  ),
                ),
              ),
            )
            .toList();

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
          body: styleAsync.when(
            loading: () =>
                ColoredBox(color: Theme.of(context).colorScheme.surface),
            error: (e, _) => Center(child: Text(e.toString())),
            data: (style) => Stack(
              children: [
                FlutterMap(
                  key: ObjectKey(style),
                  mapController: _animatedMapController.mapController,
                  options: MapOptions(
                    backgroundColor: Theme.of(context).colorScheme.surface,
                    interactionOptions: const InteractionOptions(
                      enableMultiFingerGestureRace: true,
                      flags: InteractiveFlag.all & ~InteractiveFlag.rotate,
                    ),
                    initialCameraFit: combinedBounds != null
                        ? CameraFit.bounds(
                            bounds: combinedBounds,
                            padding: const EdgeInsets.all(40),
                          )
                        : null,
                    initialCenter: const LatLng(0, 0),
                    initialZoom: 3,
                    onTap: (_, _) => _deselect(),
                  ),
                  children: [
                    VectorTileLayer(
                      tileProviders: style.providers,
                      theme: style.theme,
                      tileOffset: TileOffset.DEFAULT,
                      concurrency: kDebugMode
                          ? 0
                          : VectorTileLayer.defaultConcurrency,
                    ),
                    if (polylines.isNotEmpty)
                      PolylineLayer(polylines: polylines),
                    if (markers.isNotEmpty) MarkerLayer(markers: markers),
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
          ),
        );
      },
    );
  }

  LatLngBounds? _combinedBounds(List<Trail> trails) {
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

    return LatLngBounds(LatLng(minLat, minLon), LatLng(maxLat, maxLon));
  }
}
