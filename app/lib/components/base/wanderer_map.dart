import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_map_location_marker/flutter_map_location_marker.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/vendor/vector_map_tiles/pm_tile_provider.dart';
import 'package:vector_tile_renderer/vector_tile_renderer.dart' as vtr;

class WandererMap extends ConsumerStatefulWidget {
  final Trail trail;
  final MapController? mapController;
  final bool disabled;
  final bool offline;
  final List<Widget>? controls;
  final LatLng? elevationMarkerPosition;
  final EdgeInsets initialCameraFitPadding;

  final bool showTrail;
  final bool showLocation;

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
    this.elevationMarkerPosition,
    this.initialCameraFitPadding = const EdgeInsets.all(40),
  });

  @override
  ConsumerState<WandererMap> createState() => _WandererMapState();
}

class _WandererMapState extends ConsumerState<WandererMap> {
  Style? _style;
  PmTilesVectorTileProvider? _offlineTileProvider;
  Object? _error;
  LatLngBounds? _bounds;

  @override
  void initState() {
    super.initState();
    _bounds = widget.trail.expand?.gpx?.getBounds();
    _initStyle();
    if (widget.offline) {
      _initOffline();
    }
  }

  Future<void> _initStyle() async {
    final brightness =
        WidgetsBinding.instance.platformDispatcher.platformBrightness;
    final asset = brightness == Brightness.dark
        ? vtr.wandererDarkTheme()
        : vtr.wandererLightTheme();
    final style = await StyleReader.map(
      asset,
      apiKey: const String.fromEnvironment(
        'PROTOMAPS_API_KEY',
        defaultValue: '',
      ),
    ).read();
    if (mounted) {
      setState(() => _style = style);
    }
  }

  Future<void> _initOffline() async {
    try {
      final provider = await PmTilesVectorTileProvider.fromSource(
        widget.trail.pmTiles[0],
      );
      if (mounted) setState(() => _offlineTileProvider = provider);
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  bool get _ready =>
      widget.offline ? _offlineTileProvider != null : _style != null;

  VectorTileLayer _buildTileLayer() {
    if (widget.offline) {
      return VectorTileLayer(
        fileCacheTtl: Duration.zero,
        theme: _style!.theme,
        tileProviders: TileProviders({'protomaps': _offlineTileProvider!}),
        tileOffset: TileOffset.DEFAULT,
      );
    }
    return VectorTileLayer(
      tileProviders: _style!.providers,
      theme: _style!.theme,
      tileOffset: TileOffset.DEFAULT,
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_error != null) {
      return Center(child: Text(_error.toString()));
    }
    if (!_ready) {
      return const Center(child: CircularProgressIndicator());
    }

    return FlutterMap(
      mapController: widget.mapController,
      options: MapOptions(
        onTap: widget.onTap,
        onMapEvent: (e) => widget.onMapEvent,
        interactionOptions: widget.disabled
            ? const InteractionOptions(flags: InteractiveFlag.none)
            : const InteractionOptions(),
        initialCameraFit: _bounds != null
            ? CameraFit.bounds(
                bounds: _bounds!,
                padding: widget.initialCameraFitPadding,
              )
            : null,
        initialCenter: LatLng(widget.trail.lat ?? 0, widget.trail.lon ?? 0),
        initialZoom: 18,
      ),
      children: [
        _buildTileLayer(),

        if (widget.trail.expand?.gpx != null && widget.showTrail)
          TrailLayer(trail: widget.trail, onWaypointTap: widget.onWaypointTap),

        if (widget.showLocation) const CurrentLocationLayer(),

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
                point: widget.elevationMarkerPosition!,
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
  }
}
