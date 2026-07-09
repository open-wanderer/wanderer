import 'package:flutter/foundation.dart' show kDebugMode;
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_location_marker/flutter_map_location_marker.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/map_style_provider.dart';
import 'package:wanderer/util/map_coordinate_adapter.dart';
import 'package:wanderer/vendor/vector_map_tiles/pm_tile_provider.dart';

class WandererMap extends ConsumerStatefulWidget {
  final Trail trail;
  final MapController? mapController;
  final bool disabled;
  final bool offline;
  final List<Widget>? controls;
  final ml.Geographic? elevationMarkerPosition;
  final EdgeInsets initialCameraFitPadding;

  final bool showTrail;
  final bool showLocation;
  final Waypoint? selectedWaypoint;

  final TapCallback? onTap;
  final Function(MapEvent)? onMapEvent;
  final Function(Waypoint wp)? onWaypointTap;

  const WandererMap({
    super.key,
    required this.trail,
    this.mapController,
    this.onTap,
    this.onWaypointTap,
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
  ConsumerState<WandererMap> createState() => _WandererMapState();
}

class _WandererMapState extends ConsumerState<WandererMap> {
  MultiPmTilesVectorTileProvider? _offlineTileProvider;
  Object? _error;
  ml.LngLatBounds? _bounds;

  @override
  void initState() {
    super.initState();
    _bounds = widget.trail.bounds;
    if (widget.offline) {
      _initOffline();
    }
  }

  Future<void> _initOffline() async {
    try {
      final provider = await MultiPmTilesVectorTileProvider.fromSources(
        widget.trail.pmTiles,
      );
      if (mounted) setState(() => _offlineTileProvider = provider);
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  VectorTileLayer _buildTileLayer(Style style) {
    if (widget.offline) {
      return VectorTileLayer(
        fileCacheTtl: Duration.zero,
        theme: style.theme,
        tileProviders: TileProviders({'protomaps': _offlineTileProvider!}),
        tileOffset: TileOffset.DEFAULT,
        concurrency: kDebugMode ? 0 : VectorTileLayer.defaultConcurrency,
      );
    }
    return VectorTileLayer(
      tileProviders: style.providers,
      theme: style.theme,
      tileOffset: TileOffset.DEFAULT,
      concurrency: kDebugMode ? 0 : VectorTileLayer.defaultConcurrency,
    );
  }

  @override
  Widget build(BuildContext context) {
    final styleAsync = ref.watch(mapStyleProvider);

    if (_error != null) {
      return Center(child: Text(_error.toString()));
    }

    return styleAsync.when(
      skipLoadingOnRefresh: false,
      loading: () => ColoredBox(color: Theme.of(context).colorScheme.surface),
      error: (e, _) => Center(child: Text(e.toString())),
      data: (style) {
        if (widget.offline && _offlineTileProvider == null) {
          return const Center(child: CircularProgressIndicator());
        }
        return FlutterMap(
          key: ObjectKey(style),
          mapController: widget.mapController,
          options: MapOptions(
            backgroundColor: Theme.of(context).colorScheme.surface,
            onTap: widget.onTap,
            onMapEvent: (e) => widget.onMapEvent,
            interactionOptions: widget.disabled
                ? const InteractionOptions(flags: InteractiveFlag.none)
                : const InteractionOptions(enableMultiFingerGestureRace: true),
            initialCameraFit: _bounds != null
                ? CameraFit.bounds(
                    bounds: toLatLngBounds(_bounds!),
                    padding: widget.initialCameraFitPadding,
                  )
                : null,
            initialCenter: toLatLng(
              ml.Geographic(
                lat: widget.trail.lat ?? 0,
                lon: widget.trail.lon ?? 0,
              ),
            ),
            initialZoom: 18,
          ),
          children: [
            _buildTileLayer(style),

            if (widget.trail.expand?.gpx != null && widget.showTrail)
              TrailLayer(
                trail: widget.trail,
                onWaypointTap: widget.onWaypointTap,
                selectedWaypoint: widget.selectedWaypoint,
              ),

            if (widget.showLocation)
              CurrentLocationLayer(
                positionStream: ref.watch(foregroundPositionStreamProvider),
              ),

            if (widget.elevationMarkerPosition != null)
              MarkerLayer(
                markers: [
                  Marker(
                    width: 12,
                    height: 12,
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
                    point: toLatLng(widget.elevationMarkerPosition!),
                  ),
                ],
              ),

            Align(
              alignment: Alignment.topRight,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: widget.controls!,
              ),
            ),
          ],
        );
      },
    );
  }
}
